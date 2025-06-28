# Payer Status IO - WebSocket Health Monitor

[![Go Report Card](https://goreportcard.com/badge/github.com/copyleftdev/payer-status-io)](https://goreportcard.com/report/github.com/copyleftdev/payer-status-io)
[![Docker Pulls](https://img.shields.io/docker/pulls/copyleftdev/payer-status-io)](https://hub.docker.com/r/copyleftdev/payer-status-io)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A high-performance WebSocket-based health monitoring system for tracking the status of healthcare payer endpoints in real-time.

## 🚀 Features

- **Real-time Monitoring**: WebSocket-based push notifications for instant updates
- **Multi-payer Support**: Monitor 25+ insurance payers simultaneously
- **Comprehensive Metrics**: Track response times, status codes, and errors
- **Configurable Probes**: Customize check intervals and endpoints per payer
- **Prometheus Integration**: Built-in metrics endpoint for monitoring
- **Docker & Kubernetes Ready**: Containerized deployment with health checks
- **REST API**: Manage and query status programmatically

## 📦 Quick Start

### Prerequisites

- Go 1.21+
- Docker 20.10+
- Make (optional, but recommended)

### Using Docker (Recommended)


```bash
# Start the service
docker-compose up -d

# View logs
docker-compose logs -f
```

### Local Development

```bash
# Build and run
go build -o bin/server ./cmd/server
./bin/server --config ./docs/payer_status.yaml

# Or run with hot-reloading (requires air)
make dev
```

## 🔌 API Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics
- `GET /status` - Current status of all payers
- `GET /status/{payer}` - Status for a specific payer
- `WS /ws` - WebSocket endpoint for real-time updates

## 🔧 Configuration

Configuration is done via YAML. See `docs/configuration.md` for details.

```yaml
server:
  port: 8080
  metrics_port: 9090
  log_level: info
  
probes:
  - name: Aetna
    endpoints:
      - type: login
        url: https://aetna.com/login
        interval: 5m
        timeout: 10s
```

## 📊 Monitoring

Prometheus metrics are exposed on port 9090 by default:

- `probe_duration_seconds` - Duration of HTTP probes
- `probe_status_code` - Status code of the last probe
- `websocket_connections` - Number of active WebSocket connections

## 🛠 Development

### Building

```bash
make build
```

### Testing

```bash
make test
```

### Linting

```bash
make lint
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please read our [contributing guidelines](CONTRIBUTING.md) to get started.
