package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Company represents a company/client configuration with multiple pages
type Company struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CompanyID   string             `bson:"company_id" json:"company_id"`
	CompanyName string             `bson:"company_name" json:"company_name"`

	// Facebook Pages Configuration (array of pages)
	Pages []FacebookPage `bson:"pages" json:"pages"`

	// Settings (defaults applied to all pages if not specified per page)
	IsActive        bool   `bson:"is_active,omitempty" json:"is_active,omitempty"`
	ResponseDelay   int    `bson:"response_delay,omitempty" json:"response_delay,omitempty"`     // in seconds
	DefaultLanguage string `bson:"default_language,omitempty" json:"default_language,omitempty"` // e.g., "en", "ka", "ru"

	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// FacebookPage represents a Facebook page configuration
type FacebookPage struct {
	PageID          string `bson:"page_id" json:"page_id"`
	PageName        string `bson:"page_name" json:"page_name"`
	PageAccessToken string `bson:"page_access_token" json:"page_access_token"`
	AppSecret       string `bson:"app_secret" json:"app_secret"`
	ClaudeAPIKey    string `bson:"claude_api_key" json:"claude_api_key"`
	ClaudeModel     string `bson:"claude_model" json:"claude_model"`
	VoyageAPIKey    string `bson:"voyage_api_key,omitempty" json:"voyage_api_key,omitempty"`
	VoyageModel     string `bson:"voyage_model,omitempty" json:"voyage_model,omitempty"`
	SystemPrompt    string `bson:"system_prompt,omitempty" json:"system_prompt,omitempty"`
	IsActive        bool   `bson:"is_active" json:"is_active"`
	MaxTokens       int    `bson:"max_tokens" json:"max_tokens"`

	// Separate CRM and RAG Configuration for Facebook Comments and Messenger
	FacebookConfig  *ChannelConfig `bson:"facebook_config,omitempty" json:"facebook_config,omitempty"`
	MessengerConfig *ChannelConfig `bson:"messenger_config,omitempty" json:"messenger_config,omitempty"`

	// Legacy support - will be migrated to channel configs
	CRMLinks []CRMLink `bson:"crm_links,omitempty" json:"crm_links,omitempty"`
}

// ChannelConfig represents configuration for a specific channel (Facebook comments or Messenger)
type ChannelConfig struct {
	// Enable/disable the channel
	IsEnabled bool `bson:"is_enabled" json:"is_enabled"`

	// CRM Links for this channel
	CRMLinks []CRMLink `bson:"crm_links,omitempty" json:"crm_links,omitempty"`

	// RAG Documents settings for this channel
	RAGEnabled bool `bson:"rag_enabled" json:"rag_enabled"`

	// Optional channel-specific system prompt
	SystemPrompt string `bson:"system_prompt,omitempty" json:"system_prompt,omitempty"`
}

// CRMLink represents a CRM link for RAG (Retrieval-Augmented Generation)
type CRMLink struct {
	Name        string            `bson:"name" json:"name"`
	URL         string            `bson:"url" json:"url"`
	Type        string            `bson:"type" json:"type"`         // api, webhook, database, file
	Channels    []string          `bson:"channels" json:"channels"` // ["facebook", "messenger"] or subset
	APIKey      string            `bson:"api_key,omitempty" json:"api_key,omitempty"`
	Headers     map[string]string `bson:"headers,omitempty" json:"headers,omitempty"`
	Description string            `bson:"description,omitempty" json:"description,omitempty"`
	PageID      string            `bson:"page_id,omitempty" json:"page_id,omitempty"`
	IsActive    bool              `bson:"is_active" json:"is_active"`
}
