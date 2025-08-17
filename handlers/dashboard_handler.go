package handlers

import (
	"context"
	"facebook-bot/models"
	"facebook-bot/services"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CommentThread represents a comment with its replies
type CommentThread struct {
	CommentID  string    `json:"comment_id"`
	PostID     string    `json:"post_id"`
	SenderID   string    `json:"sender_id"`
	SenderName string    `json:"sender_name"`
	Message    string    `json:"message"`
	IsBot      bool      `json:"is_bot"`
	Timestamp  time.Time `json:"timestamp"`
	Replies    []Reply   `json:"replies"`
}

// Reply represents a reply to a comment
type Reply struct {
	CommentID  string    `json:"comment_id"`
	SenderID   string    `json:"sender_id"`
	SenderName string    `json:"sender_name"`
	Message    string    `json:"message"`
	IsBot      bool      `json:"is_bot"`
	Timestamp  time.Time `json:"timestamp"`
}

// PostWithComments represents a post with all its comments
type PostWithComments struct {
	PostID      string          `json:"post_id"`
	PostContent string          `json:"post_content"`
	PageID      string          `json:"page_id"`
	PageName    string          `json:"page_name"`
	Comments    []CommentThread `json:"comments"`
	TotalCount  int             `json:"total_count"`
	BotReplies  int             `json:"bot_replies"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// GetPostComments retrieves all comments and replies for a specific post
func GetPostComments(c *fiber.Ctx) error {
	postID := c.Params("postID")
	if postID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Post ID is required",
		})
	}

	// Get company_id from session
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get comments for the post (will verify it belongs to company)
	// Comments now come with nested replies
	topLevelComments, err := services.GetPostCommentsForCompany(ctx, postID, companyID.(string))
	if err != nil {
		slog.Error("Failed to get post comments", "error", err, "postID", postID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve comments",
		})
	}

	// Convert to CommentThread format
	var threads []CommentThread
	botReplyCount := 0
	totalCount := 0

	// Process top-level comments and their nested replies
	for _, comment := range topLevelComments {
		// Create thread from top-level comment
		thread := CommentThread{
			CommentID:  comment.CommentID,
			PostID:     comment.PostID,
			SenderID:   comment.SenderID,
			SenderName: comment.SenderName,
			Message:    comment.Message,
			IsBot:      comment.IsBot,
			Timestamp:  comment.Timestamp,
			Replies:    []Reply{},
		}

		if comment.IsBot {
			botReplyCount++
		}
		totalCount++

		// Add nested replies
		for _, reply := range comment.Replies {
			replyItem := Reply{
				CommentID:  reply.CommentID,
				SenderID:   reply.SenderID,
				SenderName: reply.SenderName,
				Message:    reply.Message,
				IsBot:      reply.IsBot,
				Timestamp:  reply.Timestamp,
			}
			thread.Replies = append(thread.Replies, replyItem)

			if reply.IsBot {
				botReplyCount++
			}
			totalCount++
		}

		threads = append(threads, thread)
	}

	// Get post content if available
	postContent := ""
	if len(topLevelComments) > 0 {
		postContent = topLevelComments[0].PostContent
	}

	result := PostWithComments{
		PostID:      postID,
		PostContent: postContent,
		Comments:    threads,
		TotalCount:  totalCount,
		BotReplies:  botReplyCount,
		UpdatedAt:   time.Now(),
	}

	return c.JSON(result)
}

// GetCompanyPages retrieves all available pages for a company
func GetCompanyPages(c *fiber.Ctx) error {
	// Get company_id from session
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get company to find its pages
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Get all pages for this company
	companyPages, err := services.GetPagesByCompanyID(ctx, companyID.(string))
	if err != nil {
		companyPages = []models.FacebookPage{}
	}

	// Create pages list without access tokens
	pages := make([]fiber.Map, 0, len(companyPages))
	for _, page := range companyPages {
		pages = append(pages, fiber.Map{
			"page_id":   page.PageID,
			"page_name": page.PageName,
			"is_active": page.IsActive,
		})
	}

	// Return pages list without access tokens
	return c.JSON(fiber.Map{
		"pages":   pages,
		"company": company.CompanyName,
	})
}

// GetCompanyComments retrieves all comments for a specific page with pagination
func GetCompanyComments(c *fiber.Ctx) error {
	// Get company_id from session
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Get page_id from query parameter
	pageID := c.Query("page_id")
	if pageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "page_id is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get company to verify page ownership
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Verify that the page belongs to this company
	pageFound := false
	var pageName string
	companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
	for _, page := range companyPages {
		if page.PageID == pageID {
			pageFound = true
			pageName = page.PageName
			break
		}
	}

	if !pageFound {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Page not found or access denied",
		})
	}

	// Get pagination parameters
	currentPage := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 100) // Default 100 comments per page
	if limit > 500 {
		limit = 500 // Max 500 comments per request
	}
	skip := (currentPage - 1) * limit

	// Get filter parameters
	filterType := c.Query("filter", "all") // all, unanswered, bot
	sortBy := c.Query("sort", "newest")    // newest, oldest

	db := services.GetDatabase()
	collection := db.Collection("comments")

	// Build query filter - now filtering by specific page_id
	filter := bson.M{
		"page_id": pageID,
	}

	// Apply filter type
	switch filterType {
	case "unanswered":
		// This would need more complex logic to find unanswered comments
		// For now, just get non-bot comments
		filter["is_bot"] = false
	case "bot":
		filter["is_bot"] = true
	}

	// Set sort order
	sortOrder := -1 // descending by default
	if sortBy == "oldest" {
		sortOrder = 1
	}

	// Get total count for pagination info
	totalCount, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		slog.Error("Failed to count comments", "error", err)
		totalCount = 0
	}

	// Sort by timestamp with pagination
	findOptions := options.Find().
		SetSort(bson.M{"timestamp": sortOrder}).
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve comments",
		})
	}
	defer cursor.Close(ctx)

	// Collect comments for this page
	var comments []map[string]interface{}

	for cursor.Next(ctx) {
		var comment bson.M
		if err := cursor.Decode(&comment); err != nil {
			continue
		}

		// Add page name
		comment["page_name"] = pageName

		comments = append(comments, comment)
	}

	// Calculate pagination info
	totalPages := (int(totalCount) + limit - 1) / limit
	hasMore := currentPage < totalPages

	slog.Info("Retrieved comments for page",
		"company_id", companyID,
		"page_id", pageID,
		"page", currentPage,
		"limit", limit,
		"returned", len(comments),
		"total", totalCount)

	return c.JSON(fiber.Map{
		"comments": comments,
		"pagination": fiber.Map{
			"page":        currentPage,
			"limit":       limit,
			"total":       totalCount,
			"total_pages": totalPages,
			"has_more":    hasMore,
		},
		"page_id":   pageID,
		"page_name": pageName,
		"company":   company.CompanyName,
	})
}

// GetNameChanges retrieves recent name changes for a company's pages
func GetNameChanges(c *fiber.Ctx) error {
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Get optional page_id from query
	pageID := c.Query("page_id")
	limit := c.QueryInt("limit", 50)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get company to find its pages
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	var pageIDs []string
	if pageID != "" {
		// Verify page belongs to company
		pageFound := false
		companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
		for _, page := range companyPages {
			if page.PageID == pageID {
				pageFound = true
				pageIDs = []string{pageID}
				break
			}
		}
		if !pageFound {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Page not found or access denied",
			})
		}
	} else {
		// Get all pages for company
		companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
		for _, page := range companyPages {
			pageIDs = append(pageIDs, page.PageID)
		}
	}

	// Get recent name changes
	changes, err := services.GetRecentNameChanges(ctx, pageIDs, limit)
	if err != nil {
		slog.Error("Failed to get name changes", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve name changes",
		})
	}

	return c.JSON(fiber.Map{
		"changes":   changes,
		"page_id":   pageID,
		"company":   company.CompanyName,
		"retrieved": len(changes),
	})
}

// GetSenderHistory retrieves name history for a specific sender
func GetSenderHistory(c *fiber.Ctx) error {
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	senderID := c.Params("senderID")
	pageID := c.Query("page_id")

	if senderID == "" || pageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "sender_id and page_id are required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify page ownership
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	pageFound := false
	companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
	for _, page := range companyPages {
		if page.PageID == pageID {
			pageFound = true
			break
		}
	}

	if !pageFound {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Page not found or access denied",
		})
	}

	// Get name history
	history, err := services.GetNameHistory(ctx, senderID, pageID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "No history found for this sender",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve history",
		})
	}

	// Get name changes for this sender
	changes, err := services.GetSenderNameChanges(ctx, senderID, pageID)
	if err != nil {
		slog.Error("Failed to get sender name changes", "error", err)
		changes = []models.NameChange{}
	}

	return c.JSON(fiber.Map{
		"history": history,
		"changes": changes,
	})
}

// GetCommentStats retrieves statistics about comments for a company or specific page
func GetCommentStats(c *fiber.Ctx) error {
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Get optional page_id from query
	pageID := c.Query("page_id")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get company to find its pages
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Build filter based on whether page_id is provided
	var filter bson.M
	if pageID != "" {
		// Verify page belongs to company
		pageFound := false
		companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
		for _, page := range companyPages {
			if page.PageID == pageID {
				pageFound = true
				break
			}
		}
		if !pageFound {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Page not found or access denied",
			})
		}
		filter = bson.M{"page_id": pageID}
	} else {
		// Get all pages for company
		var pageIDs []string
		companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
		for _, page := range companyPages {
			pageIDs = append(pageIDs, page.PageID)
		}
		filter = bson.M{"page_id": bson.M{"$in": pageIDs}}
	}

	db := services.GetDatabase()
	collection := db.Collection("comments")

	// Get total comments
	totalComments, err := collection.CountDocuments(ctx, filter)

	// Get bot replies
	botFilter := bson.M{}
	for k, v := range filter {
		botFilter[k] = v
	}
	botFilter["is_bot"] = true
	botReplies, err := collection.CountDocuments(ctx, botFilter)

	// Get unanswered comments (comments without bot replies)
	unansweredFilter := bson.M{}
	for k, v := range filter {
		unansweredFilter[k] = v
	}
	unansweredFilter["is_bot"] = false

	pipeline := []bson.M{
		{
			"$match": unansweredFilter,
		},
		{
			"$lookup": bson.M{
				"from":         "comments",
				"localField":   "comment_id",
				"foreignField": "parent_id",
				"as":           "replies",
			},
		},
		{
			"$match": bson.M{
				"replies": bson.M{"$size": 0},
			},
		},
		{
			"$count": "unanswered",
		},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		slog.Error("Failed to get unanswered count", "error", err)
	}

	var result []bson.M
	cursor.All(ctx, &result)

	unansweredCount := int64(0)
	if len(result) > 0 {
		if val, ok := result[0]["unanswered"].(int32); ok {
			unansweredCount = int64(val)
		}
	}

	// Get comments from last 24 hours
	yesterday := time.Now().Add(-24 * time.Hour)
	recentFilter := bson.M{}
	for k, v := range filter {
		recentFilter[k] = v
	}
	recentFilter["timestamp"] = bson.M{"$gte": yesterday}
	recentComments, err := collection.CountDocuments(ctx, recentFilter)

	responseRate := float64(0)
	if totalComments > 0 {
		responseRate = float64(botReplies) / float64(totalComments) * 100
	}

	pagesMonitored := 1
	if pageID == "" {
		companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
		pagesMonitored = len(companyPages)
	}

	return c.JSON(fiber.Map{
		"total_comments":  totalComments,
		"bot_replies":     botReplies,
		"unanswered":      unansweredCount,
		"recent_24h":      recentComments,
		"response_rate":   responseRate,
		"pages_monitored": pagesMonitored,
		"page_id":         pageID,
	})
}

// GetUserConversations retrieves all unique users who have sent messages
func GetUserConversations(c *fiber.Ctx) error {
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Get optional page_id from query
	pageID := c.Query("page_id")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get company to verify page ownership
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Build filter
	filter := bson.M{}
	if pageID != "" {
		// Verify page belongs to company
		pageFound := false
		companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
		for _, page := range companyPages {
			if page.PageID == pageID {
				pageFound = true
				break
			}
		}
		if !pageFound {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Page not found or access denied",
			})
		}
		filter["page_id"] = pageID
	} else {
		// Get all pages for company
		var pageIDs []string
		companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
		for _, page := range companyPages {
			pageIDs = append(pageIDs, page.PageID)
		}
		filter["page_id"] = bson.M{"$in": pageIDs}
	}

	db := services.GetDatabase()
	collection := db.Collection("comments")

	// Aggregate to get unique users with their latest message
	pipeline := []bson.M{
		{"$match": filter},
		{
			"$sort": bson.M{"timestamp": -1},
		},
		{
			"$group": bson.M{
				"_id":              "$sender_id",
				"sender_name":      bson.M{"$first": "$sender_name"},
				"latest_message":   bson.M{"$first": "$message"},
				"latest_timestamp": bson.M{"$first": "$timestamp"},
				"message_count":    bson.M{"$sum": 1},
				"page_id":          bson.M{"$first": "$page_id"},
			},
		},
		{
			"$sort": bson.M{"latest_timestamp": -1},
		},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		slog.Error("Failed to aggregate user conversations", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve user conversations",
		})
	}
	defer cursor.Close(ctx)

	var users []bson.M
	if err := cursor.All(ctx, &users); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to decode user conversations",
		})
	}

	return c.JSON(fiber.Map{
		"users": users,
		"total": len(users),
	})
}

// GetUserMessages retrieves paginated messages for a specific user
func GetUserMessages(c *fiber.Ctx) error {
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	senderID := c.Params("senderID")
	if senderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Sender ID is required",
		})
	}

	// Get pagination parameters
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)
	if limit > 200 {
		limit = 200
	}
	skip := (page - 1) * limit

	// Get optional page_id
	pageID := c.Query("page_id")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get company to verify page ownership
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Build filter
	filter := bson.M{"sender_id": senderID}

	if pageID != "" {
		// Verify page belongs to company
		pageFound := false
		companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
		for _, page := range companyPages {
			if page.PageID == pageID {
				pageFound = true
				break
			}
		}
		if !pageFound {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Page not found or access denied",
			})
		}
		filter["page_id"] = pageID
	} else {
		// Get all pages for company
		var pageIDs []string
		companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
		for _, page := range companyPages {
			pageIDs = append(pageIDs, page.PageID)
		}
		filter["page_id"] = bson.M{"$in": pageIDs}
	}

	db := services.GetDatabase()
	collection := db.Collection("comments")

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

	var messages []bson.M
	if err := cursor.All(ctx, &messages); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to decode messages",
		})
	}

	// Calculate pagination info
	totalPages := (int(totalCount) + limit - 1) / limit
	hasMore := page < totalPages

	// Get user info from first message
	var userName string
	if len(messages) > 0 {
		if name, ok := messages[0]["sender_name"].(string); ok {
			userName = name
		}
	}

	return c.JSON(fiber.Map{
		"messages": messages,
		"pagination": fiber.Map{
			"page":        page,
			"limit":       limit,
			"total":       totalCount,
			"total_pages": totalPages,
			"has_more":    hasMore,
		},
		"user": fiber.Map{
			"sender_id":   senderID,
			"sender_name": userName,
		},
	})
}

// GetChatMessages retrieves chat messages with pagination
func GetChatMessages(c *fiber.Ctx) error {
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Get query parameters
	pageID := c.Query("page_id")
	senderID := c.Query("sender_id")
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

	// Build filter for messages collection
	filter := bson.M{"type": "chat"}

	// Add page filter if provided
	if pageID != "" {
		// Verify page belongs to company
		pageFound := false
		companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
		for _, p := range companyPages {
			if p.PageID == pageID {
				pageFound = true
				break
			}
		}
		if !pageFound {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Page not found or access denied",
			})
		}
		filter["page_id"] = pageID
	} else {
		// Get all pages for company
		var pageIDs []string
		companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
		for _, p := range companyPages {
			pageIDs = append(pageIDs, p.PageID)
		}
		filter["page_id"] = bson.M{"$in": pageIDs}
	}

	// Add sender filter if provided
	if senderID != "" {
		filter["$or"] = []bson.M{
			{"sender_id": senderID},
			{"recipient_id": senderID},
		}
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
		"messages": messages,
		"pagination": fiber.Map{
			"page":        page,
			"limit":       limit,
			"total":       totalCount,
			"total_pages": totalPages,
			"has_more":    hasMore,
		},
	})
}

// GetChatConversations retrieves unique chat conversations
func GetChatConversations(c *fiber.Ctx) error {
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	pageID := c.Query("page_id")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get company to verify page ownership
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Build filter
	filter := bson.M{"type": "chat", "is_bot": false}

	if pageID != "" {
		// Verify page belongs to company
		pageFound := false
		companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
		for _, p := range companyPages {
			if p.PageID == pageID {
				pageFound = true
				break
			}
		}
		if !pageFound {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Page not found or access denied",
			})
		}
		filter["page_id"] = pageID
	} else {
		// Get all pages for company
		var pageIDs []string
		companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
		for _, p := range companyPages {
			pageIDs = append(pageIDs, p.PageID)
		}
		filter["page_id"] = bson.M{"$in": pageIDs}
	}

	db := services.GetDatabase()
	collection := db.Collection("messages")

	// Aggregate to get unique conversations
	pipeline := []bson.M{
		{"$match": filter},
		{
			"$group": bson.M{
				"_id": bson.M{
					"sender_id": "$sender_id",
					"page_id":   "$page_id",
				},
				"sender_name":      bson.M{"$first": "$sender_name"},
				"first_name":       bson.M{"$first": "$first_name"},
				"last_name":        bson.M{"$first": "$last_name"},
				"page_name":        bson.M{"$first": "$page_name"},
				"latest_message":   bson.M{"$first": "$message"},
				"latest_timestamp": bson.M{"$first": "$timestamp"},
				"message_count":    bson.M{"$sum": 1},
			},
		},
		{
			"$sort": bson.M{"latest_timestamp": -1},
		},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		slog.Error("Failed to aggregate conversations", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve conversations",
		})
	}
	defer cursor.Close(ctx)

	var conversations []bson.M
	if err := cursor.All(ctx, &conversations); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to decode conversations",
		})
	}

	// Format the conversations
	formattedConversations := make([]fiber.Map, 0, len(conversations))
	for _, conv := range conversations {
		if idMap, ok := conv["_id"].(bson.M); ok {
			formattedConversations = append(formattedConversations, fiber.Map{
				"sender_id":        idMap["sender_id"],
				"page_id":          idMap["page_id"],
				"sender_name":      conv["sender_name"],
				"first_name":       conv["first_name"],
				"last_name":        conv["last_name"],
				"page_name":        conv["page_name"],
				"latest_message":   conv["latest_message"],
				"latest_timestamp": conv["latest_timestamp"],
				"message_count":    conv["message_count"],
			})
		}
	}

	return c.JSON(fiber.Map{
		"conversations": formattedConversations,
		"total":         len(formattedConversations),
	})
}

// GetPostsList retrieves a list of unique post IDs from comments
func GetPostsList(c *fiber.Ctx) error {
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Get optional page_id from query
	pageID := c.Query("page_id")
	limit := c.QueryInt("limit", 100)
	if limit > 500 {
		limit = 500
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get company to verify page ownership
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Build filter
	filter := bson.M{}
	if pageID != "" {
		// Verify page belongs to company
		pageFound := false
		companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
		for _, page := range companyPages {
			if page.PageID == pageID {
				pageFound = true
				break
			}
		}
		if !pageFound {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Page not found or access denied",
			})
		}
		filter["page_id"] = pageID
	} else {
		// Get all pages for company
		var pageIDs []string
		companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
		for _, page := range companyPages {
			pageIDs = append(pageIDs, page.PageID)
		}
		filter["page_id"] = bson.M{"$in": pageIDs}
	}

	db := services.GetDatabase()
	collection := db.Collection("comments")

	// Aggregate to get unique post IDs with metadata
	pipeline := []bson.M{
		{"$match": filter},
		{
			"$group": bson.M{
				"_id":             "$post_id",
				"page_id":         bson.M{"$first": "$page_id"},
				"page_name":       bson.M{"$first": "$page_name"},
				"post_content":    bson.M{"$first": "$post_content"},
				"comment_count":   bson.M{"$sum": 1},
				"latest_comment":  bson.M{"$max": "$timestamp"},
				"first_comment":   bson.M{"$min": "$timestamp"},
				"bot_reply_count": bson.M{"$sum": bson.M{"$cond": []interface{}{"$is_bot", 1, 0}}},
			},
		},
		{
			"$sort": bson.M{"latest_comment": -1},
		},
		{
			"$limit": limit,
		},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		slog.Error("Failed to aggregate posts", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve posts",
		})
	}
	defer cursor.Close(ctx)

	var posts []bson.M
	if err := cursor.All(ctx, &posts); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to decode posts",
		})
	}

	// Format the results
	formattedPosts := make([]fiber.Map, 0, len(posts))
	for _, post := range posts {
		formattedPosts = append(formattedPosts, fiber.Map{
			"post_id":         post["_id"],
			"page_id":         post["page_id"],
			"page_name":       post["page_name"],
			"post_content":    post["post_content"],
			"comment_count":   post["comment_count"],
			"bot_reply_count": post["bot_reply_count"],
			"latest_comment":  post["latest_comment"],
			"first_comment":   post["first_comment"],
		})
	}

	return c.JSON(fiber.Map{
		"posts":  formattedPosts,
		"total":  len(formattedPosts),
		"limit":  limit,
		"filter": pageID,
	})
}

// GetChatHistory retrieves chat history between a user and page
func GetChatHistory(c *fiber.Ctx) error {
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	senderID := c.Params("senderID")
	pageID := c.Query("page_id")
	limit := c.QueryInt("limit", 50)

	if senderID == "" || pageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "sender_id and page_id are required",
		})
	}

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

	db := services.GetDatabase()
	collection := db.Collection("messages")

	// Build filter to get all messages between user and page
	filter := bson.M{
		"type":    "chat",
		"page_id": pageID,
		"$or": []bson.M{
			{"sender_id": senderID, "recipient_id": pageID},
			{"sender_id": pageID, "recipient_id": senderID},
		},
	}

	// Get messages sorted by timestamp (newest first, then reverse for display)
	findOptions := options.Find().
		SetSort(bson.M{"timestamp": -1}).
		SetLimit(int64(limit))

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve chat history",
		})
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	if err := cursor.All(ctx, &messages); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to decode messages",
		})
	}

	// Reverse messages for chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	// Get sender info from first message
	var senderName string
	for _, msg := range messages {
		if msg.SenderID == senderID && msg.SenderName != "" {
			senderName = msg.SenderName
			break
		}
	}

	return c.JSON(fiber.Map{
		"messages": messages,
		"conversation": fiber.Map{
			"sender_id":   senderID,
			"sender_name": senderName,
			"page_id":     pageID,
			"page_name":   pageName,
		},
		"total": len(messages),
	})
}
