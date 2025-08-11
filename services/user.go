package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"

	"facebook-bot/models"
)

// CreateUser creates a new user in the database
func CreateUser(ctx context.Context, user *models.User) error {
	collection := database.Collection("users")

	// Check if user already exists
	existingUser := collection.FindOne(ctx, bson.M{
		"$or": []bson.M{
			{"user_id": user.UserID},
			{"email": user.Email},
			{"username": user.Username},
		},
	})

	if existingUser.Err() != mongo.ErrNoDocuments {
		return fmt.Errorf("user already exists with this user_id, email, or username")
	}

	// Validate role
	if !models.IsValidRole(string(user.Role)) {
		return fmt.Errorf("invalid role: %s", user.Role)
	}

	// Generate API key
	if user.APIKey == "" {
		apiKey, err := generateAPIKey()
		if err != nil {
			return fmt.Errorf("failed to generate API key: %w", err)
		}
		user.APIKey = apiKey
	}

	// Set timestamps
	user.ID = primitive.NewObjectID()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.IsActive = true

	_, err := collection.InsertOne(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	slog.Info("User created successfully",
		"userID", user.UserID,
		"username", user.Username,
		"companyID", user.CompanyID,
		"role", user.Role)

	return nil
}

// GetUserByID retrieves a user by their user ID
func GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	collection := database.Collection("users")

	var user models.User
	err := collection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by their email
func GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	collection := database.Collection("users")

	var user models.User
	err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	return &user, nil
}

// GetUserByAPIKey retrieves a user by their API key
func GetUserByAPIKey(ctx context.Context, apiKey string) (*models.User, error) {
	collection := database.Collection("users")

	var user models.User
	err := collection.FindOne(ctx, bson.M{
		"api_key":   apiKey,
		"is_active": true,
	}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("invalid API key")
		}
		return nil, err
	}

	return &user, nil
}

// GetUsersByCompany retrieves all users for a specific company
func GetUsersByCompany(ctx context.Context, companyID string) ([]models.User, error) {
	collection := database.Collection("users")

	cursor, err := collection.Find(ctx, bson.M{"company_id": companyID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, err
	}

	return users, nil
}

// GetUsersByRole retrieves all users with a specific role in a company
func GetUsersByRole(ctx context.Context, companyID string, role models.UserRole) ([]models.User, error) {
	collection := database.Collection("users")

	cursor, err := collection.Find(ctx, bson.M{
		"company_id": companyID,
		"role":       role,
		"is_active":  true,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, err
	}

	return users, nil
}

// UpdateUser updates a user's information
func UpdateUser(ctx context.Context, userID string, update bson.M) error {
	collection := database.Collection("users")

	// Add updated timestamp
	update["updated_at"] = time.Now()

	// Validate role if it's being updated
	if role, exists := update["role"]; exists {
		if !models.IsValidRole(role.(string)) {
			return fmt.Errorf("invalid role: %s", role)
		}
	}

	result, err := collection.UpdateOne(
		ctx,
		bson.M{"user_id": userID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user not found")
	}

	slog.Info("User updated successfully", "userID", userID)
	return nil
}

// DeleteUser soft deletes a user by setting is_active to false
func DeleteUser(ctx context.Context, userID string) error {
	return UpdateUser(ctx, userID, bson.M{"is_active": false})
}

// AssignPagesToUser assigns specific pages to a bot manager
func AssignPagesToUser(ctx context.Context, userID string, pageIDs []string) error {
	collection := database.Collection("users")

	// First, verify the user is a bot manager
	var user models.User
	err := collection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&user)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if user.Role != models.RoleBotManager {
		return fmt.Errorf("user must be a bot manager to assign pages")
	}

	// Update assigned pages
	return UpdateUser(ctx, userID, bson.M{"assigned_pages": pageIDs})
}

// UpdateLastLogin updates the user's last login time
func UpdateLastLogin(ctx context.Context, userID string) error {
	return UpdateUser(ctx, userID, bson.M{"last_login": time.Now()})
}

// RegenerateAPIKey generates a new API key for a user
func RegenerateAPIKey(ctx context.Context, userID string) (string, error) {
	apiKey, err := generateAPIKey()
	if err != nil {
		return "", err
	}

	err = UpdateUser(ctx, userID, bson.M{"api_key": apiKey})
	if err != nil {
		return "", err
	}

	return apiKey, nil
}

// HashPassword generates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash compares a password with a hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// generateAPIKey generates a random API key
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "sk_" + hex.EncodeToString(bytes), nil
}

// GetCompanyAdmins retrieves all company admins for a specific company
func GetCompanyAdmins(ctx context.Context, companyID string) ([]models.User, error) {
	return GetUsersByRole(ctx, companyID, models.RoleCompanyAdmin)
}

// GetBotManagers retrieves all bot managers for a specific company
func GetBotManagers(ctx context.Context, companyID string) ([]models.User, error) {
	return GetUsersByRole(ctx, companyID, models.RoleBotManager)
}

// CanUserManagePage checks if a user can manage a specific page
func CanUserManagePage(user *models.User, pageID string) bool {
	// Company admins can manage all pages
	if user.Role == models.RoleCompanyAdmin {
		return true
	}

	// Bot managers can only manage assigned pages
	if user.Role == models.RoleBotManager {
		for _, assignedPage := range user.AssignedPages {
			if assignedPage == pageID {
				return true
			}
		}
	}

	return false
}
