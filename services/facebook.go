package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

const fbGraphAPI = "https://graph.facebook.com/v18.0"

// SendMessengerReply sends a reply message via Messenger
func SendMessengerReply(ctx context.Context, recipientID, message, pageAccessToken string) error {
	url := fmt.Sprintf("%s/me/messages?access_token=%s", fbGraphAPI, pageAccessToken)

	payload := map[string]interface{}{
		"recipient": map[string]string{
			"id": recipientID,
		},
		"message": map[string]string{
			"text": message,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("Failed to send messenger reply", "status", resp.StatusCode, "body", string(body))
		return fmt.Errorf("failed to send message: %s", resp.Status)
	}

	return nil
}

// CommentResponse represents the response from Facebook when creating a comment
type CommentResponse struct {
	ID string `json:"id"`
}

// ReplyToCommentWithResponse replies to a Facebook comment and returns the response
func ReplyToCommentWithResponse(ctx context.Context, commentID, message, pageAccessToken string) (*CommentResponse, error) {
	url := fmt.Sprintf("%s/%s/comments?access_token=%s", fbGraphAPI, commentID, pageAccessToken)

	payload := map[string]string{
		"message": message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		slog.Error("Failed to reply to comment", "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("failed to reply to comment: %s", resp.Status)
	}

	var commentResp CommentResponse
	if err := json.Unmarshal(body, &commentResp); err != nil {
		return nil, err
	}

	return &commentResp, nil
}

// GetPostContent retrieves post content from Facebook
func GetPostContent(ctx context.Context, postID, pageAccessToken string) (string, error) {
	url := fmt.Sprintf("%s/%s?fields=message&access_token=%s", fbGraphAPI, postID, pageAccessToken)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get post content: %s", resp.Status)
	}

	var result struct {
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Message, nil
}

// GetPageNameFromFB retrieves page name from Facebook API
func GetPageNameFromFB(ctx context.Context, pageID, pageAccessToken string) (string, error) {
	url := fmt.Sprintf("%s/%s?fields=name&access_token=%s", fbGraphAPI, pageID, pageAccessToken)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get page name: %s", resp.Status)
	}

	var result struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Name, nil
}

// IsCommentFromPage checks if a comment is from the page itself
func IsCommentFromPage(ctx context.Context, commentID, pageAccessToken string) (bool, error) {
	url := fmt.Sprintf("%s/%s?fields=from{id}&access_token=%s", fbGraphAPI, commentID, pageAccessToken)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Warn("Failed to check comment sender", "status", resp.StatusCode, "body", string(body))
		return false, fmt.Errorf("failed to check comment: %s", resp.Status)
	}

	var result struct {
		From struct {
			ID string `json:"id"`
		} `json:"from"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	// Get the page's own ID
	pageInfoURL := fmt.Sprintf("%s/me?fields=id&access_token=%s", fbGraphAPI, pageAccessToken)
	pageReq, err := http.NewRequestWithContext(ctx, "GET", pageInfoURL, nil)
	if err != nil {
		return false, err
	}

	pageResp, err := client.Do(pageReq)
	if err != nil {
		return false, err
	}
	defer pageResp.Body.Close()

	var pageInfo struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(pageResp.Body).Decode(&pageInfo); err != nil {
		return false, err
	}

	// Check if the comment is from the page
	isFromPage := result.From.ID == pageInfo.ID

	if isFromPage {
		slog.Info("Comment is from the page itself",
			"commentID", commentID,
			"fromID", result.From.ID,
			"pageID", pageInfo.ID,
		)
	}

	return isFromPage, nil
}
