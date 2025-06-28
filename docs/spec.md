Below is a **complete yet concise design specification** for a high‑performance, professional‑grade **WebSocket Health Monitor** written in Go.
It is organized so you can drop each section into an engineering design doc or a Git repo’s `docs/` directory with minimal editing.

---

## 1 . Purpose & Scope

| Item          | Description                                                                                                                                                                    |
| ------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Goal**      | Expose near‑real‑time health telemetry (latency, HTTP status, first/last‑seen timestamps) for a fixed but extensible list of payer endpoints via a single WebSocket interface. |
| **Non‑Goals** | Deep functional testing of payer portals, storing PHI, or replaying scraped data.                                                                                              |
| **SLA**       | 99.9 % uptime for the WebSocket service; ≤ 200 ms p50 end‑to‑end latency from probe completion to push.                                                                        |

---

## 2 . High‑Level Architecture

```
                +-------------------+
                |  WebSocket Hub    |<----+----┐
                +---------^---------+     |    |
                          |               |    | control msgs
        broadcast <- chan |               |    |
                          |      +--------v----+-------+
                          |      |  Scheduler & Worker |   YAML
                          |      +---------^-----------+  config
                          |                |
                 events ->|                | tick
                +---------+---------+      |
                |   Metrics (Prom)  |<-----+ 
                +-------------------+
```

1. **Scheduler** produces probe tasks at controlled rates.
2. **Worker pool** executes HTTP(S) calls (or other protocol) and sends a `ProbeResult` on a channel.
3. **Hub** fan‑outs JSON‑encoded results to all matching WebSocket subscribers (Observer pattern).
4. **Metrics collector** publishes Prometheus counters/histograms for internal observability.

All components share a `context.Context` tree for graceful shutdown.

---

## 3 . Key Design Principles

* **Simplicity & Modularity** – Each package has a single responsibility; interfaces allow stubbing in tests.
* **Concurrency without Contention** – Leverage goroutines, buffered channels, sync.Pool for `http.Client`s.
* **Back‑Pressure Aware** – Non‑blocking writes; slow clients are dropped after a configurable grace period.
* **Rate Limiting & Jitter** – Token‑bucket per endpoint + global ceiling, with ±10 % random jitter.
* **Config Driven** – Entire endpoint matrix is a version‑controlled YAML file hot‑reloaded on `SIGHUP`.
* **Extensibility** – New probe kinds (e.g., gRPC, GraphQL) only require implementing `Prober` interface.

---

## 4 . Data Model

```go
// config/model.go
type Endpoint struct {
    Payer       string        `yaml:"name"`
    EndpointURL string        `yaml:"url"`
    Method      string        `yaml:"method,omitempty"`  // default GET
    Type        string        `yaml:"type"`              // login, api, pdf_extraction, etc.
    Schedule    time.Duration `yaml:"schedule,omitempty"`// e.g. 5m; default 15m
}

// runtime events pushed to clients
type ProbeResult struct {
    Timestamp   time.Time `json:"ts"`
    Payer       string    `json:"payer"`
    Type        string    `json:"type"`
    URL         string    `json:"url"`
    LatencyMS   int64     `json:"latency_ms"`
    StatusCode  int       `json:"status_code"`
    Err         string    `json:"err,omitempty"`
}
```

---

## 5 . Scheduler Algorithm

```text
Min‑heap priority queue keyed by NextRun.
Loop:
  now := time.Now()
  top := heap.Peek()
  wait := top.NextRun - now
  time.After(wait + randJitter())
  task := heap.Pop()
  if limiter.Allow(task.Endpoint) { taskChan <- task }
  task.NextRun = time.Now() + task.Interval
  heap.Push(task)
```

* **Per‑endpoint token bucket** (`x/time/rate`) prevents exhaustion.
  `rate.Every(Interval)` where `Interval = max(Endpoint.Schedule, globalMinInterval)`.
* **Jitter** (`rand.Float64()*0.1`) spreads load edges.
* **Backoff** – On ≥ 2 consecutive failures, double the interval (capped).

---

## 6 . WebSocket Hub

```go
// ws/hub.go (Observer pattern)
type Hub struct {
    register   chan *Client
    unregister chan *Client
    broadcast  chan ProbeResult
}

func (h *Hub) run() {
    for {
        select {
        case c := <-h.register:
            clients[c] = struct{}{}
        case c := <-h.unregister:
            delete(clients, c)
            close(c.send)
        case msg := <-h.broadcast:
            for c := range clients {
                select { case c.send <- msg: default: h.drop(c) }
            }
        }
    }
}
```

### Client Subscription Filter

Clients send:

```json
{ "action":"subscribe", "payer":["Aetna","Cigna"], "type":["login","api"] }
```

Hub stores a predicate closure `func(ProbeResult) bool` per client so broadcasting is `O(clients)` without reflection.

---

## 7 . Package Layout

```
health-ws/
 ├─ cmd/
 │   └─ server/
 │       └─ main.go
 ├─ internal/
 │   ├─ config        // YAML parsing & live reload
 │   ├─ scheduler     // heap, limiter, task structs
 │   ├─ prober        // interface & HTTP implementation
 │   ├─ ws            // hub, client, upgrader
 │   ├─ metrics       // Prometheus instrumentation
 │   └─ util          // helpers (retry, backoff, logging)
 ├─ api/              // OpenAPI & JSON schema for WS messages
 ├─ scripts/          // Docker, k8s manifests, makefile targets
 └─ docs/             // This spec, sequence diagrams, ADRs
```

---

## 8 . Key Third‑Party Dependencies

| Purpose            | Lib / Module                                                |
| ------------------ | ----------------------------------------------------------- |
| YAML config        | `gopkg.in/yaml.v3`                                          |
| WebSocket          | `nhooyr.io/websocket` **or** `github.com/gorilla/websocket` |
| Rate limiting      | `golang.org/x/time/rate`                                    |
| Metrics            | `github.com/prometheus/client_golang/prometheus`            |
| Scheduler heap     | stdlib `container/heap`                                     |
| Structured logging | `go.uber.org/zap`                                           |
| Graceful shutdown  | `golang.org/x/sync/errgroup`                                |

---

## 9 . Performance Techniques

* **HTTP Keep‑Alive & Connection Pool** – Single `http.Client` per hostname with tuned `Transport`.
* **Avoid goroutine leaks** – every worker uses `ctx` and returns on cancel.
* **Sync.Pool** for `ProbeResult` to reduce allocations.
* **Zero‑copy JSON** – Pre‑encode static fields; use `json.Encoder` with pooled `bytes.Buffer`.
* **Prom‑push path** on `/metrics` served via `net/http` default mux, separate from WS path.

---

## 10 . Security

* Force TLS (`wss://`) and HSTS headers.
* Optional JWT auth; verify `sub`, `exp`, allowed scopes.
* Tight read limits (`websocket.ReaderLimit`).
* CORS: Allow only known domains if running behind CDN.

---

## 11 . Testing Strategy

| Layer     | Technique                                                                                              |
| --------- | ------------------------------------------------------------------------------------------------------ |
| Scheduler | Deterministic fake clock (e.g., `clockwork`) + heap assertions                                         |
| Prober    | `httptest.Server` with scripted latencies & errors                                                     |
| Hub       | Property tests: given *n* clients and *m* messages, each predicate‑positive client receives *m*        |
| E2E       | Docker‑compose service with mocked endpoints; run `hey` to open 5 k WS clients and measure p95 latency |

---

## 12 . Deployment & Ops

* **Container** – Alpine image, scratch‑stage binary < 15 MB.
* **Helm chart** – HPA based on CPU and WebSocket connection count.
* **ConfigMap** – endpoints YAML; version control PR flow.
* **Prometheus & Grafana** – dashboards: error rate, probe latency histogram, active connections.

---

## 13 . Future Enhancements

* Adaptive probe interval (PID controller on failure/success).
* Persist last N probe results to Redis for replay to late‑joiners.
* gRPC‑stream mirror for backend microservices.
* Automatic PagerDuty alert on sustained 5xx from any endpoint.

---

## 14 . Minimal Example (`main.go` excerpt)

```go
package main

func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()

    cfg := config.MustLoad("endpoints.yaml")
    hub := ws.NewHub()
    go hub.Run()

    scheduler := scheduler.New(cfg.Endpoints, hub.Broadcast)
    g, ctx := errgroup.WithContext(ctx)
    g.Go(func() error { return scheduler.Start(ctx) })

    srv := &http.Server{
        Addr:         ":8080",
        Handler:      ws.NewRouter(hub), // registers /ws & /metrics
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 10 * time.Second,
    }
    g.Go(func() error { return srv.ListenAndServe() })
    g.Go(func() error {
        <-ctx.Done()
        ctxShut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        return srv.Shutdown(ctxShut)
    })
    log.Fatal(g.Wait())
}
```

---

### Ready to build ✨

This specification gives you **clear architecture, algorithms, packages, and exemplar code**—all optimized for performance, simplicity, and professional maintainability in Go.
