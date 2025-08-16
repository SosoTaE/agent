package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Message represents a chat message
type Message struct {
	ID            primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	Type          string                 `bson:"type" json:"type"`       // "chat", "crm_data", "crm_response"
	ChatID        string                 `bson:"chat_id" json:"chat_id"` // Always the customer ID for grouping conversations
	SenderID      string                 `bson:"sender_id" json:"sender_id"`
	SenderName    string                 `bson:"sender_name,omitempty" json:"sender_name,omitempty"`   // Full name
	FirstName     string                 `bson:"first_name,omitempty" json:"first_name,omitempty"`     // Facebook first name
	LastName      string                 `bson:"last_name,omitempty" json:"last_name,omitempty"`       // Facebook last name
	RecipientID   string                 `bson:"recipient_id,omitempty" json:"recipient_id,omitempty"` // For bot messages, who it's sent to
	PageID        string                 `bson:"page_id" json:"page_id"`
	PageName      string                 `bson:"page_name" json:"page_name"`
	Message       string                 `bson:"message" json:"message"`
	ProcessedData map[string]interface{} `bson:"processed_data,omitempty" json:"processed_data,omitempty"` // For CRM data processing results
	IsBot         bool                   `bson:"is_bot" json:"is_bot"`                                     // true if message is from bot
	IsHuman       bool                   `bson:"is_human,omitempty" json:"is_human,omitempty"`             // true if message is from human agent via dashboard
	Source        string                 `bson:"source,omitempty" json:"source,omitempty"`                 // Source of message: "dashboard", "facebook", "bot", etc.
	AgentID       string                 `bson:"agent_id,omitempty" json:"agent_id,omitempty"`             // ID of human agent who sent the message
	AgentEmail    string                 `bson:"agent_email,omitempty" json:"agent_email,omitempty"`       // Email of human agent
	AgentName     string                 `bson:"agent_name,omitempty" json:"agent_name,omitempty"`         // Name of human agent
	Timestamp     time.Time              `bson:"timestamp" json:"timestamp"`
	UpdatedAt     time.Time              `bson:"updated_at,omitempty" json:"updated_at,omitempty"` // Last update time
}

// Comment represents a Facebook comment or reply
type Comment struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CommentID   string             `bson:"comment_id" json:"comment_id"`                   // Unique Facebook comment ID
	PostID      string             `bson:"post_id" json:"post_id"`                         // The post this comment belongs to
	ParentID    string             `bson:"parent_id,omitempty" json:"parent_id,omitempty"` // For replies: the parent comment ID
	SenderID    string             `bson:"sender_id" json:"sender_id"`
	SenderName  string             `bson:"sender_name" json:"sender_name"`
	FirstName   string             `bson:"first_name,omitempty" json:"first_name,omitempty"` // Facebook first name
	LastName    string             `bson:"last_name,omitempty" json:"last_name,omitempty"`   // Facebook last name
	PageID      string             `bson:"page_id" json:"page_id"`
	PageName    string             `bson:"page_name" json:"page_name"`
	Message     string             `bson:"message" json:"message"`
	PostContent string             `bson:"post_content,omitempty" json:"post_content,omitempty"`
	IsReply     bool               `bson:"is_reply" json:"is_reply"`                   // True if this is a reply
	IsBot       bool               `bson:"is_bot" json:"is_bot"`                       // True if this comment is from the bot
	Replies     []Comment          `bson:"replies,omitempty" json:"replies,omitempty"` // Nested replies to this comment
	Timestamp   time.Time          `bson:"timestamp" json:"timestamp"`
	UpdatedAt   time.Time          `bson:"updated_at,omitempty" json:"updated_at,omitempty"` // Last update time
}

// Response represents an AI response
type Response struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Type       string             `bson:"type" json:"type"` // "chat" or "comment"
	CommentID  string             `bson:"comment_id,omitempty" json:"comment_id,omitempty"`
	PostID     string             `bson:"post_id,omitempty" json:"post_id,omitempty"`
	SenderID   string             `bson:"sender_id" json:"sender_id"`
	SenderName string             `bson:"sender_name,omitempty" json:"sender_name,omitempty"`
	PageID     string             `bson:"page_id" json:"page_id"`
	PageName   string             `bson:"page_name" json:"page_name"`
	Original   string             `bson:"original" json:"original"`
	Response   string             `bson:"response" json:"response"`
	Timestamp  time.Time          `bson:"timestamp" json:"timestamp"`
}

// PageCache represents cached page information
type PageCache struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PageID    string             `bson:"page_id" json:"page_id"`
	PageName  string             `bson:"page_name" json:"page_name"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// ProcessedComment tracks comments that have already been processed
type ProcessedComment struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CommentID   string             `bson:"comment_id" json:"comment_id"`
	ProcessedAt time.Time          `bson:"processed_at" json:"processed_at"`
	TTL         time.Time          `bson:"ttl" json:"ttl"` // For automatic cleanup after 24 hours
}
