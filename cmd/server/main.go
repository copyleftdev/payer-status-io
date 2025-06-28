package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"payer-status-io/internal/config"
	"payer-status-io/internal/hub"
	"payer-status-io/internal/metrics"
	"payer-status-io/internal/prober"
	"payer-status-io/internal/scheduler"
)

const (
	defaultConfigPath = "./docs/payer_status.yaml"
	defaultWSPort     = "8080"
	defaultMetricsPort = "9090"
	defaultLogLevel   = "info"
	workerPoolSize    = 50
	taskChannelSize   = 1000
	probeTimeout      = 10 * time.Second
)

func main() {
	// Initialize logger
	logger := initLogger()
	defer logger.Sync()

	logger.Info("Starting Payer Status WebSocket Health Monitor")

	// Create cancellable context for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), 
		os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	// Load configuration
	configPath := getEnv("CONFIG_PATH", defaultConfigPath)
	configLoader := config.NewLoader(configPath, logger)
	cfg := configLoader.MustLoad()

	// Initialize core components
	metricsCollector := metrics.New(logger)
	wsHub := hub.New(logger)
	taskScheduler := scheduler.New(logger, taskChannelSize)
	httpProber := prober.New(logger, probeTimeout)

	// Load initial configuration into scheduler
	taskScheduler.LoadConfig(cfg)

	// Set up configuration hot-reload
	configLoader.OnConfigChange(func(newCfg *config.Config) {
		logger.Info("Configuration changed, reloading scheduler")
		taskScheduler.LoadConfig(newCfg)
		metricsCollector.RecordConfigReload(true)
	})
	configLoader.WatchForChanges(ctx)

	// Start worker pool for probe execution
	startWorkerPool(ctx, logger, taskScheduler, httpProber, wsHub, metricsCollector)

	// Create HTTP servers
	wsServer := createWebSocketServer(wsHub, configLoader, logger)
	metricsServer := createMetricsServer(metricsCollector, logger)

	// Start all services using errgroup for coordinated shutdown
	g, gCtx := errgroup.WithContext(ctx)

	// Start WebSocket hub
	g.Go(func() error {
		return wsHub.Run(gCtx)
	})

	// Start task scheduler
	g.Go(func() error {
		return taskScheduler.Start(gCtx)
	})

	// Start WebSocket server
	g.Go(func() error {
		logger.Info("Starting WebSocket server", zap.String("port", getEnv("WS_PORT", defaultWSPort)))
		if err := wsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("WebSocket server failed: %w", err)
		}
		return nil
	})

	// Start metrics server
	g.Go(func() error {
		logger.Info("Starting metrics server", zap.String("port", getEnv("METRICS_PORT", defaultMetricsPort)))
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("metrics server failed: %w", err)
		}
		return nil
	})

	// Graceful shutdown handler
	g.Go(func() error {
		<-gCtx.Done()
		logger.Info("Initiating graceful shutdown")

		// Create shutdown context with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		// Shutdown servers
		var wg sync.WaitGroup
		
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := wsServer.Shutdown(shutdownCtx); err != nil {
				logger.Error("WebSocket server shutdown error", zap.Error(err))
			} else {
				logger.Info("WebSocket server shutdown complete")
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := metricsServer.Shutdown(shutdownCtx); err != nil {
				logger.Error("Metrics server shutdown error", zap.Error(err))
			} else {
				logger.Info("Metrics server shutdown complete")
			}
		}()

		// Close HTTP clients
		wg.Add(1)
		go func() {
			defer wg.Done()
			httpProber.Close()
			logger.Info("HTTP prober cleanup complete")
		}()

		wg.Wait()
		logger.Info("Graceful shutdown complete")
		return nil
	})

	// Wait for all goroutines to complete
	if err := g.Wait(); err != nil {
		logger.Error("Application error", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("Payer Status Monitor stopped")
}

// startWorkerPool starts the worker pool for executing probe tasks
func startWorkerPool(ctx context.Context, logger *zap.Logger, scheduler *scheduler.Scheduler, 
	prober *prober.Prober, hub *hub.Hub, metrics *metrics.Metrics) {
	
	taskChan := scheduler.GetTaskChannel()
	
	for i := 0; i < workerPoolSize; i++ {
		go func(workerID int) {
			logger.Debug("Starting worker", zap.Int("worker_id", workerID))
			
			for {
				select {
				case <-ctx.Done():
					logger.Debug("Worker stopping", zap.Int("worker_id", workerID))
					return
					
				case task, ok := <-taskChan:
					if !ok {
						logger.Debug("Task channel closed, worker stopping", zap.Int("worker_id", workerID))
						return
					}
					
					// Execute probe with timeout context
					probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
					result := prober.ProbeTask(probeCtx, task)
					cancel()
					
					// Record metrics
					metrics.RecordProbe(result)
					
					// Broadcast result
					hub.Broadcast(result)
					metrics.IncrementWebSocketMessage(result.Payer, result.Type)
				}
			}
		}(i)
	}
	
	logger.Info("Worker pool started", zap.Int("workers", workerPoolSize))
}

// createWebSocketServer creates the WebSocket HTTP server
func createWebSocketServer(hub *hub.Hub, configLoader *config.Loader, logger *zap.Logger) *http.Server {
	mux := http.NewServeMux()
	
	// WebSocket endpoint
	mux.HandleFunc("/ws", hub.HandleWebSocket)
	
	// Static file serving for web client
	mux.Handle("/", http.FileServer(http.Dir("./web/")))
	
	// Serve test client files
	mux.Handle("/test/", http.StripPrefix("/test/", http.FileServer(http.Dir("./test/"))))
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"payer-status-monitor"}`))
	})
	
	// Configuration API endpoint for dynamic UI
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		cfg := configLoader.GetConfig()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		
		// Extract unique payers and endpoint types
		payers := make([]string, 0, len(cfg.Payers))
		typeSet := make(map[string]bool)
		
		for _, payer := range cfg.Payers {
			payers = append(payers, payer.Name)
			for _, endpoint := range payer.Endpoints {
				typeSet[endpoint.Type] = true
			}
		}
		
		types := make([]string, 0, len(typeSet))
		for t := range typeSet {
			types = append(types, t)
		}
		
		// Count actual endpoints
		totalEndpoints := 0
		for _, payer := range cfg.Payers {
			totalEndpoints += len(payer.Endpoints)
		}
		
		// Create proper JSON response
		response := map[string]interface{}{
			"payers":          payers,
			"types":           types,
			"total_payers":    len(cfg.Payers),
			"total_endpoints": totalEndpoints,
		}
		
		json.NewEncoder(w).Encode(response)
	})
	
	// Debug endpoints (as per .windsurfrules)
	mux.HandleFunc("/debug/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := hub.GetStats()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Simple JSON encoding for stats
		fmt.Fprintf(w, `{"hub_stats":%v}`, stats)
	})

	return &http.Server{
		Addr:         ":" + getEnv("WS_PORT", defaultWSPort),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// createMetricsServer creates the Prometheus metrics HTTP server
func createMetricsServer(metrics *metrics.Metrics, logger *zap.Logger) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	
	return &http.Server{
		Addr:         ":" + getEnv("METRICS_PORT", defaultMetricsPort),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
}

// initLogger initializes the structured logger
func initLogger() *zap.Logger {
	logLevel := getEnv("LOG_LEVEL", defaultLogLevel)
	
	var config zap.Config
	if logLevel == "debug" {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}
	
	// Parse log level
	switch logLevel {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	}
	
	logger, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	
	return logger
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
