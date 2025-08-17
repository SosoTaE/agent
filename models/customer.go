package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Customer represents a Facebook user who has sent messages to a page
type Customer struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CustomerID   string             `bson:"customer_id" json:"customer_id"`     // Facebook user ID
	CustomerName string             `bson:"customer_name" json:"customer_name"` // Full name
	FirstName    string             `bson:"first_name,omitempty" json:"first_name,omitempty"`
	LastName     string             `bson:"last_name,omitempty" json:"last_name,omitempty"`
	PageID       string             `bson:"page_id" json:"page_id"`                               // The page they're messaging
	PageName     string             `bson:"page_name" json:"page_name"`                           // Page name for reference
	CompanyID    string             `bson:"company_id" json:"company_id"`                         // Company that owns the page
	MessageCount int                `bson:"message_count" json:"message_count"`                   // Total messages sent
	LastMessage  string             `bson:"last_message,omitempty" json:"last_message,omitempty"` // Last message text
	LastSeen     time.Time          `bson:"last_seen" json:"last_seen"`                           // Last interaction time
	FirstSeen    time.Time          `bson:"first_seen" json:"first_seen"`                         // First interaction time
	Stop         bool               `bson:"stop" json:"stop"`                                     // Whether customer wants to talk to real person
	StoppedAt    *time.Time         `bson:"stopped_at,omitempty" json:"stopped_at,omitempty"`     // When customer requested real person
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

// CustomerPage represents the relationship between a customer and multiple pages
// This is used when a customer interacts with multiple pages of the same company
type CustomerPage struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CustomerID    string             `bson:"customer_id" json:"customer_id"`
	CompanyID     string             `bson:"company_id" json:"company_id"`
	Pages         []PageInteraction  `bson:"pages" json:"pages"`
	TotalMessages int                `bson:"total_messages" json:"total_messages"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
}

// PageInteraction represents a customer's interaction with a specific page
type PageInteraction struct {
	PageID       string    `bson:"page_id" json:"page_id"`
	PageName     string    `bson:"page_name" json:"page_name"`
	MessageCount int       `bson:"message_count" json:"message_count"`
	LastMessage  string    `bson:"last_message" json:"last_message"`
	LastSeen     time.Time `bson:"last_seen" json:"last_seen"`
	FirstSeen    time.Time `bson:"first_seen" json:"first_seen"`
}
