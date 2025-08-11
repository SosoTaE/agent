package services

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"facebook-bot/models"
)

var (
	mongoClient *mongo.Client
	database    *mongo.Database
)

// GetDatabase returns the MongoDB database instance
func GetDatabase() *mongo.Database {
	return database
}

// InitMongoDB initializes MongoDB connection
func InitMongoDB(ctx context.Context, uri string) (*mongo.Client, error) {
	clientOptions := options.Client().ApplyURI(uri)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	slog.Info("Connected to MongoDB")
	mongoClient = client

	return client, nil
}

// InitServices initializes all services
func InitServices(client *mongo.Client, databaseName string) {
	database = client.Database(databaseName)

	// Create indexes
	createIndexes()
}

// createIndexes creates necessary database indexes
func createIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Messages collection indexes
	messagesCollection := database.Collection("messages")
	messagesCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.M{"sender_id": 1}},
		{Keys: bson.M{"page_id": 1}},
		{Keys: bson.M{"timestamp": -1}},
	})

	// Comments collection indexes
	commentsCollection := database.Collection("comments")
	commentsCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.M{"comment_id": 1}, Options: options.Index().SetUnique(true)},
		{Keys: bson.M{"post_id": 1}},
		{Keys: bson.M{"parent_id": 1}},
		{Keys: bson.M{"page_id": 1}},
		{Keys: bson.M{"is_reply": 1}},
		{Keys: bson.M{"is_bot": 1}},
		{Keys: bson.M{"timestamp": -1}},
	})

	// Responses collection indexes
	responsesCollection := database.Collection("responses")
	responsesCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.M{"sender_id": 1}},
		{Keys: bson.M{"page_id": 1}},
		{Keys: bson.M{"timestamp": -1}},
	})

	// Page cache collection indexes
	pageCacheCollection := database.Collection("page_cache")
	pageCacheCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{"page_id": 1},
		Options: options.Index().SetUnique(true),
	})
}

// SaveMessage saves a message to database
func SaveMessage(ctx context.Context, message *models.Message) error {
	collection := database.Collection("messages")
	_, err := collection.InsertOne(ctx, message)
	return err
}

// SaveComment saves a comment or reply to the comments collection
func SaveComment(ctx context.Context, commentID, postID, parentID, postContent, senderID, senderName, pageID, pageName, message string, isBot bool) error {
	collection := database.Collection("comments")

	// Determine if this is a reply
	isReply := parentID != "" && parentID != postID

	// Create the comment document
	comment := models.Comment{
		CommentID:   commentID,
		PostID:      postID,
		ParentID:    parentID,
		SenderID:    senderID,
		SenderName:  senderName,
		PageID:      pageID,
		PageName:    pageName,
		Message:     message,
		PostContent: postContent,
		IsReply:     isReply,
		IsBot:       isBot,
		Timestamp:   time.Now(),
	}

	// Use upsert to avoid duplicates
	filter := bson.M{"comment_id": commentID}
	update := bson.M{"$set": comment}
	opts := options.Update().SetUpsert(true)

	result, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		slog.Error("Failed to save comment",
			"error", err,
			"commentID", commentID,
		)
		return err
	}

	if result.UpsertedCount > 0 {
		slog.Info("New comment saved",
			"commentID", commentID,
			"isReply", isReply,
			"isBot", isBot,
			"postID", postID,
			"parentID", parentID,
		)
	} else {
		slog.Info("Comment updated",
			"commentID", commentID,
		)
	}

	return nil
}

// CheckCommentProcessed checks if we've already responded to a comment
func CheckCommentProcessed(ctx context.Context, commentID string) (bool, error) {
	collection := database.Collection("comments")

	// Check if there's a bot reply to this comment
	count, err := collection.CountDocuments(ctx, bson.M{
		"parent_id": commentID,
		"is_bot":    true,
	})

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// SaveBotReply saves the bot's reply as a new comment
func SaveBotReply(ctx context.Context, responseCommentID, parentCommentID, postID, botMessage, pageID, pageName string) error {
	return SaveComment(ctx, responseCommentID, postID, parentCommentID, "", pageID, pageName, pageID, pageName, botMessage, true)
}

// SaveResponse saves an AI response to database
func SaveResponse(ctx context.Context, response *models.Response) error {
	collection := database.Collection("responses")
	_, err := collection.InsertOne(ctx, response)
	return err
}

// GetPageNameFromDB retrieves page name from cache
func GetPageNameFromDB(ctx context.Context, pageID string) (string, error) {
	collection := database.Collection("page_cache")

	var pageCache models.PageCache
	err := collection.FindOne(ctx, bson.M{"page_id": pageID}).Decode(&pageCache)
	if err != nil {
		return "", err
	}

	// Check if cache is still valid (24 hours)
	if time.Since(pageCache.UpdatedAt) > 24*time.Hour {
		return "", mongo.ErrNoDocuments
	}

	return pageCache.PageName, nil
}

// CachePageName saves page name to cache
func CachePageName(ctx context.Context, pageID, pageName string) error {
	collection := database.Collection("page_cache")

	filter := bson.M{"page_id": pageID}
	update := bson.M{
		"$set": bson.M{
			"page_id":    pageID,
			"page_name":  pageName,
			"updated_at": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// ChatHistory represents a chat history entry
type ChatHistory struct {
	Role      string    `bson:"role" json:"role"` // "user" or "assistant"
	Content   string    `bson:"content" json:"content"`
	Timestamp time.Time `bson:"timestamp" json:"timestamp"`
}

// GetChatHistory fetches the chat history for a specific user and page
func GetChatHistory(ctx context.Context, senderID, pageID string, limit int) ([]ChatHistory, error) {
	messagesCollection := database.Collection("messages")

	// If limit is not specified, default to 10
	if limit <= 0 {
		limit = 10
	}

	history := []ChatHistory{}

	// Get messages from both user and bot for this specific conversation
	// We need messages where either:
	// 1. sender_id is the user (senderID) - user messages to page
	// 2. recipient_id is the user (senderID) and it's a bot message - bot messages to user
	messageFilter := bson.M{
		"page_id": pageID,
		"type":    "chat",
		"$or": []bson.M{
			{"sender_id": senderID, "recipient_id": pageID},                 // User messages to page
			{"sender_id": pageID, "recipient_id": senderID, "is_bot": true}, // Bot messages to user
		},
	}

	findOptions := options.Find().
		SetSort(bson.M{"timestamp": -1}).
		SetLimit(int64(limit * 2)) // Get more messages since we're getting both user and bot

	messageCursor, err := messagesCollection.Find(ctx, messageFilter, findOptions)
	if err != nil {
		return nil, err
	}
	defer messageCursor.Close(ctx)

	var messages []models.Message
	if err := messageCursor.All(ctx, &messages); err != nil {
		return nil, err
	}

	// Convert messages to chat history
	for _, msg := range messages {
		role := "user"
		if msg.IsBot {
			role = "assistant"
		}
		history = append(history, ChatHistory{
			Role:      role,
			Content:   msg.Message,
			Timestamp: msg.Timestamp,
		})
	}

	// Sort by timestamp (oldest first for conversation context)
	for i := 0; i < len(history)-1; i++ {
		for j := i + 1; j < len(history); j++ {
			if history[i].Timestamp.After(history[j].Timestamp) {
				history[i], history[j] = history[j], history[i]
			}
		}
	}

	// Limit to requested number of messages
	if len(history) > limit {
		history = history[len(history)-limit:]
	}

	slog.Info("Chat history retrieved",
		"senderID", senderID,
		"pageID", pageID,
		"messagesFound", len(messages),
		"historyLength", len(history),
	)

	return history, nil
}

// GetCommentHistory fetches recent comments for context
func GetCommentHistory(ctx context.Context, postID string, limit int) ([]ChatHistory, error) {
	commentsCollection := database.Collection("comments")

	if limit <= 0 {
		limit = 5
	}

	history := []ChatHistory{}

	// Get recent comments and replies on the post
	filter := bson.M{"post_id": postID}

	findOptions := options.Find().
		SetSort(bson.M{"timestamp": -1}).
		SetLimit(int64(limit * 2)) // Get more to include both user and bot comments

	cursor, err := commentsCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var comments []models.Comment
	if err := cursor.All(ctx, &comments); err != nil {
		return nil, err
	}

	// Build history from comments
	for _, comment := range comments {
		role := "user"
		if comment.IsBot {
			role = "assistant"
		}

		content := comment.Message
		if !comment.IsBot && comment.SenderName != "" {
			content = comment.SenderName + ": " + comment.Message
		}

		history = append(history, ChatHistory{
			Role:      role,
			Content:   content,
			Timestamp: comment.Timestamp,
		})
	}

	// Sort by timestamp (oldest first)
	for i := 0; i < len(history)-1; i++ {
		for j := i + 1; j < len(history); j++ {
			if history[i].Timestamp.After(history[j].Timestamp) {
				history[i], history[j] = history[j], history[i]
			}
		}
	}

	// Limit the final history
	if len(history) > limit {
		history = history[len(history)-limit:]
	}

	return history, nil
}

// GetComment fetches a comment by its comment ID
func GetComment(ctx context.Context, commentID string) (*models.Comment, error) {
	collection := database.Collection("comments")

	var comment models.Comment
	err := collection.FindOne(ctx, bson.M{"comment_id": commentID}).Decode(&comment)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &comment, nil
}

// GetCommentWithReplies fetches a comment and all its replies
func GetCommentWithReplies(ctx context.Context, commentID string) (*models.Comment, []models.Comment, error) {
	collection := database.Collection("comments")

	// Get the parent comment
	parent, err := GetComment(ctx, commentID)
	if err != nil {
		return nil, nil, err
	}

	// Get all replies to this comment
	filter := bson.M{"parent_id": commentID}
	findOptions := options.Find().SetSort(bson.M{"timestamp": 1})

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return parent, nil, err
	}
	defer cursor.Close(ctx)

	var replies []models.Comment
	if err := cursor.All(ctx, &replies); err != nil {
		return parent, nil, err
	}

	return parent, replies, nil
}

// GetPostComments fetches all comments for a post (including replies)
func GetPostComments(ctx context.Context, postID string) ([]models.Comment, error) {
	collection := database.Collection("comments")

	filter := bson.M{"post_id": postID}
	findOptions := options.Find().SetSort(bson.M{"timestamp": 1})

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var comments []models.Comment
	if err := cursor.All(ctx, &comments); err != nil {
		return nil, err
	}

	return comments, nil
}

// GetPostCommentsThreaded fetches all comments for a post and organizes them by parent-child relationship
func GetPostCommentsThreaded(ctx context.Context, postID string) (map[string][]models.Comment, error) {
	comments, err := GetPostComments(ctx, postID)
	if err != nil {
		return nil, err
	}

	// Organize comments by parent ID
	threads := make(map[string][]models.Comment)

	for _, comment := range comments {
		if comment.IsReply && comment.ParentID != "" {
			// This is a reply, add it under its parent
			threads[comment.ParentID] = append(threads[comment.ParentID], comment)
		} else {
			// This is a parent comment, ensure it has an entry
			if _, exists := threads[comment.CommentID]; !exists {
				threads[comment.CommentID] = []models.Comment{}
			}
			// Add the parent comment with empty string key for root level
			threads[""] = append(threads[""], comment)
		}
	}

	return threads, nil
}
