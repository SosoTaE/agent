package handlers

import (
	"context"
	"encoding/json"
	"fmt"
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

		case "get_stopped_customers":
			// Handle request for customers who want to talk to real person
			handleGetStoppedCustomers(conn, msg)

		case "assign_agent":
			// Handle agent assignment to customer
			handleAssignAgent(conn, msg)

		case "unassign_agent":
			// Handle agent unassignment from customer
			handleUnassignAgent(conn, msg)

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

	// Auto-assign agent if not already assigned
	if customer.AgentID == "" {
		updatedCustomer, assignErr := services.AssignAgentToCustomer(ctx, msg.CustomerID, msg.PageID,
			conn.UserID, conn.UserEmail, conn.UserName)
		if assignErr != nil {
			slog.Warn("Failed to auto-assign agent", "error", assignErr)
		} else {
			customer = updatedCustomer
			// Broadcast agent assignment
			wsManager := services.GetWebSocketManager()
			wsManager.BroadcastToCompany(conn.CompanyID, services.BroadcastMessage{
				CompanyID: conn.CompanyID,
				PageID:    msg.PageID,
				Type:      "agent_assignment_changed",
				Data: map[string]interface{}{
					"customer_id": msg.CustomerID,
					"customer":    customer,
					"agent_id":    conn.UserID,
					"agent_email": conn.UserEmail,
					"agent_name":  conn.UserName,
					"action":      "assigned",
					"timestamp":   time.Now().Unix(),
				},
			})
			slog.Info("Auto-assigned agent to customer on message send",
				"customerID", msg.CustomerID,
				"agentID", conn.UserID)
		}
	} else if customer.AgentID != conn.UserID {
		// Check if assigned to another agent
		sendWebSocketError(conn, fmt.Sprintf("Customer is already assigned to %s", customer.AgentEmail))
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

// handleGetStoppedCustomers handles requests for customers who want to talk to a real person
func handleGetStoppedCustomers(conn *services.WebSocketConnection, msg WebSocketMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Parse request parameters from Data field
	var params struct {
		PageID string `json:"page_id,omitempty"`
		Limit  int    `json:"limit,omitempty"`
		Skip   int    `json:"skip,omitempty"`
	}

	// Set defaults
	params.Limit = 50
	params.Skip = 0

	// Parse parameters if provided
	if msg.Data != nil {
		if err := json.Unmarshal(msg.Data, &params); err != nil {
			slog.Warn("Failed to parse stopped customers request params", "error", err)
		}
	}

	// Validate limit
	if params.Limit <= 0 || params.Limit > 200 {
		params.Limit = 50
	}

	// If page_id is provided, validate it belongs to the company
	if params.PageID != "" {
		_, err := services.ValidatePageOwnership(ctx, params.PageID, conn.CompanyID)
		if err != nil {
			sendWebSocketError(conn, "Page not found or access denied")
			return
		}
	}

	// Get stopped customers
	customers, totalCount, err := services.GetStoppedCustomers(ctx, conn.CompanyID, params.PageID, params.Limit, params.Skip)
	if err != nil {
		slog.Error("Failed to get stopped customers",
			"companyID", conn.CompanyID,
			"pageID", params.PageID,
			"error", err)
		sendWebSocketError(conn, "Failed to retrieve stopped customers")
		return
	}

	// Calculate time since last conversation for each customer
	type CustomerWithTimeSince struct {
		models.Customer
		TimeSinceLastConversation    string `json:"time_since_last_conversation"`
		MinutesSinceLastConversation int    `json:"minutes_since_last_conversation"`
		IsAssigned                   bool   `json:"is_assigned"`
		IsAssignedToMe               bool   `json:"is_assigned_to_me"`
	}

	customersWithTime := make([]CustomerWithTimeSince, 0, len(customers))
	now := time.Now()

	for _, customer := range customers {
		timeDiff := now.Sub(customer.UpdatedAt)
		minutes := int(timeDiff.Minutes())

		var timeStr string
		if minutes < 60 {
			timeStr = fmt.Sprintf("%d minutes ago", minutes)
		} else if minutes < 1440 { // Less than 24 hours
			hours := minutes / 60
			timeStr = fmt.Sprintf("%d hours ago", hours)
		} else {
			days := minutes / 1440
			timeStr = fmt.Sprintf("%d days ago", days)
		}

		isAssigned := customer.AgentID != ""
		isAssignedToMe := customer.AgentID == conn.UserID

		customersWithTime = append(customersWithTime, CustomerWithTimeSince{
			Customer:                     customer,
			TimeSinceLastConversation:    timeStr,
			MinutesSinceLastConversation: minutes,
			IsAssigned:                   isAssigned,
			IsAssignedToMe:               isAssignedToMe,
		})
	}

	// Send response
	response := map[string]interface{}{
		"type": "stopped_customers",
		"data": map[string]interface{}{
			"customers": customersWithTime,
			"pagination": map[string]interface{}{
				"total":    totalCount,
				"limit":    params.Limit,
				"skip":     params.Skip,
				"has_more": int64(params.Skip+params.Limit) < totalCount,
			},
			"page_id": params.PageID,
		},
		"timestamp": time.Now().Unix(),
	}

	if responseData, err := json.Marshal(response); err == nil {
		conn.Send <- responseData
	}

	slog.Info("Sent stopped customers to client",
		"companyID", conn.CompanyID,
		"count", len(customers),
		"total", totalCount)
}

// handleAssignAgent handles agent assignment to a customer
func handleAssignAgent(conn *services.WebSocketConnection, msg WebSocketMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Parse request parameters
	var params struct {
		CustomerID string `json:"customer_id"`
		PageID     string `json:"page_id"`
	}

	if msg.Data != nil {
		if err := json.Unmarshal(msg.Data, &params); err != nil {
			sendWebSocketError(conn, "Invalid request data")
			return
		}
	}

	if params.CustomerID == "" || params.PageID == "" {
		sendWebSocketError(conn, "customer_id and page_id are required")
		return
	}

	// Verify page belongs to company
	_, err := services.ValidatePageOwnership(ctx, params.PageID, conn.CompanyID)
	if err != nil {
		sendWebSocketError(conn, "Page not found or access denied")
		return
	}

	// Check if customer exists and needs help
	customer, err := services.GetCustomer(ctx, params.CustomerID, params.PageID)
	if err != nil || customer == nil {
		sendWebSocketError(conn, "Customer not found")
		return
	}

	if !customer.Stop {
		sendWebSocketError(conn, "Customer has not requested human assistance")
		return
	}

	// Check if customer is already assigned to another agent
	if customer.AgentID != "" && customer.AgentID != conn.UserID {
		sendWebSocketError(conn, fmt.Sprintf("Customer is already assigned to %s", customer.AgentEmail))
		return
	}

	// Assign the agent to the customer
	updatedCustomer, err := services.AssignAgentToCustomer(ctx, params.CustomerID, params.PageID,
		conn.UserID, conn.UserEmail, conn.UserName)
	if err != nil {
		slog.Error("Failed to assign agent to customer", "error", err)
		sendWebSocketError(conn, "Failed to assign agent")
		return
	}

	// Send success response to the requesting agent
	response := map[string]interface{}{
		"type": "agent_assigned",
		"data": map[string]interface{}{
			"customer":    updatedCustomer,
			"agent_id":    conn.UserID,
			"agent_email": conn.UserEmail,
			"agent_name":  conn.UserName,
		},
		"timestamp": time.Now().Unix(),
	}

	if responseData, err := json.Marshal(response); err == nil {
		conn.Send <- responseData
	}

	// Broadcast to all connected users in the company
	wsManager := services.GetWebSocketManager()
	wsManager.BroadcastToCompany(conn.CompanyID, services.BroadcastMessage{
		CompanyID: conn.CompanyID,
		PageID:    params.PageID,
		Type:      "agent_assignment_changed",
		Data: map[string]interface{}{
			"customer_id": params.CustomerID,
			"customer":    updatedCustomer,
			"agent_id":    conn.UserID,
			"agent_email": conn.UserEmail,
			"agent_name":  conn.UserName,
			"action":      "assigned",
			"timestamp":   time.Now().Unix(),
		},
	})

	slog.Info("Agent assigned to customer via WebSocket",
		"customerID", params.CustomerID,
		"pageID", params.PageID,
		"agentID", conn.UserID,
		"agentEmail", conn.UserEmail)
}

// handleUnassignAgent handles agent unassignment from a customer
func handleUnassignAgent(conn *services.WebSocketConnection, msg WebSocketMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Parse request parameters
	var params struct {
		CustomerID string `json:"customer_id"`
		PageID     string `json:"page_id"`
	}

	if msg.Data != nil {
		if err := json.Unmarshal(msg.Data, &params); err != nil {
			sendWebSocketError(conn, "Invalid request data")
			return
		}
	}

	if params.CustomerID == "" || params.PageID == "" {
		sendWebSocketError(conn, "customer_id and page_id are required")
		return
	}

	// Verify page belongs to company
	_, err := services.ValidatePageOwnership(ctx, params.PageID, conn.CompanyID)
	if err != nil {
		sendWebSocketError(conn, "Page not found or access denied")
		return
	}

	// Check if customer exists
	customer, err := services.GetCustomer(ctx, params.CustomerID, params.PageID)
	if err != nil || customer == nil {
		sendWebSocketError(conn, "Customer not found")
		return
	}

	// Check if the agent is actually assigned to this customer
	if customer.AgentID != conn.UserID {
		sendWebSocketError(conn, "You are not assigned to this customer")
		return
	}

	// Unassign the agent from the customer
	updatedCustomer, err := services.UnassignAgentFromCustomer(ctx, params.CustomerID, params.PageID)
	if err != nil {
		slog.Error("Failed to unassign agent from customer", "error", err)
		sendWebSocketError(conn, "Failed to unassign agent")
		return
	}

	// Send success response to the requesting agent
	response := map[string]interface{}{
		"type": "agent_unassigned",
		"data": map[string]interface{}{
			"customer": updatedCustomer,
		},
		"timestamp": time.Now().Unix(),
	}

	if responseData, err := json.Marshal(response); err == nil {
		conn.Send <- responseData
	}

	// Broadcast to all connected users in the company
	wsManager := services.GetWebSocketManager()
	wsManager.BroadcastToCompany(conn.CompanyID, services.BroadcastMessage{
		CompanyID: conn.CompanyID,
		PageID:    params.PageID,
		Type:      "agent_assignment_changed",
		Data: map[string]interface{}{
			"customer_id": params.CustomerID,
			"customer":    updatedCustomer,
			"action":      "unassigned",
			"timestamp":   time.Now().Unix(),
		},
	})

	slog.Info("Agent unassigned from customer via WebSocket",
		"customerID", params.CustomerID,
		"pageID", params.PageID,
		"agentID", conn.UserID,
		"agentEmail", conn.UserEmail)
}
