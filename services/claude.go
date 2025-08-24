package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"facebook-bot/models"
)

const claudeAPIURL = "https://api.anthropic.com/v1/messages"

// ClaudeRequest represents the request to Claude API
type ClaudeRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
	Tools     []Tool    `json:"tools,omitempty"`
	System    string    `json:"system,omitempty"`
}

// Message represents a message in the conversation
type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// Tool represents a tool that Claude can use
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"input_schema"`
}

// InputSchema represents the schema for tool input
type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required"`
}

// Property represents a property in the input schema
type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

// ContentBlock represents a content block in Claude's response
type ContentBlock struct {
	Type  string  `json:"type"`
	Text  string  `json:"text,omitempty"`
	ID    string  `json:"id,omitempty"`
	Name  string  `json:"name,omitempty"`
	Input ToolUse `json:"input,omitempty"`
}

// ToolUse represents tool use input
type ToolUse struct {
	Intent string `json:"intent,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// ClaudeResponse represents the response from Claude API
type ClaudeResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason"`
	StopSequence string         `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// GetClaudeResponseWithConfig gets a response from Claude AI using company configuration
func GetClaudeResponseWithConfig(ctx context.Context, input, messageType string, company *models.Company, pageConfig *models.FacebookPage) (string, error) {
	// Test mode: if API key is "TEST_MODE", return a mock response
	if pageConfig.ClaudeAPIKey == "TEST_MODE" {
		slog.Info("Running in TEST_MODE - returning mock response")
		return fmt.Sprintf("TEST RESPONSE: I received your %s message: '%s'. This is a test response.", messageType, input), nil
	}

	if pageConfig.ClaudeAPIKey == "" {
		return "", fmt.Errorf("Claude API key not configured for page %s", pageConfig.PageID)
	}

	fmt.Println(pageConfig.ClaudeAPIKey)

	// Build the system prompt based on context and page configuration
	systemPrompt := buildSystemPromptWithConfig(messageType, pageConfig.PageName, pageConfig.SystemPrompt)

	// Set max tokens from page config or use default
	maxTokens := pageConfig.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	// Create the request
	requestBody := ClaudeRequest{
		Model:     pageConfig.ClaudeModel,
		MaxTokens: maxTokens,
		Messages: []Message{
			{
				Role:    "user",
				Content: systemPrompt + "\n\n" + input,
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", claudeAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	// Use retry logic for API call
	resp, body, err := callClaudeAPIWithRetry(req, pageConfig.ClaudeAPIKey, 3)
	if err != nil {
		// Check if it's a timeout error
		if os.IsTimeout(err) || strings.Contains(err.Error(), "deadline exceeded") {
			slog.Error("Claude API timeout (config)",
				"error", err,
				"messageLength", len(input),
				"messageType", messageType,
			)
			return "", fmt.Errorf("Claude API timeout - request took too long. Try with a shorter message")
		}
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("Claude API error", "status", resp.StatusCode, "body", string(body))
		return "", fmt.Errorf("Claude API error: %s", resp.Status)
	}

	var claudeResp ClaudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", err
	}

	if len(claudeResp.Content) > 0 {
		response := claudeResp.Content[0].Text
		slog.Info("Claude response generated",
			"inputTokens", claudeResp.Usage.InputTokens,
			"outputTokens", claudeResp.Usage.OutputTokens,
		)
		return response, nil
	}

	return "", fmt.Errorf("no response content from Claude")
}

// GetClaudeResponseWithHistory gets a response from Claude AI with conversation history
func GetClaudeResponseWithHistory(ctx context.Context, input, messageType string, company *models.Company, pageConfig *models.FacebookPage, history []ChatHistory) (string, error) {
	return GetClaudeResponseWithRAG(ctx, input, messageType, company, pageConfig, history, "")
}

// callClaudeAPIWithRetry makes an API call with retry logic for transient errors
func callClaudeAPIWithRetry(req *http.Request, apiKey string, maxRetries int) (*http.Response, []byte, error) {
	client := &http.Client{
		Timeout: 45 * time.Second,
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 2^attempt seconds (2s, 4s, 8s)
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			slog.Info("Retrying Claude API call",
				"attempt", attempt+1,
				"backoff", backoff,
				"lastError", lastErr)
			time.Sleep(backoff)
		}

		// Clone the request for retry
		reqCopy := req.Clone(req.Context())
		reqCopy.Header.Set("Content-Type", "application/json")
		reqCopy.Header.Set("x-api-key", apiKey)
		reqCopy.Header.Set("anthropic-version", "2023-06-01")

		resp, err := client.Do(reqCopy)
		if err != nil {
			lastErr = err
			// Don't retry on timeout or context cancellation
			if os.IsTimeout(err) || strings.Contains(err.Error(), "context canceled") {
				return nil, nil, err
			}
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		// Check for retryable status codes
		if resp.StatusCode == http.StatusTooManyRequests ||
			resp.StatusCode == http.StatusServiceUnavailable ||
			resp.StatusCode == http.StatusGatewayTimeout ||
			(resp.StatusCode == http.StatusInternalServerError && strings.Contains(string(body), "Overloaded")) {
			lastErr = fmt.Errorf("Claude API error (retryable): %d - %s", resp.StatusCode, string(body))
			continue
		}

		// Success or non-retryable error
		return resp, body, nil
	}

	return nil, nil, fmt.Errorf("Claude API failed after %d attempts: %w", maxRetries, lastErr)
}

// GetClaudeResponseWithToolUse gets a response using tool calling for intent detection
func GetClaudeResponseWithToolUse(ctx context.Context, input, messageType string, company *models.Company, pageConfig *models.FacebookPage, history []ChatHistory, ragContext string) (string, bool, error) {
	// Test mode: if API key is "TEST_MODE", return a mock response
	if pageConfig.ClaudeAPIKey == "TEST_MODE" {
		slog.Info("Running in TEST_MODE - returning mock response")
		return fmt.Sprintf("TEST RESPONSE: I received your %s message: '%s'. This is a test response.", messageType, input), false, nil
	}

	if pageConfig.ClaudeAPIKey == "" {
		return "", false, fmt.Errorf("Claude API key not configured for page %s", pageConfig.PageID)
	}

	// Build formatted input for the user message
	var formattedInput strings.Builder

	// System Prompt Section
	formattedInput.WriteString("COMPANY CONTEXT:\n")
	if pageConfig.SystemPrompt != "" {
		formattedInput.WriteString(pageConfig.SystemPrompt)
	} else {
		formattedInput.WriteString("You are a helpful customer service assistant for " + pageConfig.PageName)
	}
	formattedInput.WriteString("\n\nCRITICAL: You MUST respond in the SAME LANGUAGE the customer used in their message. If they write in Georgian, respond in Georgian. If they write in English, respond in English. Match their language exactly.\n\n")

	// Chat History Section
	if len(history) > 0 {
		formattedInput.WriteString("CONVERSATION HISTORY:\n")
		for _, h := range history {
			if h.Role == "user" {
				formattedInput.WriteString(fmt.Sprintf("Customer: %s\n", h.Content))
			} else {
				formattedInput.WriteString(fmt.Sprintf("Assistant: %s\n", h.Content))
			}
		}
		formattedInput.WriteString("\n")
	}

	// RAG Data Section
	if ragContext != "" {
		formattedInput.WriteString("KNOWLEDGE BASE:\n")
		formattedInput.WriteString(ragContext)
		formattedInput.WriteString("\n\n")
	}

	// Customer Question Section
	formattedInput.WriteString("CURRENT CUSTOMER MESSAGE:\n")
	formattedInput.WriteString(input)
	formattedInput.WriteString("\n\n")

	// Response Instructions
	formattedInput.WriteString("YOUR TASK:\n")
	formattedInput.WriteString("1. Determine if the customer EXPLICITLY wants a human agent\n")
	formattedInput.WriteString("   ONLY mark as 'wants_agent' if they explicitly request: human, agent, operator, representative, real person, support team\n")
	formattedInput.WriteString("   Common greetings in ANY language (hello, hi, გამარჯობა, привет, etc.) are NOT requests for agents\n")
	formattedInput.WriteString("2. Call detect_agent_request tool with:\n")
	formattedInput.WriteString("   - intent='wants_agent' ONLY if they explicitly ask for a human\n")
	formattedInput.WriteString("   - intent='continue_bot' for EVERYTHING else (greetings, questions, math, general inquiries)\n")
	formattedInput.WriteString("3. After using the tool, ALWAYS write a response:\n")
	formattedInput.WriteString("   - For wants_agent: 'დაგაკავშირებთ რეალურ ადამიანთან'\n")
	formattedInput.WriteString("   - For continue_bot: Respond appropriately (greet back, answer questions, etc.)\n")

	if ragContext != "" {
		formattedInput.WriteString("\n⚠️ CRITICAL KNOWLEDGE BASE ENFORCEMENT ⚠️\n")
		formattedInput.WriteString("YOU ARE STRICTLY LIMITED TO THE KNOWLEDGE BASE PROVIDED.\n")
		formattedInput.WriteString("- CHECK: Is the question about information in the KNOWLEDGE BASE above? \n")
		formattedInput.WriteString("  - IF YES → Answer using ONLY that information\n")
		formattedInput.WriteString("  - IF NO → You MUST respond with something like: 'I can only provide information about [main topic from knowledge base]. Could you please ask about that instead?'\n")
		formattedInput.WriteString("- FORBIDDEN: Answering about weather, math, general knowledge, news, or ANYTHING not in the knowledge base\n")
		formattedInput.WriteString("- REQUIRED: Redirect ALL off-topic questions back to your knowledge base topic\n")
	}

	// Define the tool for detecting agent requests
	agentDetectionTool := Tool{
		Name:        "detect_agent_request",
		Description: "Always use this tool to indicate whether the customer wants to speak with a real human agent or continue with the bot. Be very careful: greetings are NOT agent requests!",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"intent": {
					Type:        "string",
					Description: "Set to 'wants_agent' ONLY if customer explicitly requests human/agent/operator/representative. Set to 'continue_bot' for greetings, questions, and all other messages",
					Enum:        []string{"wants_agent", "continue_bot"},
				},
				"reason": {
					Type:        "string",
					Description: "Brief explanation of why this intent was detected (e.g., 'Customer said hello - this is a greeting' or 'Customer explicitly asked for human agent')",
				},
			},
			Required: []string{"intent", "reason"},
		},
	}

	// Set max tokens from page config or use default
	maxTokens := pageConfig.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	// Create system message that enforces both tool use and text response
	systemMessage := "You are a customer service assistant. You MUST ALWAYS do these two things in order:\n" +
		"1. FIRST: Use the detect_agent_request tool to determine if the customer wants a human agent\n" +
		"   - ONLY detect 'wants_agent' if they EXPLICITLY ask for human/agent/operator/representative\n" +
		"   - Greetings in ANY language are NOT agent requests - they should be 'continue_bot'\n" +
		"2. THEN: Write a text response to the customer\n\n" +
		"CRITICAL LANGUAGE RULE: You MUST respond in the SAME LANGUAGE the customer used in their message.\n" +
		"- If the customer writes in Georgian, respond in Georgian\n" +
		"- If the customer writes in English, respond in English\n" +
		"- If the customer writes in Russian, respond in Russian\n" +
		"- Match the customer's language exactly - this is essential for good customer service\n\n"

	// Add RAG-specific instructions to system message
	if ragContext != "" {
		systemMessage += "🛒 ONLINE STORE ASSISTANT - STRICT LIMITATIONS 🛒\n" +
			"You are an online store customer service bot. You can ONLY answer questions about:\n" +
			"✅ Products in our store\n" +
			"✅ Prices and discounts\n" +
			"✅ Shipping and delivery\n" +
			"✅ Payment methods\n" +
			"✅ Returns and refunds\n" +
			"✅ Order status\n" +
			"✅ Product availability\n" +
			"✅ Store policies\n\n" +
			"ABSOLUTELY FORBIDDEN (INSTANT REJECTION):\n" +
			"❌ Any question NOT about our online store\n" +
			"❌ Weather, news, general knowledge\n" +
			"❌ Math problems, calculations (except order totals)\n" +
			"❌ Personal advice, opinions, recommendations outside our products\n" +
			"❌ Entertainment, sports, politics, technology\n" +
			"❌ Anything not directly related to shopping in our store\n\n" +
			"YOUR ONLY ALLOWED RESPONSE FOR NON-STORE QUESTIONS:\n" +
			"'I am an online store assistant. I can only help with questions about our products, orders, shipping, and store policies. Please ask me about our store.'\n\n" +
			"ENFORCEMENT RULES:\n" +
			"1. BEFORE answering, CHECK: Is this about our ONLINE STORE?\n" +
			"2. If YES → Is the answer in the knowledge base? → Answer ONLY with that info\n" +
			"3. If NO → Use the rejection response above\n" +
			"4. If UNSURE → Use the rejection response above\n" +
			"5. NEVER discuss topics outside online shopping\n" +
			"6. ONLY use information from the knowledge base\n\n"
	}

	systemMessage += "If they want an agent: Acknowledge their request politely in their language\n" +
		"If they don't want an agent: Respond naturally to their message in their language (greet back, answer questions, etc.)\n\n" +
		"IMPORTANT: Be very careful - simple greetings like 'hello', 'hi', 'გამარჯობა' are NOT requests for agents!"

	// Create the request with tool
	requestBody := ClaudeRequest{
		Model:     pageConfig.ClaudeModel,
		MaxTokens: maxTokens,
		System:    systemMessage,
		Messages: []Message{
			{
				Role:    "user",
				Content: formattedInput.String(),
			},
		},
		Tools: []Tool{agentDetectionTool},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", false, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", claudeAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", false, err
	}

	// Use retry logic for API call
	resp, body, err := callClaudeAPIWithRetry(req, pageConfig.ClaudeAPIKey, 3)
	if err != nil {
		if os.IsTimeout(err) || strings.Contains(err.Error(), "deadline exceeded") {
			slog.Error("Claude API timeout with tool use",
				"error", err,
				"messageLength", len(input),
			)
			return "", false, fmt.Errorf("Claude API timeout - request took too long")
		}
		slog.Error("Claude API call failed after retries",
			"error", err,
			"pageID", pageConfig.PageID,
			"inputLength", len(input))
		return "", false, err
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("Claude API error with tool use",
			"status", resp.StatusCode,
			"body", string(body),
			"pageID", pageConfig.PageID,
			"inputLength", len(input))
		return "", false, fmt.Errorf("Claude API error: %s - %s", resp.Status, string(body))
	}

	var claudeResp ClaudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		slog.Error("Failed to parse Claude response",
			"error", err,
			"body", string(body))
		return "", false, err
	}

	// Log the response structure for debugging
	slog.Debug("Claude API response structure",
		"contentCount", len(claudeResp.Content),
		"stopReason", claudeResp.StopReason,
		"model", claudeResp.Model)

	// Process the response to check for tool use and extract text
	var responseText string
	wantsAgent := false
	toolUsed := false

	for _, content := range claudeResp.Content {
		if content.Type == "tool_use" && content.Name == "detect_agent_request" {
			toolUsed = true
			// Check if customer wants an agent
			if content.Input.Intent == "wants_agent" {
				wantsAgent = true
				slog.Info("Tool detected customer wants real agent",
					"input", input,
					"reason", content.Input.Reason)
			} else {
				slog.Info("Tool detected customer does NOT want agent",
					"input", input,
					"reason", content.Input.Reason,
					"intent", content.Input.Intent)
			}
		} else if content.Type == "text" {
			responseText = content.Text
			preview := responseText
			if len(responseText) > 100 {
				preview = responseText[:100] + "..."
			}
			slog.Debug("Claude provided text response",
				"textLength", len(responseText),
				"preview", preview)
		}
	}

	// Log if tool was used but no text response
	if toolUsed && responseText == "" {
		var contentTypes []string
		for _, c := range claudeResp.Content {
			contentTypes = append(contentTypes, c.Type)
		}
		slog.Warn("Tool was used but no text response provided",
			"wantsAgent", wantsAgent,
			"contentTypes", contentTypes)
	}

	// If Claude didn't provide text content (only used the tool), make a second call for the response
	if responseText == "" && toolUsed {
		slog.Info("Tool used without text, making follow-up call for response",
			"wantsAgent", wantsAgent,
			"input", input)

		// Prepare a follow-up prompt based on the detected intent
		var followUpPrompt string
		if wantsAgent {
			followUpPrompt = "The customer has requested to speak with a human agent. Please acknowledge their request politely and let them know you'll connect them with someone."
		} else {
			// For continue_bot, we need to actually answer their question
			followUpPrompt = fmt.Sprintf("Please respond to the customer's message: '%s'\n", input)
			if ragContext != "" {
				followUpPrompt += fmt.Sprintf("\nUse this information to answer:\n%s", ragContext)
			}
			if pageConfig.SystemPrompt != "" {
				followUpPrompt += fmt.Sprintf("\n\nContext: %s", pageConfig.SystemPrompt)
			}
		}

		// Make a simple call without tools to get the text response
		followUpRequest := ClaudeRequest{
			Model:     pageConfig.ClaudeModel,
			MaxTokens: maxTokens,
			System:    "You are a helpful customer service assistant. Provide a direct, helpful response to the customer. CRITICAL: Respond in the SAME LANGUAGE the customer used in their message.",
			Messages: []Message{
				{
					Role:    "user",
					Content: followUpPrompt,
				},
			},
		}

		followUpJSON, err := json.Marshal(followUpRequest)
		if err != nil {
			slog.Error("Failed to marshal follow-up request", "error", err)
		} else {
			followUpReq, err := http.NewRequestWithContext(ctx, "POST", claudeAPIURL, bytes.NewBuffer(followUpJSON))
			if err == nil {
				followUpReq.Header.Set("Content-Type", "application/json")
				followUpReq.Header.Set("x-api-key", pageConfig.ClaudeAPIKey)
				followUpReq.Header.Set("anthropic-version", "2023-06-01")

				client := &http.Client{Timeout: 30 * time.Second}
				followUpResp, err := client.Do(followUpReq)
				if err == nil {
					defer followUpResp.Body.Close()
					followUpBody, _ := io.ReadAll(followUpResp.Body)

					if followUpResp.StatusCode == http.StatusOK {
						var followUpClaudeResp ClaudeResponse
						if err := json.Unmarshal(followUpBody, &followUpClaudeResp); err == nil {
							if len(followUpClaudeResp.Content) > 0 {
								responseText = followUpClaudeResp.Content[0].Text
								slog.Info("Got text response from follow-up call",
									"responseLength", len(responseText),
									"inputTokens", followUpClaudeResp.Usage.InputTokens,
									"outputTokens", followUpClaudeResp.Usage.OutputTokens)
							}
						}
					}
				}
			}
		}

		// If still no response, use fallback
		if responseText == "" {
			if wantsAgent {
				responseText = "I understand you'd like to speak with a human agent. Let me connect you with someone who can help you right away."
			} else {
				// Provide a more contextual response based on common inputs
				lowerInput := strings.ToLower(input)
				if strings.Contains(lowerInput, "hello") || strings.Contains(lowerInput, "hi") ||
					strings.Contains(lowerInput, "გამარჯობა") || strings.Contains(lowerInput, "სალამი") {
					responseText = "Hello! Welcome to " + pageConfig.PageName + ". How can I help you today?"
				} else if strings.Contains(lowerInput, "2+2") || strings.Contains(lowerInput, "2 + 2") {
					responseText = "2 + 2 equals 4. Is there anything else I can help you with?"
				} else {
					responseText = "Thank you for your message. I'm here to help you. How can I assist you today?"
				}
			}

			slog.Warn("Using fallback response after failed follow-up",
				"pageID", pageConfig.PageID,
				"model", pageConfig.ClaudeModel,
				"input", input)
		}
	}

	slog.Info("Claude response with tool use generated",
		"inputTokens", claudeResp.Usage.InputTokens,
		"outputTokens", claudeResp.Usage.OutputTokens,
		"wantsAgent", wantsAgent,
		"hasResponse", responseText != "",
	)

	return responseText, wantsAgent, nil
}

// GetClaudeResponseWithRAG gets a response from Claude AI with RAG context
func GetClaudeResponseWithRAG(ctx context.Context, input, messageType string, company *models.Company, pageConfig *models.FacebookPage, history []ChatHistory, ragContext string) (string, error) {
	// Test mode: if API key is "TEST_MODE", return a mock response
	if pageConfig.ClaudeAPIKey == "TEST_MODE" {
		slog.Info("Running in TEST_MODE - returning mock response")
		historyInfo := ""
		if len(history) > 0 {
			historyInfo = fmt.Sprintf(" (with %d history messages)", len(history))
		}
		ragInfo := ""
		if ragContext != "" {
			ragInfo = " (with RAG context)"
		}
		return fmt.Sprintf("TEST RESPONSE: I received your %s message: '%s'%s%s. This is a test response.", messageType, input, historyInfo, ragInfo), nil
	}

	if pageConfig.ClaudeAPIKey == "" {
		return "", fmt.Errorf("Claude API key not configured for page %s", pageConfig.PageID)
	}

	// STRICT PRE-FILTERING: If we have RAG context, check if the question is about online store
	if ragContext != "" {
		// Check if question is NOT about online store
		if !isOnlineStoreQuestion(input, ragContext) {
			slog.Info("Pre-filtered non-store question",
				"question", input)

			// Return a standard rejection message for non-store questions
			rejectionMsg := "I am an online store assistant. I can only help with questions about our products, orders, shipping, and store policies. Please ask me about our store."

			// Match the customer's language if possible
			if containsGeorgian(input) {
				rejectionMsg = "მე ვარ ონლაინ მაღაზიის ასისტენტი. შემიძლია დაგეხმაროთ მხოლოდ ჩვენი პროდუქტების, შეკვეთების, მიწოდებისა და მაღაზიის პოლიტიკის შესახებ კითხვებზე."
			} else if containsRussian(input) {
				rejectionMsg = "Я ассистент интернет-магазина. Могу помочь только с вопросами о наших товарах, заказах, доставке и политике магазина."
			}

			return rejectionMsg, nil
		}
	}

	// Build formatted input with clear labels
	var formattedInput strings.Builder

	// System Prompt Section
	formattedInput.WriteString("SYSTEM PROMPT:\n")
	formattedInput.WriteString(pageConfig.SystemPrompt)
	if pageConfig.SystemPrompt == "" {
		formattedInput.WriteString("You are a helpful customer service assistant for " + pageConfig.PageName)
	}
	formattedInput.WriteString("\n\nCRITICAL LANGUAGE RULE: You MUST respond in the SAME LANGUAGE the customer used in their message. If they write in Georgian, respond in Georgian. If they write in English, respond in English. If they write in Russian, respond in Russian. Match their language exactly.\n\n")

	// Post Context Section (if from comment)
	if messageType == "comment" {
		formattedInput.WriteString("POST CONTEXT:\n")
		formattedInput.WriteString(fmt.Sprintf("This is a comment on a Facebook post from %s page.", pageConfig.PageName))
		formattedInput.WriteString("\n\n")
	}

	// Chat History Section
	if len(history) > 0 {
		formattedInput.WriteString("CHAT HISTORY:\n")
		for _, h := range history {
			if h.Role == "user" {
				formattedInput.WriteString(fmt.Sprintf("Customer: %s\n", h.Content))
			} else {
				formattedInput.WriteString(fmt.Sprintf("Assistant: %s\n", h.Content))
			}
		}
		formattedInput.WriteString("\n")
	}

	// RAG Data Section
	if ragContext != "" {
		formattedInput.WriteString("RAG DATA:\n")
		formattedInput.WriteString(ragContext)
		formattedInput.WriteString("\n\n")
	}

	fmt.Println(ragContext)

	// Customer Question Section
	formattedInput.WriteString("CUSTOMER QUESTION:\n")
	formattedInput.WriteString(input)
	formattedInput.WriteString("\n\n")

	// Response Instructions
	formattedInput.WriteString("INSTRUCTIONS:\n")
	if ragContext != "" {
		formattedInput.WriteString("🛒 ONLINE STORE ASSISTANT MODE 🛒\n\n")
		formattedInput.WriteString("DECISION FLOWCHART:\n")
		formattedInput.WriteString("┌─ Is this about ONLINE SHOPPING/STORE?\n")
		formattedInput.WriteString("├─ NO → Reply: 'I am an online store assistant. I can only help with questions about our products, orders, shipping, and store policies.'\n")
		formattedInput.WriteString("├─ MAYBE → Reply: 'I am an online store assistant. I can only help with questions about our products, orders, shipping, and store policies.'\n")
		formattedInput.WriteString("└─ YES → Is this about OUR store specifically?\n")
		formattedInput.WriteString("    ├─ NO → Reply: 'I am an online store assistant. I can only help with questions about our products, orders, shipping, and store policies.'\n")
		formattedInput.WriteString("    └─ YES → Is the answer in the RAG DATA?\n")
		formattedInput.WriteString("        ├─ NO → Reply: 'I am an online store assistant. I can only help with questions about our products, orders, shipping, and store policies.'\n")
		formattedInput.WriteString("        └─ YES → Answer using ONLY the RAG DATA\n\n")
		formattedInput.WriteString("NON-STORE TOPICS (INSTANT REJECTION):\n")
		formattedInput.WriteString("× Weather/Climate → REJECT\n")
		formattedInput.WriteString("× News/Politics → REJECT\n")
		formattedInput.WriteString("× Math (except prices) → REJECT\n")
		formattedInput.WriteString("× Entertainment → REJECT\n")
		formattedInput.WriteString("× General knowledge → REJECT\n")
		formattedInput.WriteString("× Personal advice (non-shopping) → REJECT\n")
		formattedInput.WriteString("× ANYTHING not about our online store → REJECT\n\n")
		formattedInput.WriteString("YOUR IDENTITY: You are ONLY a shopping assistant for THIS online store. Nothing else.\n")
	} else {
		formattedInput.WriteString("Please provide a helpful response based on the information above. ")
		formattedInput.WriteString("Be professional and friendly. ")
	}
	formattedInput.WriteString("Respond in the customer's language.")

	// Get the complete formatted prompt
	systemPrompt := formattedInput.String()

	// Log RAG context status
	if ragContext != "" {
		slog.Info("Sending to Claude with RAG context",
			"ragContextLength", len(ragContext),
			"messageType", messageType,
		)
	}

	// Set max tokens from page config or use default
	maxTokens := pageConfig.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	// Build messages array - now simplified since everything is in systemPrompt
	messages := []Message{
		{
			Role:    "user",
			Content: systemPrompt,
		},
	}

	// Create system message for RAG
	var systemMsg string
	if ragContext != "" {
		systemMsg = "🛒 YOU ARE AN ONLINE STORE ASSISTANT - EXTREME RESTRICTIONS 🛒\n\n" +
			"You work EXCLUSIVELY for an online store. You can ONLY discuss:\n" +
			"• Products we sell\n" +
			"• Prices and promotions\n" +
			"• Shipping/delivery options\n" +
			"• Payment methods\n" +
			"• Returns/refunds/exchanges\n" +
			"• Order tracking\n" +
			"• Product availability\n" +
			"• Store policies\n\n" +
			"CRITICAL RULE: If the question is NOT about online shopping → REJECT IT\n\n" +
			"REJECTION PROTOCOL:\n" +
			"1. Question NOT about our store? → 'I am an online store assistant. I can only help with questions about our products, orders, shipping, and store policies.'\n" +
			"2. Question about store but answer not in knowledge base? → Same rejection\n" +
			"3. Unsure if it's store-related? → Same rejection\n\n" +
			"FORBIDDEN TOPICS (AUTOMATIC REJECTION):\n" +
			"⛔ Weather, news, general knowledge\n" +
			"⛔ Math (except calculating order totals)\n" +
			"⛔ Personal advice unrelated to shopping\n" +
			"⛔ Entertainment, sports, politics\n" +
			"⛔ Technology (unless selling tech products)\n" +
			"⛔ ANYTHING not about online shopping\n\n" +
			"VERIFICATION BEFORE EVERY RESPONSE:\n" +
			"☐ Is this about our ONLINE STORE? If NO → REJECT\n" +
			"☐ Is this about shopping/products/orders? If NO → REJECT\n" +
			"☐ Is the answer in the knowledge base? If NO → REJECT\n\n" +
			"You are a shopping assistant, NOT a general AI. Stay in character."
	} else {
		systemMsg = "You are a helpful customer service assistant. Always respond in the SAME LANGUAGE the customer used."
	}

	// Create the request
	requestBody := ClaudeRequest{
		Model:     pageConfig.ClaudeModel,
		MaxTokens: maxTokens,
		System:    systemMsg,
		Messages:  messages,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", claudeAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", pageConfig.ClaudeAPIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Create client with longer timeout
	client := &http.Client{
		Timeout: 45 * time.Second, // 45 second timeout for HTTP client
	}
	resp, err := client.Do(req)
	if err != nil {
		// Check if it's a timeout error
		if os.IsTimeout(err) || strings.Contains(err.Error(), "deadline exceeded") {
			slog.Error("Claude API timeout",
				"error", err,
				"messageLength", len(input),
				"historyCount", len(history),
				"ragContextLength", len(ragContext),
			)
			return "", fmt.Errorf("Claude API timeout - request took too long. Try with a shorter message or less context")
		}
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("Claude API error", "status", resp.StatusCode, "body", string(body))
		return "", fmt.Errorf("Claude API error: %s", resp.Status)
	}

	var claudeResp ClaudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", err
	}

	if len(claudeResp.Content) > 0 {
		response := claudeResp.Content[0].Text

		// Check if customer wants to talk to a real person
		if DetectRealPersonIntent(input, response) {
			// Return a special indicator that can be detected by the message handler
			response = "CUSTOMER_WANTS_REAL_PERSON||" + response
		}

		// Log the Claude response
		logClaudeResponse(response, claudeResp.Usage.InputTokens, claudeResp.Usage.OutputTokens)

		slog.Info("Claude response generated with history",
			"inputTokens", claudeResp.Usage.InputTokens,
			"outputTokens", claudeResp.Usage.OutputTokens,
			"historyCount", len(history),
		)
		return response, nil
	}

	return "", fmt.Errorf("no response content from Claude")
}

// buildSystemPromptWithConfig builds a context-aware system prompt with company configuration
func buildSystemPromptWithConfig(messageType, pageName, customPrompt string) string {
	return buildSystemPromptWithRAG(messageType, pageName, customPrompt, "")
}

// calculateTotalLength calculates the total character length of all messages
func calculateTotalLength(messages []Message) int {
	total := 0
	for _, msg := range messages {
		// Content can be either string or other types
		if content, ok := msg.Content.(string); ok {
			total += len(content)
		}
	}
	return total
}

// logClaudeResponse logs the response received from Claude
func logClaudeResponse(response string, inputTokens, outputTokens int) {
	// Only log if DEBUG_CLAUDE environment variable is set
	if os.Getenv("DEBUG_CLAUDE") != "true" {
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("✅ CLAUDE API RESPONSE")
	fmt.Println(strings.Repeat("=", 80))

	// Log token usage
	fmt.Printf("📊 Token Usage:\n")
	fmt.Printf("  • Input Tokens: %d\n", inputTokens)
	fmt.Printf("  • Output Tokens: %d\n", outputTokens)
	fmt.Printf("  • Total Tokens: %d\n", inputTokens+outputTokens)
	fmt.Printf("  • Response Length: %d characters\n", len(response))
	fmt.Println()

	// Log the response
	fmt.Println("💬 Response Content:")
	fmt.Println(strings.Repeat("-", 40))

	// Truncate very long responses for console readability
	if len(response) > 1500 {
		fmt.Printf("%s\n", response[:1000])
		fmt.Println("\n... [RESPONSE TRUNCATED] ...")
		fmt.Printf("%s\n", response[len(response)-300:])
	} else {
		fmt.Printf("%s\n", response)
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("✅ RESPONSE DELIVERED TO USER")
	fmt.Println(strings.Repeat("=", 80) + "\n")
}

// buildSystemPromptWithRAG builds a context-aware system prompt with RAG context
func buildSystemPromptWithRAG(messageType, pageName, customPrompt, ragContext string) string {
	basePrompt := fmt.Sprintf("You are a professional customer service representative for '%s'. ", pageName)

	// If company has custom prompt, use it as base
	if customPrompt != "" {
		basePrompt = customPrompt + " "
	}

	// Add RAG context if available with stronger instructions
	if ragContext != "" {
		basePrompt = fmt.Sprintf(`🔴 CRITICAL INSTRUCTIONS - YOU MUST FOLLOW THESE RULES:

You are a knowledgeable sales and customer service representative with access to the company's complete database of properties, products, and services.

📊 AVAILABLE INFORMATION DATABASE:
%s

⚠️ MANDATORY RESPONSE GUIDELINES:
1. READ AND USE THE DATABASE: You MUST carefully read ALL information provided above before responding
2. ANSWER BASED ON DATA: Your responses MUST be based on the actual data provided, not generic assumptions
3. BE SPECIFIC: When customers ask about properties, apartments, prices, or availability, provide SPECIFIC details from the database
4. ACCURATE PRICING: Always quote exact prices and terms from the database when available
5. PROPERTY DETAILS: Include relevant details like floor, area, rooms, status, and pricing when discussing properties
6. AVAILABILITY STATUS: Clearly indicate if properties are available, sold, or reserved based on the data
7. PROACTIVE INFORMATION: Anticipate follow-up questions and provide comprehensive information upfront
8. PROFESSIONAL TONE: Maintain a helpful, professional, and friendly tone while being informative

RESPONSE STRUCTURE:
- First, directly answer the customer's question using specific data
- Provide relevant details from the database
- If multiple options exist, present them clearly
- Offer to provide additional information if needed
- If information is not in the database, clearly state that and offer to connect them with someone who can help

IMPORTANT: Never make up information. Only use what's provided in the database above.

%s`, ragContext, basePrompt)
	} else {
		// Even without RAG context, emphasize customer service
		basePrompt = fmt.Sprintf(`You are a professional customer service representative for '%s'.

IMPORTANT: Since you don't have access to specific property/product data at this moment, you should:
1. Acknowledge the customer's inquiry professionally
2. Explain that you'll need to check the latest information
3. Ask for their contact details or specific requirements
4. Offer to have someone with access to the current database contact them promptly
5. Be helpful and maintain a professional, friendly tone

%s`, pageName, basePrompt)
	}

	switch messageType {
	case "chat":
		return basePrompt + `

CHAT MESSAGE GUIDELINES:
- Respond directly to the user's question with specific information
- Use data from the database to provide accurate answers about properties, prices, and availability
- Be friendly but professional
- If discussing properties, include key details (price, area, rooms, floor, status)
- Offer to provide additional information or schedule a viewing if appropriate`
	case "comment":
		return basePrompt + `

COMMENT RESPONSE GUIDELINES:
- Acknowledge the comment and provide relevant information
- If the comment is about properties/products, share specific details from the database
- Encourage further engagement by inviting questions
- Keep responses informative but concise for public visibility`
	case "reply":
		return basePrompt + `

REPLY THREAD GUIDELINES:
- Maintain context from the original post and parent comment
- Provide specific answers using database information
- Continue the conversation naturally while being informative
- Reference previous points in the thread when relevant`
	default:
		return basePrompt + `

Provide a helpful, professional response using the available information from the database. Focus on answering customer questions accurately and completely.`
	}
}

// DetectRealPersonIntent analyzes customer input to detect if they want to talk to a real person
func DetectRealPersonIntent(customerInput string, botResponse string) bool {
	// Convert to lowercase for case-insensitive matching
	input := strings.ToLower(customerInput)

	// Keywords and phrases that indicate wanting to talk to a real person
	realPersonPhrases := []string{
		"real person",
		"human",
		"speak to someone",
		"talk to someone",
		"customer service",
		"representative",
		"agent",
		"operator",
		"real human",
		"actual person",
		"live person",
		"live chat",
		"transfer me",
		"connect me",
		"help from a person",
		"talk to a person",
		"speak with a person",
		"not a bot",
		"not a robot",
		"stop bot",
		"i want a human",
		"give me a human",
		"need a human",
		"prefer human",
		"want to speak",
		"want to talk",
		"need to speak",
		"need to talk",
		"contact support",
		"real support",
		"человек",       // Russian
		"оператор",      // Russian
		"менеджер",      // Russian
		"живой человек", // Russian
	}

	// Check for any of the phrases in the customer input
	for _, phrase := range realPersonPhrases {
		if strings.Contains(input, phrase) {
			slog.Info("Detected customer wants real person",
				"input", customerInput,
				"matched_phrase", phrase)
			return true
		}
	}

	// Also check for frustration indicators combined with help requests
	frustrationWords := []string{"frustrated", "angry", "upset", "annoyed", "tired of", "sick of", "enough", "stop"}
	helpWords := []string{"help", "assist", "support", "please", "need"}

	hasFrustration := false
	hasHelp := false

	for _, word := range frustrationWords {
		if strings.Contains(input, word) {
			hasFrustration = true
			break
		}
	}

	for _, word := range helpWords {
		if strings.Contains(input, word) {
			hasHelp = true
			break
		}
	}

	// If both frustration and help are detected, likely wants human assistance
	if hasFrustration && hasHelp {
		slog.Info("Detected customer frustration with help request - likely wants real person",
			"input", customerInput)
		return true
	}

	return false
}

// GetClaudeIntentDetection uses Claude to detect if customer wants a real person
func GetClaudeIntentDetection(ctx context.Context, customerInput string, apiKey string) (string, error) {
	// Special prompt for intent detection
	intentPrompt := `Analyze the following customer message and determine if they are requesting to speak with a real human agent instead of a bot.

Customer message: "` + customerInput + `"

Respond with ONLY one of these two exact strings:
- "customer wants real person" (if they are requesting human assistance)
- "continue with bot" (if they are not requesting human assistance)

Common phrases that indicate wanting a human:
- "I want to speak to a real person/human"
- "Transfer me to an agent"
- "Can I talk to customer service"
- "I need human help"
- "Stop the bot"
- Expressions of frustration combined with requests for help

Do not include any other text in your response.`

	// Create a minimal request for intent detection
	requestBody := ClaudeRequest{
		Model:     "claude-3-haiku-20240307", // Use faster model for intent detection
		MaxTokens: 50,
		Messages: []Message{
			{
				Role:    "user",
				Content: intentPrompt,
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", claudeAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Create client with appropriate timeout for intent detection with tools
	client := &http.Client{
		Timeout: 30 * time.Second, // Increased from 10s to handle tool calls properly
	}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Failed to get intent detection from Claude", "error", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("Claude API error for intent detection", "status", resp.StatusCode, "body", string(body))
		return "", fmt.Errorf("Claude API error: %s", resp.Status)
	}

	var claudeResp ClaudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", err
	}

	if len(claudeResp.Content) > 0 {
		intent := strings.TrimSpace(strings.ToLower(claudeResp.Content[0].Text))
		slog.Info("Claude intent detection result",
			"input", customerInput,
			"intent", intent)
		return intent, nil
	}

	return "", fmt.Errorf("no response content from Claude for intent detection")
}

// extractMainTopic tries to identify the main topic from RAG context
func extractMainTopic(ragContext string) string {
	// Take first 200 characters to identify topic
	preview := ragContext
	if len(ragContext) > 200 {
		preview = ragContext[:200]
	}

	// Look for common topic indicators
	lowerPreview := strings.ToLower(preview)
	if strings.Contains(lowerPreview, "property") || strings.Contains(lowerPreview, "apartment") || strings.Contains(lowerPreview, "real estate") {
		return "properties and real estate"
	} else if strings.Contains(lowerPreview, "product") || strings.Contains(lowerPreview, "catalog") || strings.Contains(lowerPreview, "item") {
		return "our products and services"
	} else if strings.Contains(lowerPreview, "service") || strings.Contains(lowerPreview, "support") {
		return "our services"
	} else if strings.Contains(lowerPreview, "company") || strings.Contains(lowerPreview, "about") {
		return "our company information"
	}

	// Default fallback
	return "the information in our knowledge base"
}

// isOffTopicQuestion checks if a question is clearly off-topic based on common patterns
func isOffTopicQuestion(question, ragContext string) bool {
	lowerQuestion := strings.ToLower(question)
	lowerRAG := strings.ToLower(ragContext)

	// List of clearly off-topic question patterns
	offTopicPatterns := []string{
		// Weather
		"weather", "temperature", "forecast", "rain", "snow", "sunny", "cloudy",
		"ამინდი", "ტემპერატურა", "წვიმა", "თოვლი", // Georgian weather terms
		"погода", "температура", "дождь", "снег", // Russian weather terms

		// Math and calculations (unless RAG is about math)
		"calculate", "solve", "equation", "math", "algebra", "geometry",
		"გამოთვალე", "ამოხსენი", "განტოლება", // Georgian math terms
		"вычислить", "решить", "уравнение", // Russian math terms

		// General knowledge
		"capital of", "president", "population", "history of", "when was", "who invented",
		"დედაქალაქი", "პრეზიდენტი", "მოსახლეობა", // Georgian general knowledge
		"столица", "президент", "население", // Russian general knowledge

		// Personal advice
		"should i", "what do you think", "your opinion", "advice about", "recommend me",
		"რა ვქნა", "რას მირჩევ", // Georgian advice
		"что делать", "посоветуй", // Russian advice

		// Current events (unless RAG is news)
		"latest news", "today's news", "current events", "what happened",
		"ახალი ამბები", "დღეს რა მოხდა", // Georgian news
		"новости", "что случилось", // Russian news

		// Programming/Tech (unless RAG is about tech)
		"write code", "python", "javascript", "debug", "programming",
		"კოდი", "პროგრამირება", // Georgian programming
		"код", "программирование", // Russian programming

		// Entertainment
		"movie", "song", "game", "sport", "football", "basketball",
		"ფილმი", "სიმღერა", "თამაში", "სპორტი", // Georgian entertainment
		"фильм", "песня", "игра", "спорт", // Russian entertainment
	}

	// Check if question contains off-topic patterns
	for _, pattern := range offTopicPatterns {
		if strings.Contains(lowerQuestion, pattern) {
			// But check if this pattern is actually IN the RAG context
			if !strings.Contains(lowerRAG, pattern) {
				slog.Debug("Off-topic pattern detected",
					"pattern", pattern,
					"question", question)
				return true
			}
		}
	}

	// Check for questions that are too general or philosophical
	philosophicalPatterns := []string{
		"meaning of life", "what is love", "why do we exist", "purpose of",
		"სიცოცხლის აზრი", "რა არის სიყვარული", // Georgian philosophical
		"смысл жизни", "что такое любовь", // Russian philosophical
	}

	for _, pattern := range philosophicalPatterns {
		if strings.Contains(lowerQuestion, pattern) {
			return true
		}
	}

	// Check if question is just greeting or small talk (these are OK)
	greetings := []string{
		"hello", "hi", "hey", "good morning", "good evening",
		"გამარჯობა", "სალამი", "დილა მშვიდობისა", // Georgian greetings
		"привет", "здравствуйте", "добрый день", // Russian greetings
	}

	for _, greeting := range greetings {
		if strings.TrimSpace(lowerQuestion) == greeting {
			return false // Greetings are allowed
		}
	}

	return false
}

// isOnlineStoreQuestion checks if a question is related to online store/shopping
func isOnlineStoreQuestion(question, ragContext string) bool {
	lowerQuestion := strings.ToLower(question)

	// Online store related keywords that indicate valid questions
	storeKeywords := []string{
		// Products and shopping
		"product", "item", "buy", "purchase", "order", "cart", "checkout",
		"price", "cost", "discount", "sale", "promo", "coupon", "deal",
		"stock", "available", "availability", "in stock", "out of stock",

		// Shipping and delivery
		"ship", "shipping", "delivery", "deliver", "track", "tracking",
		"express", "standard", "overnight", "international",

		// Payment
		"pay", "payment", "credit card", "debit", "paypal", "visa", "mastercard",
		"billing", "invoice", "receipt",

		// Returns and support
		"return", "refund", "exchange", "warranty", "guarantee",
		"cancel", "cancellation", "customer service", "support",

		// Store policies
		"policy", "policies", "terms", "conditions", "privacy",

		// Georgian shopping terms
		"პროდუქტი", "ყიდვა", "შეკვეთა", "ფასი", "მიწოდება", "გადახდა",
		"დაბრუნება", "გაცვლა", "ფასდაკლება",

		// Russian shopping terms
		"товар", "продукт", "купить", "заказ", "цена", "доставка",
		"оплата", "возврат", "обмен", "скидка", "корзина",

		// Categories (common in online stores)
		"category", "categories", "brand", "size", "color", "model",
		"კატეგორია", "ზომა", "ფერი", // Georgian
		"категория", "размер", "цвет", // Russian
	}

	// Check if question contains store-related keywords
	hasStoreKeyword := false
	for _, keyword := range storeKeywords {
		if strings.Contains(lowerQuestion, keyword) {
			hasStoreKeyword = true
			break
		}
	}

	// List of clearly NON-store topics that override store keywords
	nonStoreTopics := []string{
		// Weather
		"weather", "temperature", "forecast", "rain", "snow", "sunny",
		"ამინდი", "ტემპერატურა", "წვიმა", // Georgian
		"погода", "температура", "дождь", // Russian

		// News and current events
		"news", "politics", "president", "election", "war", "covid",
		"ახალი ამბები", "პოლიტიკა", // Georgian
		"новости", "политика", // Russian

		// Entertainment
		"movie", "film", "song", "music", "game", "sport", "football",
		"ფილმი", "სიმღერა", "თამაში", "სპორტი", // Georgian
		"фильм", "песня", "игра", "спорт", // Russian

		// General knowledge
		"capital of", "population", "history", "when was", "who invented",
		"დედაქალაქი", "მოსახლეობა", "ისტორია", // Georgian
		"столица", "население", "история", // Russian

		// Math/Science (unless about pricing)
		"equation", "formula", "theorem", "physics", "chemistry",
		"განტოლება", "ფორმულა", "ფიზიკა", // Georgian
		"уравнение", "формула", "физика", // Russian

		// Personal/Philosophical
		"meaning of life", "love", "happiness", "depression",
		"სიცოცხლის აზრი", "სიყვარული", // Georgian
		"смысл жизни", "любовь", // Russian
	}

	// Check for non-store topics
	for _, topic := range nonStoreTopics {
		if strings.Contains(lowerQuestion, topic) {
			// Even if it has store keywords, these topics override
			return false
		}
	}

	// Special case: greetings are OK even without store keywords
	greetings := []string{
		"hello", "hi", "hey", "good morning", "good evening", "how are you",
		"გამარჯობა", "სალამი", "როგორ ხარ", // Georgian
		"привет", "здравствуйте", "как дела", // Russian
	}

	for _, greeting := range greetings {
		if strings.Contains(lowerQuestion, greeting) {
			return true // Allow greetings
		}
	}

	// If no store keywords found and not a greeting, it's probably off-topic
	if !hasStoreKeyword {
		return false
	}

	return true
}

// containsGeorgian checks if text contains Georgian characters
func containsGeorgian(text string) bool {
	for _, r := range text {
		if r >= 0x10A0 && r <= 0x10FF {
			return true
		}
	}
	return false
}

// containsRussian checks if text contains Cyrillic characters
func containsRussian(text string) bool {
	for _, r := range text {
		if (r >= 0x0400 && r <= 0x04FF) || (r >= 0x0500 && r <= 0x052F) {
			return true
		}
	}
	return false
}
