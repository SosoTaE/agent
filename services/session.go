package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"facebook-bot/models"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	SessionDuration   = 24 * time.Hour
	SessionCookieName = "session"
)

// GenerateSessionID generates a secure random session ID
func GenerateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateSession creates a new session in the database
func CreateSession(ctx context.Context, userID, username, email, companyID, role, ipAddress, userAgent string) (*models.Session, error) {
	sessionID, err := GenerateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	now := time.Now()
	session := &models.Session{
		ID:           primitive.NewObjectID(),
		SessionID:    sessionID,
		UserID:       userID,
		Username:     username,
		Email:        email,
		CompanyID:    companyID,
		Role:         role,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		CreatedAt:    now,
		LastAccessed: now,
		ExpiresAt:    now.Add(SessionDuration),
		IsActive:     true,
	}

	collection := GetDatabase().Collection("sessions")
	_, err = collection.InsertOne(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// GetSessionByID retrieves a session from the database by session ID
func GetSessionByID(ctx context.Context, sessionID string) (*models.Session, error) {
	collection := GetDatabase().Collection("sessions")

	var session models.Session
	err := collection.FindOne(ctx, bson.M{
		"session_id": sessionID,
		"is_active":  true,
		"expires_at": bson.M{"$gt": time.Now()},
	}).Decode(&session)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Update last accessed time
	_, err = collection.UpdateOne(
		ctx,
		bson.M{"_id": session.ID},
		bson.M{"$set": bson.M{"last_accessed": time.Now()}},
	)
	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to update session last accessed time: %v\n", err)
	}

	return &session, nil
}

// UpdateSession updates session data
func UpdateSession(ctx context.Context, sessionID string, data map[string]interface{}) error {
	collection := GetDatabase().Collection("sessions")

	updateData := bson.M{
		"last_accessed": time.Now(),
	}

	if data != nil {
		updateData["data"] = data
	}

	result, err := collection.UpdateOne(
		ctx,
		bson.M{
			"session_id": sessionID,
			"is_active":  true,
		},
		bson.M{"$set": updateData},
	)

	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	if result.ModifiedCount == 0 {
		return fmt.Errorf("session not found or inactive")
	}

	return nil
}

// ExtendSession extends the expiration time of a session
func ExtendSession(ctx context.Context, sessionID string) error {
	collection := GetDatabase().Collection("sessions")

	_, err := collection.UpdateOne(
		ctx,
		bson.M{
			"session_id": sessionID,
			"is_active":  true,
		},
		bson.M{
			"$set": bson.M{
				"last_accessed": time.Now(),
				"expires_at":    time.Now().Add(SessionDuration),
			},
		},
	)

	if err != nil {
		return fmt.Errorf("failed to extend session: %w", err)
	}

	return nil
}

// DestroySession marks a session as inactive
func DestroySession(ctx context.Context, sessionID string) error {
	collection := GetDatabase().Collection("sessions")

	_, err := collection.UpdateOne(
		ctx,
		bson.M{"session_id": sessionID},
		bson.M{
			"$set": bson.M{
				"is_active":  false,
				"expires_at": time.Now(),
			},
		},
	)

	if err != nil {
		return fmt.Errorf("failed to destroy session: %w", err)
	}

	return nil
}

// DestroyUserSessions destroys all sessions for a specific user
func DestroyUserSessions(ctx context.Context, userID string) error {
	collection := GetDatabase().Collection("sessions")

	_, err := collection.UpdateMany(
		ctx,
		bson.M{
			"user_id":   userID,
			"is_active": true,
		},
		bson.M{
			"$set": bson.M{
				"is_active":  false,
				"expires_at": time.Now(),
			},
		},
	)

	if err != nil {
		return fmt.Errorf("failed to destroy user sessions: %w", err)
	}

	return nil
}

// CleanupExpiredSessions removes expired sessions from the database
func CleanupExpiredSessions(ctx context.Context) (int64, error) {
	collection := GetDatabase().Collection("sessions")

	// Delete sessions that have expired more than 7 days ago
	cutoffTime := time.Now().Add(-7 * 24 * time.Hour)

	result, err := collection.DeleteMany(
		ctx,
		bson.M{
			"expires_at": bson.M{"$lt": cutoffTime},
		},
	)

	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	return result.DeletedCount, nil
}

// GetActiveSessionsForUser retrieves all active sessions for a user
func GetActiveSessionsForUser(ctx context.Context, userID string) ([]*models.Session, error) {
	collection := GetDatabase().Collection("sessions")

	cursor, err := collection.Find(ctx, bson.M{
		"user_id":    userID,
		"is_active":  true,
		"expires_at": bson.M{"$gt": time.Now()},
	}, options.Find().SetSort(bson.M{"last_accessed": -1}))

	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}
	defer cursor.Close(ctx)

	var sessions []*models.Session
	if err := cursor.All(ctx, &sessions); err != nil {
		return nil, fmt.Errorf("failed to decode sessions: %w", err)
	}

	return sessions, nil
}

// CountActiveSessions counts the number of active sessions
func CountActiveSessions(ctx context.Context) (int64, error) {
	collection := GetDatabase().Collection("sessions")

	count, err := collection.CountDocuments(ctx, bson.M{
		"is_active":  true,
		"expires_at": bson.M{"$gt": time.Now()},
	})

	if err != nil {
		return 0, fmt.Errorf("failed to count active sessions: %w", err)
	}

	return count, nil
}

// CreateSessionIndexes creates necessary indexes for the sessions collection
func CreateSessionIndexes(ctx context.Context) error {
	collection := GetDatabase().Collection("sessions")

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.M{"session_id": 1},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.M{"user_id": 1},
		},
		{
			Keys: bson.M{"company_id": 1},
		},
		{
			Keys: bson.M{"expires_at": 1},
		},
		{
			Keys: bson.M{"is_active": 1, "expires_at": 1},
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create session indexes: %w", err)
	}

	return nil
}
