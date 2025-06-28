package hub

import (
	"context"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"

	"payer-status-io/internal/config"
)

// Client represents a WebSocket client connection
type Client struct {
	id        string
	conn      *websocket.Conn
	send      chan *config.ProbeResult
	predicate func(*config.ProbeResult) bool
	logger    *zap.Logger
	hub       *Hub
}

// Hub manages WebSocket connections and broadcasts
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan *config.ProbeResult
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	logger     *zap.Logger
}

// SubscriptionRequest represents a client subscription filter
type SubscriptionRequest struct {
	Action string   `json:"action"` // "subscribe"
	Payers []string `json:"payer,omitempty"`
	Types  []string `json:"type,omitempty"`
}

// New creates a new WebSocket hub
func New(logger *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *config.ProbeResult, 1000), // Buffered for back-pressure
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run(ctx context.Context) error {
	h.logger.Info("Starting WebSocket hub")
	
	for {
		select {
		case <-ctx.Done():
			h.logger.Info("WebSocket hub stopping due to context cancellation")
			h.closeAllClients()
			return ctx.Err()

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Info("Client registered", zap.String("client_id", client.id))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			h.logger.Info("Client unregistered", zap.String("client_id", client.id))

		case result := <-h.broadcast:
			h.broadcastToClients(result)
		}
	}
}

// Broadcast sends a probe result to the broadcast channel
func (h *Hub) Broadcast(result *config.ProbeResult) {
	select {
	case h.broadcast <- result:
		// Successfully queued for broadcast
	default:
		// Channel full, drop message (back-pressure handling)
		h.logger.Warn("Broadcast channel full, dropping message",
			zap.String("payer", result.Payer),
			zap.String("type", result.Type))
	}
}

// HandleWebSocket handles new WebSocket connections
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Accept WebSocket connection with security options
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"}, // Configure based on your CORS requirements
		Subprotocols:   []string{"payer-status-v1"},
	})
	if err != nil {
		h.logger.Error("Failed to accept WebSocket connection", zap.Error(err))
		return
	}

	// Set read limit for DoS protection (as per .windsurfrules)
	conn.SetReadLimit(1024) // 1KB limit for subscription messages

	client := &Client{
		id:        generateClientID(),
		conn:      conn,
		send:      make(chan *config.ProbeResult, 256),
		predicate: func(*config.ProbeResult) bool { return true }, // Accept all by default
		logger:    h.logger,
		hub:       h,
	}

	// Register client
	h.register <- client

	// Start client goroutines
	go client.writePump(context.Background())
	go client.readPump(context.Background())
}

// broadcastToClients sends a result to all matching clients
func (h *Hub) broadcastToClients(result *config.ProbeResult) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.predicate(result) {
			select {
			case client.send <- result:
				// Successfully sent
			default:
				// Client's send channel is full, close it (back-pressure handling)
				h.logger.Warn("Client send channel full, closing connection",
					zap.String("client_id", client.id))
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}

// closeAllClients closes all client connections
func (h *Hub) closeAllClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients {
		close(client.send)
		client.conn.Close(websocket.StatusGoingAway, "Server shutting down")
	}
	h.clients = make(map[*Client]bool)
}

// GetStats returns hub statistics
func (h *Hub) GetStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return map[string]interface{}{
		"active_clients":     len(h.clients),
		"broadcast_chan_len": len(h.broadcast),
		"broadcast_chan_cap": cap(h.broadcast),
	}
}

const (
	writeWait      = 30 * time.Second   // Increased write timeout
	pongWait       = 24 * time.Hour     // 24 hour read timeout for persistent connections
	pingPeriod     = (pongWait * 9) / 10 // Ping every ~21.6 hours
	maxMessageSize = 512
)

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump(ctx context.Context) {
	defer func() {
		c.conn.Close(websocket.StatusInternalError, "write pump closed")
		c.hub.unregister <- c
	}()

	pingTicker := time.NewTicker(pingPeriod)
	defer pingTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.hub.logger.Debug("Write pump stopping due to context cancellation", zap.String("client_id", c.id))
			return

		case message, ok := <-c.send:
			if !ok {
				c.hub.logger.Debug("Send channel closed", zap.String("client_id", c.id))
				return
			}

			// Create write context with timeout
			writeCtx, cancel := context.WithTimeout(ctx, writeWait)
			if err := wsjson.Write(writeCtx, c.conn, message); err != nil {
				c.hub.logger.Error("Error writing message", zap.Error(err), zap.String("client_id", c.id))
				cancel()
				return
			}
			cancel()

		case <-pingTicker.C:
			// Send ping
			pingCtx, cancel := context.WithTimeout(ctx, writeWait)
			if err := c.conn.Ping(pingCtx); err != nil {
				c.hub.logger.Error("Error sending ping", zap.Error(err), zap.String("client_id", c.id))
				cancel()
				return
			}
			cancel()
		}
	}
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump(ctx context.Context) {
	defer func() {
		c.conn.Close(websocket.StatusInternalError, "read pump closed")
		c.hub.unregister <- c
	}()

	for {
		select {
		case <-ctx.Done():
			c.hub.logger.Debug("Read pump stopping due to context cancellation", zap.String("client_id", c.id))
			return

		default:
			// Create read context with timeout
			readCtx, cancel := context.WithTimeout(ctx, pongWait)

			// Read message (for subscription updates)
			var msg map[string]interface{}
			if err := wsjson.Read(readCtx, c.conn, &msg); err != nil {
				cancel()
				if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
					websocket.CloseStatus(err) == websocket.StatusGoingAway {
					c.hub.logger.Debug("WebSocket closed normally", zap.String("client_id", c.id))
				} else {
					c.hub.logger.Error("Error reading message", zap.Error(err), zap.String("client_id", c.id))
				}
				return
			}
			cancel()

			// Handle subscription updates
			c.handleSubscriptionUpdate(msg)
		}
	}
}

func (c *Client) handleSubscriptionUpdate(msg map[string]interface{}) {
	// Simple manual parsing instead of mapstructure
	action, ok := msg["action"].(string)
	if !ok || action != "subscribe" {
		return
	}

	var payers, types []string
	if payersRaw, ok := msg["payers"]; ok {
		if payersSlice, ok := payersRaw.([]interface{}); ok {
			for _, p := range payersSlice {
				if payer, ok := p.(string); ok {
					payers = append(payers, payer)
				}
			}
		}
	}

	if typesRaw, ok := msg["types"]; ok {
		if typesSlice, ok := typesRaw.([]interface{}); ok {
			for _, t := range typesSlice {
				if msgType, ok := t.(string); ok {
					types = append(types, msgType)
				}
			}
		}
	}

	// Update predicate
	c.predicate = c.createPredicate(payers, types)
	c.logger.Info("Client subscription updated",
		zap.String("client_id", c.id),
		zap.Strings("payers", payers),
		zap.Strings("types", types))
}

// createPredicate creates a filter function based on subscription criteria
func (c *Client) createPredicate(payers, types []string) func(*config.ProbeResult) bool {
	return func(result *config.ProbeResult) bool {
		// If no filters specified, accept all
		if len(payers) == 0 && len(types) == 0 {
			return true
		}

		// Check payer filter
		if len(payers) > 0 {
			payerMatch := false
			for _, payer := range payers {
				if result.Payer == payer {
					payerMatch = true
					break
				}
			}
			if !payerMatch {
				return false
			}
		}

		// Check type filter
		if len(types) > 0 {
			typeMatch := false
			for _, typ := range types {
				if result.Type == typ {
					typeMatch = true
					break
				}
			}
			if !typeMatch {
				return false
			}
		}

		return true
	}
}

// generateClientID generates a unique client identifier
func generateClientID() string {
	return time.Now().Format("20060102150405") + "-" + 
		   string(rune('A' + time.Now().Nanosecond()%26))
}
