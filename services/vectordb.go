package services

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	"facebook-bot/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// VectorDocument represents a document with embeddings in the vector DB
type VectorDocument struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CompanyID string             `bson:"company_id" json:"company_id"`
	PageID    string             `bson:"page_id" json:"page_id"`
	Content   string             `bson:"content" json:"content"`
	Embedding []float32          `bson:"embedding" json:"embedding"`
	Metadata  map[string]string  `bson:"metadata" json:"metadata"`
	Source    string             `bson:"source" json:"source"`                       // "crm", "product", "faq", etc.
	Channels  map[string]bool    `bson:"channels" json:"channels"`                   // {"facebook": true, "messenger": false} - true means enabled for that channel
	CRMURL    string             `bson:"crm_url,omitempty" json:"crm_url,omitempty"` // URL of the CRM link this document came from
	CRMID     string             `bson:"crm_id,omitempty" json:"crm_id,omitempty"`   // ID of the CRM link this document belongs to
	IsActive  bool               `bson:"is_active" json:"is_active"`                 // Whether this document should be used in searches
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// SearchResult represents a search result from vector DB
type SearchResult struct {
	Content  string            `json:"content"`
	Score    float32           `json:"score"`
	Source   string            `json:"source"`
	Metadata map[string]string `json:"metadata"`
}

// GetEmbeddings generates embeddings for text using Voyage AI
func GetEmbeddings(ctx context.Context, text string, companyID string, pageID string) ([]float32, error) {
	// Get company configuration
	company, err := GetCompanyByID(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get company config: %w", err)
	}

	// Find the page configuration
	var pageConfig *models.FacebookPage
	for _, page := range company.Pages {
		if page.PageID == pageID {
			pageConfig = &page
			break
		}
	}

	// If no pageID provided or page not found, use first page
	if pageConfig == nil && len(company.Pages) > 0 {
		pageConfig = &company.Pages[0]
	}

	if pageConfig == nil {
		return nil, fmt.Errorf("no page configuration found")
	}

	// Check if GPT is configured (prioritize GPT over Voyage)
	if pageConfig.GPTAPIKey != "" {
		model := pageConfig.GPTModel
		if model == "" {
			model = "text-embedding-3-large" // Default GPT embedding model
		}

		embeddings, err := GetOpenAIEmbeddings(ctx, []string{text}, pageConfig.GPTAPIKey, model)
		if err != nil {
			return nil, fmt.Errorf("GPT embedding failed: %w", err)
		}

		if len(embeddings) == 0 {
			return nil, fmt.Errorf("no embeddings generated")
		}

		return embeddings[0], nil
	}

	// Fall back to Voyage if configured
	if pageConfig.VoyageAPIKey != "" {
		model := pageConfig.VoyageModel
		if model == "" {
			model = "voyage-2" // Default Voyage model
		}

		embeddings, err := GetVoyageEmbeddings(ctx, []string{text}, pageConfig.VoyageAPIKey, model)
		if err != nil {
			return nil, fmt.Errorf("Voyage embedding failed: %w", err)
		}

		if len(embeddings) == 0 {
			return nil, fmt.Errorf("no embeddings generated")
		}

		return embeddings[0], nil
	}

	// If neither GPT nor Voyage is configured, use mock embeddings
	slog.Warn("No embedding API configured, using mock embeddings",
		"companyID", companyID,
		"pageID", pageID,
	)
	mockEmbeddings := GetMockEmbeddings([]string{text})
	return mockEmbeddings[0], nil
}

// CosineSimilarity calculates the cosine similarity between two vectors
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// StoreEmbeddings stores document embeddings in MongoDB
func StoreEmbeddings(ctx context.Context, companyID, pageID, content, source string, metadata map[string]string) error {
	return StoreEmbeddingsWithOptions(ctx, companyID, pageID, content, source, metadata, "", true)
}

// StoreEmbeddingsWithOptions stores document embeddings with additional options
func StoreEmbeddingsWithOptions(ctx context.Context, companyID, pageID, content, source string, metadata map[string]string, crmURL string, isActive bool) error {
	// Call the new function with empty channel for backward compatibility
	return StoreEmbeddingsWithChannelAndOptions(ctx, companyID, pageID, "", content, source, metadata, crmURL, isActive)
}

// StoreEmbeddingsWithChannelAndOptions stores embeddings with channel specification
func StoreEmbeddingsWithChannelAndOptions(ctx context.Context, companyID, pageID, channel, content, source string, metadata map[string]string, crmURL string, isActive bool) error {
	// Convert single channel to array for backward compatibility
	channels := []string{}
	if channel != "" {
		channels = []string{channel}
	}
	return StoreEmbeddingsWithChannelsAndOptions(ctx, companyID, pageID, channels, content, source, metadata, crmURL, isActive)
}

// StoreEmbeddingsWithCRMID stores embeddings with CRM ID specification
func StoreEmbeddingsWithCRMID(ctx context.Context, companyID, pageID, channel, content, source string, metadata map[string]string, crmURL, crmID string, isActive bool) error {
	// Convert single channel to array for backward compatibility
	channels := []string{}
	if channel != "" {
		channels = []string{channel}
	}
	return StoreEmbeddingsWithChannelsAndCRMID(ctx, companyID, pageID, channels, content, source, metadata, crmURL, crmID, isActive)
}

// StoreEmbeddingsWithChannelsAndOptions stores embeddings with multiple channels
func StoreEmbeddingsWithChannelsAndOptions(ctx context.Context, companyID, pageID string, channels []string, content, source string, metadata map[string]string, crmURL string, isActive bool) error {
	// Call the new function with empty CRM ID for backward compatibility
	return StoreEmbeddingsWithChannelsAndCRMID(ctx, companyID, pageID, channels, content, source, metadata, crmURL, "", isActive)
}

// StoreEmbeddingsWithChannelsAndCRMID stores embeddings with multiple channels and CRM ID
func StoreEmbeddingsWithChannelsAndCRMID(ctx context.Context, companyID, pageID string, channels []string, content, source string, metadata map[string]string, crmURL, crmID string, isActive bool) error {
	collection := database.Collection("vector_documents")

	// Generate embeddings using company's configured provider
	embedding, err := GetEmbeddings(ctx, content, companyID, pageID)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Convert channels array to map
	channelMap := make(map[string]bool)
	if len(channels) == 0 {
		// If no channels specified, default to both enabled
		channelMap["facebook"] = true
		channelMap["messenger"] = true
	} else {
		// Initialize all channels as false first
		channelMap["facebook"] = false
		channelMap["messenger"] = false
		// Then set the specified channels to true
		for _, ch := range channels {
			ch = normalizeChannel(ch)
			if ch == "facebook" || ch == "messenger" {
				channelMap[ch] = true
			}
		}
	}

	doc := VectorDocument{
		CompanyID: companyID,
		PageID:    pageID,
		Channels:  channelMap,
		Content:   content,
		Embedding: embedding,
		Metadata:  metadata,
		Source:    source,
		CRMURL:    crmURL,
		CRMID:     crmID,
		IsActive:  isActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Upsert based on CRM URL if provided, otherwise based on content
	var filter bson.M
	if crmURL != "" {
		// For CRM documents, use URL as unique identifier
		filter = bson.M{
			"company_id": companyID,
			"page_id":    pageID,
			"crm_url":    crmURL,
		}
	} else {
		// For other documents, use content as unique identifier
		filter = bson.M{
			"company_id": companyID,
			"content":    content,
		}
	}

	update := bson.M{"$set": doc}
	opts := options.Update().SetUpsert(true)

	_, err = collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to store embeddings: %w", err)
	}

	slog.Info("Stored embeddings",
		"companyID", companyID,
		"pageID", pageID,
		"source", source,
		"crmURL", crmURL,
		"isActive", isActive,
		"contentLength", len(content),
	)

	return nil
}

// SearchSimilarDocumentsByPage searches for similar documents filtered by page ID using cosine similarity
func SearchSimilarDocumentsByPage(ctx context.Context, query string, companyID string, pageID string, limit int) ([]SearchResult, error) {
	collection := database.Collection("vector_documents")

	// Generate query embedding using company's configured provider
	queryEmbedding, err := GetEmbeddings(ctx, query, companyID, pageID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Fetch documents filtered by company, page ID, and active status
	filter := bson.M{
		"company_id": companyID,
		"page_id":    pageID,
		"is_active":  true,
	}
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var documents []VectorDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, err
	}

	if len(documents) == 0 {
		slog.Info("No documents found for page",
			"pageID", pageID,
			"companyID", companyID,
		)
		return []SearchResult{}, nil
	}

	// Calculate cosine similarity scores for all documents
	type scoredDoc struct {
		doc   VectorDocument
		score float32
	}

	var scoredDocs []scoredDoc
	for _, doc := range documents {
		// Calculate cosine similarity between query and document embeddings
		score := CosineSimilarity(queryEmbedding, doc.Embedding)
		scoredDocs = append(scoredDocs, scoredDoc{doc: doc, score: score})
	}

	// Sort by cosine similarity score in descending order (highest similarity first)
	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].score > scoredDocs[j].score
	})

	// Dynamic threshold based on top score
	var results []SearchResult
	relevanceThreshold := float32(0.3) // Lower threshold to capture more relevant results

	// If top score is high, adjust threshold dynamically
	if len(scoredDocs) > 0 && scoredDocs[0].score > 0.7 {
		relevanceThreshold = scoredDocs[0].score * 0.5 // Take results with at least 50% of best score
	}

	// Collect results above threshold, up to limit
	for i := 0; i < len(scoredDocs) && len(results) < limit; i++ {
		if scoredDocs[i].score >= relevanceThreshold {
			results = append(results, SearchResult{
				Content:  scoredDocs[i].doc.Content,
				Score:    scoredDocs[i].score,
				Source:   scoredDocs[i].doc.Source,
				Metadata: scoredDocs[i].doc.Metadata,
			})
		}
	}

	// If no results meet threshold but we have documents, take top results anyway
	if len(results) == 0 && len(scoredDocs) > 0 {
		// Take at least top 3 results or limit, whichever is smaller
		takeCount := 3
		if limit < takeCount {
			takeCount = limit
		}
		if len(scoredDocs) < takeCount {
			takeCount = len(scoredDocs)
		}

		for i := 0; i < takeCount; i++ {
			results = append(results, SearchResult{
				Content:  scoredDocs[i].doc.Content,
				Score:    scoredDocs[i].score,
				Source:   scoredDocs[i].doc.Source,
				Metadata: scoredDocs[i].doc.Metadata,
			})
		}

		slog.Info("No results met threshold, using best available",
			"threshold", relevanceThreshold,
			"bestScore", scoredDocs[0].score,
			"resultsReturned", len(results),
			"pageID", pageID,
		)
	}

	slog.Info("Vector search completed using cosine similarity",
		"query", query,
		"companyID", companyID,
		"pageID", pageID,
		"resultsFound", len(results),
		"totalDocuments", len(documents),
		"topScore", func() float32 {
			if len(results) > 0 {
				return results[0].Score
			}
			return 0
		}(),
	)

	return results, nil
}

// SearchSimilarDocuments searches for similar documents using vector similarity
func SearchSimilarDocuments(ctx context.Context, query string, companyID string, limit int) ([]SearchResult, error) {
	collection := database.Collection("vector_documents")

	// Generate query embedding using company's configured provider
	// Use empty pageID to get default page config
	queryEmbedding, err := GetEmbeddings(ctx, query, companyID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Fetch all active documents for the company (in production, use vector index)
	filter := bson.M{
		"company_id": companyID,
		"is_active":  true,
	}
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var documents []VectorDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, err
	}

	// Calculate similarity scores
	type scoredDoc struct {
		doc   VectorDocument
		score float32
	}

	var scoredDocs []scoredDoc
	for _, doc := range documents {
		score := CosineSimilarity(queryEmbedding, doc.Embedding)
		scoredDocs = append(scoredDocs, scoredDoc{doc: doc, score: score})
	}

	// Sort by score
	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].score > scoredDocs[j].score
	})

	// Take top N results with dynamic threshold
	var results []SearchResult
	relevanceThreshold := float32(0.5) // Lowered threshold to get more results

	for i := 0; i < limit && i < len(scoredDocs); i++ {
		if scoredDocs[i].score > relevanceThreshold {
			results = append(results, SearchResult{
				Content:  scoredDocs[i].doc.Content,
				Score:    scoredDocs[i].score,
				Source:   scoredDocs[i].doc.Source,
				Metadata: scoredDocs[i].doc.Metadata,
			})
		}
	}

	// If no results meet the threshold, take the best matches anyway
	if len(results) == 0 && len(scoredDocs) > 0 {
		// Take top 2 results regardless of score for debugging
		for i := 0; i < 2 && i < len(scoredDocs); i++ {
			results = append(results, SearchResult{
				Content:  scoredDocs[i].doc.Content,
				Score:    scoredDocs[i].score,
				Source:   scoredDocs[i].doc.Source,
				Metadata: scoredDocs[i].doc.Metadata,
			})
		}
		slog.Warn("No results met threshold, using best available",
			"threshold", relevanceThreshold,
			"bestScore", scoredDocs[0].score,
		)
	}

	slog.Info("Vector search completed",
		"query", query,
		"companyID", companyID,
		"resultsFound", len(results),
	)

	return results, nil
}

// splitIntoLargeChunks splits text into larger chunks by size
func splitIntoLargeChunks(text string, maxSize int) []string {
	if len(text) <= maxSize {
		return []string{text}
	}

	var chunks []string
	lines := strings.Split(text, "\n")
	currentChunk := ""

	for _, line := range lines {
		// If adding this line would exceed max size, start new chunk
		if len(currentChunk)+len(line)+1 > maxSize && currentChunk != "" {
			chunks = append(chunks, currentChunk)
			currentChunk = line
		} else {
			if currentChunk != "" {
				currentChunk += "\n"
			}
			currentChunk += line
		}
	}

	// Add the last chunk
	if currentChunk != "" {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}

// GetRAGContext retrieves relevant context for a query using cosine similarity search
func GetRAGContext(ctx context.Context, query string, companyID string, pageID string) (string, error) {
	// Search using stored embeddings with cosine similarity (no API calls)
	results, err := SearchWithStoredEmbeddings(ctx, query, companyID, pageID, 5)
	if err != nil {
		slog.Error("Failed to search with stored embeddings", "error", err)
		return "", err
	}

	if len(results) == 0 {
		slog.Info("No relevant context found for query",
			"query", query,
			"companyID", companyID,
			"pageID", pageID,
		)
		return "", nil
	}

	// Build context from multiple results for comprehensive coverage
	var contexts []string
	totalLength := 0
	maxContextLength := 10000 // Max total context to prevent token overflow

	for i, result := range results {
		// Include confidence score to help AI gauge relevance
		confidence := "very high"
		if result.Score < 0.8 {
			confidence = "high"
		}
		if result.Score < 0.6 {
			confidence = "medium"
		}
		if result.Score < 0.4 {
			confidence = "low"
		}

		// For top results, include more content
		extractSize := 3000
		if i > 0 {
			extractSize = 2000 // Less for lower-ranked results
		}
		if i > 2 {
			extractSize = 1000 // Even less for results 4+
		}

		// Extract relevant portion of the document
		relevantContent := extractRelevantPortion(result.Content, query, extractSize)

		// Check if adding this would exceed max length
		contextEntry := fmt.Sprintf("[Result %d - Source: %s, Relevance: %s (%.3f)]\n%s",
			i+1, result.Source, confidence, result.Score, relevantContent)

		if totalLength+len(contextEntry) > maxContextLength {
			break // Stop adding more context to prevent overflow
		}

		contexts = append(contexts, contextEntry)
		totalLength += len(contextEntry)
	}

	ragContext := strings.Join(contexts, "\n\n---\n\n")

	// Log detailed RAG information for debugging
	slog.Info("RAG context retrieved",
		"query", query,
		"queryLength", len(query),
		"contextLength", len(ragContext),
		"sources", len(results),
		"topScore", results[0].Score,
		"companyID", companyID,
		"pageID", pageID,
	)

	// Debug log to see what content is being retrieved
	if len(ragContext) > 0 {
		preview := ragContext
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		slog.Debug("RAG context preview", "content", preview)
	}

	return ragContext, nil
}

// GetRAGContextForChannel retrieves relevant context for a query filtered by channel
func GetRAGContextForChannel(ctx context.Context, query string, companyID string, pageID string, channel string) (string, error) {
	// Search using stored embeddings with cosine similarity filtered by channel
	results, err := SearchWithStoredEmbeddingsForChannel(ctx, query, companyID, pageID, channel, 5)
	if err != nil {
		slog.Error("Failed to search with stored embeddings", "error", err)
		return "", err
	}

	if len(results) == 0 {
		slog.Info("No relevant context found for query",
			"query", query,
			"companyID", companyID,
			"pageID", pageID,
			"channel", channel,
		)
		return "", nil
	}

	// Build context from multiple results for comprehensive coverage
	var contexts []string
	totalLength := 0
	maxContextLength := 10000 // Max total context to prevent token overflow

	for i, result := range results {
		// Include confidence score to help AI gauge relevance
		confidence := "very high"
		if result.Score < 0.8 {
			confidence = "high"
		}
		if result.Score < 0.6 {
			confidence = "medium"
		}
		if result.Score < 0.4 {
			confidence = "low"
		}

		// For top results, include more content
		extractSize := 3000
		if i > 0 {
			extractSize = 2000 // Less for lower-ranked results
		}
		if i > 2 {
			extractSize = 1000 // Even less for results 4+
		}

		// Extract relevant portion of the document
		relevantContent := extractRelevantPortion(result.Content, query, extractSize)

		// Check if adding this would exceed max length
		contextEntry := fmt.Sprintf("[Result %d - Source: %s, Relevance: %s (%.3f)]\n%s",
			i+1, result.Source, confidence, result.Score, relevantContent)

		if totalLength+len(contextEntry) > maxContextLength {
			break // Stop adding more context to prevent overflow
		}

		contexts = append(contexts, contextEntry)
		totalLength += len(contextEntry)
	}

	ragContext := strings.Join(contexts, "\n\n---\n\n")

	// Log detailed RAG information for debugging
	slog.Info("RAG context retrieved for channel",
		"query", query,
		"queryLength", len(query),
		"contextLength", len(ragContext),
		"sources", len(results),
		"topScore", results[0].Score,
		"companyID", companyID,
		"pageID", pageID,
		"channel", channel,
	)

	// Debug log to see what content is being retrieved
	if len(ragContext) > 0 {
		preview := ragContext
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		slog.Debug("RAG context preview", "content", preview)
	}

	return ragContext, nil
}

// extractRelevantPortion extracts the most relevant portion of content based on query
func extractRelevantPortion(content, query string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}

	// Convert to lowercase for searching (works with Georgian text too)
	lowerContent := strings.ToLower(content)
	lowerQuery := strings.ToLower(query)

	// Split query into words for better matching
	queryWords := strings.Fields(lowerQuery)

	// Find the best position with most query word matches
	bestIndex := -1
	bestScore := 0

	// Sliding window to find the best section
	windowSize := 500 // Look in 500 char windows
	for i := 0; i < len(lowerContent)-windowSize; i += 100 {
		window := lowerContent[i:min(i+windowSize, len(lowerContent))]
		score := 0

		// Count how many query words appear in this window
		for _, word := range queryWords {
			if strings.Contains(window, word) {
				score++
			}
		}

		if score > bestScore {
			bestScore = score
			bestIndex = i
		}
	}

	// If no match found, try to find any single word match
	if bestIndex == -1 {
		for _, word := range queryWords {
			index := strings.Index(lowerContent, word)
			if index != -1 {
				bestIndex = index
				break
			}
		}
	}

	// If still no match, return the beginning of content
	if bestIndex == -1 {
		if len(content) > maxLength {
			// Try to cut at sentence boundary
			cutPoint := maxLength
			for i := maxLength - 1; i > maxLength-100 && i > 0; i-- {
				if content[i] == '.' || content[i] == '!' || content[i] == '?' || content[i] == '\n' {
					cutPoint = i + 1
					break
				}
			}
			return content[:cutPoint] + "..."
		}
		return content
	}

	// Extract content around the best match
	start := bestIndex - maxLength/3 // Show more content after match than before
	if start < 0 {
		start = 0
	}

	// Try to start at sentence or paragraph boundary
	if start > 0 {
		for i := start; i > max(0, start-200); i-- {
			if content[i] == '.' || content[i] == '\n' || content[i] == '!' || content[i] == '?' {
				start = i + 1
				if start < len(content) && (content[start] == ' ' || content[start] == '\n') {
					start++
				}
				break
			}
		}
	}

	end := start + maxLength
	if end > len(content) {
		end = len(content)
		start = max(0, end-maxLength)
	}

	// Try to end at sentence boundary
	if end < len(content) {
		for i := end; i < min(len(content), end+200); i++ {
			if content[i] == '.' || content[i] == '\n' || content[i] == '!' || content[i] == '?' {
				end = i + 1
				break
			}
		}
	}

	extracted := content[start:end]

	// Add ellipsis if truncated
	if start > 0 {
		extracted = "..." + extracted
	}
	if end < len(content) {
		extracted = extracted + "..."
	}

	return extracted
}

// Helper functions for min/max
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// GetAllVectorDocuments retrieves all vector documents for a company
func GetAllVectorDocuments(ctx context.Context, companyID string) ([]VectorDocument, error) {
	collection := database.Collection("vector_documents")

	filter := bson.M{"company_id": companyID}
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var documents []VectorDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, err
	}

	return documents, nil
}

// ToggleVectorDocumentStatus toggles the active status of a vector document
func ToggleVectorDocumentStatus(ctx context.Context, companyID, pageID, crmURL string, isActive bool) error {
	collection := database.Collection("vector_documents")

	filter := bson.M{
		"company_id": companyID,
		"page_id":    pageID,
		"crm_url":    crmURL,
	}

	update := bson.M{
		"$set": bson.M{
			"is_active":  isActive,
			"updated_at": time.Now(),
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to toggle vector document status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("vector document not found for CRM URL: %s", crmURL)
	}

	slog.Info("Toggled vector document status",
		"companyID", companyID,
		"pageID", pageID,
		"crmURL", crmURL,
		"isActive", isActive,
	)

	return nil
}

// HasActiveCRMDocuments checks if there are any active CRM documents in the vector database for a given page
func HasActiveCRMDocuments(ctx context.Context, companyID, pageID string) (bool, error) {
	collection := database.Collection("vector_documents")

	filter := bson.M{
		"company_id": companyID,
		"page_id":    pageID,
		"source":     "crm",
		"is_active":  true,
	}

	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to count active CRM documents: %w", err)
	}

	return count > 0, nil
}

// HasActiveCRMDocumentsForChannel checks if there are active CRM documents for a specific channel
func HasActiveCRMDocumentsForChannel(ctx context.Context, companyID, pageID, channel string) (bool, error) {
	collection := database.Collection("vector_documents")

	// Normalize the channel name
	channel = normalizeChannel(channel)

	// Check if channel is enabled in the channels map
	filter := bson.M{
		"company_id":          companyID,
		"page_id":             pageID,
		"channels." + channel: true, // Check if the channel is enabled in the channels map
		"source":              "crm",
		"is_active":           true,
	}

	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to count active CRM documents: %w", err)
	}

	return count > 0, nil
}

// UpdateVectorDocumentChannels updates the channels array for documents matching the filename
func UpdateVectorDocumentChannels(ctx context.Context, companyID, pageID, filename string, channels []string) (int64, error) {
	collection := database.Collection("vector_documents")

	// Convert channels array to map
	channelMap := make(map[string]bool)
	// Initialize both channels as false
	channelMap["facebook"] = false
	channelMap["messenger"] = false

	// Set specified channels to true
	for _, ch := range channels {
		ch = normalizeChannel(ch)
		if ch == "facebook" || ch == "messenger" {
			channelMap[ch] = true
		}
	}

	// Find and update all documents with this filename
	filter := bson.M{
		"company_id":        companyID,
		"page_id":           pageID,
		"metadata.filename": filename,
	}

	update := bson.M{
		"$set": bson.M{
			"channels":   channelMap,
			"updated_at": time.Now(),
		},
	}

	result, err := collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, fmt.Errorf("failed to update document channels: %w", err)
	}

	slog.Info("Updated document channels",
		"companyID", companyID,
		"pageID", pageID,
		"filename", filename,
		"channels", channels,
		"updatedCount", result.ModifiedCount,
	)

	return result.ModifiedCount, nil
}

// UpdateVectorDocumentChannelValue updates a specific channel value for documents matching the filename
func UpdateVectorDocumentChannelValue(ctx context.Context, companyID, pageID, filename, platform string, value bool) (int64, error) {
	collection := database.Collection("vector_documents")

	// Normalize the platform name
	platform = normalizeChannel(platform)
	if platform != "facebook" && platform != "messenger" {
		return 0, fmt.Errorf("invalid platform: %s", platform)
	}

	// Find all documents with this filename
	filter := bson.M{
		"company_id":        companyID,
		"page_id":           pageID,
		"metadata.filename": filename,
	}

	// First, get the current documents to preserve other channel values
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to find documents: %w", err)
	}
	defer cursor.Close(ctx)

	var documents []VectorDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return 0, fmt.Errorf("failed to decode documents: %w", err)
	}

	if len(documents) == 0 {
		return 0, nil // No documents found
	}

	// Update the specific channel value while preserving others
	update := bson.M{
		"$set": bson.M{
			"channels." + platform: value,
			"updated_at":           time.Now(),
		},
	}

	result, err := collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, fmt.Errorf("failed to update document channel value: %w", err)
	}

	slog.Info("Updated document channel value",
		"companyID", companyID,
		"pageID", pageID,
		"filename", filename,
		"platform", platform,
		"value", value,
		"updatedCount", result.ModifiedCount,
	)

	return result.ModifiedCount, nil
}

// SyncVectorDocumentWithCRMLink syncs vector document active status with CRM link status
func SyncVectorDocumentWithCRMLink(ctx context.Context, companyID, pageID, crmURL string, isActive bool) error {
	// When a CRM link is toggled, update the corresponding vector document
	return ToggleVectorDocumentStatus(ctx, companyID, pageID, crmURL, isActive)
}

// DeleteVectorDocumentByContent deletes a vector document by its content
func DeleteVectorDocumentByContent(ctx context.Context, companyID, pageID, content string) (int64, error) {
	collection := database.Collection("vector_documents")

	result, err := collection.DeleteOne(ctx, bson.M{
		"company_id": companyID,
		"page_id":    pageID,
		"content":    content,
	})

	if err != nil {
		return 0, fmt.Errorf("failed to delete vector document: %w", err)
	}

	return result.DeletedCount, nil
}

// DeleteVectorDocumentsByMetadata deletes vector documents by metadata field
func DeleteVectorDocumentsByMetadata(ctx context.Context, companyID, pageID, metadataKey, metadataValue string) (int64, error) {
	collection := database.Collection("vector_documents")

	filter := bson.M{
		"company_id":                            companyID,
		"page_id":                               pageID,
		fmt.Sprintf("metadata.%s", metadataKey): metadataValue,
	}

	result, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete vector documents: %w", err)
	}

	return result.DeletedCount, nil
}

// normalizeChannels ensures channel values are properly formatted
// normalizeChannel normalizes a single channel name
func normalizeChannel(channel string) string {
	// Normalize channel names to lowercase
	ch := strings.ToLower(strings.TrimSpace(channel))

	// Fix common misspellings and ensure correct values
	switch ch {
	case "facebook", "fb":
		return "facebook"
	case "messenger", "messanger", "msg":
		return "messenger"
	default:
		return ch
	}
}

func normalizeChannels(channels map[string]bool) map[string]bool {
	if channels == nil {
		return make(map[string]bool)
	}

	normalized := make(map[string]bool)
	for ch, enabled := range channels {
		ch = normalizeChannel(ch)
		if ch == "facebook" || ch == "messenger" {
			normalized[ch] = enabled
		}
	}

	return normalized
}

// GetAllVectorDocumentsByCompany retrieves all vector documents for a company
func GetAllVectorDocumentsByCompany(ctx context.Context, companyID string) ([]VectorDocument, error) {
	collection := database.Collection("vector_documents")

	filter := bson.M{
		"company_id": companyID,
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find vector documents: %w", err)
	}
	defer cursor.Close(ctx)

	var documents []VectorDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, fmt.Errorf("failed to decode vector documents: %w", err)
	}

	return documents, nil
}

// CheckRAGDocumentExists checks if a RAG document with specific metadata exists
func CheckRAGDocumentExists(ctx context.Context, companyID, pageID, metadataKey, metadataValue string) (bool, error) {
	collection := database.Collection("vector_documents")

	filter := bson.M{
		"company_id":              companyID,
		"page_id":                 pageID,
		"metadata." + metadataKey: metadataValue,
	}

	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to check document existence: %w", err)
	}

	return count > 0, nil
}

// GetVectorDocuments retrieves all vector documents for a page
func GetVectorDocuments(ctx context.Context, companyID, pageID string) ([]VectorDocument, error) {
	collection := database.Collection("vector_documents")

	filter := bson.M{
		"company_id": companyID,
		"page_id":    pageID,
	}

	cursor, err := collection.Find(ctx, filter, options.Find().SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve vector documents: %w", err)
	}
	defer cursor.Close(ctx)

	var documents []VectorDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, fmt.Errorf("failed to decode vector documents: %w", err)
	}

	// Normalize channels for each document
	for i := range documents {
		documents[i].Channels = normalizeChannels(documents[i].Channels)
	}

	return documents, nil
}

// ToggleVectorDocumentByID toggles the active status of a single document by ID
func ToggleVectorDocumentByID(ctx context.Context, documentID string, isActive bool) (int64, error) {
	collection := database.Collection("vector_documents")

	// Convert string ID to ObjectID
	objID, err := primitive.ObjectIDFromHex(documentID)
	if err != nil {
		return 0, fmt.Errorf("invalid document ID: %w", err)
	}

	result, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": objID},
		bson.M{
			"$set": bson.M{
				"is_active":  isActive,
				"updated_at": time.Now(),
			},
		},
	)

	if err != nil {
		return 0, fmt.Errorf("failed to toggle document status: %w", err)
	}

	return result.ModifiedCount, nil
}

// ToggleVectorDocumentsByMetadata toggles the active status of documents by metadata field
func ToggleVectorDocumentsByMetadata(ctx context.Context, companyID, pageID, metadataKey, metadataValue string, isActive bool) (int64, error) {
	collection := database.Collection("vector_documents")

	filter := bson.M{
		"company_id":                            companyID,
		"page_id":                               pageID,
		fmt.Sprintf("metadata.%s", metadataKey): metadataValue,
	}

	result, err := collection.UpdateMany(
		ctx,
		filter,
		bson.M{
			"$set": bson.M{
				"is_active":  isActive,
				"updated_at": time.Now(),
			},
		},
	)

	if err != nil {
		return 0, fmt.Errorf("failed to toggle documents status: %w", err)
	}

	return result.ModifiedCount, nil
}

// InitVectorDB creates indexes for vector documents collection
func InitVectorDB(ctx context.Context) error {
	collection := database.Collection("vector_documents")

	// Create indexes
	indexes := []mongo.IndexModel{
		{Keys: bson.M{"company_id": 1}},
		{Keys: bson.M{"page_id": 1}},
		{Keys: bson.M{"source": 1}},
		{Keys: bson.M{"created_at": -1}},
		{Keys: bson.D{
			{Key: "company_id", Value: 1},
			{Key: "content", Value: 1},
		}, Options: options.Index().SetUnique(true)},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create vector DB indexes: %w", err)
	}

	slog.Info("Vector DB initialized")
	return nil
}
