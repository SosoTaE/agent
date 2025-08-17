package services

import (
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
)

// WebSocket errors
var (
	ErrConnectionNotFound   = errors.New("connection not found")
	ErrConnectionBufferFull = errors.New("connection buffer full")
)

// WebSocketManager manages WebSocket connections
type WebSocketManager struct {
	// Map of company ID to map of connection ID to connection
	connections map[string]map[string]*WebSocketConnection
	mu          sync.RWMutex
	broadcast   chan BroadcastMessage
}

// WebSocketConnection represents a single WebSocket connection
type WebSocketConnection struct {
	Conn      *websocket.Conn
	CompanyID string
	UserID    string
	UserEmail string
	UserName  string
	Send      chan []byte
}

// BroadcastMessage represents a message to broadcast
type BroadcastMessage struct {
	CompanyID string
	PageID    string
	Type      string
	Data      interface{}
}

// MessagePayload represents the structure of WebSocket messages
type MessagePayload struct {
	Type      string      `json:"type"`
	PageID    string      `json:"page_id,omitempty"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

var wsManager *WebSocketManager
var once sync.Once

// GetWebSocketManager returns the singleton WebSocket manager
func GetWebSocketManager() *WebSocketManager {
	once.Do(func() {
		wsManager = &WebSocketManager{
			connections: make(map[string]map[string]*WebSocketConnection),
			broadcast:   make(chan BroadcastMessage, 100),
		}
		go wsManager.handleBroadcast()
	})
	return wsManager
}

// RegisterConnection registers a new WebSocket connection
func (m *WebSocketManager) RegisterConnection(conn *WebSocketConnection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.connections[conn.CompanyID] == nil {
		m.connections[conn.CompanyID] = make(map[string]*WebSocketConnection)
	}

	m.connections[conn.CompanyID][conn.UserID] = conn

	slog.Info("WebSocket connection registered",
		"companyID", conn.CompanyID,
		"userID", conn.UserID,
		"totalConnections", len(m.connections[conn.CompanyID]))
}

// UnregisterConnection removes a WebSocket connection
func (m *WebSocketManager) UnregisterConnection(companyID, userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if companyConns, exists := m.connections[companyID]; exists {
		if conn, exists := companyConns[userID]; exists {
			close(conn.Send)
			delete(companyConns, userID)

			slog.Info("WebSocket connection unregistered",
				"companyID", companyID,
				"userID", userID,
				"remainingConnections", len(companyConns))

			// Clean up empty company map
			if len(companyConns) == 0 {
				delete(m.connections, companyID)
			}
		}
	}
}

// BroadcastToCompany sends a message to all connections for a company
func (m *WebSocketManager) BroadcastToCompany(companyID string, message BroadcastMessage) {
	m.broadcast <- message
}

// handleBroadcast processes broadcast messages
func (m *WebSocketManager) handleBroadcast() {
	for message := range m.broadcast {
		m.mu.RLock()
		companyConns, exists := m.connections[message.CompanyID]
		m.mu.RUnlock()

		if !exists {
			continue
		}

		// Create payload
		payload := MessagePayload{
			Type:      message.Type,
			PageID:    message.PageID,
			Data:      message.Data,
			Timestamp: getCurrentTimestamp(),
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			slog.Error("Failed to marshal WebSocket message", "error", err)
			continue
		}

		// Send to all connections for this company
		m.mu.RLock()
		for _, conn := range companyConns {
			select {
			case conn.Send <- jsonData:
				// Message sent successfully
			default:
				// Connection buffer full, skip
				slog.Warn("WebSocket connection buffer full",
					"companyID", message.CompanyID,
					"userID", conn.UserID)
			}
		}
		m.mu.RUnlock()
	}
}

// SendToConnection sends a message to a specific connection
func (m *WebSocketManager) SendToConnection(companyID, userID string, data []byte) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if companyConns, exists := m.connections[companyID]; exists {
		if conn, exists := companyConns[userID]; exists {
			select {
			case conn.Send <- data:
				return nil
			default:
				return ErrConnectionBufferFull
			}
		}
	}
	return ErrConnectionNotFound
}

// GetConnectionCount returns the number of active connections for a company
func (m *WebSocketManager) GetConnectionCount(companyID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if companyConns, exists := m.connections[companyID]; exists {
		return len(companyConns)
	}
	return 0
}

func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}
