package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Voyage (Anthropic) Embedding API structures
type VoyageEmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type VoyageEmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// GetVoyageEmbeddings generates embeddings using Voyage AI API with rate limiting
func GetVoyageEmbeddings(ctx context.Context, texts []string, apiKey string, model string) ([][]float32, error) {
	if model == "" {
		model = "voyage-2" // Default Voyage model
	}

	// Apply rate limiting
	if err := voyageRateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	reqBody := VoyageEmbeddingRequest{
		Input: texts,
		Model: model,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.voyageai.com/v1/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}

	// Retry logic for rate limit errors
	maxRetries := 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to call Voyage API: %w", err)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		// Check for rate limit error
		if resp.StatusCode == 429 {
			slog.Warn("Voyage API rate limit hit, waiting before retry",
				"attempt", i+1,
				"maxRetries", maxRetries,
			)
			// Wait longer for rate limit
			select {
			case <-time.After(time.Duration(20*(i+1)) * time.Second):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Voyage API error (status %d): %s", resp.StatusCode, string(body))
		}

		var embResp VoyageEmbeddingResponse
		if err := json.Unmarshal(body, &embResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		// Extract embeddings in order
		embeddings := make([][]float32, len(texts))
		for _, data := range embResp.Data {
			if data.Index < len(embeddings) {
				embeddings[data.Index] = data.Embedding
			}
		}

		slog.Info("Generated Voyage embeddings",
			"count", len(embeddings),
			"model", model,
			"totalTokens", embResp.Usage.TotalTokens,
		)

		return embeddings, nil
	}

	// If we get here, all retries failed
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("failed to get embeddings after %d retries", maxRetries)
}

// OpenAI Embedding API structures
type OpenAIEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type OpenAIEmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// GetOpenAIEmbeddings generates embeddings using OpenAI API
func GetOpenAIEmbeddings(ctx context.Context, texts []string, apiKey string, model string) ([][]float32, error) {
	if model == "" {
		model = "text-embedding-ada-002" // Default OpenAI embedding model
	}

	reqBody := OpenAIEmbeddingRequest{
		Model: model,
		Input: texts,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	var embResp OpenAIEmbeddingResponse
	if err := json.Unmarshal(body, &embResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Extract embeddings in order
	embeddings := make([][]float32, len(texts))
	for _, data := range embResp.Data {
		if data.Index < len(embeddings) {
			embeddings[data.Index] = data.Embedding
		}
	}

	slog.Info("Generated OpenAI embeddings",
		"count", len(embeddings),
		"model", model,
		"promptTokens", embResp.Usage.PromptTokens,
	)

	return embeddings, nil
}

// Cohere Embedding API structures
type CohereEmbeddingRequest struct {
	Texts []string `json:"texts"`
	Model string   `json:"model"`
}

type CohereEmbeddingResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
	Meta       struct {
		APIVersion struct {
			Version string `json:"version"`
		} `json:"api_version"`
	} `json:"meta"`
}

// GetCohereEmbeddings generates embeddings using Cohere API
func GetCohereEmbeddings(ctx context.Context, texts []string, apiKey string, model string) ([][]float32, error) {
	if model == "" {
		model = "embed-english-v2.0" // Default Cohere embedding model
	}

	reqBody := CohereEmbeddingRequest{
		Texts: texts,
		Model: model,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.cohere.ai/v1/embed", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Cohere API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Cohere API error (status %d): %s", resp.StatusCode, string(body))
	}

	var embResp CohereEmbeddingResponse
	if err := json.Unmarshal(body, &embResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	slog.Info("Generated Cohere embeddings",
		"count", len(embResp.Embeddings),
		"model", model,
	)

	return embResp.Embeddings, nil
}

// GetMockEmbeddings generates mock embeddings for testing
func GetMockEmbeddings(texts []string) [][]float32 {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		// Generate a simple mock embedding based on text length and position
		embedding := make([]float32, 768) // Standard embedding size
		for j := range embedding {
			// Create some variation based on text content
			seed := float32(len(text)) * 0.001
			embedding[j] = (float32(i) + float32(j)/768.0 + seed) / float32(len(texts)+1)
		}
		embeddings[i] = embedding
	}
	return embeddings
}
