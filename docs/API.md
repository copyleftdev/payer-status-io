# API Documentation

## Table of Contents
- [WebSocket API](#websocket-api)
- [REST API](#rest-api)
- [Metrics](#metrics)
- [Health Check](#health-check)

## WebSocket API

### Connection
- **URL**: `ws://localhost:8080/ws`
- **Protocol**: `payer-status-v1`
- **Authentication**: None (for local development)

### Messages

#### Subscription Message
```json
{
  "action": "subscribe",
  "payers": ["Aetna", "Cigna"],
  "types": ["login", "api"]
}
```

#### Unsubscribe Message
```json
{
  "action": "unsubscribe"
}
```

#### Probe Result Message
```json
{
  "ts": "2023-06-27T22:45:43.123456Z",
  "payer": "Aetna",
  "type": "login",
  "url": "https://aetna.com/login",
  "latency_ms": 123,
  "status_code": 200,
  "error": ""
}
```

## REST API

### Health Check
- **Endpoint**: `GET /health`
- **Response**:
  ```json
  {
    "status": "ok",
    "version": "1.0.0",
    "uptime": "1h23m45s",
    "timestamp": "2023-06-27T22:45:43Z"
  }
  ```

### Get Status
- **Endpoint**: `GET /status`
- **Query Parameters**:
  - `payer`: Filter by payer name
  - `type`: Filter by endpoint type
- **Response**:
  ```json
  [
    {
      "payer": "Aetna",
      "type": "login",
      "status": "healthy",
      "last_check": "2023-06-27T22:45:43Z",
      "latency_ms": 123,
      "status_code": 200
    }
  ]
  ```

## Metrics

### Prometheus Metrics
- **Endpoint**: `GET /metrics`
- **Metrics Exposed**:
  - `probe_duration_seconds` - Duration of HTTP probes
  - `probe_status_code` - Status code of the last probe
  - `websocket_connections` - Number of active WebSocket connections
  - `http_requests_total` - Total HTTP requests processed
  - `http_request_duration_seconds` - HTTP request latencies

### Health Check

```http
GET /health HTTP/1.1
Host: localhost:8080
Accept: application/json
```

**Response:**
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "status": "ok",
  "version": "1.0.0",
  "uptime": "1h23m45s",
  "timestamp": "2023-06-27T22:45:43Z"
}
```

## Error Responses

### 400 Bad Request
```json
{
  "error": "Invalid request",
  "message": "Invalid subscription format"
}
```

### 404 Not Found
```json
{
  "error": "Not found",
  "message": "The requested resource was not found"
}
```

### 500 Internal Server Error
```json
{
  "error": "Internal server error",
  "message": "Something went wrong"
}
```
