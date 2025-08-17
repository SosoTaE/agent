package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"facebook-bot/models"
	"facebook-bot/services"
)

// WebSocketMessage represents an incoming WebSocket message
type WebSocketMessage struct {
	Type       string          `json:"type"`
	CustomerID string          `json:"customer_id,omitempty"`
	PageID     string          `json:"page_id,omitempty"`
	Message    string          `json:"message,omitempty"`
	Data       json.RawMessage `json:"data,omitempty"`
}

// WebSocketUpgrade upgrades HTTP connection to WebSocket
func WebSocketUpgrade(c *fiber.Ctx) error {
	// Check if request is WebSocket upgrade
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

// HandleWebSocket handles WebSocket connections
func HandleWebSocket(c *websocket.Conn) {
	// Get company ID from locals (set by auth middleware)
	companyID, ok := c.Locals("company_id").(string)
	if !ok || companyID == "" {
		slog.Error("WebSocket connection without company ID")
		c.Close()
		return
	}

	// Get user information from session
	userEmail, _ := c.Locals("user_email").(string)
	userName, _ := c.Locals("user_name").(string)
	userID, _ := c.Locals("user_id").(string)

	// If no user ID from session, generate one
	if userID == "" {
		userID = uuid.New().String()
	}

	// Create connection object
	conn := &services.WebSocketConnection{
		Conn:      c,
		CompanyID: companyID,
		UserID:    userID,
		UserEmail: userEmail,
		UserName:  userName,
		Send:      make(chan []byte, 256),
	}

	// Register connection
	wsManager := services.GetWebSocketManager()
	wsManager.RegisterConnection(conn)
	defer wsManager.UnregisterConnection(companyID, userID)

	slog.Info("WebSocket connection established",
		"companyID", companyID,
		"userID", userID)

	// Send initial connection success message
	welcomeMsg := map[string]interface{}{
		"type":    "connected",
		"message": "WebSocket connection established",
		"user_id": userID,
	}
	if welcomeData, err := json.Marshal(welcomeMsg); err == nil {
		c.WriteMessage(websocket.TextMessage, welcomeData)
	}

	// Start goroutine to handle sending messages
	go handleWebSocketSend(conn)

	// Handle incoming messages
	handleWebSocketReceive(conn)
}

// handleWebSocketSend handles sending messages to the WebSocket client
func handleWebSocketSend(conn *services.WebSocketConnection) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		conn.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-conn.Send:
			conn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Channel closed
				conn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := conn.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				slog.Error("Failed to write WebSocket message", "error", err)
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			conn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleWebSocketReceive handles receiving messages from the WebSocket client
func handleWebSocketReceive(conn *services.WebSocketConnection) {
	defer func() {
		conn.Conn.Close()
	}()

	conn.Conn.SetReadLimit(512 * 1024) // 512KB max message size
	conn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.Conn.SetPongHandler(func(string) error {
		conn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, messageBytes, err := conn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("WebSocket read error", "error", err)
			}
			break
		}

		// Reset read deadline on successful read
		conn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		// Parse message
		var msg WebSocketMessage
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			slog.Error("Failed to parse WebSocket message", "error", err)
			continue
		}

		// Handle different message types
		switch msg.Type {
		case "ping":
			// Respond with pong
			pongMsg := map[string]string{"type": "pong"}
			if pongData, err := json.Marshal(pongMsg); err == nil {
				conn.Send <- pongData
			}

		case "subscribe":
			// Client subscribing to specific page updates
			slog.Info("WebSocket client subscribed",
				"companyID", conn.CompanyID,
				"pageID", msg.PageID)

		case "send_message":
			// Handle sending message from dashboard to customer
			handleDashboardMessage(conn, msg)

		default:
			slog.Warn("Unknown WebSocket message type",
				"type", msg.Type,
				"companyID", conn.CompanyID)
		}
	}
}

// handleDashboardMessage handles messages sent from dashboard to customers
func handleDashboardMessage(conn *services.WebSocketConnection, msg WebSocketMessage) {
	if msg.CustomerID == "" || msg.PageID == "" || msg.Message == "" {
		errorMsg := map[string]string{
			"type":  "error",
			"error": "Missing required fields: customer_id, page_id, and message",
		}
		if errorData, err := json.Marshal(errorMsg); err == nil {
			conn.Send <- errorData
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get company configuration
	company, err := services.GetCompanyByID(ctx, conn.CompanyID)
	if err != nil {
		slog.Error("Failed to get company", "error", err)
		sendWebSocketError(conn, "Failed to get company configuration")
		return
	}

	// Find the page configuration
	companyPages, err := services.GetPagesByCompanyID(ctx, company.CompanyID)
	if err != nil {
		sendWebSocketError(conn, "Failed to get company pages")
		return
	}

	var pageConfig *models.FacebookPage
	for _, page := range companyPages {
		if page.PageID == msg.PageID {
			p := page // Create a copy to avoid pointer to loop variable
			pageConfig = &p
			break
		}
	}

	if pageConfig == nil {
		sendWebSocketError(conn, "Page not found in company configuration")
		return
	}

	// Check if customer has stop=true (wants real person)
	customer, err := services.GetCustomer(ctx, msg.CustomerID, msg.PageID)
	if err != nil || customer == nil {
		sendWebSocketError(conn, "Customer not found")
		return
	}

	if !customer.Stop {
		sendWebSocketError(conn, "Customer has not requested human assistance")
		return
	}

	// Send message via Facebook Messenger
	if err := services.SendMessengerReply(ctx, msg.CustomerID, msg.Message, pageConfig.PageAccessToken); err != nil {
		slog.Error("Failed to send message to customer", "error", err)
		sendWebSocketError(conn, "Failed to send message to customer")
		return
	}

	// Save the message to database (from dashboard/human agent)
	messageDoc := &models.Message{
		Type:        "chat",
		ChatID:      msg.CustomerID,
		SenderID:    msg.PageID,     // Page is the sender
		RecipientID: msg.CustomerID, // Customer is the recipient
		PageID:      msg.PageID,
		PageName:    pageConfig.PageName,
		Message:     msg.Message,
		IsBot:       false,
		IsHuman:     true,        // Mark as human response
		Source:      "dashboard", // Mark source as dashboard
		AgentID:     conn.UserID,
		AgentEmail:  conn.UserEmail,
		AgentName:   conn.UserName,
		Timestamp:   time.Now(),
	}

	if err := services.SaveMessage(ctx, messageDoc); err != nil {
		slog.Error("Failed to save dashboard message", "error", err)
	}

	// Send success response
	successMsg := map[string]interface{}{
		"type":        "message_sent",
		"customer_id": msg.CustomerID,
		"page_id":     msg.PageID,
		"message":     msg.Message,
		"timestamp":   time.Now().Unix(),
	}

	if successData, err := json.Marshal(successMsg); err == nil {
		conn.Send <- successData
	}

	// Broadcast the message to all connected dashboard users
	wsManager := services.GetWebSocketManager()
	wsManager.BroadcastToCompany(conn.CompanyID, services.BroadcastMessage{
		CompanyID: conn.CompanyID,
		PageID:    msg.PageID,
		Type:      "new_message",
		Data: map[string]interface{}{
			"chat_id":      msg.CustomerID,
			"sender_id":    msg.PageID,
			"recipient_id": msg.CustomerID,
			"message":      msg.Message,
			"is_bot":       false,
			"is_human":     true,
			"agent_id":     conn.UserID,
			"agent_email":  conn.UserEmail,
			"agent_name":   conn.UserName,
			"timestamp":    time.Now().Unix(),
		},
	})

	slog.Info("Dashboard message sent successfully",
		"customerID", msg.CustomerID,
		"pageID", msg.PageID,
		"companyID", conn.CompanyID)
}

// sendWebSocketError sends an error message to the WebSocket client
func sendWebSocketError(conn *services.WebSocketConnection, errorMessage string) {
	errorMsg := map[string]string{
		"type":  "error",
		"error": errorMessage,
	}
	if errorData, err := json.Marshal(errorMsg); err == nil {
		conn.Send <- errorData
	}
}
