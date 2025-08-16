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
	Source    string             `bson:"source" json:"source"` // "crm", "product", "faq", etc.
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

	// Check if Voyage is configured
	if pageConfig.VoyageAPIKey == "" {
		slog.Warn("No Voyage API key configured, using mock embeddings",
			"companyID", companyID,
			"pageID", pageID,
		)
		mockEmbeddings := GetMockEmbeddings([]string{text})
		return mockEmbeddings[0], nil
	}

	// Use Voyage embeddings
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
	collection := database.Collection("vector_documents")

	// Generate embeddings using company's configured provider
	embedding, err := GetEmbeddings(ctx, content, companyID, pageID)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	doc := VectorDocument{
		CompanyID: companyID,
		PageID:    pageID,
		Content:   content,
		Embedding: embedding,
		Metadata:  metadata,
		Source:    source,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Upsert based on content hash to avoid duplicates
	filter := bson.M{
		"company_id": companyID,
		"content":    content,
	}

	update := bson.M{"$set": doc}
	opts := options.Update().SetUpsert(true)

	_, err = collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to store embeddings: %w", err)
	}

	slog.Info("Stored embeddings",
		"companyID", companyID,
		"source", source,
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

	// Fetch documents filtered by both company and page ID
	filter := bson.M{
		"company_id": companyID,
		"page_id":    pageID,
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

	// Fetch all documents for the company (in production, use vector index)
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
