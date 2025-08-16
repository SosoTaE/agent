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
	if company.ClaudeAPIKey == "TEST_MODE" {
		slog.Info("Running in TEST_MODE - returning mock response")
		return fmt.Sprintf("TEST RESPONSE: I received your %s message: '%s'. This is a test response.", messageType, input), nil
	}

	if company.ClaudeAPIKey == "" {
		return "", fmt.Errorf("Claude API key not configured for company %s", company.CompanyID)
	}

	fmt.Println(company.ClaudeAPIKey)

	// Build the system prompt based on context and company configuration
	systemPrompt := buildSystemPromptWithConfig(messageType, pageConfig.PageName, company.SystemPrompt)

	// Set max tokens from company config or use default
	maxTokens := company.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	// Create the request
	requestBody := ClaudeRequest{
		Model:     company.ClaudeModel,
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

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", company.ClaudeAPIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Create client with longer timeout
	client := &http.Client{
		Timeout: 45 * time.Second, // 45 second timeout for HTTP client
	}
	resp, err := client.Do(req)
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

// GetClaudeResponseWithToolUse gets a response using tool calling for intent detection
func GetClaudeResponseWithToolUse(ctx context.Context, input, messageType string, company *models.Company, pageConfig *models.FacebookPage, history []ChatHistory, ragContext string) (string, bool, error) {
	// Test mode: if API key is "TEST_MODE", return a mock response
	if company.ClaudeAPIKey == "TEST_MODE" {
		slog.Info("Running in TEST_MODE - returning mock response")
		return fmt.Sprintf("TEST RESPONSE: I received your %s message: '%s'. This is a test response.", messageType, input), false, nil
	}

	if company.ClaudeAPIKey == "" {
		return "", false, fmt.Errorf("Claude API key not configured for company %s", company.CompanyID)
	}

	// Build formatted input with clear labels
	var formattedInput strings.Builder

	// System Prompt Section
	formattedInput.WriteString("SYSTEM PROMPT:\n")
	formattedInput.WriteString(company.SystemPrompt)
	if company.SystemPrompt == "" {
		formattedInput.WriteString("You are a helpful customer service assistant for " + pageConfig.PageName)
	}
	formattedInput.WriteString("\n\n")

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

	// Customer Question Section
	formattedInput.WriteString("CUSTOMER MESSAGE:\n")
	formattedInput.WriteString(input)
	formattedInput.WriteString("\n\n")

	// Instructions
	formattedInput.WriteString("INSTRUCTIONS:\n")
	formattedInput.WriteString("1. First, use the detect_agent_request tool to determine if the customer wants to speak with a real human agent.\n")
	formattedInput.WriteString("2. Then provide a helpful response based on the information above.\n")
	if ragContext != "" {
		formattedInput.WriteString("3. Use the RAG DATA to answer accurately. Do not make up information not present in the RAG DATA.\n")
	}
	formattedInput.WriteString("Be professional and friendly.")

	// Define the tool for detecting agent requests
	agentDetectionTool := Tool{
		Name:        "detect_agent_request",
		Description: "Detect if the customer is requesting to speak with a real human agent",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"intent": {
					Type:        "string",
					Description: "Whether the customer wants a real person or not",
					Enum:        []string{"wants_agent", "continue_bot"},
				},
				"reason": {
					Type:        "string",
					Description: "Brief explanation of why this intent was detected",
				},
			},
			Required: []string{"intent", "reason"},
		},
	}

	// Set max tokens from company config or use default
	maxTokens := company.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	// Create the request with tool
	requestBody := ClaudeRequest{
		Model:     company.ClaudeModel,
		MaxTokens: maxTokens,
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

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", company.ClaudeAPIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Create client with longer timeout
	client := &http.Client{
		Timeout: 45 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		if os.IsTimeout(err) || strings.Contains(err.Error(), "deadline exceeded") {
			slog.Error("Claude API timeout with tool use",
				"error", err,
				"messageLength", len(input),
			)
			return "", false, fmt.Errorf("Claude API timeout - request took too long")
		}
		return "", false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, err
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("Claude API error", "status", resp.StatusCode, "body", string(body))
		return "", false, fmt.Errorf("Claude API error: %s", resp.Status)
	}

	var claudeResp ClaudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", false, err
	}

	// Process the response to check for tool use and extract text
	var responseText string
	wantsAgent := false

	for _, content := range claudeResp.Content {
		if content.Type == "tool_use" && content.Name == "detect_agent_request" {
			// Check if customer wants an agent
			if content.Input.Intent == "wants_agent" {
				wantsAgent = true
				slog.Info("Tool detected customer wants real agent",
					"input", input,
					"reason", content.Input.Reason)
			}
		} else if content.Type == "text" {
			responseText = content.Text
		}
	}

	slog.Info("Claude response with tool use generated",
		"inputTokens", claudeResp.Usage.InputTokens,
		"outputTokens", claudeResp.Usage.OutputTokens,
		"wantsAgent", wantsAgent,
	)

	return responseText, wantsAgent, nil
}

// GetClaudeResponseWithRAG gets a response from Claude AI with RAG context
func GetClaudeResponseWithRAG(ctx context.Context, input, messageType string, company *models.Company, pageConfig *models.FacebookPage, history []ChatHistory, ragContext string) (string, error) {
	// Test mode: if API key is "TEST_MODE", return a mock response
	if company.ClaudeAPIKey == "TEST_MODE" {
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

	if company.ClaudeAPIKey == "" {
		return "", fmt.Errorf("Claude API key not configured for company %s", company.CompanyID)
	}

	// Build formatted input with clear labels
	var formattedInput strings.Builder

	// System Prompt Section
	formattedInput.WriteString("SYSTEM PROMPT:\n")
	formattedInput.WriteString(company.SystemPrompt)
	if company.SystemPrompt == "" {
		formattedInput.WriteString("You are a helpful customer service assistant for " + pageConfig.PageName)
	}
	formattedInput.WriteString("\n\n")

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
	formattedInput.WriteString("Please provide a helpful response based on the information above. ")
	if ragContext != "" {
		formattedInput.WriteString("Use the RAG DATA to answer accurately. Do not make up information not present in the RAG DATA. ")
	}
	formattedInput.WriteString("Be professional and friendly.")

	// Get the complete formatted prompt
	systemPrompt := formattedInput.String()

	// Log RAG context status
	if ragContext != "" {
		slog.Info("Sending to Claude with RAG context",
			"ragContextLength", len(ragContext),
			"messageType", messageType,
		)
	}

	// Set max tokens from company config or use default
	maxTokens := company.MaxTokens
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

	// Create the request
	requestBody := ClaudeRequest{
		Model:     company.ClaudeModel,
		MaxTokens: maxTokens,
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
	req.Header.Set("x-api-key", company.ClaudeAPIKey)
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

	// Create client with shorter timeout for intent detection
	client := &http.Client{
		Timeout: 10 * time.Second,
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
