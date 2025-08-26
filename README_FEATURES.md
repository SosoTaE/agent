# Customer Management Features

## Overview

This bot system includes comprehensive customer management capabilities for handling Facebook Messenger conversations, with special support for human agent takeover when customers request real person assistance.

## Key Features

### 1. Customer Tracking
- Automatically tracks all customers who message your Facebook pages
- Stores customer information including names, message history, and interaction timestamps
- Tracks message count and last seen times
- Maintains `updated_at` timestamp to track time since last conversation

### 2. Human Agent Request Detection
- AI automatically detects when customers want to speak to a real person
- Sets `stop` flag to prevent bot responses
- Tracks when customer requested help with `stopped_at` timestamp
- Notifies all connected agents via WebSocket in real-time

### 3. Agent Assignment System
- Agents can be assigned to specific customers
- Prevents multiple agents from handling the same customer
- Tracks agent details: ID, name, email, and assignment time
- Auto-assignment when agent sends first message to customer

### 4. Real-Time WebSocket Updates
- Live updates for all connected agents
- Notifications for new help requests
- Agent assignment/unassignment broadcasts
- Customer status change notifications
- Message delivery confirmations

## Database Schema

### Customer Collection Fields

```javascript
{
  _id: ObjectId,
  customer_id: "facebook_user_id",
  customer_name: "John Doe",
  first_name: "John",
  last_name: "Doe",
  page_id: "facebook_page_id",
  page_name: "Business Page",
  company_id: "company_123",
  message_count: 42,
  last_message: "I need help with my order",
  last_seen: ISODate("2024-01-01T10:30:00Z"),
  first_seen: ISODate("2024-01-01T09:00:00Z"),
  stop: true,  // Customer wants human assistance
  stopped_at: ISODate("2024-01-01T10:00:00Z"),
  agent_name: "Agent Smith",
  agent_id: "agent_123",
  agent_email: "agent@example.com",
  assigned_at: ISODate("2024-01-01T10:05:00Z"),
  created_at: ISODate("2024-01-01T09:00:00Z"),
  updated_at: ISODate("2024-01-01T10:30:00Z")
}
```

## API Endpoints

### REST API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/dashboard/customers` | Get all customers (with pagination) |
| GET | `/api/dashboard/customers/:id` | Get specific customer details |
| PUT | `/api/dashboard/customers/:id/stop` | Update customer stop status |
| POST | `/api/dashboard/customers/:id/toggle-stop` | Toggle stop status |
| PUT | `/api/dashboard/customers/:id/agent` | Update agent name |
| POST | `/api/dashboard/customers/:id/message` | Send message to customer |

### WebSocket Messages

| Type | Direction | Description |
|------|-----------|-------------|
| `get_stopped_customers` | Client → Server | Request list of customers needing help |
| `stopped_customers` | Server → Client | Response with customer list |
| `assign_agent` | Client → Server | Assign agent to customer |
| `agent_assigned` | Server → Client | Confirmation of assignment |
| `unassign_agent` | Client → Server | Release customer assignment |
| `agent_unassigned` | Server → Client | Confirmation of unassignment |
| `send_message` | Client → Server | Send message to customer |
| `message_sent` | Server → Client | Message delivery confirmation |

### WebSocket Broadcasts

| Type | Description |
|------|-------------|
| `agent_requested` | Customer requested human help |
| `agent_assignment_changed` | Agent assigned/unassigned |
| `customer_stop_status_changed` | Stop status changed |
| `new_message` | New message in conversation |

## Usage Examples

### Get Customers Waiting for Help

**WebSocket Request:**
```json
{
  "type": "get_stopped_customers",
  "data": {
    "page_id": "optional_page_filter",
    "limit": 50,
    "skip": 0
  }
}
```

**Response includes:**
- Customer information
- Time since last conversation (human-readable and minutes)
- Assignment status (`is_assigned`, `is_assigned_to_me`)
- Agent details if assigned

### Assign Agent to Customer

**WebSocket Request:**
```json
{
  "type": "assign_agent",
  "data": {
    "customer_id": "cust_123",
    "page_id": "page_456"
  }
}
```

**Features:**
- Prevents double assignment
- Broadcasts to all agents
- Returns full customer object with agent info

### Send Message with Auto-Assignment

**WebSocket Request:**
```json
{
  "type": "send_message",
  "customer_id": "cust_123",
  "page_id": "page_456",
  "message": "Hello, I'm here to help!"
}
```

**Behavior:**
- Automatically assigns sender if customer unassigned
- Prevents sending if assigned to another agent
- Sends via Facebook Messenger API
- Saves to message history

## Workflow Scenarios

### Scenario 1: Customer Requests Help

1. Customer messages: "I want to speak to a human"
2. AI detects intent and sets `stop: true`
3. Bot responds: "I've notified our team. Someone will assist you shortly."
4. All agents receive `agent_requested` broadcast
5. Customer appears in stopped customers list

### Scenario 2: Agent Takes Over

1. Agent views list of waiting customers
2. Agent sends `assign_agent` message
3. System assigns agent and broadcasts update
4. Other agents see customer is now "handled"
5. Agent can send messages to customer

### Scenario 3: Multiple Agents

1. Agent A assigns customer
2. Agent B tries to assign same customer
3. Agent B receives error: "Already assigned to Agent A"
4. Agent B sees assignment status in UI
5. Only Agent A can message customer

### Scenario 4: Agent Completes Help

1. Agent finishes helping customer
2. Agent sends `unassign_agent` message
3. System removes assignment
4. Broadcasts unassignment to all agents
5. Customer available for future help

## Security Features

- **Authentication Required**: All endpoints require valid session
- **Company Isolation**: Agents only see their company's customers
- **Page Ownership Validation**: Operations validate page belongs to company
- **Assignment Protection**: Only assigned agent can message/unassign
- **Audit Trail**: All actions logged with agent information

## Performance Optimizations

- **Database Indexes**: Optimized queries with compound indexes
- **Pagination**: Default 50 items, max 200 per request
- **Efficient Broadcasting**: Company-scoped WebSocket broadcasts
- **Caching**: Session and company data cached
- **Connection Pooling**: MongoDB connection pooling

## Error Handling

Common error scenarios and resolutions:

| Error | Cause | Resolution |
|-------|-------|------------|
| "Customer not found" | Invalid customer/page ID | Verify IDs |
| "Already assigned" | Another agent has customer | Choose different customer |
| "Not assigned to you" | Trying to unassign another's customer | Only assigned agent can unassign |
| "Page access denied" | Page not in your company | Verify page ownership |
| "Customer has not requested help" | Trying to message without stop flag | Wait for help request |

## Best Practices

1. **Regular Status Checks**: Poll `get_stopped_customers` periodically
2. **Handle Disconnections**: Implement WebSocket reconnection logic
3. **Optimistic UI**: Update UI immediately, rollback on error
4. **Clean Up**: Unassign customers when done helping
5. **Monitor Time**: Prioritize customers waiting longest
6. **Conflict Resolution**: Handle assignment conflicts gracefully

## Monitoring and Analytics

Track these metrics for system health:

- **Response Time**: Time from help request to agent assignment
- **Handle Time**: Duration of agent assignment
- **Queue Length**: Number of customers waiting
- **Agent Utilization**: Customers per agent
- **Resolution Rate**: Customers helped vs. abandoned

## Future Enhancements

Planned improvements:

- [ ] Agent availability status
- [ ] Customer priority levels
- [ ] Conversation history in WebSocket
- [ ] Agent performance metrics
- [ ] Automated assignment rules
- [ ] Customer satisfaction tracking
- [ ] Shift scheduling integration
- [ ] Mobile app support

## Development Setup

1. **MongoDB Indexes**: Run `CreateIndexesForCustomers()` on startup
2. **WebSocket Server**: Ensure WebSocket upgrade middleware is configured
3. **Authentication**: Implement session-based auth for WebSocket
4. **CORS**: Configure for your frontend domain
5. **SSL/TLS**: Use `wss://` in production

## Troubleshooting

### WebSocket Won't Connect
- Check authentication/session
- Verify WebSocket URL
- Check firewall/proxy settings

### Not Seeing Updates
- Ensure same company ID
- Check WebSocket connection status
- Verify broadcast message handling

### Assignment Failing
- Check customer stop status
- Verify page ownership
- Ensure not already assigned

### Messages Not Sending
- Verify Facebook Page Access Token
- Check customer stop status
- Ensure agent is assigned

## Support

For issues or questions:
1. Check documentation in `/docs` folder
2. Review error logs in application
3. Verify MongoDB connection and indexes
4. Test WebSocket connection with provided HTML tester