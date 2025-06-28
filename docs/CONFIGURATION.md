# Configuration Guide

## Table of Contents
- [Configuration File](#configuration-file)
- [Environment Variables](#environment-variables)
- [Payer Configuration](#payer-configuration)
- [Endpoint Configuration](#endpoint-configuration)
- [WebSocket Configuration](#websocket-configuration)
- [Metrics Configuration](#metrics-configuration)
- [Logging Configuration](#logging-configuration)
- [TLS Configuration](#tls-configuration)
- [Rate Limiting](#rate-limiting)
- [Example Configuration](#example-configuration)

## Configuration File

The primary configuration is done through a YAML file. By default, the application looks for `payer_status.yaml` in the current directory.

### Specifying a Custom Config File

```bash
./payer-status-io --config /path/to/config.yaml
```

## Environment Variables

All configuration options can be overridden using environment variables with the `PAYER_STATUS_` prefix. For example:

```yaml
server:
  port: 8080
```

Can be overridden with:

```bash
export PAYER_STATUS_SERVER_PORT=9090
```

Nested configuration can be accessed using `_` as a separator:

```yaml
server:
  metrics:
    enabled: true
```

```bash
export PAYER_STATUS_SERVER_METRICS_ENABLED=false
```

## Payer Configuration

### Required Fields

```yaml
payers:
  - name: string          # Payer identifier (e.g., "Aetna", "Cigna")
    endpoints:
      - type: string      # login|api|patient_search|pdf_extraction|claims_address|eligibility
        url: string       # Full URL or environment variable reference
        method: string    # HTTP method (default: GET)
        schedule: string  # Probe interval (e.g., "5m", "1h")
```

### Optional Fields

```yaml
        description: string  # Human-readable description
        timeout: string      # Request timeout (default: "10s")
        headers:             # Custom headers
          Header-Name: value
        tls_skip_verify: bool # Skip TLS verification (default: false)
        expect_status: int   # Expected HTTP status code (default: 200)
        expect_body: string  # Expected response body (regex)
        expect_json:         # Expected JSON response (partial match)
          field: value
        auth:                # Basic auth
          username: string
          password: string   # Can use ${ENV_VAR} for secrets
        retry:               # Retry configuration
          attempts: int      # Number of retry attempts (default: 2)
          delay: string     # Initial delay between retries (default: "1s")
          max_delay: string # Maximum delay between retries (default: "30s")
```

## Endpoint Configuration

### Supported HTTP Methods
- GET
- POST
- PUT
- DELETE
- PATCH
- HEAD

### Schedule Format

Uses Go's `time.ParseDuration` format:
- `300ms`
- `5s`
- `2m`
- `1h`
- `24h`

### Environment Variables

Use `${VAR_NAME}` or `$VAR_NAME` to reference environment variables in configuration values:

```yaml
url: ${API_BASE_URL}/login
headers:
  Authorization: Bearer $API_KEY
```

## WebSocket Configuration

```yaml
websocket:
  enabled: true                 # Enable WebSocket server (default: true)
  path: /ws                    # WebSocket endpoint path
  write_timeout: 10s            # Timeout for writing messages
  read_timeout: 10s             # Timeout for reading messages
  max_message_size: 1024        # Maximum message size in bytes
  read_buffer_size: 4096        # Read buffer size in bytes
  write_buffer_size: 4096       # Write buffer size in bytes
  enable_compression: true      # Enable WebSocket per-message deflate
  check_origin: true            # Check Origin header
  allowed_origins:              # List of allowed origins (if check_origin is true)
    - http://localhost:3000
    - https://example.com
```

## Metrics Configuration

```yaml
metrics:
  enabled: true                 # Enable metrics endpoint (default: true)
  path: /metrics              # Metrics endpoint path
  namespace: payer_status      # Metrics namespace
  subsystem: server            # Metrics subsystem
  enable_go_metrics: true      # Enable Go runtime metrics
  enable_process_metrics: true # Enable process metrics
  enable_http_metrics: true    # Enable HTTP request metrics
  enable_db_metrics: true      # Enable database metrics (if applicable)
  labels:                     # Additional labels for all metrics
    environment: production
    region: us-west-2
```

## Logging Configuration

```yaml
logging:
  level: info                  # Log level: debug, info, warn, error, fatal, panic
  format: json                 # Log format: json, console
  timestamp_format: rfc3339    # Timestamp format
  enable_caller: true          # Enable caller information
  enable_stacktrace: false     # Enable stacktrace for errors
  output_paths:                # Log output paths (stdout, stderr, or file paths)
    - stdout
  error_output_paths:          # Error log output paths
    - stderr
  max_size: 100               # Max log file size in MB
  max_backups: 3              # Max number of old log files to retain
  max_age: 28                 # Max number of days to retain log files
  compress: true              # Whether to compress rotated log files
```

## TLS Configuration

### Server TLS

```yaml
tls:
  enabled: false               # Enable TLS (default: false)
  cert_file: server.crt       # Path to certificate file
  key_file: server.key         # Path to private key file
  client_auth: none            # none, request, require, verify_if_given, require_and_verify
  client_ca_file: ca.crt       # Path to CA certificate file for client verification
  min_version: TLS1.2          # Minimum TLS version (TLS1.0, TLS1.1, TLS1.2, TLS1.3)
  max_version: TLS1.3          # Maximum TLS version
  cipher_suites:               # List of allowed cipher suites
    - TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
    - TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
  prefer_server_cipher_suites: true  # Prefer server's cipher suite order
  curve_preferences:           # Curve preferences for ECDHE
    - P256
    - P384
  session_tickets: true        # Enable session tickets
  session_ticket_key:          # Session ticket key (32 bytes, base64 encoded)
  session_cache:
    enabled: true
    capacity: 1000
    bucket_size: 64
```

### Client TLS (for probes)

```yaml
client:
  tls:
    insecure_skip_verify: false  # Skip TLS verification (not recommended for production)
    cert_file: client.crt      # Client certificate file
    key_file: client.key        # Client private key file
    ca_file: ca.crt            # CA certificate file for server verification
    server_name: example.com    # Server name for SNI
    min_version: TLS1.2
    max_version: TLS1.3
```

## Rate Limiting

```yaml
rate_limiting:
  enabled: true
  rps: 10                      # Requests per second
  burst: 20                    # Burst size
  enabled_endpoints:           # List of endpoint patterns to apply rate limiting
    - /api/*
    - /admin/*
  exempt_ips:                  # List of IPs/CIDRs exempt from rate limiting
    - 127.0.0.1/32
    - 10.0.0.0/8
  storage:                    # Rate limiting storage backend
    type: memory              # memory, redis, memcached
    # Redis configuration (if type is redis)
    redis:
      addr: localhost:6379
      password: ""
      db: 0
      pool_size: 10
      min_idle_conns: 2
      dial_timeout: 5s
      read_timeout: 3s
      write_timeout: 3s
      pool_timeout: 4s
      idle_timeout: 5m
      max_conn_age: 0
      max_retries: 3
      min_retry_backoff: 8ms
      max_retry_backoff: 512ms
```

## Example Configuration

```yaml
# payer_status.yaml

# Server configuration
server:
  host: "0.0.0.0"
  port: 8080
  shutdown_timeout: 30s
  read_timeout: 10s
  write_timeout: 10s
  idle_timeout: 120s
  max_header_bytes: 1048576

# WebSocket configuration
websocket:
  enabled: true
  path: /ws
  write_timeout: 10s
  read_timeout: 10s
  max_message_size: 1024
  read_buffer_size: 4096
  write_buffer_size: 4096
  enable_compression: true
  check_origin: true
  allowed_origins:
    - "http://localhost:3000"
    - "https://example.com"

# Metrics configuration
metrics:
  enabled: true
  path: /metrics
  namespace: payer_status
  subsystem: server
  enable_go_metrics: true
  enable_process_metrics: true
  enable_http_metrics: true
  labels:
    environment: development

# Logging configuration
logging:
  level: info
  format: json
  timestamp_format: rfc3339
  enable_caller: true
  enable_stacktrace: false
  output_paths:
    - stdout
  error_output_paths:
    - stderr
  max_size: 100
  max_backups: 3
  max_age: 28
  compress: true

# Rate limiting configuration
rate_limiting:
  enabled: true
  rps: 10
  burst: 20
  enabled_endpoints:
    - /api/*
  exempt_ips:
    - 127.0.0.1/32
  storage:
    type: memory

# Payers configuration
payers:
  - name: Aetna
    description: Aetna Health Insurance
    endpoints:
      - type: login
        url: https://www.aetna.com/login
        method: GET
        schedule: 5m
        timeout: 10s
        headers:
          User-Agent: Payer-Status-IO/1.0
        retry:
          attempts: 2
          delay: 1s
          max_delay: 10s

  - name: Cigna
    description: Cigna Health Insurance
    endpoints:
      - type: login
        url: https://my.cigna.com/web/public/consumer/portal/login
        method: GET
        schedule: 5m
        timeout: 15s
        expect_status: 200
        headers:
          User-Agent: Payer-Status-IO/1.0

  - name: UnitedHealthcare
    description: UnitedHealthcare
    endpoints:
      - type: login
        url: ${UHC_LOGIN_URL}  # Using environment variable
        method: POST
        schedule: 10m
        timeout: 20s
        headers:
          Content-Type: application/json
          Authorization: Bearer $API_KEY  # Using environment variable
        body: |
          {
            "username": "${UHC_USERNAME}",
            "password": "${UHC_PASSWORD}"
          }
        expect_status: 200
        expect_json:
          status: success

# TLS configuration (disabled by default)
tls:
  enabled: false
  cert_file: /path/to/cert.pem
  key_file: /path/to/key.pem
  min_version: TLS1.2
  max_version: TLS1.3
```

## Configuration Reloading

The application supports configuration reloading via SIGHUP signal. Send the SIGHUP signal to the running process to reload the configuration file:

```bash
kill -HUP $(pgrep payer-status-io)
```

## Configuration Validation

The application validates the configuration file on startup and will fail with descriptive error messages if the configuration is invalid. Common validation errors include:

- Missing required fields
- Invalid URLs
- Invalid duration formats
- Invalid regular expressions
- Invalid TLS configurations

## Best Practices

1. **Use Environment Variables for Secrets**: Never commit sensitive information like passwords or API keys directly in the configuration file. Use environment variables instead.

2. **Version Control**: Consider committing a `config.example.yaml` with placeholder values and add the actual configuration file to `.gitignore`.

3. **Least Privilege**: When running the application, ensure it has the minimum required permissions to access the configuration file and any referenced resources.

4. **Regular Audits**: Periodically review and update the configuration to remove unused endpoints and update credentials.

5. **Backup**: Regularly back up your configuration files, especially in production environments.

6. **Documentation**: Document any non-obvious configuration options or customizations for future reference.
