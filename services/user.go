package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"

	"facebook-bot/models"
)

// CreateUserWithHashedPassword creates a new user in the database with a pre-hashed password
// This function does not hash the password and assumes it's already hashed
func CreateUserWithHashedPassword(ctx context.Context, user *models.User) error {
	collection := database.Collection("users")

	// Check if user already exists with the same email in the same company
	existingUser := collection.FindOne(ctx, bson.M{
		"email":      user.Email,
		"company_id": user.CompanyID,
	})

	if existingUser.Err() != mongo.ErrNoDocuments {
		return fmt.Errorf("user already exists with this email in your company")
	}

	// Validate role
	if !models.IsValidRole(string(user.Role)) {
		return fmt.Errorf("invalid role: %s", user.Role)
	}

	// Don't hash the password - it's already hashed
	// Set ID if not provided
	if user.ID.IsZero() {
		user.ID = primitive.NewObjectID()
	}

	// Set timestamps if not provided
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = time.Now()
	}

	_, err := collection.InsertOne(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	slog.Info("User created successfully with pre-hashed password",
		"userID", user.ID.Hex(),
		"username", user.Username,
		"companyID", user.CompanyID,
		"role", user.Role)

	return nil
}

// CreateUser creates a new user in the database
func CreateUser(ctx context.Context, user *models.User) error {
	collection := database.Collection("users")

	// Check if user already exists with the same email in the same company
	existingUser := collection.FindOne(ctx, bson.M{
		"email":      user.Email,
		"company_id": user.CompanyID,
	})

	if existingUser.Err() != mongo.ErrNoDocuments {
		return fmt.Errorf("user already exists with this email in your company")
	}

	// Validate role
	if !models.IsValidRole(string(user.Role)) {
		return fmt.Errorf("invalid role: %s", user.Role)
	}

	// Hash password if it's not already hashed
	if user.Password != "" && !strings.HasPrefix(user.Password, "$2a$") {
		hashedPassword, err := HashPassword(user.Password)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		user.Password = hashedPassword
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
		"userID", user.ID.Hex(),
		"username", user.Username,
		"companyID", user.CompanyID,
		"role", user.Role)

	return nil
}

// GetUserByID retrieves a user by their ObjectID
func GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	collection := database.Collection("users")

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	var user models.User
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	return &user, nil
}

// GetUserByIDAndCompanyID retrieves a user by their ObjectID and company ID
func GetUserByIDAndCompanyID(ctx context.Context, userID, companyID string) (*models.User, error) {
	collection := database.Collection("users")

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	var user models.User
	err = collection.FindOne(ctx, bson.M{
		"_id":        objectID,
		"company_id": companyID,
	}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found in company")
		}
		return nil, err
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by their email (for backward compatibility)
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

// GetUserByEmailAndCompany retrieves a user by their email and company_id
func GetUserByEmailAndCompany(ctx context.Context, email, companyID string) (*models.User, error) {
	collection := database.Collection("users")
	slog.Info("Searching for user",
		"database", database.Name(),
		"collection", "users",
		"email", email,
		"company_id", companyID)

	var user models.User
	err := collection.FindOne(ctx, bson.M{
		"email":      email,
		"company_id": companyID,
	}).Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Try to count users with this email across all companies
			emailCount, _ := collection.CountDocuments(ctx, bson.M{"email": email})
			// Count users in this company
			companyCount, _ := collection.CountDocuments(ctx, bson.M{"company_id": companyID})

			slog.Info("User not found",
				"email", email,
				"company_id", companyID,
				"users_with_email", emailCount,
				"users_in_company", companyCount)
			return nil, fmt.Errorf("user not found in company")
		}
		return nil, err
	}

	slog.Info("User found",
		"email", email,
		"company_id", companyID,
		"username", user.Username,
		"has_password", user.Password != "")

	return &user, nil
}

// GetUserByUsername retrieves a user by their username
func GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	collection := database.Collection("users")

	var user models.User
	err := collection.FindOne(ctx, bson.M{
		"username":  username,
		"is_active": true,
	}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found")
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

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}

	// Add updated timestamp
	update["updated_at"] = time.Now()

	// Validate role if it's being updated
	if role, exists := update["role"]; exists {
		if !models.IsValidRole(role.(string)) {
			return fmt.Errorf("invalid role: %s", role)
		}
	}

	// Hash password if it's being updated
	if password, exists := update["password"]; exists {
		passwordStr := password.(string)
		if passwordStr != "" && !strings.HasPrefix(passwordStr, "$2a$") {
			hashedPassword, err := HashPassword(passwordStr)
			if err != nil {
				return fmt.Errorf("failed to hash password: %w", err)
			}
			update["password"] = hashedPassword
		}
	}

	result, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
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

// UpdateLastLogin updates the user's last login time
func UpdateLastLogin(ctx context.Context, userID string) error {
	return UpdateUser(ctx, userID, bson.M{"last_login": time.Now()})
}

// UpdateUserLastLogin is an alias for UpdateLastLogin
func UpdateUserLastLogin(ctx context.Context, userID string) error {
	return UpdateLastLogin(ctx, userID)
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
	// Company admins and bot managers can manage all pages in their company
	if user.Role == models.RoleCompanyAdmin || user.Role == models.RoleBotManager {
		return true
	}

	return false
}
