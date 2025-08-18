package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"facebook-bot/models"
	"facebook-bot/services"
)

// Messaging represents a messaging event (moved from webhooks to avoid import cycle)
type Messaging struct {
	Sender    User     `json:"sender"`
	Recipient User     `json:"recipient"`
	Timestamp int64    `json:"timestamp"`
	Message   *Message `json:"message,omitempty"`
}

// User represents a Facebook user
type User struct {
	ID string `json:"id"`
}

// Message represents a message
type Message struct {
	MID         string       `json:"mid"`
	Text        string       `json:"text"`
	QuickReply  *QuickReply  `json:"quick_reply,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// QuickReply represents a quick reply
type QuickReply struct {
	Payload string `json:"payload"`
}

// Attachment represents a message attachment
type Attachment struct {
	Type    string  `json:"type"`
	Payload Payload `json:"payload"`
}

// Payload represents attachment payload
type Payload struct {
	URL string `json:"url"`
}

// HandleMessage processes incoming messages
func HandleMessage(messaging Messaging, pageID string) {
	// Increase timeout to 60 seconds for Claude API calls
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Hour)
	defer cancel()

	senderID := messaging.Sender.ID
	messageText := messaging.Message.Text

	// Get company configuration by page ID
	company, err := services.GetCompanyByPageID(ctx, pageID)
	if err != nil {
		slog.Error("Failed to get company configuration", "error", err, "pageID", pageID)
		return
	}

	// Get specific page configuration
	pageConfig, err := services.GetPageConfig(company, pageID)
	if err != nil {
		slog.Error("Failed to get page configuration", "error", err, "pageID", pageID)
		return
	}

	slog.Info("Handling message",
		"senderID", senderID,
		"pageID", pageID,
		"pageName", pageConfig.PageName,
		"companyID", company.CompanyID,
		"message", messageText,
	)

	// Fetch user details (first name and last name) from Facebook synchronously
	var firstName, lastName, senderName string
	userDetails, err := services.GetFacebookUserDetails(ctx, senderID, pageConfig.PageAccessToken)
	if err != nil {
		slog.Warn("Failed to fetch Facebook user details",
			"senderID", senderID,
			"error", err)
		// Use fallback name when Facebook API doesn't return user details
		senderName = fmt.Sprintf("User %s", senderID[:min(8, len(senderID))])
	} else if userDetails != nil {
		firstName = userDetails.FirstName
		lastName = userDetails.LastName
		// Create sender name from first and last name
		if firstName != "" || lastName != "" {
			if firstName != "" && lastName != "" {
				senderName = firstName + " " + lastName
			} else if firstName != "" {
				senderName = firstName
			} else {
				senderName = lastName
			}
		} else {
			// If no names available, use fallback
			senderName = fmt.Sprintf("User %s", senderID[:min(8, len(senderID))])
		}
	} else {
		// No user details returned, use fallback
		senderName = fmt.Sprintf("User %s", senderID[:min(8, len(senderID))])
	}

	// Also update existing records asynchronously
	go func() {
		updateCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if firstName != "" || lastName != "" {
			if err := services.UpdateUserNameInDatabase(updateCtx, senderID, firstName, lastName); err != nil {
				slog.Warn("Failed to update user names in database",
					"senderID", senderID,
					"error", err)
			}
		}
	}()

	// CRM message processing has been removed

	// Save or update customer in customers collection
	if err := services.SaveOrUpdateCustomer(ctx, senderID, senderName, firstName, lastName,
		pageID, pageConfig.PageName, company.CompanyID, messageText); err != nil {
		slog.Error("Failed to save/update customer", "error", err)
	}

	// Check if customer has stop=true (wants human assistance only)
	customer, err := services.GetCustomer(ctx, senderID, pageID)
	if err != nil {
		slog.Warn("Failed to get customer status", "error", err)
	}

	// If customer has stop=true, skip bot processing
	if customer != nil && customer.Stop {
		slog.Info("Customer has stop=true, skipping bot response",
			"customerID", senderID,
			"pageID", pageID)

		// Still save the message to database
		messageDoc := &models.Message{
			Type:        "chat",
			ChatID:      senderID,
			SenderID:    senderID,
			SenderName:  senderName,
			FirstName:   firstName,
			LastName:    lastName,
			RecipientID: pageID,
			PageID:      pageID,
			PageName:    pageConfig.PageName,
			Message:     messageText,
			IsBot:       false,
			Timestamp:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := services.SaveMessage(ctx, messageDoc); err != nil {
			slog.Error("Failed to save user message", "error", err)
		}

		// Broadcast the message via WebSocket for dashboard monitoring
		wsManager := services.GetWebSocketManager()
		wsManager.BroadcastToCompany(company.CompanyID, services.BroadcastMessage{
			CompanyID: company.CompanyID,
			PageID:    pageID,
			Type:      "new_message",
			Data: map[string]interface{}{
				"chat_id":        senderID,
				"sender_id":      senderID,
				"sender_name":    senderName,
				"recipient_id":   pageID,
				"message":        messageText,
				"is_bot":         false,
				"requires_human": true,
				"timestamp":      time.Now().Unix(),
			},
		})

		// Exit early - don't process with bot
		return
	}

	// Save user's message to database with first and last name
	messageDoc := &models.Message{
		Type:        "chat",
		ChatID:      senderID, // Always use customer ID as chat_id
		SenderID:    senderID,
		SenderName:  senderName,
		FirstName:   firstName,
		LastName:    lastName,
		RecipientID: pageID, // User is sending to the page
		PageID:      pageID,
		PageName:    pageConfig.PageName,
		Message:     messageText,
		IsBot:       false,
		Source:      "facebook", // Mark source as facebook
		Timestamp:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := services.SaveMessage(ctx, messageDoc); err != nil {
		slog.Error("Failed to save user message", "error", err)
	}

	// Broadcast incoming message to WebSocket clients
	wsManager := services.GetWebSocketManager()
	wsManager.BroadcastToCompany(company.CompanyID, services.BroadcastMessage{
		CompanyID: company.CompanyID,
		PageID:    pageID,
		Type:      "new_message",
		Data: map[string]interface{}{
			"chat_id":      senderID,
			"sender_id":    senderID,
			"sender_name":  senderName,
			"recipient_id": pageID,
			"message":      messageText,
			"is_bot":       false,
			"timestamp":    time.Now().Unix(),
		},
	})

	// Fetch chat history for context (limit to 5 messages to prevent timeouts)
	chatHistory, err := services.GetChatHistory(ctx, senderID, pageID, 5)
	if err != nil {
		slog.Warn("Failed to fetch chat history", "error", err)
		// Continue without history if fetch fails
		chatHistory = []services.ChatHistory{}
	}

	slog.Info("Fetched chat history",
		"count", len(chatHistory),
		"senderID", senderID,
		"pageID", pageID,
	)

	// Fetch RAG context if enabled for this company and has active CRM links
	var ragContext string
	hasActiveCRMLinks := false
	for _, crmLink := range pageConfig.CRMLinks {
		if crmLink.IsActive {
			hasActiveCRMLinks = true
			break
		}
	}

	if hasActiveCRMLinks {
		// Get relevant context based on the user's message, filtered by page ID
		ragContext, err = services.GetRAGContext(ctx, messageText, company.CompanyID, pageID)
		if err != nil {
			slog.Warn("Failed to fetch RAG context", "error", err)
			// Continue without RAG context
		} else if ragContext != "" {
			slog.Info("RAG context retrieved",
				"contextLength", len(ragContext),
				"companyID", company.CompanyID,
				"pageID", pageID,
			)
		}
	} else if len(pageConfig.CRMLinks) > 0 {
		slog.Info("CRM links exist but none are active, skipping RAG context",
			"pageID", pageID,
			"totalCRMLinks", len(pageConfig.CRMLinks),
		)
	}

	// Get AI response from Claude with tool use for agent detection
	aiResponse, wantsAgent, err := services.GetClaudeResponseWithToolUse(ctx, messageText, "chat", company, pageConfig, chatHistory, ragContext)
	if err != nil {
		slog.Error("Failed to get Claude response", "error", err)
		aiResponse = "I apologize, but I'm having trouble processing your message right now. Please try again later."
		wantsAgent = false
	}

	// Check if tool detected that customer wants to talk to a real person
	if wantsAgent || strings.Contains(aiResponse, "CUSTOMER_WANTS_REAL_PERSON||") {

		// Update customer's stop status
		updatedCustomer, err := services.UpdateCustomerStopStatus(ctx, senderID, pageID, true)
		if err != nil {
			slog.Error("Failed to update customer stop status", "error", err)
		} else {
			slog.Info("Customer marked as wanting real person assistance (via tool)",
				"customerID", senderID,
				"pageID", pageID)
		}

		// Append a message to the response indicating human assistance will be provided
		aiResponse = aiResponse + "\n\nI've notified our human support team. Someone will assist you shortly."

		// Broadcast notification about human assistance request
		wsManager.BroadcastToCompany(company.CompanyID, services.BroadcastMessage{
			CompanyID: company.CompanyID,
			PageID:    pageID,
			Type:      "agent_requested",
			Data: map[string]interface{}{
				"chat_id":       senderID,
				"customer_name": senderName,
				"message":       messageText,
				"timestamp":     time.Now().Unix(),
			},
		})

		// Also broadcast the customer status update
		if updatedCustomer != nil {
			wsManager.BroadcastToCompany(company.CompanyID, services.BroadcastMessage{
				CompanyID: company.CompanyID,
				PageID:    pageID,
				Type:      "customer_stop_status_changed",
				Data: map[string]interface{}{
					"customer":  updatedCustomer,
					"stop":      true,
					"timestamp": time.Now().Unix(),
				},
			})
		}
	}

	// Response delay removed for faster processing
	// Send response back via Messenger immediately
	if err := services.SendMessengerReply(ctx, senderID, aiResponse, pageConfig.PageAccessToken); err != nil {
		slog.Error("Failed to send messenger reply", "error", err)
	}

	// Save bot's response as a message in the database (only if not empty)
	if aiResponse != "" {
		botMessageDoc := &models.Message{
			Type:        "chat",
			ChatID:      senderID, // Always use customer ID as chat_id, even for bot messages
			SenderID:    pageID,   // Bot's sender ID is the page ID
			RecipientID: senderID, // Bot is sending to the user
			PageID:      pageID,
			PageName:    pageConfig.PageName,
			Message:     aiResponse,
			IsBot:       true,
			Source:      "bot", // Mark source as bot
			Timestamp:   time.Now(),
		}

		if err := services.SaveMessage(ctx, botMessageDoc); err != nil {
			slog.Error("Failed to save bot message", "error", err)
		}
	}

	// Broadcast bot response to WebSocket clients
	wsManager.BroadcastToCompany(company.CompanyID, services.BroadcastMessage{
		CompanyID: company.CompanyID,
		PageID:    pageID,
		Type:      "new_message",
		Data: map[string]interface{}{
			"chat_id":      senderID,
			"sender_id":    pageID,
			"sender_name":  pageConfig.PageName,
			"recipient_id": senderID,
			"message":      aiResponse,
			"is_bot":       true,
			"timestamp":    time.Now().Unix(),
		},
	})

	// Save response to database for analytics
	responseDoc := &models.Response{
		Type:      "chat",
		SenderID:  senderID,
		PageID:    pageID,
		PageName:  pageConfig.PageName,
		Original:  messageText,
		Response:  aiResponse,
		Timestamp: time.Now(),
	}

	if err := services.SaveResponse(ctx, responseDoc); err != nil {
		slog.Error("Failed to save response", "error", err)
	}
}

// GetAllMessagesByPage retrieves all messages for a specific page with pagination
func GetAllMessagesByPage(c *fiber.Ctx) error {
	// Get page_id from URL params
	pageID := c.Params("pageID")
	if pageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Page ID is required",
		})
	}

	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Get pagination parameters
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)
	if limit > 200 {
		limit = 200
	}
	skip := (page - 1) * limit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify page belongs to company
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Check if page belongs to company
	pageFound := false
	var pageName string
	companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
	for _, p := range companyPages {
		if p.PageID == pageID {
			pageFound = true
			pageName = p.PageName
			break
		}
	}
	if !pageFound {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Page not found or access denied",
		})
	}

	// Build filter for messages collection
	filter := bson.M{
		"page_id": pageID,
	}

	db := services.GetDatabase()
	collection := db.Collection("messages")

	// Get total count
	totalCount, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		slog.Error("Failed to count messages", "error", err)
		totalCount = 0
	}

	// Get messages with pagination
	findOptions := options.Find().
		SetSort(bson.M{"timestamp": -1}).
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve messages",
		})
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	if err := cursor.All(ctx, &messages); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to decode messages",
		})
	}

	// Calculate pagination info
	totalPages := (int(totalCount) + limit - 1) / limit
	hasMore := page < totalPages

	return c.JSON(fiber.Map{
		"page_id":   pageID,
		"page_name": pageName,
		"messages":  messages,
		"pagination": fiber.Map{
			"page":        page,
			"limit":       limit,
			"total":       totalCount,
			"total_pages": totalPages,
			"has_more":    hasMore,
		},
	})
}

// GetCustomerConversation retrieves a customer's conversation with the bot to reconstruct chat
func GetCustomerConversation(c *fiber.Ctx) error {
	// Get customer_id from URL params
	customerID := c.Params("customerID")
	if customerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Customer ID is required",
		})
	}

	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Page ID is required for customer conversations
	pageID := c.Query("page_id")
	if pageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Page ID is required for customer conversations",
		})
	}

	// Get pagination parameters
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)
	if limit > 200 {
		limit = 200
	}
	skip := (page - 1) * limit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get company to verify page ownership
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Verify page belongs to company
	pageFound := false
	var pageName string
	companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
	for _, p := range companyPages {
		if p.PageID == pageID {
			pageFound = true
			pageName = p.PageName
			break
		}
	}
	if !pageFound {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Page not found or access denied",
		})
	}

	// Build filter to get conversation using chat_id
	// Since chat_id is always the customer ID, we can simply filter by it
	filter := bson.M{
		"chat_id": customerID,
		"page_id": pageID,
	}

	db := services.GetDatabase()
	collection := db.Collection("messages")

	// Get total count
	totalCount, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		slog.Error("Failed to count messages", "error", err)
		totalCount = 0
	}

	// Get messages with pagination
	findOptions := options.Find().
		SetSort(bson.M{"timestamp": -1}).
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve messages",
		})
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	if err := cursor.All(ctx, &messages); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to decode messages",
		})
	}

	// Get customer info from the first message
	var customerName string
	if len(messages) > 0 {
		for _, msg := range messages {
			if msg.SenderID == customerID && msg.SenderName != "" {
				customerName = msg.SenderName
				break
			}
		}
	}

	// Organize messages into conversation format
	conversation := make([]fiber.Map, 0, len(messages))
	for _, msg := range messages {
		msgData := fiber.Map{
			"id":           msg.ID.Hex(),
			"message":      msg.Message,
			"timestamp":    msg.Timestamp,
			"is_bot":       msg.IsBot,
			"sender_id":    msg.SenderID,
			"recipient_id": msg.RecipientID,
		}

		// Determine if this is from customer or bot
		if msg.SenderID == customerID {
			msgData["type"] = "customer"
			msgData["sender_name"] = msg.SenderName
		} else {
			msgData["type"] = "bot"
			msgData["sender_name"] = pageName
		}

		conversation = append(conversation, msgData)
	}

	// Calculate pagination info
	totalPages := (int(totalCount) + limit - 1) / limit
	hasMore := page < totalPages

	return c.JSON(fiber.Map{
		"customer_id":   customerID,
		"customer_name": customerName,
		"page_id":       pageID,
		"page_name":     pageName,
		"conversation":  conversation,
		"pagination": fiber.Map{
			"page":        page,
			"limit":       limit,
			"total":       totalCount,
			"total_pages": totalPages,
			"has_more":    hasMore,
		},
	})
}

// GetPageConversations retrieves all unique conversations for a page
func GetPageConversations(c *fiber.Ctx) error {
	// Get page_id from URL params
	pageID := c.Params("pageID")
	if pageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Page ID is required",
		})
	}

	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Get pagination parameters
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100
	}
	skip := (page - 1) * limit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify page belongs to company
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Check if page belongs to company
	pageFound := false
	var pageName string
	companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
	for _, p := range companyPages {
		if p.PageID == pageID {
			pageFound = true
			pageName = p.PageName
			break
		}
	}
	if !pageFound {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Page not found or access denied",
		})
	}

	db := services.GetDatabase()
	collection := db.Collection("messages")

	// Aggregate to get unique conversations (unique chat_ids) with last message
	pipeline := []bson.M{
		// Match messages for this page
		{"$match": bson.M{"page_id": pageID}},
		// Sort by timestamp descending
		{"$sort": bson.M{"timestamp": -1}},
		// Group by chat_id to get unique conversations
		{"$group": bson.M{
			"_id":            "$chat_id",
			"last_message":   bson.M{"$first": "$message"},
			"last_timestamp": bson.M{"$first": "$timestamp"},
			"customer_name":  bson.M{"$first": "$sender_name"},
			"first_name":     bson.M{"$first": "$first_name"},
			"last_name":      bson.M{"$first": "$last_name"},
			"message_count":  bson.M{"$sum": 1},
			"last_is_bot":    bson.M{"$first": "$is_bot"},
		}},
		// Sort by last message timestamp
		{"$sort": bson.M{"last_timestamp": -1}},
		// Pagination
		{"$skip": skip},
		{"$limit": limit},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve conversations",
		})
	}
	defer cursor.Close(ctx)

	var conversations []fiber.Map
	for cursor.Next(ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			continue
		}

		conversation := fiber.Map{
			"chat_id":        result["_id"],
			"customer_name":  result["customer_name"],
			"first_name":     result["first_name"],
			"last_name":      result["last_name"],
			"last_message":   result["last_message"],
			"last_timestamp": result["last_timestamp"],
			"message_count":  result["message_count"],
			"last_is_bot":    result["last_is_bot"],
		}
		conversations = append(conversations, conversation)
	}

	// Get total count of unique conversations
	countPipeline := []bson.M{
		{"$match": bson.M{"page_id": pageID}},
		{"$group": bson.M{"_id": "$chat_id"}},
		{"$count": "total"},
	}

	countCursor, err := collection.Aggregate(ctx, countPipeline)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to count conversations",
		})
	}
	defer countCursor.Close(ctx)

	var totalCount int64
	if countCursor.Next(ctx) {
		var countResult bson.M
		if err := countCursor.Decode(&countResult); err == nil {
			if total, ok := countResult["total"].(int32); ok {
				totalCount = int64(total)
			} else if total, ok := countResult["total"].(int64); ok {
				totalCount = total
			}
		}
	}

	// Calculate pagination info
	totalPages := (int(totalCount) + limit - 1) / limit
	hasMore := page < totalPages

	return c.JSON(fiber.Map{
		"page_id":       pageID,
		"page_name":     pageName,
		"conversations": conversations,
		"pagination": fiber.Map{
			"page":        page,
			"limit":       limit,
			"total":       totalCount,
			"total_pages": totalPages,
			"has_more":    hasMore,
		},
	})
}

// GetChatIDs retrieves all unique chat IDs for a page
func GetChatIDs(c *fiber.Ctx) error {
	// Get page_id from URL params
	pageID := c.Params("pageID")
	if pageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Page ID is required",
		})
	}

	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Get pagination parameters
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)
	if limit > 200 {
		limit = 200
	}
	skip := (page - 1) * limit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify page belongs to company
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Check if page belongs to company
	pageFound := false
	var pageName string
	companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
	for _, p := range companyPages {
		if p.PageID == pageID {
			pageFound = true
			pageName = p.PageName
			break
		}
	}
	if !pageFound {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Page not found or access denied",
		})
	}

	db := services.GetDatabase()
	collection := db.Collection("messages")

	// Aggregate to get unique chat_ids with metadata
	pipeline := []bson.M{
		// Match messages for this page
		{"$match": bson.M{"page_id": pageID}},
		// Sort by timestamp descending to get latest first
		{"$sort": bson.M{"timestamp": -1}},
		// Group by chat_id to get unique chats
		{"$group": bson.M{
			"_id":            "$chat_id",
			"last_message":   bson.M{"$first": "$message"},
			"last_timestamp": bson.M{"$first": "$timestamp"},
			"last_sender_id": bson.M{"$first": "$sender_id"},
			"last_is_bot":    bson.M{"$first": "$is_bot"},
			"message_count":  bson.M{"$sum": 1},
		}},
		// Sort by last message timestamp
		{"$sort": bson.M{"last_timestamp": -1}},
		// Pagination
		{"$skip": skip},
		{"$limit": limit},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve chat IDs",
		})
	}
	defer cursor.Close(ctx)

	// Build chat list with customer info
	var chats []fiber.Map
	customersCollection := db.Collection("customers")

	for cursor.Next(ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			continue
		}

		chatID := result["_id"].(string)

		// Get customer info from customers collection
		var customer models.Customer
		err := customersCollection.FindOne(ctx, bson.M{
			"customer_id": chatID,
			"page_id":     pageID,
		}).Decode(&customer)

		customerName := ""
		firstName := ""
		lastName := ""
		if err == nil {
			customerName = customer.CustomerName
			firstName = customer.FirstName
			lastName = customer.LastName
		}

		// If no name in customer collection, use fallback
		if customerName == "" {
			customerName = fmt.Sprintf("User %s", chatID[:min(8, len(chatID))])
		}

		chat := fiber.Map{
			"chat_id":        chatID,
			"customer_name":  customerName,
			"first_name":     firstName,
			"last_name":      lastName,
			"last_message":   result["last_message"],
			"last_timestamp": result["last_timestamp"],
			"last_sender_id": result["last_sender_id"],
			"last_is_bot":    result["last_is_bot"],
			"message_count":  result["message_count"],
		}
		chats = append(chats, chat)
	}

	// Get total count of unique chats
	countPipeline := []bson.M{
		{"$match": bson.M{"page_id": pageID}},
		{"$group": bson.M{"_id": "$chat_id"}},
		{"$count": "total"},
	}

	countCursor, err := collection.Aggregate(ctx, countPipeline)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to count chats",
		})
	}
	defer countCursor.Close(ctx)

	var totalCount int64
	if countCursor.Next(ctx) {
		var countResult bson.M
		if err := countCursor.Decode(&countResult); err == nil {
			if total, ok := countResult["total"].(int32); ok {
				totalCount = int64(total)
			} else if total, ok := countResult["total"].(int64); ok {
				totalCount = total
			}
		}
	}

	// Calculate pagination info
	totalPages := (int(totalCount) + limit - 1) / limit
	hasMore := page < totalPages

	return c.JSON(fiber.Map{
		"page_id":   pageID,
		"page_name": pageName,
		"chats":     chats,
		"pagination": fiber.Map{
			"page":        page,
			"limit":       limit,
			"total":       totalCount,
			"total_pages": totalPages,
			"has_more":    hasMore,
		},
	})
}

// GetMessagesByChatID retrieves all messages for a specific chat_id
func GetMessagesByChatID(c *fiber.Ctx) error {
	// Get chat_id from URL params
	chatID := c.Params("chatID")
	if chatID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Chat ID is required",
		})
	}

	// Get page_id from query params (required for security)
	pageID := c.Query("page_id")
	if pageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Page ID is required",
		})
	}

	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Get pagination parameters
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)
	if limit > 200 {
		limit = 200
	}
	skip := (page - 1) * limit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify page belongs to company
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Check if page belongs to company
	pageFound := false
	var pageName string
	companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
	for _, p := range companyPages {
		if p.PageID == pageID {
			pageFound = true
			pageName = p.PageName
			break
		}
	}
	if !pageFound {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Page not found or access denied",
		})
	}

	db := services.GetDatabase()
	collection := db.Collection("messages")

	// Build filter to get messages for this chat
	filter := bson.M{
		"chat_id": chatID,
		"page_id": pageID,
	}

	// Get total count
	totalCount, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		slog.Error("Failed to count messages", "error", err)
		totalCount = 0
	}

	// Get messages with pagination - sort ascending for chat view
	findOptions := options.Find().
		SetSort(bson.M{"timestamp": 1}). // Ascending order for chat flow
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve messages",
		})
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	if err := cursor.All(ctx, &messages); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to decode messages",
		})
	}

	// Get customer info
	customersCollection := db.Collection("customers")
	var customer models.Customer
	err = customersCollection.FindOne(ctx, bson.M{
		"customer_id": chatID,
		"page_id":     pageID,
	}).Decode(&customer)

	customerName := ""
	if err == nil && customer.CustomerName != "" {
		customerName = customer.CustomerName
	} else {
		customerName = fmt.Sprintf("User %s", chatID[:min(8, len(chatID))])
	}

	// Calculate pagination info
	totalPages := (int(totalCount) + limit - 1) / limit
	hasMore := page < totalPages

	return c.JSON(fiber.Map{
		"chat_id":       chatID,
		"customer_name": customerName,
		"page_id":       pageID,
		"page_name":     pageName,
		"messages":      messages,
		"pagination": fiber.Map{
			"page":        page,
			"limit":       limit,
			"total":       totalCount,
			"total_pages": totalPages,
			"has_more":    hasMore,
		},
	})
}
