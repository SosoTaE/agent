package handlers

import (
	"context"
	"log/slog"
	"time"

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

	// CRM message processing has been removed

	// Save user's message to database
	messageDoc := &models.Message{
		Type:        "chat",
		SenderID:    senderID,
		RecipientID: pageID, // User is sending to the page
		PageID:      pageID,
		PageName:    pageConfig.PageName,
		Message:     messageText,
		IsBot:       false,
		Timestamp:   time.Now(),
	}

	if err := services.SaveMessage(ctx, messageDoc); err != nil {
		slog.Error("Failed to save user message", "error", err)
	}

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

	//// Log history details for debugging
	//for i, h := range chatHistory {
	//	contentPreview := h.Content
	//	if len(contentPreview) > 50 {
	//		contentPreview = contentPreview[:50] + "..."
	//	}
	//	slog.Debug("History item",
	//		"index", i,
	//		"role", h.Role,
	//		"content", contentPreview,
	//	)
	//}

	// Fetch RAG context if enabled for this company
	var ragContext string
	if len(company.CRMLinks) > 0 {
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
	}

	// Get AI response from Claude with conversation history and RAG context
	aiResponse, err := services.GetClaudeResponseWithRAG(ctx, messageText, "chat", company, pageConfig, chatHistory, ragContext)
	if err != nil {
		slog.Error("Failed to get Claude response", "error", err)
		aiResponse = "I apologize, but I'm having trouble processing your message right now. Please try again later."
	}

	// Response delay removed for faster processing
	// Send response back via Messenger immediately
	if err := services.SendMessengerReply(ctx, senderID, aiResponse, pageConfig.PageAccessToken); err != nil {
		slog.Error("Failed to send messenger reply", "error", err)
	}

	// Save bot's response as a message in the database
	botMessageDoc := &models.Message{
		Type:        "chat",
		SenderID:    pageID,   // Bot's sender ID is the page ID
		RecipientID: senderID, // Bot is sending to the user
		PageID:      pageID,
		PageName:    pageConfig.PageName,
		Message:     aiResponse,
		IsBot:       true,
		Timestamp:   time.Now(),
	}

	if err := services.SaveMessage(ctx, botMessageDoc); err != nil {
		slog.Error("Failed to save bot message", "error", err)
	}

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
