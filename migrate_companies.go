package main

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OldCompany represents the old company structure with pages array
type OldCompany struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	CompanyID       string             `bson:"company_id"`
	CompanyName     string             `bson:"company_name"`
	Pages           []FacebookPage     `bson:"pages"`
	AppSecret       string             `bson:"app_secret"`
	ClaudeAPIKey    string             `bson:"claude_api_key"`
	ClaudeModel     string             `bson:"claude_model"`
	VoyageAPIKey    string             `bson:"voyage_api_key,omitempty"`
	VoyageModel     string             `bson:"voyage_model,omitempty"`
	GPTAPIKey       string             `bson:"gpt_api_key,omitempty"`
	GPTModel        string             `bson:"gpt_model,omitempty"`
	SystemPrompt    string             `bson:"system_prompt,omitempty"`
	CRMLinks        []interface{}      `bson:"crm_links,omitempty"`
	IsActive        bool               `bson:"is_active"`
	MaxTokens       int                `bson:"max_tokens"`
	ResponseDelay   int                `bson:"response_delay"`
	DefaultLanguage string             `bson:"default_language,omitempty"`
	CreatedAt       time.Time          `bson:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at"`
}

// FacebookPage represents a Facebook page configuration
type FacebookPage struct {
	PageID          string `bson:"page_id"`
	PageName        string `bson:"page_name"`
	PageAccessToken string `bson:"page_access_token"`
	IsActive        bool   `bson:"is_active"`
}

// NewCompany represents the new company structure with single page
type NewCompany struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	CompanyID       string             `bson:"company_id"`
	CompanyName     string             `bson:"company_name"`
	PageID          string             `bson:"page_id"`
	PageName        string             `bson:"page_name"`
	PageAccessToken string             `bson:"page_access_token"`
	AppSecret       string             `bson:"app_secret"`
	ClaudeAPIKey    string             `bson:"claude_api_key"`
	ClaudeModel     string             `bson:"claude_model"`
	VoyageAPIKey    string             `bson:"voyage_api_key,omitempty"`
	VoyageModel     string             `bson:"voyage_model,omitempty"`
	GPTAPIKey       string             `bson:"gpt_api_key,omitempty"`
	GPTModel        string             `bson:"gpt_model,omitempty"`
	SystemPrompt    string             `bson:"system_prompt,omitempty"`
	CRMLinks        []interface{}      `bson:"crm_links,omitempty"`
	IsActive        bool               `bson:"is_active"`
	MaxTokens       int                `bson:"max_tokens"`
	ResponseDelay   int                `bson:"response_delay"`
	DefaultLanguage string             `bson:"default_language,omitempty"`
	CreatedAt       time.Time          `bson:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at"`
}
