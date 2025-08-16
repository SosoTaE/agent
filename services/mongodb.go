package services

import (
	"context"
	"fmt"
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

// GetMongoClient returns the MongoDB client instance
func GetMongoClient() *mongo.Client {
	return mongoClient
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

// TrackNameChange tracks when a user changes their display name
func TrackNameChange(ctx context.Context, comment *models.Comment) (*models.NameChange, error) {
	if comment.SenderID == "" || comment.SenderName == "" {
		return nil, nil
	}

	collection := database.Collection("name_history")
	now := time.Now()

	// Find existing history for this sender
	var history models.NameHistory
	err := collection.FindOne(ctx, bson.M{
		"sender_id": comment.SenderID,
		"page_id":   comment.PageID,
	}).Decode(&history)

	if err == mongo.ErrNoDocuments {
		// Create new history entry
		history = models.NameHistory{
			SenderID: comment.SenderID,
			PageID:   comment.PageID,
			Names: []models.NameRecord{{
				Name:      comment.SenderName,
				FirstSeen: now,
				LastSeen:  now,
				Count:     1,
			}},
			CreatedAt: now,
			UpdatedAt: now,
		}

		_, saveErr := collection.InsertOne(ctx, history)
		return nil, saveErr
	} else if err != nil {
		return nil, err
	}

	// Check if this is a new name
	var lastUsedName string
	var nameFound bool

	for i, nameRecord := range history.Names {
		if nameRecord.Name == comment.SenderName {
			// Update existing name record
			history.Names[i].LastSeen = now
			history.Names[i].Count++
			nameFound = true
		}
		// Track the most recently used name (before this comment)
		if lastUsedName == "" || nameRecord.LastSeen.After(history.Names[0].LastSeen) {
			if nameRecord.Name != comment.SenderName {
				lastUsedName = nameRecord.Name
			}
		}
	}

	var nameChange *models.NameChange

	if !nameFound {
		// This is a new name - add it to the history
		history.Names = append(history.Names, models.NameRecord{
			Name:      comment.SenderName,
			FirstSeen: now,
			LastSeen:  now,
			Count:     1,
		})

		// Create name change record if there was a previous name
		if lastUsedName != "" {
			nameChange = &models.NameChange{
				SenderID:  comment.SenderID,
				PageID:    comment.PageID,
				OldName:   lastUsedName,
				NewName:   comment.SenderName,
				ChangedAt: now,
				CommentID: comment.CommentID,
				PostID:    comment.PostID,
			}
		}
	}

	// Update the history record
	history.UpdatedAt = now
	_, err = collection.ReplaceOne(ctx,
		bson.M{"_id": history.ID},
		history,
	)

	return nameChange, err
}

// SaveComment saves a comment or reply to the comments collection
func SaveComment(ctx context.Context, commentID, postID, parentID, postContent, senderID, senderName, pageID, pageName, message string, isBot bool) error {
	return SaveCommentWithNames(ctx, commentID, postID, parentID, postContent, senderID, senderName, "", "", pageID, pageName, message, isBot)
}

// SaveCommentWithNames saves a comment or reply to the comments collection with first and last name
func SaveCommentWithNames(ctx context.Context, commentID, postID, parentID, postContent, senderID, senderName, firstName, lastName, pageID, pageName, message string, isBot bool) error {
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
		FirstName:   firstName,
		LastName:    lastName,
		PageID:      pageID,
		PageName:    pageName,
		Message:     message,
		PostContent: postContent,
		IsReply:     isReply,
		IsBot:       isBot,
		Timestamp:   time.Now(),
		UpdatedAt:   time.Now(),
		Replies:     []models.Comment{}, // Initialize empty replies array
	}

	// Track name changes for non-bot comments
	if !isBot && senderID != "" && senderName != "" {
		nameChange, err := TrackNameChange(ctx, &comment)
		if err != nil {
			slog.Error("Failed to track name change", "error", err, "sender_id", senderID)
			// Continue saving comment even if name tracking fails
		}
		if nameChange != nil {
			slog.Info("Name change detected while saving comment",
				"sender_id", nameChange.SenderID,
				"old_name", nameChange.OldName,
				"new_name", nameChange.NewName,
			)
		}
	}

	// If this is a reply, add it to the parent comment's replies array
	if isReply {
		// Find the parent comment and add this reply to its replies array
		filter := bson.M{"comment_id": parentID}
		update := bson.M{
			"$push": bson.M{"replies": comment},
			"$set":  bson.M{"updated_at": time.Now()},
		}

		result, err := collection.UpdateOne(ctx, filter, update)
		if err != nil {
			slog.Error("Failed to add reply to parent comment",
				"error", err,
				"parentID", parentID,
				"replyID", commentID,
			)
			return err
		}

		if result.MatchedCount == 0 {
			slog.Warn("Parent comment not found, saving reply as standalone comment",
				"parentID", parentID,
				"replyID", commentID,
			)
			// If parent not found, save as standalone comment
			filter := bson.M{"comment_id": commentID}
			update := bson.M{"$set": comment}
			opts := options.Update().SetUpsert(true)
			_, err = collection.UpdateOne(ctx, filter, update, opts)
			return err
		}

		slog.Info("Reply added to parent comment",
			"parentID", parentID,
			"replyID", commentID,
			"isBot", isBot,
		)
	} else {
		// This is a top-level comment, save it directly
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
	}

	return nil
}

// CheckCommentProcessed checks if we've already responded to a comment
func CheckCommentProcessed(ctx context.Context, commentID string) (bool, error) {
	collection := database.Collection("comments")

	// First check if the comment exists and has bot replies
	var comment models.Comment
	err := collection.FindOne(ctx, bson.M{"comment_id": commentID}).Decode(&comment)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}

	// Check if any of the replies are from the bot
	for _, reply := range comment.Replies {
		if reply.IsBot {
			return true, nil
		}
	}

	return false, nil
}

// SaveBotReply saves the bot's reply as a nested reply to the parent comment
func SaveBotReply(ctx context.Context, responseCommentID, parentCommentID, postID, botMessage, pageID, pageName string) error {
	// Bot replies are always replies to user comments, so we save them as nested replies
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

	// Get recent top-level comments (replies are now nested)
	filter := bson.M{
		"post_id": postID,
		"$or": []bson.M{
			{"parent_id": ""},
			{"parent_id": postID},
			{"parent_id": bson.M{"$exists": false}},
		},
	}

	findOptions := options.Find().
		SetSort(bson.M{"timestamp": -1}).
		SetLimit(int64(limit))

	cursor, err := commentsCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var comments []models.Comment
	if err := cursor.All(ctx, &comments); err != nil {
		return nil, err
	}

	// Build history from comments and their replies
	for _, comment := range comments {
		// Add the parent comment
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

		// Add all replies to this comment
		for _, reply := range comment.Replies {
			replyRole := "user"
			if reply.IsBot {
				replyRole = "assistant"
			}

			replyContent := reply.Message
			if !reply.IsBot && reply.SenderName != "" {
				replyContent = reply.SenderName + ": " + reply.Message
			}

			history = append(history, ChatHistory{
				Role:      replyRole,
				Content:   replyContent,
				Timestamp: reply.Timestamp,
			})
		}
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

// GetCommentWithReplies fetches a comment and all its replies (now from the nested structure)
func GetCommentWithReplies(ctx context.Context, commentID string) (*models.Comment, []models.Comment, error) {
	// Get the parent comment which now includes replies in its structure
	parent, err := GetComment(ctx, commentID)
	if err != nil {
		return nil, nil, err
	}

	if parent == nil {
		return nil, nil, nil
	}

	// The replies are now embedded in the parent comment
	return parent, parent.Replies, nil
}

// GetPostComments fetches all top-level comments for a post (replies are nested within)
func GetPostComments(ctx context.Context, postID string) ([]models.Comment, error) {
	collection := database.Collection("comments")

	// Only fetch top-level comments (those without a parent or where parent equals post)
	filter := bson.M{
		"post_id": postID,
		"$or": []bson.M{
			{"parent_id": ""},
			{"parent_id": postID},
			{"parent_id": bson.M{"$exists": false}},
		},
	}
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

// CommentWithReplies represents a comment with its nested replies
type CommentWithReplies struct {
	models.Comment
	Replies []CommentWithReplies `json:"replies"`
}

// GetPostCommentsHierarchical fetches all comments for a post in a hierarchical structure
func GetPostCommentsHierarchical(ctx context.Context, postID string) ([]CommentWithReplies, error) {
	return GetPostCommentsHierarchicalWithFilter(ctx, postID, nil)
}

// GetPostCommentsHierarchicalWithFilter fetches comments with optional page filtering
func GetPostCommentsHierarchicalWithFilter(ctx context.Context, postID string, pageIDs []string) ([]CommentWithReplies, error) {
	collection := database.Collection("comments")

	// Build filter - now we only fetch top-level comments since replies are nested
	filter := bson.M{
		"post_id": postID,
		"$or": []bson.M{
			{"parent_id": ""},
			{"parent_id": postID},
			{"parent_id": bson.M{"$exists": false}},
		},
	}
	if len(pageIDs) > 0 {
		filter["page_id"] = bson.M{"$in": pageIDs}
	}
	findOptions := options.Find().SetSort(bson.M{"timestamp": 1})

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var topLevelComments []models.Comment
	if err := cursor.All(ctx, &topLevelComments); err != nil {
		return nil, err
	}

	slog.Info("GetPostCommentsHierarchical",
		"postID", postID,
		"topLevelComments", len(topLevelComments),
	)

	// Convert to CommentWithReplies structure
	var rootComments []CommentWithReplies
	totalReplies := 0

	for _, comment := range topLevelComments {
		// Convert nested replies recursively
		cwp := convertToCommentWithReplies(comment)
		rootComments = append(rootComments, cwp)
		totalReplies += countReplies(comment.Replies)
	}

	slog.Info("Hierarchical structure built",
		"rootComments", len(rootComments),
		"totalReplies", totalReplies,
	)

	return rootComments, nil
}

// convertToCommentWithReplies converts a Comment with nested replies to CommentWithReplies
func convertToCommentWithReplies(comment models.Comment) CommentWithReplies {
	cwp := CommentWithReplies{
		Comment: comment,
		Replies: []CommentWithReplies{},
	}

	// Convert each nested reply
	for _, reply := range comment.Replies {
		replyWithReplies := convertToCommentWithReplies(reply)
		cwp.Replies = append(cwp.Replies, replyWithReplies)
	}

	return cwp
}

// countReplies counts total replies including nested ones
func countReplies(replies []models.Comment) int {
	count := len(replies)
	for _, reply := range replies {
		count += countReplies(reply.Replies)
	}
	return count
}

// Helper function to get map keys for debugging
func getMapKeys(m map[string]*CommentWithReplies) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Helper function to get reason why comment is root
func getRootReason(comment models.Comment, postID string) string {
	if !comment.IsReply {
		return "not marked as reply"
	}
	if comment.ParentID == "" {
		return "empty parent_id"
	}
	if comment.ParentID == postID {
		return "parent_id equals post_id"
	}
	return "unknown"
}

// Helper minInt function
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// PostInfo contains basic information about a post
type PostInfo struct {
	PostID       string    `json:"post_id"`
	PageID       string    `json:"page_id"`
	PageName     string    `json:"page_name"`
	FirstComment time.Time `json:"first_comment"`
	LastComment  time.Time `json:"last_comment"`
	CommentCount int       `json:"comment_count"`
}

// GetAllPostIDs retrieves all unique post IDs from the comments collection
func GetAllPostIDs(ctx context.Context) ([]string, error) {
	collection := database.Collection("comments")

	// Use aggregation to get unique post IDs
	pipeline := mongo.Pipeline{
		{{"$group", bson.D{
			{"_id", "$post_id"},
		}}},
		{{"$sort", bson.D{{"_id", 1}}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var postIDs []string
	for cursor.Next(ctx) {
		var result struct {
			ID string `bson:"_id"`
		}
		if err := cursor.Decode(&result); err != nil {
			slog.Error("Failed to decode post ID", "error", err)
			continue
		}
		if result.ID != "" {
			postIDs = append(postIDs, result.ID)
		}
	}

	return postIDs, nil
}

// GetAllPostIDsWithInfo retrieves all unique post IDs with additional information
func GetAllPostIDsWithInfo(ctx context.Context) ([]PostInfo, error) {
	collection := database.Collection("comments")

	// Use aggregation to get post info
	pipeline := mongo.Pipeline{
		{{"$group", bson.D{
			{"_id", "$post_id"},
			{"page_id", bson.D{{"$first", "$page_id"}}},
			{"page_name", bson.D{{"$first", "$page_name"}}},
			{"first_comment", bson.D{{"$min", "$timestamp"}}},
			{"last_comment", bson.D{{"$max", "$timestamp"}}},
			{"comment_count", bson.D{{"$sum", 1}}},
		}}},
		{{"$sort", bson.D{{"last_comment", -1}}}}, // Sort by most recent activity
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var posts []PostInfo
	for cursor.Next(ctx) {
		var result struct {
			ID           string    `bson:"_id"`
			PageID       string    `bson:"page_id"`
			PageName     string    `bson:"page_name"`
			FirstComment time.Time `bson:"first_comment"`
			LastComment  time.Time `bson:"last_comment"`
			CommentCount int       `bson:"comment_count"`
		}
		if err := cursor.Decode(&result); err != nil {
			slog.Error("Failed to decode post info", "error", err)
			continue
		}
		if result.ID != "" {
			posts = append(posts, PostInfo{
				PostID:       result.ID,
				PageID:       result.PageID,
				PageName:     result.PageName,
				FirstComment: result.FirstComment,
				LastComment:  result.LastComment,
				CommentCount: result.CommentCount,
			})
		}
	}

	return posts, nil
}

// GetPostIDsPaginated retrieves post IDs with pagination
func GetPostIDsPaginated(ctx context.Context, page, limit int) ([]PostInfo, int64, error) {
	collection := database.Collection("comments")

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	skip := (page - 1) * limit

	// First, get total unique posts count
	pipeline := mongo.Pipeline{
		{{"$group", bson.D{
			{"_id", "$post_id"},
		}}},
		{{"$count", "total"}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}

	var countResult struct {
		Total int64 `bson:"total"`
	}
	if cursor.Next(ctx) {
		cursor.Decode(&countResult)
	}
	cursor.Close(ctx)

	// Now get paginated results with info
	pipeline = mongo.Pipeline{
		{{"$group", bson.D{
			{"_id", "$post_id"},
			{"page_id", bson.D{{"$first", "$page_id"}}},
			{"page_name", bson.D{{"$first", "$page_name"}}},
			{"first_comment", bson.D{{"$min", "$timestamp"}}},
			{"last_comment", bson.D{{"$max", "$timestamp"}}},
			{"comment_count", bson.D{{"$sum", 1}}},
		}}},
		{{"$sort", bson.D{{"last_comment", -1}}}},
		{{"$skip", skip}},
		{{"$limit", limit}},
	}

	cursor, err = collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var posts []PostInfo
	for cursor.Next(ctx) {
		var result struct {
			ID           string    `bson:"_id"`
			PageID       string    `bson:"page_id"`
			PageName     string    `bson:"page_name"`
			FirstComment time.Time `bson:"first_comment"`
			LastComment  time.Time `bson:"last_comment"`
			CommentCount int       `bson:"comment_count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		if result.ID != "" {
			posts = append(posts, PostInfo{
				PostID:       result.ID,
				PageID:       result.PageID,
				PageName:     result.PageName,
				FirstComment: result.FirstComment,
				LastComment:  result.LastComment,
				CommentCount: result.CommentCount,
			})
		}
	}

	return posts, countResult.Total, nil
}

// GetPostIDsByPage retrieves all post IDs for a specific page
func GetPostIDsByPage(ctx context.Context, pageID string) ([]PostInfo, error) {
	collection := database.Collection("comments")

	pipeline := mongo.Pipeline{
		{{"$match", bson.D{{"page_id", pageID}}}},
		{{"$group", bson.D{
			{"_id", "$post_id"},
			{"page_name", bson.D{{"$first", "$page_name"}}},
			{"first_comment", bson.D{{"$min", "$timestamp"}}},
			{"last_comment", bson.D{{"$max", "$timestamp"}}},
			{"comment_count", bson.D{{"$sum", 1}}},
		}}},
		{{"$sort", bson.D{{"last_comment", -1}}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var posts []PostInfo
	for cursor.Next(ctx) {
		var result struct {
			ID           string    `bson:"_id"`
			PageName     string    `bson:"page_name"`
			FirstComment time.Time `bson:"first_comment"`
			LastComment  time.Time `bson:"last_comment"`
			CommentCount int       `bson:"comment_count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		if result.ID != "" {
			posts = append(posts, PostInfo{
				PostID:       result.ID,
				PageID:       pageID,
				PageName:     result.PageName,
				FirstComment: result.FirstComment,
				LastComment:  result.LastComment,
				CommentCount: result.CommentCount,
			})
		}
	}

	return posts, nil
}

// GetPostIDsByCompany retrieves all post IDs for a company's pages
func GetPostIDsByCompany(ctx context.Context, companyID string) ([]PostInfo, error) {
	// Get all pages for this company
	pages, err := GetPagesByCompanyID(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get company pages: %w", err)
	}

	// Extract page IDs
	pageIDs := make([]string, 0, len(pages))
	for _, page := range pages {
		pageIDs = append(pageIDs, page.PageID)
	}

	if len(pageIDs) == 0 {
		return []PostInfo{}, nil
	}

	collection := database.Collection("comments")

	pipeline := mongo.Pipeline{
		{{"$match", bson.D{{"page_id", bson.D{{"$in", pageIDs}}}}}},
		{{"$group", bson.D{
			{"_id", "$post_id"},
			{"page_id", bson.D{{"$first", "$page_id"}}},
			{"page_name", bson.D{{"$first", "$page_name"}}},
			{"first_comment", bson.D{{"$min", "$timestamp"}}},
			{"last_comment", bson.D{{"$max", "$timestamp"}}},
			{"comment_count", bson.D{{"$sum", 1}}},
		}}},
		{{"$sort", bson.D{{"last_comment", -1}}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var posts []PostInfo
	for cursor.Next(ctx) {
		var result struct {
			ID           string    `bson:"_id"`
			PageID       string    `bson:"page_id"`
			PageName     string    `bson:"page_name"`
			FirstComment time.Time `bson:"first_comment"`
			LastComment  time.Time `bson:"last_comment"`
			CommentCount int       `bson:"comment_count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		if result.ID != "" {
			posts = append(posts, PostInfo{
				PostID:       result.ID,
				PageID:       result.PageID,
				PageName:     result.PageName,
				FirstComment: result.FirstComment,
				LastComment:  result.LastComment,
				CommentCount: result.CommentCount,
			})
		}
	}

	return posts, nil
}

// ValidatePostOwnership checks if a post belongs to a company's pages
func ValidatePostOwnership(ctx context.Context, postID string, companyID string) (string, error) {
	// Get all pages for this company
	pages, err := GetPagesByCompanyID(ctx, companyID)
	if err != nil {
		return "", fmt.Errorf("failed to get company pages: %w", err)
	}

	// Get one comment to check page_id
	collection := database.Collection("comments")
	var sampleComment models.Comment
	err = collection.FindOne(ctx, bson.M{"post_id": postID}).Decode(&sampleComment)
	if err != nil {
		return "", fmt.Errorf("post not found: %w", err)
	}

	// Check if the page_id belongs to this company
	for _, page := range pages {
		if page.PageID == sampleComment.PageID {
			return sampleComment.PageID, nil // Return the page ID
		}
	}

	return "", fmt.Errorf("post does not belong to company")
}

// GetCompanyPageIDs returns all page IDs for a company
func GetCompanyPageIDs(ctx context.Context, companyID string) ([]string, error) {
	pages, err := GetPagesByCompanyID(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get company pages: %w", err)
	}

	pageIDs := make([]string, 0, len(pages))
	for _, page := range pages {
		pageIDs = append(pageIDs, page.PageID)
	}

	return pageIDs, nil
}

// GetPostCommentsForCompany fetches comments for a post only if it belongs to company pages
func GetPostCommentsForCompany(ctx context.Context, postID string, companyID string) ([]models.Comment, error) {
	// Validate post ownership
	pageID, err := ValidatePostOwnership(ctx, postID, companyID)
	if err != nil {
		return nil, err
	}

	// Get top-level comments filtered by the page (replies are nested)
	collection := database.Collection("comments")
	filter := bson.M{
		"post_id": postID,
		"page_id": pageID,
		"$or": []bson.M{
			{"parent_id": ""},
			{"parent_id": postID},
			{"parent_id": bson.M{"$exists": false}},
		},
	}
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

// GetPostCommentsHierarchicalForCompany fetches hierarchical comments for a post only if it belongs to company pages
func GetPostCommentsHierarchicalForCompany(ctx context.Context, postID string, companyID string) ([]CommentWithReplies, error) {
	// Validate post ownership and get the page ID
	pageID, err := ValidatePostOwnership(ctx, postID, companyID)
	if err != nil {
		return nil, err
	}

	// Get hierarchical comments filtered by the page
	return GetPostCommentsHierarchicalWithFilter(ctx, postID, []string{pageID})
}

// GetPostCommentsThreaded fetches all comments for a post and organizes them by parent-child relationship
func GetPostCommentsThreaded(ctx context.Context, postID string) (map[string][]models.Comment, error) {
	// Get top-level comments (replies are now nested within)
	topLevelComments, err := GetPostComments(ctx, postID)
	if err != nil {
		return nil, err
	}

	// Organize comments by parent ID
	threads := make(map[string][]models.Comment)

	// Add all top-level comments to the root thread
	threads[""] = topLevelComments

	// For each top-level comment, add its replies to the threads map
	for _, comment := range topLevelComments {
		if len(comment.Replies) > 0 {
			threads[comment.CommentID] = comment.Replies
			// Recursively add nested replies
			addRepliesToThreads(comment.Replies, threads)
		} else {
			// Ensure every comment has an entry even if no replies
			threads[comment.CommentID] = []models.Comment{}
		}
	}

	return threads, nil
}

// addRepliesToThreads recursively adds nested replies to the threads map
func addRepliesToThreads(replies []models.Comment, threads map[string][]models.Comment) {
	for _, reply := range replies {
		if len(reply.Replies) > 0 {
			threads[reply.CommentID] = reply.Replies
			addRepliesToThreads(reply.Replies, threads)
		} else {
			threads[reply.CommentID] = []models.Comment{}
		}
	}
}
