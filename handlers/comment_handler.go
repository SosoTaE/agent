package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"facebook-bot/models"
	"facebook-bot/services"
)

// ChangeValue represents the value of a change (moved from webhooks to avoid import cycle)
type ChangeValue struct {
	Item        string        `json:"item"`
	CommentID   string        `json:"comment_id"`
	PostID      string        `json:"post_id"`
	ParentID    string        `json:"parent_id"`
	SenderID    string        `json:"sender_id"`
	SenderName  string        `json:"sender_name"`
	From        *FacebookUser `json:"from,omitempty"` // User who made the comment
	Message     string        `json:"message"`
	CreatedTime int64         `json:"created_time"` // Unix timestamp
}

// FacebookUser represents a Facebook user in webhook payloads
type FacebookUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// HandleComment processes incoming comments and replies
func HandleComment(change ChangeValue, pageID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // Increased to handle Claude API + Facebook API calls
	defer cancel()

	commentID := change.CommentID
	postID := change.PostID
	parentID := change.ParentID
	senderID := change.SenderID
	senderName := change.SenderName
	message := change.Message

	// Use From field if available (Facebook's preferred structure)
	if change.From != nil {
		if change.From.ID != "" {
			senderID = change.From.ID
		}
		if change.From.Name != "" {
			senderName = change.From.Name
		}
	}

	// Check if we have a sender ID
	if senderID == "" {
		slog.Error("No sender ID found in webhook data",
			"commentID", commentID,
			"hasFromField", change.From != nil,
		)
		return
	}

	// CRITICAL: Check if the comment is from the page itself (the bot)
	// This prevents the bot from replying to its own comments
	if senderID == pageID {
		slog.Info("Skipping bot's own comment",
			"commentID", commentID,
			"pageID", pageID,
			"senderID", senderID,
		)
		return
	}

	// Determine if this is a reply
	isReply := parentID != "" && parentID != postID

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

	// Check if Facebook comments are enabled for this page
	if pageConfig.FacebookConfig != nil && !pageConfig.FacebookConfig.IsEnabled {
		slog.Info("Facebook comments are disabled for this page, skipping bot response",
			"pageID", pageID)
		return
	}

	// Additional check: if sender name matches page name, it's likely the bot
	if senderName == pageConfig.PageName {
		slog.Info("Skipping comment from page (matched by name)",
			"commentID", commentID,
			"senderName", senderName,
			"pageName", pageConfig.PageName,
		)
		return
	}

	// Final check: Query Facebook to verify if comment is from the page
	isFromPage, err := services.IsCommentFromPage(ctx, commentID, pageConfig.PageAccessToken)
	if err != nil {
		slog.Warn("Failed to check if comment is from page", "error", err)
		// Don't return, continue processing but with caution
	} else if isFromPage {
		slog.Info("Skipping comment - verified from page via API",
			"commentID", commentID,
		)
		return
	}

	commentType := "comment"
	if isReply {
		commentType = "reply"
	}

	slog.Info("Handling "+commentType,
		"commentID", commentID,
		"postID", postID,
		"parentID", parentID,
		"isReply", isReply,
		"senderID", senderID,
		"senderName", senderName,
		"pageID", pageID,
		"pageName", pageConfig.PageName,
		"companyID", company.CompanyID,
		"message", message,
	)

	// Fetch user details (first name and last name) from Facebook synchronously
	var firstName, lastName string
	userDetails, err := services.GetFacebookUserDetails(ctx, senderID, pageConfig.PageAccessToken)
	if err != nil {
		slog.Warn("Failed to fetch Facebook user details",
			"senderID", senderID,
			"error", err)
		// Continue without first/last name if fetch fails
	} else if userDetails != nil {
		firstName = userDetails.FirstName
		lastName = userDetails.LastName
		// Update sender name if we have both names
		if firstName != "" && lastName != "" {
			senderName = fmt.Sprintf("%s %s", firstName, lastName)
		}
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

	// Get post content for context
	postContent, err := services.GetPostContent(ctx, postID, pageConfig.PageAccessToken)
	if err != nil {
		slog.Error("Failed to get post content", "error", err)
		postContent = ""
	}

	// Check if we've already responded to this comment
	alreadyProcessed, err := services.CheckCommentProcessed(ctx, commentID)
	if err != nil {
		slog.Error("Failed to check if comment processed", "error", err)
	}

	if alreadyProcessed {
		slog.Info("Comment already processed, skipping",
			"commentID", commentID,
			"isReply", isReply,
		)
		return
	}

	// Save the user's comment with first and last name
	err = services.SaveCommentWithNames(
		ctx, commentID, postID, parentID, postContent,
		senderID, senderName, firstName, lastName, pageID, pageConfig.PageName,
		message, false, // isBot = false for user comments
	)

	if err != nil {
		slog.Error("Failed to save comment", "error", err)
		return
	}

	// If this is a reply, fetch the parent comment for additional context
	var parentCommentText string
	if isReply {
		parentComment, err := services.GetComment(ctx, parentID)
		if err != nil {
			slog.Warn("Failed to fetch parent comment", "error", err, "parentID", parentID)
		} else if parentComment != nil {
			parentCommentText = parentComment.SenderName + ": " + parentComment.Message
			slog.Info("Fetched parent comment",
				"parentCommentID", parentID,
				"parentSender", parentComment.SenderName,
			)
		}
	}

	// Fetch comment history for context
	commentHistory, err := services.GetCommentHistory(ctx, postID, 5)
	if err != nil {
		slog.Warn("Failed to fetch comment history", "error", err)
		// Continue without history if fetch fails
		commentHistory = []services.ChatHistory{}
	}

	slog.Info("Fetched comment history",
		"count", len(commentHistory),
		"postID", postID,
		"pageID", pageID,
	)

	// Fetch RAG context based on facebook configuration
	var ragContext string
	shouldUseRAG := false

	// Check facebook-specific configuration
	if pageConfig.FacebookConfig != nil {
		// Check if RAG is enabled for facebook
		if pageConfig.FacebookConfig.RAGEnabled {
			shouldUseRAG = true
		}
		// Also check if there are active CRM links for facebook
		if len(pageConfig.FacebookConfig.CRMLinks) > 0 {
			for _, link := range pageConfig.FacebookConfig.CRMLinks {
				if link.IsActive {
					shouldUseRAG = true
					break
				}
			}
		}
	} else {
		// Fallback to legacy behavior - check for active CRM documents
		hasActiveCRMDocs, err := services.HasActiveCRMDocumentsForChannel(ctx, company.CompanyID, pageID, "facebook")
		if err != nil {
			slog.Warn("Failed to check for active CRM documents", "error", err)
			hasActiveCRMDocs = false
		}
		shouldUseRAG = hasActiveCRMDocs
	}

	if shouldUseRAG {
		// Get relevant context based on the user's message, filtered by page ID and channel
		ragContext, err = services.GetRAGContextForChannel(ctx, message, company.CompanyID, pageID, "facebook")
		if err != nil {
			slog.Warn("Failed to fetch RAG context", "error", err)
			// Continue without RAG context
		} else if ragContext != "" {
			slog.Info("RAG context retrieved for facebook",
				"contextLength", len(ragContext),
				"companyID", company.CompanyID,
				"pageID", pageID,
				"channel", "facebook",
			)
		}
	} else {
		slog.Info("RAG disabled for facebook on this page",
			"pageID", pageID,
		)
	}

	// Generate context-aware response with history
	var contextStr string
	if isReply && parentCommentText != "" {
		// For replies, include the parent comment in the context
		contextStr = fmt.Sprintf("Post: %s\nParent Comment: %s\nReply: %s", postContent, parentCommentText, message)
	} else {
		// For regular comments
		contextStr = fmt.Sprintf("Post: %s\nComment: %s", postContent, message)
	}

	messageType := "comment"
	if isReply {
		messageType = "reply"
	}

	aiResponse, _, err := services.GetClaudeResponseWithToolUse(ctx, contextStr, messageType, company, pageConfig, commentHistory, ragContext)
	if err != nil {
		slog.Error("Failed to get Claude response", "error", err)
		if isReply {
			aiResponse = "Thank you for your reply!"
		} else {
			aiResponse = "Thank you for your comment!"
		}
	}

	// For comments, we don't do agent handoff - remove any CUSTOMER_WANTS_REAL_PERSON marker
	if strings.Contains(aiResponse, "CUSTOMER_WANTS_REAL_PERSON||") {
		// Remove the marker from the response
		aiResponse = strings.Replace(aiResponse, "CUSTOMER_WANTS_REAL_PERSON||", "", -1)
		aiResponse = strings.TrimSpace(aiResponse)

		slog.Info("Removed agent request marker from comment response",
			"commentID", commentID,
			"senderID", senderID,
			"pageID", pageID)
	}

	// Response delay removed for faster processing
	// Reply to the comment on Facebook immediately
	responseData, err := services.ReplyToCommentWithResponse(ctx, commentID, aiResponse, pageConfig.PageAccessToken)
	if err != nil {
		slog.Error("Failed to reply to comment", "error", err)
		return
	}

	// Save the bot's response as a new comment in the database
	if responseData != nil && responseData.ID != "" {
		err = services.SaveBotReply(ctx, responseData.ID, commentID, postID, aiResponse, pageID, pageConfig.PageName)
		if err != nil {
			slog.Error("Failed to save bot reply", "error", err)
		} else {
			slog.Info("Bot reply saved",
				"responseID", responseData.ID,
				"parentCommentID", commentID,
			)
		}
	}

	// Save response to database for analytics
	responseDoc := &models.Response{
		Type:       "comment",
		CommentID:  commentID,
		PostID:     postID,
		SenderID:   senderID,
		SenderName: senderName,
		PageID:     pageID,
		PageName:   pageConfig.PageName,
		Original:   message,
		Response:   aiResponse,
		Timestamp:  time.Now(),
	}

	if err := services.SaveResponse(ctx, responseDoc); err != nil {
		slog.Error("Failed to save response", "error", err)
	}
}
