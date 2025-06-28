# WebSocket Health Monitor - Testing Guide

## 📋 Overview
This guide provides comprehensive testing instructions for the WebSocket Health Monitor system using the provided Postman collection and additional WebSocket testing tools.

## 🚀 Quick Start

### 1. Import Postman Collection
1. Open Postman
2. Click **Import** → **Upload Files**
3. Select `Payer-Status-WebSocket-Monitor.postman_collection.json`
4. The collection will be imported with pre-configured tests

### 2. Environment Setup
The collection uses these default variables:
- `base_url`: `http://localhost:8080` (WebSocket server)
- `metrics_url`: `http://localhost:9090` (Prometheus metrics)
- `websocket_url`: `ws://localhost:8080/ws` (WebSocket endpoint)

## 🧪 HTTP Endpoint Tests

### Health Check
```bash
GET http://localhost:8080/health
```
**Expected Response:**
```json
{
  "status": "healthy",
  "service": "payer-status-monitor"
}
```

### Prometheus Metrics
```bash
GET http://localhost:9090/metrics
```
**Key Metrics to Verify:**
- `probe_duration_seconds` - Histogram of probe latencies
- `probe_total` - Counter of total probes by payer/type/status
- `websocket_connections_active` - Current WebSocket connections
- `websocket_messages_sent_total` - Total messages broadcast
- `config_reload_total` - Configuration reload attempts

### Web Dashboard
```bash
GET http://localhost:8080/
```
**Expected:** HTML dashboard with real-time monitoring interface

## 🔌 WebSocket Testing

### Using Browser Console
```javascript
// Connect to WebSocket
const ws = new WebSocket('ws://localhost:8080/ws');

// Handle connection events
ws.onopen = () => {
    console.log('Connected to WebSocket');
    
    // Subscribe to all payers and types
    ws.send(JSON.stringify({
        action: 'subscribe',
        payers: [],  // Empty = all payers
        types: []    // Empty = all types
    }));
};

// Handle incoming probe results
ws.onmessage = (event) => {
    const result = JSON.parse(event.data);
    console.log('Probe Result:', result);
};

// Handle errors and disconnection
ws.onerror = (error) => console.error('WebSocket Error:', error);
ws.onclose = () => console.log('WebSocket Disconnected');
```

### Using wscat (Command Line)
```bash
# Install wscat
npm install -g wscat

# Connect to WebSocket
wscat -c ws://localhost:8080/ws

# Send subscription message
{"action":"subscribe","payers":[],"types":[]}

# Send filtered subscription (Aetna only)
{"action":"subscribe","payers":["Aetna"],"types":[]}

# Send type-filtered subscription (login endpoints only)
{"action":"subscribe","payers":[],"types":["login"]}
```

### Expected WebSocket Messages
```json
{
  "ts": "2024-01-15T10:30:45Z",
  "payer": "Aetna",
  "type": "login",
  "url": "https://claimconnect.dentalxchange.com/dci/wicket/page",
  "latency_ms": 245,
  "status_code": 200,
  "err": ""
}
```

## 🧪 Subscription Filtering Tests

### Test Case 1: All Results
```json
{"action":"subscribe","payers":[],"types":[]}
```
**Expected:** Receive all probe results from all payers and endpoint types

### Test Case 2: Single Payer Filter
```json
{"action":"subscribe","payers":["Aetna"],"types":[]}
```
**Expected:** Only receive probe results from Aetna endpoints

### Test Case 3: Single Type Filter
```json
{"action":"subscribe","payers":[],"types":["login"]}
```
**Expected:** Only receive login endpoint probe results from all payers

### Test Case 4: Combined Filter
```json
{"action":"subscribe","payers":["Cigna"],"types":["api","eligibility"]}
```
**Expected:** Only receive API and eligibility probe results from Cigna

### Test Case 5: Update Subscription
```json
{"action":"subscribe","payers":["Delta Dental"],"types":["patient_search"]}
```
**Expected:** Previous subscription is replaced, now only Delta Dental patient_search results

## 📊 Load Testing Scenarios

### Postman Runner Configuration
1. Select the **Load Testing** folder
2. Click **Run Collection**
3. Set iterations: `100-1000`
4. Set delay: `10ms` between requests
5. Monitor response times and success rates

### Expected Performance
- Health check: `< 50ms` response time
- Metrics endpoint: `< 200ms` response time
- WebSocket connections: Support `1000+` concurrent clients
- Memory usage: Stable under load (no leaks)

## 🔍 Debugging & Troubleshooting

### Common Issues

#### WebSocket Connection Fails
```bash
# Check if server is running
curl http://localhost:8080/health

# Check WebSocket endpoint
curl -i -N -H "Connection: Upgrade" -H "Upgrade: websocket" -H "Sec-WebSocket-Key: test" -H "Sec-WebSocket-Version: 13" http://localhost:8080/ws
```

#### No Probe Results Received
1. Verify configuration has valid endpoints
2. Check server logs for probe execution
3. Ensure minimum 1-minute schedule intervals
4. Verify network connectivity to target URLs

#### Metrics Not Available
```bash
# Check metrics server
curl http://localhost:9090/metrics | head -20

# Verify Prometheus format
curl http://localhost:9090/metrics | grep probe_total
```

### Server Logs Analysis
```bash
# Follow server logs
tail -f /var/log/payer-status-monitor.log

# Filter for specific events
grep "probe_result" /var/log/payer-status-monitor.log
grep "websocket" /var/log/payer-status-monitor.log
```

## 🎯 Test Scenarios

### Scenario 1: Basic Functionality
1. ✅ Health check returns healthy status
2. ✅ Metrics endpoint returns Prometheus format
3. ✅ WebSocket connection establishes successfully
4. ✅ Subscription message is accepted
5. ✅ Probe results are received within expected timeframe

### Scenario 2: Subscription Filtering
1. ✅ Subscribe to all results
2. ✅ Filter by specific payer
3. ✅ Filter by endpoint type
4. ✅ Combine payer and type filters
5. ✅ Update subscription dynamically

### Scenario 3: Performance & Reliability
1. ✅ Handle 100+ concurrent WebSocket connections
2. ✅ Maintain <200ms response times under load
3. ✅ Graceful handling of slow/disconnected clients
4. ✅ Memory usage remains stable over time
5. ✅ Metrics accurately reflect system state

### Scenario 4: Error Handling
1. ✅ Invalid subscription messages are rejected
2. ✅ Network failures don't crash the server
3. ✅ Malformed probe URLs are handled gracefully
4. ✅ WebSocket disconnections are cleaned up properly
5. ✅ Configuration errors are logged appropriately

## 📈 Success Criteria

### Functional Requirements
- ✅ All HTTP endpoints return expected responses
- ✅ WebSocket connections establish and receive data
- ✅ Subscription filtering works correctly
- ✅ Probe results contain all required fields
- ✅ Metrics are properly exposed for Prometheus

### Performance Requirements
- ✅ Health check: `< 50ms` p95 response time
- ✅ WebSocket message delivery: `< 200ms` p50 latency
- ✅ Support `1000+` concurrent WebSocket connections
- ✅ Memory usage: `< 100MB` under normal load
- ✅ CPU usage: `< 50%` under normal load

### Reliability Requirements
- ✅ Zero crashes during 24-hour test period
- ✅ Graceful handling of network failures
- ✅ Proper cleanup of disconnected clients
- ✅ Configuration hot-reload without service interruption
- ✅ Accurate metrics reporting at all times

## 🛠️ Advanced Testing Tools

### Artillery.js Load Testing
```yaml
# artillery-websocket-test.yml
config:
  target: 'ws://localhost:8080'
  phases:
    - duration: 60
      arrivalRate: 10
scenarios:
  - name: "WebSocket Load Test"
    engine: ws
    flow:
      - connect:
          url: "/ws"
      - send:
          payload: '{"action":"subscribe","payers":[],"types":[]}'
      - think: 30
```

### WebSocket King (GUI Tool)
1. Download WebSocket King
2. Connect to `ws://localhost:8080/ws`
3. Send subscription messages
4. Monitor real-time probe results
5. Test connection stability

This comprehensive testing guide ensures your WebSocket Health Monitor meets all functional, performance, and reliability requirements specified in your `.windsurfrules`.
