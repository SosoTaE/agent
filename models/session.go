package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Session struct {
	ID           primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	SessionID    string                 `bson:"session_id" json:"session_id"`
	UserID       string                 `bson:"user_id" json:"user_id"`
	Username     string                 `bson:"username" json:"username"`
	Email        string                 `bson:"email" json:"email"`
	CompanyID    string                 `bson:"company_id" json:"company_id"`
	Role         string                 `bson:"role" json:"role"`
	IPAddress    string                 `bson:"ip_address,omitempty" json:"ip_address,omitempty"`
	UserAgent    string                 `bson:"user_agent,omitempty" json:"user_agent,omitempty"`
	Data         map[string]interface{} `bson:"data,omitempty" json:"data,omitempty"`
	CreatedAt    time.Time              `bson:"created_at" json:"created_at"`
	LastAccessed time.Time              `bson:"last_accessed" json:"last_accessed"`
	ExpiresAt    time.Time              `bson:"expires_at" json:"expires_at"`
	IsActive     bool                   `bson:"is_active" json:"is_active"`
}
