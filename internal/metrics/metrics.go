package metrics

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"payer-status-io/internal/config"
)

// Metrics holds all Prometheus metrics for the application
type Metrics struct {
	// Probe metrics
	probeDuration *prometheus.HistogramVec
	probeTotal    *prometheus.CounterVec

	// WebSocket metrics
	wsConnectionsActive *prometheus.GaugeVec
	wsMessagesSent      *prometheus.CounterVec

	// Config metrics
	configReloadTotal *prometheus.CounterVec

	// System metrics
	schedulerTasksTotal *prometheus.GaugeVec
	httpClientsTotal    *prometheus.GaugeVec

	logger *zap.Logger
}

// New creates a new metrics collector
func New(logger *zap.Logger) *Metrics {
	m := &Metrics{
		logger: logger,
	}

	m.initMetrics()
	m.registerMetrics()

	return m
}

// initMetrics initializes all Prometheus metrics
func (m *Metrics) initMetrics() {
	// Probe duration histogram (as per .windsurfrules)
	m.probeDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "probe_duration_seconds",
			Help:    "Duration of health probe requests in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to ~32s
		},
		[]string{"payer", "type", "status_code"},
	)

	// Probe total counter (as per .windsurfrules)
	m.probeTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "probe_total",
			Help: "Total number of health probes executed",
		},
		[]string{"payer", "type", "status_code"},
	)

	// WebSocket active connections (as per .windsurfrules)
	m.wsConnectionsActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "websocket_connections_active",
			Help: "Number of active WebSocket connections",
		},
		[]string{"status"},
	)

	// WebSocket messages sent (as per .windsurfrules)
	m.wsMessagesSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "websocket_messages_sent_total",
			Help: "Total number of WebSocket messages sent",
		},
		[]string{"payer", "type"},
	)

	// Config reload counter (as per .windsurfrules)
	m.configReloadTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "config_reload_total",
			Help: "Total number of configuration reload attempts",
		},
		[]string{"status"}, // success, failure
	)

	// Scheduler tasks gauge
	m.schedulerTasksTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "scheduler_tasks_total",
			Help: "Total number of scheduled tasks",
		},
		[]string{"status"}, // pending, active
	)

	// HTTP clients gauge
	m.httpClientsTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_clients_total",
			Help: "Total number of HTTP clients (connection pools)",
		},
		[]string{"hostname"},
	)
}

// registerMetrics registers all metrics with Prometheus
func (m *Metrics) registerMetrics() {
	prometheus.MustRegister(
		m.probeDuration,
		m.probeTotal,
		m.wsConnectionsActive,
		m.wsMessagesSent,
		m.configReloadTotal,
		m.schedulerTasksTotal,
		m.httpClientsTotal,
	)

	m.logger.Info("Prometheus metrics registered")
}

// RecordProbe records metrics for a completed probe
func (m *Metrics) RecordProbe(result *config.ProbeResult) {
	statusCode := "unknown"
	if result.StatusCode > 0 {
		statusCode = strconv.Itoa(result.StatusCode)
	}

	labels := prometheus.Labels{
		"payer":       result.Payer,
		"type":        result.Type,
		"status_code": statusCode,
	}

	// Record duration
	duration := float64(result.LatencyMS) / 1000.0 // Convert to seconds
	m.probeDuration.With(labels).Observe(duration)

	// Increment counter
	m.probeTotal.With(labels).Inc()

	m.logger.Debug("Recorded probe metrics",
		zap.String("payer", result.Payer),
		zap.String("type", result.Type),
		zap.String("status_code", statusCode),
		zap.Float64("duration_seconds", duration))
}

// SetWebSocketConnections updates the active WebSocket connections gauge
func (m *Metrics) SetWebSocketConnections(count int) {
	m.wsConnectionsActive.WithLabelValues("active").Set(float64(count))
}

// IncrementWebSocketMessage increments the WebSocket messages sent counter
func (m *Metrics) IncrementWebSocketMessage(payer, msgType string) {
	m.wsMessagesSent.WithLabelValues(payer, msgType).Inc()
}

// RecordConfigReload records a configuration reload attempt
func (m *Metrics) RecordConfigReload(success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	m.configReloadTotal.WithLabelValues(status).Inc()

	m.logger.Debug("Recorded config reload metric", zap.String("status", status))
}

// SetSchedulerTasks updates the scheduler tasks gauge
func (m *Metrics) SetSchedulerTasks(pending, active int) {
	m.schedulerTasksTotal.WithLabelValues("pending").Set(float64(pending))
	m.schedulerTasksTotal.WithLabelValues("active").Set(float64(active))
}

// SetHTTPClients updates the HTTP clients gauge
func (m *Metrics) SetHTTPClients(hostname string, count int) {
	m.httpClientsTotal.WithLabelValues(hostname).Set(float64(count))
}

// Handler returns the Prometheus metrics HTTP handler
func (m *Metrics) Handler() http.Handler {
	return promhttp.Handler()
}

// GetStats returns metrics statistics
func (m *Metrics) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"metrics_registered": 7,
		"endpoint":          "/metrics",
	}
}
