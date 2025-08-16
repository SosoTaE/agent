package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Company represents a company/client configuration with a single page
type Company struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CompanyID   string             `bson:"company_id" json:"company_id"`
	CompanyName string             `bson:"company_name" json:"company_name"`

	// Facebook Page Configuration (single page per document)
	PageID          string `bson:"page_id" json:"page_id"`
	PageName        string `bson:"page_name" json:"page_name"`
	PageAccessToken string `bson:"page_access_token" json:"page_access_token"`

	// Facebook App Configuration
	AppSecret string `bson:"app_secret" json:"app_secret"`

	// Claude AI Configuration
	ClaudeAPIKey string `bson:"claude_api_key" json:"claude_api_key"`
	ClaudeModel  string `bson:"claude_model" json:"claude_model"`

	// Voyage Embedding Configuration
	VoyageAPIKey string `bson:"voyage_api_key,omitempty" json:"voyage_api_key,omitempty"`
	VoyageModel  string `bson:"voyage_model,omitempty" json:"voyage_model,omitempty"` // e.g., "voyage-2", "voyage-large-2"

	// Custom AI Instructions
	SystemPrompt string `bson:"system_prompt,omitempty" json:"system_prompt,omitempty"`

	// CRM and RAG Configuration
	CRMLinks []CRMLink `bson:"crm_links,omitempty" json:"crm_links,omitempty"`

	// Settings
	IsActive        bool   `bson:"is_active" json:"is_active"`
	MaxTokens       int    `bson:"max_tokens" json:"max_tokens"`
	ResponseDelay   int    `bson:"response_delay" json:"response_delay"`                         // in seconds
	DefaultLanguage string `bson:"default_language,omitempty" json:"default_language,omitempty"` // e.g., "en", "ka", "ru"

	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// FacebookPage represents a Facebook page configuration
type FacebookPage struct {
	PageID          string `bson:"page_id" json:"page_id"`
	PageName        string `bson:"page_name" json:"page_name"`
	PageAccessToken string `bson:"page_access_token" json:"page_access_token"`
	IsActive        bool   `bson:"is_active" json:"is_active"`
}

// CRMLink represents a CRM link for RAG (Retrieval-Augmented Generation)
type CRMLink struct {
	Name        string            `bson:"name" json:"name"`
	URL         string            `bson:"url" json:"url"`
	Type        string            `bson:"type" json:"type"` // api, webhook, database, file
	APIKey      string            `bson:"api_key,omitempty" json:"api_key,omitempty"`
	Headers     map[string]string `bson:"headers,omitempty" json:"headers,omitempty"`
	Description string            `bson:"description,omitempty" json:"description,omitempty"`
	IsActive    bool              `bson:"is_active" json:"is_active"`
}
