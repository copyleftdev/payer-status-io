# System Architecture

## Overview

The Payer Status IO is a high-performance WebSocket-based health monitoring system designed to track the status of healthcare payer endpoints in real-time. The system is built with scalability, reliability, and observability in mind.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Client Applications                      │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐  │
│  │ Web Browsers│     │ Mobile Apps │     │ 3rd Party   │  │
│  └──────┬──────┘     └──────┬──────┘     │  Services   │  │
│         │                    │             └──────┬──────┘  │
└─────────┼────────────────────┼─────────────────────┼─────────┘
          │                    │                     │
          ▼                    ▼                     ▼
┌─────────────────────────────────────────────────────────────┐
│                    API Gateway / Load Balancer             │
└───────────────┬───────────────────────┬───────────────────┘
                │                       │
                ▼                       ▼
┌─────────────────────────┐   ┌─────────────────────────┐
│     WebSocket Server    │   │      REST API Server    │
│  (Real-time Updates)    │   │   (Configuration/Query) │
└──────────────┬──────────┘   └──────────┬──────────────┘
               │                           │
               └───────────┬───────────────┘
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                     Core Services                          │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐  │
│  │   Scheduler │     │    Prober   │     │  Metrics    │  │
│  │  (Timers)   │     │ (HTTP Client)│     │  Collector  │  │
│  └──────┬──────┘     └──────┬──────┘     └──────┬──────┘  │
│         │                    │                    │         │
│  ┌──────▼──────┐     ┌──────▼──────┐     ┌──────▼──────┐  │
│  │   Config    │     │    Cache    │     │   Storage   │  │
│  │  Manager    │     │  (In-memory)│     │  (Optional) │  │
│  └─────────────┘     └─────────────┘     └─────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. WebSocket Hub
Manages client connections and message broadcasting with support for:
- Connection lifecycle management
- Message broadcasting with back-pressure handling
- Subscription filtering by payer and endpoint type
- Graceful connection cleanup

### 2. Scheduler
Responsible for scheduling and executing health checks:
- Priority queue for probe scheduling
- Rate limiting and jitter to prevent thundering herd
- Context-based cancellation for graceful shutdown
- Support for different scheduling strategies (fixed interval, exponential backoff)

### 3. Prober
Executes HTTP probes against configured endpoints:
- Connection pooling and keep-alive
- Configurable timeouts and retries
- TLS configuration support
- Environment variable substitution in URLs
- Request/response logging

### 4. Metrics Collector
Collects and exposes system metrics:
- Prometheus metrics endpoint
- Custom metrics for probes and WebSocket connections
- Health check endpoints
- Structured logging with Zap

## Data Flow

1. **Initialization**
   - Load configuration from YAML file
   - Initialize metrics collectors
   - Start WebSocket server
   - Schedule initial probes

2. **Probe Execution**
   - Scheduler triggers probe based on schedule
   - Prober executes HTTP request to target endpoint
   - Result is published to metrics collector
   - Result is broadcast to subscribed WebSocket clients

3. **Client Subscription**
   - Client connects to WebSocket endpoint
   - Client sends subscription message
   - Hub adds client to appropriate subscription groups
   - Client receives real-time updates for subscribed payers/endpoints

## Performance Considerations

### Concurrency Model
- Goroutines for concurrent probe execution
- Worker pool to limit concurrent probes
- Channel-based communication between components
- Context propagation for cancellation

### Resource Management
- Connection pooling for HTTP clients
- Memory-efficient data structures
- Back-pressure handling for slow clients
- Graceful degradation under load

## Security Considerations

### Authentication & Authorization
- JWT-based authentication (optional)
- CORS configuration
- Rate limiting
- Request validation

### Data Protection
- TLS for all external communications
- Sensitive data redaction in logs
- Input validation and sanitization
- Secure defaults

## Monitoring & Observability

### Logging
- Structured JSON logging
- Configurable log levels
- Request correlation IDs
- Performance tracing

### Metrics
- Prometheus metrics endpoint
- Custom metrics for business logic
- Alerting rules (via Prometheus Alertmanager)
- Grafana dashboards

## Deployment Architecture

### Containerization
- Multi-stage Docker build
- Minimal base image (scratch/distroless)
- Non-root user
- Health checks

### Orchestration
- Kubernetes manifests
- Horizontal Pod Autoscaler
- Pod Disruption Budget
- Resource requests/limits

## Scalability

### Horizontal Scaling
- Stateless design
- Shared-nothing architecture
- Distributed tracing support
- Service discovery

### Performance Targets
- 99.9% uptime SLA
- <200ms p50 latency for probes
- Support for 10k+ concurrent WebSocket connections
- Sub-second probe execution time

## Dependencies

### Runtime
- Go 1.21+
- Prometheus client library
- nhooyr.io/websocket
- zap for logging
- viper for configuration

### Development
- Docker
- golangci-lint
- air for hot-reloading
- go-mock for testing
