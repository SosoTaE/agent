package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

// SearchWithStoredEmbeddings searches using cosine similarity against stored embeddings
func SearchWithStoredEmbeddings(ctx context.Context, query string, companyID string, pageID string, limit int) ([]SearchResult, error) {
	collection := database.Collection("vector_documents")

	// Fetch all active documents for the page with their embeddings
	filter := bson.M{
		"company_id": companyID,
		"page_id":    pageID,
		"is_active":  true,
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

	// Find the document that best matches the query text
	queryLower := strings.ToLower(query)
	queryWords := strings.Fields(queryLower)

	// First, find documents that contain query keywords to use as reference
	var referenceEmbedding []float32
	bestTextScore := float32(0)

	for _, doc := range documents {
		contentLower := strings.ToLower(doc.Content)
		score := float32(0)

		// Calculate text matching score
		for _, word := range queryWords {
			if len(word) < 2 {
				continue
			}
			if strings.Contains(contentLower, word) {
				score += 1.0
			}
		}

		// Bonus for exact phrase match
		if strings.Contains(contentLower, queryLower) {
			score += float32(len(queryWords))
		}

		// If this is the best match so far, use its embedding as reference
		if score > bestTextScore && len(doc.Embedding) > 0 {
			bestTextScore = score
			referenceEmbedding = doc.Embedding
		}
	}

	// If no text match found, use the first document's embedding as reference
	if len(referenceEmbedding) == 0 && len(documents) > 0 && len(documents[0].Embedding) > 0 {
		referenceEmbedding = documents[0].Embedding
		slog.Info("No text match found, using first document as reference")
	}

	// If still no embedding, fall back to text-only search
	if len(referenceEmbedding) == 0 {
		slog.Warn("No embeddings found, falling back to text search")
		return searchByTextOnly(documents, query, limit), nil
	}

	// Calculate cosine similarity for all documents using the reference embedding
	type scoredDoc struct {
		doc           VectorDocument
		cosineSim     float32
		textScore     float32
		combinedScore float32
	}

	var scoredDocs []scoredDoc

	for _, doc := range documents {
		if len(doc.Embedding) == 0 {
			continue
		}

		// Calculate cosine similarity
		cosineSim := CosineSimilarity(referenceEmbedding, doc.Embedding)

		// Calculate text relevance score
		contentLower := strings.ToLower(doc.Content)
		textScore := float32(0)

		for _, word := range queryWords {
			if len(word) < 2 {
				continue
			}
			count := strings.Count(contentLower, word)
			if count > 0 {
				textScore += 1.0 + float32(count-1)*0.2 // Diminishing returns for multiple occurrences
			}
		}

		// Bonus for exact phrase
		if strings.Contains(contentLower, queryLower) {
			textScore += 2.0
		}

		// Normalize text score
		if len(queryWords) > 0 {
			textScore = textScore / float32(len(queryWords))
		}

		// Combine cosine similarity and text score
		// Weight: 70% cosine similarity, 30% text match
		combinedScore := (cosineSim * 0.7) + (textScore * 0.3)

		scoredDocs = append(scoredDocs, scoredDoc{
			doc:           doc,
			cosineSim:     cosineSim,
			textScore:     textScore,
			combinedScore: combinedScore,
		})
	}

	// Sort by combined score in descending order
	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].combinedScore > scoredDocs[j].combinedScore
	})

	// Take top N results
	var results []SearchResult
	for i := 0; i < limit && i < len(scoredDocs); i++ {
		results = append(results, SearchResult{
			Content:  scoredDocs[i].doc.Content,
			Score:    scoredDocs[i].combinedScore,
			Source:   scoredDocs[i].doc.Source,
			Metadata: scoredDocs[i].doc.Metadata,
		})

		slog.Debug("Search result",
			"rank", i+1,
			"cosineSim", scoredDocs[i].cosineSim,
			"textScore", scoredDocs[i].textScore,
			"combinedScore", scoredDocs[i].combinedScore,
		)
	}

	slog.Info("Cosine similarity search completed with stored embeddings",
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

// searchByTextOnly performs text-only search when embeddings are not available
func searchByTextOnly(documents []VectorDocument, query string, limit int) []SearchResult {
	queryLower := strings.ToLower(query)
	queryWords := strings.Fields(queryLower)

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
			if len(word) < 2 {
				continue
			}

			count := strings.Count(contentLower, word)
			if count > 0 {
				score += 1.0 + float32(count-1)*0.2
			}
		}

		// Bonus for exact phrase
		if strings.Contains(contentLower, queryLower) {
			score += 3.0
		}

		// Normalize by query length
		if len(queryWords) > 0 {
			score = score / float32(len(queryWords))
		}

		if score > 0 {
			scoredDocs = append(scoredDocs, scoredDoc{doc: doc, score: score})
		}
	}

	// If no matches, include all with low score
	if len(scoredDocs) == 0 {
		for _, doc := range documents {
			scoredDocs = append(scoredDocs, scoredDoc{doc: doc, score: 0.1})
		}
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

	return results
}
