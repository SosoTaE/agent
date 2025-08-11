package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

// SearchDocumentsByText searches for documents using text matching without generating embeddings
func SearchDocumentsByText(ctx context.Context, query string, companyID string, pageID string, limit int) ([]SearchResult, error) {
	collection := database.Collection("vector_documents")

	// Fetch all documents for the page
	filter := bson.M{
		"company_id": companyID,
		"page_id":    pageID,
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch documents: %w", err)
	}
	defer cursor.Close(ctx)

	var documents []VectorDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, fmt.Errorf("failed to read documents: %w", err)
	}

	if len(documents) == 0 {
		slog.Info("No documents found for page",
			"pageID", pageID,
			"companyID", companyID,
		)
		return []SearchResult{}, nil
	}

	// Convert query to lowercase for case-insensitive matching
	queryLower := strings.ToLower(query)
	queryWords := strings.Fields(queryLower)

	// Score each document based on keyword matching
	type scoredDoc struct {
		doc   VectorDocument
		score float32
	}

	var scoredDocs []scoredDoc

	for _, doc := range documents {
		contentLower := strings.ToLower(doc.Content)
		score := float32(0)

		// Calculate score based on word matches
		for _, word := range queryWords {
			// Skip very short words
			if len(word) < 3 {
				continue
			}

			// Count occurrences of the word
			count := strings.Count(contentLower, word)
			if count > 0 {
				// Give higher score for more occurrences, but with diminishing returns
				wordScore := float32(1.0)
				if count > 1 {
					wordScore = 1.0 + (float32(count-1) * 0.2)
				}
				score += wordScore
			}

			// Bonus for exact phrase match
			if strings.Contains(contentLower, queryLower) {
				score += 3.0
			}
		}

		// Normalize score by number of query words
		if len(queryWords) > 0 {
			score = score / float32(len(queryWords))
		}

		// Store if score is above zero
		if score > 0 {
			scoredDocs = append(scoredDocs, scoredDoc{
				doc:   doc,
				score: score,
			})
		}
	}

	// If no matches found, return all documents with low scores
	if len(scoredDocs) == 0 {
		slog.Info("No keyword matches found, returning all documents",
			"query", query,
			"pageID", pageID,
		)

		for _, doc := range documents {
			scoredDocs = append(scoredDocs, scoredDoc{
				doc:   doc,
				score: 0.1, // Low default score
			})
		}
	}

	// Sort by score in descending order
	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].score > scoredDocs[j].score
	})

	// Take top N results
	var results []SearchResult
	for i := 0; i < limit && i < len(scoredDocs); i++ {
		results = append(results, SearchResult{
			Content:  scoredDocs[i].doc.Content,
			Score:    scoredDocs[i].score,
			Source:   scoredDocs[i].doc.Source,
			Metadata: scoredDocs[i].doc.Metadata,
		})
	}

	slog.Info("Text search completed",
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

// GetRAGContextWithoutEmbeddings retrieves relevant context using text search without generating embeddings
func GetRAGContextWithoutEmbeddings(ctx context.Context, query string, companyID string, pageID string) (string, error) {
	// Search for relevant documents using text matching
	results, err := SearchDocumentsByText(ctx, query, companyID, pageID, 5)
	if err != nil {
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

	// Build context from results
	var contexts []string
	totalLength := 0
	maxContextLength := 10000 // Max total context to prevent token overflow

	for i, result := range results {
		// Include score information
		confidence := "relevant"
		if result.Score > 2.0 {
			confidence = "highly relevant"
		} else if result.Score < 0.5 {
			confidence = "possibly relevant"
		}

		// For top results, include more content
		extractSize := 3000
		if i > 0 {
			extractSize = 2000
		}
		if i > 2 {
			extractSize = 1000
		}

		// Extract relevant portion
		relevantContent := extractRelevantPortion(result.Content, query, extractSize)

		// Format context entry
		contextEntry := fmt.Sprintf("[Result %d - Source: %s, Relevance: %s (score: %.2f)]\n%s",
			i+1, result.Source, confidence, result.Score, relevantContent)

		// Check if adding this would exceed max length
		if totalLength+len(contextEntry) > maxContextLength {
			break
		}

		contexts = append(contexts, contextEntry)
		totalLength += len(contextEntry)
	}

	ragContext := strings.Join(contexts, "\n\n---\n\n")

	slog.Info("RAG context retrieved without embeddings",
		"query", query,
		"queryLength", len(query),
		"contextLength", len(ragContext),
		"sources", len(results),
		"topScore", results[0].Score,
		"companyID", companyID,
		"pageID", pageID,
	)

	return ragContext, nil
}

// SearchUsingExistingEmbeddings searches using the average of existing document embeddings
func SearchUsingExistingEmbeddings(ctx context.Context, query string, companyID string, pageID string, limit int) ([]SearchResult, error) {
	collection := database.Collection("vector_documents")

	// First, try text-based search to find the most relevant document
	textResults, err := SearchDocumentsByText(ctx, query, companyID, pageID, 1)
	if err != nil || len(textResults) == 0 {
		// Fallback to returning all documents with low scores
		return SearchDocumentsByText(ctx, query, companyID, pageID, limit)
	}

	// Get the most relevant document to use its embedding as reference
	filter := bson.M{
		"company_id": companyID,
		"page_id":    pageID,
		"content":    textResults[0].Content,
	}

	var referenceDoc VectorDocument
	err = collection.FindOne(ctx, filter).Decode(&referenceDoc)
	if err != nil {
		// Fallback to text search
		return textResults, nil
	}

	// Now fetch all documents and calculate similarity using the reference embedding
	allFilter := bson.M{
		"company_id": companyID,
		"page_id":    pageID,
	}

	cursor, err := collection.Find(ctx, allFilter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var documents []VectorDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, err
	}

	// Calculate similarity scores using the reference embedding
	type scoredDoc struct {
		doc   VectorDocument
		score float32
	}

	var scoredDocs []scoredDoc
	for _, doc := range documents {
		// Calculate cosine similarity with reference embedding
		score := CosineSimilarity(referenceDoc.Embedding, doc.Embedding)

		// Boost score if there's also text match
		contentLower := strings.ToLower(doc.Content)
		queryLower := strings.ToLower(query)
		if strings.Contains(contentLower, queryLower) {
			score *= 1.5
		}

		scoredDocs = append(scoredDocs, scoredDoc{doc: doc, score: score})
	}

	// Sort by score
	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].score > scoredDocs[j].score
	})

	// Take top results
	var results []SearchResult
	for i := 0; i < limit && i < len(scoredDocs); i++ {
		results = append(results, SearchResult{
			Content:  scoredDocs[i].doc.Content,
			Score:    scoredDocs[i].score,
			Source:   scoredDocs[i].doc.Source,
			Metadata: scoredDocs[i].doc.Metadata,
		})
	}

	slog.Info("Embedding search completed using existing embeddings",
		"query", query,
		"companyID", companyID,
		"pageID", pageID,
		"resultsFound", len(results),
	)

	return results, nil
}
