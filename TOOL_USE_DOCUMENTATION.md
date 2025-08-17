# Tool Use Implementation for Agent Detection

## Overview
The webhook processing system now uses Claude's tool calling feature to detect when customers want to talk to a real human agent instead of the bot.

## How It Works

### 1. Tool Definition
The system defines a tool called `detect_agent_request` that Claude uses to analyze customer messages:

```json
{
  "name": "detect_agent_request",
  "description": "Detect if the customer is requesting to speak with a real human agent",
  "input_schema": {
    "type": "object",
    "properties": {
      "intent": {
        "type": "string",
        "enum": ["wants_agent", "continue_bot"],
        "description": "Whether the customer wants a real person or not"
      },
      "reason": {
        "type": "string",
        "description": "Brief explanation of why this intent was detected"
      }
    },
    "required": ["intent", "reason"]
  }
}
```

### 2. Message Processing Flow

When a message is received through the webhook:

1. **Webhook receives message** → `webhooks/webhook.go`
2. **Message is processed** → `handlers/message_handler.go`
3. **Claude API is called with tool** → `services/claude.go::GetClaudeResponseWithToolUse()`
4. **Tool detects intent** → Returns response text and boolean `wantsAgent`
5. **If agent requested**:
   - Customer's `stop` status is set to `true`
   - Future messages from this customer bypass the bot
   - Human support team is notified via WebSocket
   - Customer receives confirmation message

### 3. Key Functions

#### `GetClaudeResponseWithToolUse()`
Located in `services/claude.go`, this function:
- Formats the conversation context with system prompt, chat history, and RAG data
- Includes the agent detection tool in the API request
- Processes Claude's response to extract both the text response and tool use results
- Returns: `(responseText string, wantsAgent bool, error)`

#### Message Handler Updates
In `handlers/message_handler.go`:
```go
// Get AI response from Claude with tool use for agent detection
aiResponse, wantsAgent, err := services.GetClaudeResponseWithToolUse(
    ctx, messageText, "chat", company, pageConfig, chatHistory, ragContext
)

if wantsAgent {
    // Update customer's stop status
    services.UpdateCustomerStopStatus(ctx, senderID, pageID, true)
    
    // Notify human support team
    wsManager.BroadcastToCompany(company.CompanyID, services.BroadcastMessage{
        Type: "agent_requested",
        Data: map[string]interface{}{
            "chat_id": senderID,
            "customer_name": senderName,
            "message": messageText,
        },
    })
    
    // Add confirmation to response
    aiResponse += "\n\nI've notified our human support team. Someone will assist you shortly."
}
```

### 4. Phrases That Trigger Agent Detection

The tool is trained to detect phrases like:
- "I want to talk to a real person"
- "Can I speak with a human agent?"
- "Transfer me to customer service"
- "I need help from a real human"
- "Stop the bot"
- "Give me an operator"
- Frustration combined with help requests

### 5. WebSocket Notifications

When a customer requests an agent, a WebSocket broadcast is sent:
```json
{
  "type": "agent_requested",
  "company_id": "company_123",
  "page_id": "page_456",
  "data": {
    "chat_id": "customer_789",
    "customer_name": "John Doe",
    "message": "I need to speak with a real person",
    "timestamp": 1692345678
  }
}
```

### 6. Customer Stop Status

Once a customer's `stop` status is set to `true`:
- Bot no longer responds to their messages
- Messages are still saved to the database
- WebSocket notifications still occur
- Human agents can see and respond via the dashboard

### 7. Comment/Reply Handling

For Facebook comments and replies:
- Tool detection still works
- If agent is requested, response includes: "For personal assistance, please send us a direct message and our team will help you."
- This directs public comment interactions to private messaging

## Testing

To test with a real Claude API key:

1. Set up a company with a valid Claude API key in the database
2. Send messages through the webhook that should trigger agent detection
3. Check logs for "Tool detected customer wants real agent"
4. Verify customer's `stop` status is updated in the database
5. Confirm WebSocket notifications are sent

## Configuration

No additional configuration is needed. The tool use feature works automatically when:
- Company has a valid Claude API key
- Claude model supports tool use (e.g., claude-3-haiku, claude-3-sonnet, claude-3-opus)

## Benefits

1. **More Accurate Detection**: Tool use is more reliable than keyword matching
2. **Context Awareness**: Considers conversation history and context
3. **Consistent Behavior**: Same detection logic across all message types
4. **Clear Intent Logging**: Tool provides reasons for its decisions
5. **Seamless Integration**: Works with existing RAG and chat history features