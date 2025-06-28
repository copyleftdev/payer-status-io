package config

import (
	"time"
)

// Config represents the complete configuration structure
type Config struct {
	Payers []Payer `yaml:"payers"`
}

// Payer represents a healthcare payer with multiple endpoints
type Payer struct {
	Name      string     `yaml:"name"`
	Endpoints []Endpoint `yaml:"endpoints"`
}

// Endpoint represents a single endpoint to monitor
type Endpoint struct {
	Type        string        `yaml:"type"`                  // login, api, patient_search, etc.
	URL         string        `yaml:"url,omitempty"`         // Full URL
	Path        string        `yaml:"path,omitempty"`        // Relative path (for API endpoints)
	URLContains string        `yaml:"url_contains,omitempty"` // URL pattern matching
	Method      string        `yaml:"method,omitempty"`      // HTTP method (default: GET)
	Schedule    time.Duration `yaml:"schedule,omitempty"`    // Probe interval (default: 15m)
	Description string        `yaml:"description,omitempty"` // Optional context
}

// GetURL returns the complete URL for the endpoint
func (e *Endpoint) GetURL() string {
	if e.URL != "" {
		return e.URL
	}
	// For path-only endpoints, they would be combined with a base URL
	// This will be handled by the prober package
	return e.Path
}

// GetMethod returns the HTTP method, defaulting to GET
func (e *Endpoint) GetMethod() string {
	if e.Method == "" {
		return "GET"
	}
	return e.Method
}

// GetSchedule returns the probe interval, defaulting to 15 minutes
func (e *Endpoint) GetSchedule() time.Duration {
	if e.Schedule == 0 {
		return 15 * time.Minute
	}
	// Enforce minimum interval of 1 minute as per .windsurfrules
	if e.Schedule < time.Minute {
		return time.Minute
	}
	return e.Schedule
}

// ProbeResult represents the result of a health probe
type ProbeResult struct {
	Timestamp  time.Time `json:"ts"`
	Payer      string    `json:"payer"`
	Type       string    `json:"type"`
	URL        string    `json:"url"`
	LatencyMS  int64     `json:"latency_ms"`
	StatusCode int       `json:"status_code"`
	Err        string    `json:"err,omitempty"`
}
