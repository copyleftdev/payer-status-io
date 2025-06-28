package config

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Loader handles configuration loading and hot-reloading
type Loader struct {
	configPath string
	config     *Config
	mu         sync.RWMutex
	logger     *zap.Logger
	callbacks  []func(*Config)
}

// NewLoader creates a new configuration loader
func NewLoader(configPath string, logger *zap.Logger) *Loader {
	return &Loader{
		configPath: configPath,
		logger:     logger,
		callbacks:  make([]func(*Config), 0),
	}
}

// Load loads the configuration from the YAML file
func (l *Loader) Load() error {
	data, err := os.ReadFile(l.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", l.configPath, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse YAML config: %w", err)
	}

	if err := l.validate(&config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	l.mu.Lock()
	l.config = &config
	l.mu.Unlock()

	l.logger.Info("Configuration loaded successfully",
		zap.String("config_path", l.configPath),
		zap.Int("payers_count", len(config.Payers)),
		zap.Int("total_endpoints", l.countEndpoints(&config)))

	// Notify callbacks of config change
	for _, callback := range l.callbacks {
		callback(&config)
	}

	return nil
}

// MustLoad loads the configuration and panics on error
func (l *Loader) MustLoad() *Config {
	if err := l.Load(); err != nil {
		l.logger.Fatal("Failed to load configuration", zap.Error(err))
	}
	return l.GetConfig()
}

// GetConfig returns the current configuration (thread-safe)
func (l *Loader) GetConfig() *Config {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.config
}

// OnConfigChange registers a callback for configuration changes
func (l *Loader) OnConfigChange(callback func(*Config)) {
	l.callbacks = append(l.callbacks, callback)
}

// WatchForChanges starts watching for SIGHUP signals to reload configuration
func (l *Loader) WatchForChanges(ctx context.Context) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-sigChan:
				l.logger.Info("Received SIGHUP, reloading configuration")
				if err := l.Load(); err != nil {
					l.logger.Error("Failed to reload configuration", zap.Error(err))
				} else {
					l.logger.Info("Configuration reloaded successfully")
				}
			}
		}
	}()
}

// validate performs basic validation on the configuration
func (l *Loader) validate(config *Config) error {
	if len(config.Payers) == 0 {
		return fmt.Errorf("no payers configured")
	}

	for i, payer := range config.Payers {
		if payer.Name == "" {
			return fmt.Errorf("payer at index %d has empty name", i)
		}

		if len(payer.Endpoints) == 0 {
			return fmt.Errorf("payer %s has no endpoints", payer.Name)
		}

		for j, endpoint := range payer.Endpoints {
			if endpoint.Type == "" {
				return fmt.Errorf("payer %s endpoint at index %d has empty type", payer.Name, j)
			}

			if endpoint.URL == "" && endpoint.Path == "" && endpoint.URLContains == "" {
				return fmt.Errorf("payer %s endpoint %s has no URL, path, or url_contains", payer.Name, endpoint.Type)
			}

			// Validate schedule if provided
			if endpoint.Schedule != 0 && endpoint.Schedule < time.Minute {
				return fmt.Errorf("payer %s endpoint %s has schedule less than 1 minute", payer.Name, endpoint.Type)
			}
		}
	}

	return nil
}

// countEndpoints returns the total number of endpoints across all payers
func (l *Loader) countEndpoints(config *Config) int {
	count := 0
	for _, payer := range config.Payers {
		count += len(payer.Endpoints)
	}
	return count
}

// MustLoad is a convenience function to create and load configuration
func MustLoad(configPath string) *Config {
	logger, _ := zap.NewProduction()
	loader := NewLoader(configPath, logger)
	return loader.MustLoad()
}
