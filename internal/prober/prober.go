package prober

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"payer-status-io/internal/config"
	"payer-status-io/internal/scheduler"
)

// Prober handles HTTP health probes with connection pooling
type Prober struct {
	clients map[string]*http.Client // Per-hostname HTTP clients
	mu      sync.RWMutex
	logger  *zap.Logger
	timeout time.Duration
}

// New creates a new prober with optimized HTTP clients
func New(logger *zap.Logger, timeout time.Duration) *Prober {
	return &Prober{
		clients: make(map[string]*http.Client),
		logger:  logger,
		timeout: timeout,
	}
}

// ProbeTask executes a health probe for the given task
func (p *Prober) ProbeTask(ctx context.Context, task *scheduler.Task) *config.ProbeResult {
	start := time.Now()
	
	result := &config.ProbeResult{
		Timestamp: start,
		Payer:     task.Payer,
		Type:      task.Endpoint.Type,
		URL:       p.resolveURL(task.Endpoint),
	}

	// Create HTTP request
	req, err := p.createRequest(ctx, task.Endpoint)
	if err != nil {
		result.Err = fmt.Sprintf("failed to create request: %v", err)
		result.LatencyMS = time.Since(start).Milliseconds()
		return result
	}

	// Get appropriate HTTP client
	client := p.getClient(req.URL.Hostname())

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		result.Err = fmt.Sprintf("request failed: %v", err)
		result.LatencyMS = time.Since(start).Milliseconds()
		return result
	}
	defer resp.Body.Close()

	// Record results
	result.StatusCode = resp.StatusCode
	result.LatencyMS = time.Since(start).Milliseconds()

	p.logger.Debug("Probe completed",
		zap.String("payer", task.Payer),
		zap.String("type", task.Endpoint.Type),
		zap.String("url", result.URL),
		zap.Int("status_code", result.StatusCode),
		zap.Int64("latency_ms", result.LatencyMS))

	return result
}

// createRequest creates an HTTP request for the endpoint
func (p *Prober) createRequest(ctx context.Context, endpoint config.Endpoint) (*http.Request, error) {
	url := p.resolveURL(endpoint)
	method := endpoint.GetMethod()

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	// Set standard headers
	req.Header.Set("User-Agent", "Payer-Status-Monitor/1.0")
	req.Header.Set("Accept", "text/html,application/json,*/*")
	req.Header.Set("Cache-Control", "no-cache")

	return req, nil
}

// resolveURL resolves the complete URL for an endpoint
func (p *Prober) resolveURL(endpoint config.Endpoint) string {
	url := endpoint.GetURL()
	
	// Handle environment variable substitution
	if strings.Contains(url, "${") {
		url = os.ExpandEnv(url)
	}

	return url
}

// getClient returns an optimized HTTP client for the hostname
func (p *Prober) getClient(hostname string) *http.Client {
	p.mu.RLock()
	client, exists := p.clients[hostname]
	p.mu.RUnlock()

	if exists {
		return client
	}

	// Create new client with connection pooling
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := p.clients[hostname]; exists {
		return client
	}

	// Create optimized transport for this hostname
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		},
	}

	client = &http.Client{
		Transport: transport,
		Timeout:   p.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 5 redirects
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	p.clients[hostname] = client
	
	p.logger.Debug("Created HTTP client for hostname",
		zap.String("hostname", hostname))

	return client
}

// Close closes all HTTP clients and cleans up resources
func (p *Prober) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for hostname, client := range p.clients {
		if transport, ok := client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
		p.logger.Debug("Closed HTTP client", zap.String("hostname", hostname))
	}

	p.clients = make(map[string]*http.Client)
}

// GetStats returns prober statistics
func (p *Prober) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"http_clients": len(p.clients),
		"timeout_ms":   p.timeout.Milliseconds(),
	}
}
