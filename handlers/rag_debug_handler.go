package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"

	"facebook-bot/services"
)

// TestRAGRetrieval tests RAG retrieval for a query
func TestRAGRetrieval(c *fiber.Ctx) error {
	companyID := c.Params("companyID")

	var req struct {
		Query  string `json:"query"`
		PageID string `json:"page_id,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Query is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get company to verify it exists
	company, err := services.GetCompanyByID(ctx, companyID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Test vector search - use page-specific search if page ID provided
	var results []services.SearchResult
	var ragContext string

	if req.PageID != "" {
		// Search with page ID filter
		results, err = services.SearchSimilarDocumentsByPage(ctx, req.Query, companyID, req.PageID, 5)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to search documents by page: " + err.Error(),
			})
		}

		// Get RAG context with page ID filter
		ragContext, err = services.GetRAGContext(ctx, req.Query, companyID, req.PageID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to get RAG context: " + err.Error(),
			})
		}
	} else {
		// Search without page filter (company-wide)
		results, err = services.SearchSimilarDocuments(ctx, req.Query, companyID, 5)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to search documents: " + err.Error(),
			})
		}

		// For backward compatibility, use the current company document's page
		if company.PageID != "" {
			// Use the company's page
			ragContext, err = services.GetRAGContext(ctx, req.Query, companyID, company.PageID)
		} else {
			ragContext = "No page configured for this company document"
		}
	}

	// Format results for response
	var searchResults []map[string]interface{}
	for _, result := range results {
		// Truncate content for readability
		content := result.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}

		searchResults = append(searchResults, map[string]interface{}{
			"score":    result.Score,
			"source":   result.Source,
			"content":  content,
			"metadata": result.Metadata,
		})
	}

	return c.JSON(fiber.Map{
		"query":              req.Query,
		"page_id":            req.PageID,
		"company":            company.CompanyName,
		"results_found":      len(results),
		"search_results":     searchResults,
		"rag_context_length": len(ragContext),
		"rag_context":        ragContext,
		"voyage_configured":  company.VoyageAPIKey != "",
		"voyage_model":       company.VoyageModel,
	})
}

// GetStoredDocuments retrieves all stored documents for a company
func GetStoredDocuments(c *fiber.Ctx) error {
	companyID := c.Params("companyID")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get all documents for the company
	documents, err := services.GetAllVectorDocuments(ctx, companyID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get documents: " + err.Error(),
		})
	}

	// Format for response
	var docs []map[string]interface{}
	for _, doc := range documents {
		// Truncate content for readability
		content := doc.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}

		docs = append(docs, map[string]interface{}{
			"id":       doc.ID.Hex(),
			"source":   doc.Source,
			"content":  content,
			"metadata": doc.Metadata,
			"created":  doc.CreatedAt,
		})
	}

	return c.JSON(fiber.Map{
		"company_id":      companyID,
		"total_documents": len(documents),
		"documents":       docs,
	})
}
