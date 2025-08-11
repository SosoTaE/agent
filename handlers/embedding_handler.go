package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"

	"facebook-bot/services"
)

// UpdateVoyageConfig updates the Voyage embedding configuration for a company
func UpdateVoyageConfig(c *fiber.Ctx) error {
	companyID := c.Params("companyID")

	var config struct {
		APIKey string `json:"api_key"`
		Model  string `json:"model"`
	}

	if err := c.BodyParser(&config); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate API key
	if config.APIKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Voyage API key is required",
		})
	}

	// Set default model if not specified
	if config.Model == "" {
		config.Model = "voyage-2"
	}

	// Validate model
	validModels := map[string]bool{
		"voyage-2":       true,
		"voyage-large-2": true,
		"voyage-code-2":  true,
	}

	if !validModels[config.Model] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid model. Must be one of: voyage-2, voyage-large-2, voyage-code-2",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Update company configuration
	update := bson.M{
		"voyage_api_key": config.APIKey,
		"voyage_model":   config.Model,
	}

	if err := services.UpdateCompany(ctx, companyID, update); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update Voyage configuration",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Voyage configuration updated successfully",
		"model":   config.Model,
	})
}

// Keeping the old function name as an alias for backward compatibility
func UpdateEmbeddingConfig(c *fiber.Ctx) error {
	return UpdateVoyageConfig(c)
}

// TestVoyageEmbedding tests the Voyage embedding configuration
func TestVoyageEmbedding(c *fiber.Ctx) error {
	companyID := c.Params("companyID")

	var req struct {
		Text string `json:"text"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Text == "" {
		req.Text = "This is a test text for Voyage embedding generation"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get company to check Voyage configuration
	company, err := services.GetCompanyByID(ctx, companyID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Generate embedding using Voyage
	embedding, err := services.GetEmbeddings(ctx, req.Text, companyID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate Voyage embedding: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message":        "Voyage embedding generated successfully",
		"provider":       "Voyage AI",
		"model":          company.VoyageModel,
		"text":           req.Text,
		"embedding_size": len(embedding),
		"sample":         embedding[:10], // Show first 10 values as sample
	})
}

// Keeping the old function name as an alias
func TestEmbedding(c *fiber.Ctx) error {
	return TestVoyageEmbedding(c)
}

// ReindexDocuments reindexes all documents with new embeddings
func ReindexDocuments(c *fiber.Ctx) error {
	companyID := c.Params("companyID")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Get company configuration
	_, err := services.GetCompanyByID(ctx, companyID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Note: Reindexing logic has been removed as CRM processing is no longer supported

	return c.JSON(fiber.Map{
		"message":    "Reindexing not available - CRM processing has been removed",
		"company_id": companyID,
	})
}
