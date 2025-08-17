package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// FacebookUserDetails represents the user details from Facebook Graph API
type FacebookUserDetails struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// GetFacebookUserDetails fetches user details from Facebook Graph API
func GetFacebookUserDetails(ctx context.Context, userID string, accessToken string) (*FacebookUserDetails, error) {
	// Build the Graph API URL
	apiURL := fmt.Sprintf("https://graph.facebook.com/v19.0/%s", userID)

	// Create URL with query parameters
	u, err := url.Parse(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	q.Set("fields", "first_name,last_name")
	q.Set("access_token", accessToken)
	u.RawQuery = q.Encode()

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user details: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		slog.Error("Facebook API error",
			"status", resp.StatusCode,
			"body", string(body),
			"userID", userID)
		return nil, fmt.Errorf("Facebook API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var userDetails FacebookUserDetails
	if err := json.Unmarshal(body, &userDetails); err != nil {
		return nil, fmt.Errorf("failed to parse user details: %w", err)
	}

	// Set the ID if not present in response
	if userDetails.ID == "" {
		userDetails.ID = userID
	}

	slog.Info("Successfully fetched Facebook user details",
		"userID", userID,
		"firstName", userDetails.FirstName,
		"lastName", userDetails.LastName)

	return &userDetails, nil
}

// UpdateUserNameInDatabase updates the user's first and last name in the database
func UpdateUserNameInDatabase(ctx context.Context, userID, firstName, lastName string) error {
	db := GetDatabase()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	fullName := fmt.Sprintf("%s %s", firstName, lastName)
	updateData := map[string]interface{}{
		"$set": map[string]interface{}{
			"first_name":  firstName,
			"last_name":   lastName,
			"sender_name": fullName,
			"updated_at":  time.Now(),
		},
	}

	filter := map[string]interface{}{
		"sender_id": userID,
	}

	// Update all comments from this user with the new name information
	commentsCollection := db.Collection("comments")
	commentResult, err := commentsCollection.UpdateMany(ctx, filter, updateData)
	if err != nil {
		return fmt.Errorf("failed to update user names in comments: %w", err)
	}

	// Update all messages from this user with the new name information
	messagesCollection := db.Collection("messages")
	messageResult, err := messagesCollection.UpdateMany(ctx, filter, updateData)
	if err != nil {
		return fmt.Errorf("failed to update user names in messages: %w", err)
	}

	slog.Info("Updated user names in database",
		"userID", userID,
		"firstName", firstName,
		"lastName", lastName,
		"commentsModified", commentResult.ModifiedCount,
		"messagesModified", messageResult.ModifiedCount)

	return nil
}

// FetchAndSaveUserDetails fetches user details from Facebook and saves them to the database
func FetchAndSaveUserDetails(ctx context.Context, userID string, accessToken string) error {
	// Skip if userID is empty or looks like a page ID
	if userID == "" || len(userID) < 5 {
		return nil
	}

	// Fetch user details from Facebook
	userDetails, err := GetFacebookUserDetails(ctx, userID, accessToken)
	if err != nil {
		slog.Warn("Failed to fetch Facebook user details",
			"userID", userID,
			"error", err)
		// Don't return error as this is not critical
		return nil
	}

	// Update the database with the new names
	if userDetails.FirstName != "" || userDetails.LastName != "" {
		err = UpdateUserNameInDatabase(ctx, userID, userDetails.FirstName, userDetails.LastName)
		if err != nil {
			slog.Warn("Failed to update user names in database",
				"userID", userID,
				"error", err)
		}
	}

	return nil
}
