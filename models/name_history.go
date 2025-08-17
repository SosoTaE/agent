package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NameHistory tracks changes in sender names over time
type NameHistory struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SenderID  string             `bson:"sender_id" json:"sender_id"`
	PageID    string             `bson:"page_id" json:"page_id"`
	Names     []NameRecord       `bson:"names" json:"names"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// NameRecord represents a single name used by a sender
type NameRecord struct {
	Name      string    `bson:"name" json:"name"`
	FirstSeen time.Time `bson:"first_seen" json:"first_seen"`
	LastSeen  time.Time `bson:"last_seen" json:"last_seen"`
	Count     int       `bson:"count" json:"count"` // Number of times this name was seen
}

// NameChange represents a detected name change
type NameChange struct {
	SenderID  string    `json:"sender_id"`
	PageID    string    `json:"page_id"`
	OldName   string    `json:"old_name"`
	NewName   string    `json:"new_name"`
	ChangedAt time.Time `json:"changed_at"`
	CommentID string    `json:"comment_id"` // Comment where change was detected
	PostID    string    `json:"post_id"`    // Post where change was detected
}
