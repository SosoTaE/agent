# Agent Assignment Management

## Overview

The agent assignment system allows human agents to take over conversations from the AI bot. When an agent is assigned to a customer, only that agent can send messages to the customer, ensuring exclusive communication and preventing conflicts.

## Data Model

### Customer Model Fields

```go
type Customer struct {
    // ... other fields ...
    
    // Agent Assignment Fields
    AgentName    string     `bson:"agent_name,omitempty"`    // Name of assigned agent
    AgentID      string     `bson:"agent_id,omitempty"`      // User ID of assigned agent
    AgentEmail   string     `bson:"agent_email,omitempty"`   // Email of assigned agent
    AssignedAt   *time.Time `bson:"assigned_at,omitempty"`   // When agent was assigned
    
    // Status Fields
    Stop         bool       `bson:"stop"`                    // If true, bot won't respond
    UpdatedAt    time.Time  `bson:"updated_at"`              // Last interaction time
}
```

## API Endpoints

### 1. Get Stopped Customers

Retrieve customers who have requested to talk to a real person (stop=true).

**Endpoint:** Via WebSocket message

**WebSocket Message:**
```json
{
  "type": "get_stopped_customers",
  "page_id": "page_123456",
  "limit": 10,
  "skip": 0
}
```

**Response:**
```json
{
  "type": "stopped_customers",
  "data": {
    "customers": [
      {
        "customer_id": "cust_123",
        "customer_name": "John Doe",
        "stop": true,
        "agent_name": "",
        "agent_id": "",
        "time_since_last_conversation": "5m ago",
        "last_message": "I need help with my order",
        "updated_at": "2024-01-15T10:30:00Z"
      }
    ],
    "total": 15,
    "page_id": "page_123456"
  }
}
```

### 2. Assign Agent to Customer

Assign an agent to take over a customer conversation.

**REST Endpoint:** `PUT /api/dashboard/customers/:customerID/agent`

**Request Body:**
```json
{
  "page_id": "page_123456",
  "agent_name": "John Agent"
}
```

**WebSocket Alternative:**
```json
{
  "type": "assign_agent",
  "customer_id": "cust_123",
  "page_id": "page_123456"
}
```

**Response:**
```json
{
  "message": "Agent assigned successfully",
  "customer": {
    "customer_id": "cust_123",
    "agent_name": "John Agent",
    "agent_id": "user_456",
    "agent_email": "john@company.com",
    "assigned_at": "2024-01-15T10:35:00Z"
  }
}
```

### 3. Unassign Agent from Customer

Remove agent assignment, allowing the bot to resume or another agent to take over.

**REST Endpoint:** `DELETE /api/dashboard/customers/:customerID/agent`

**Request Body:**
```json
{
  "page_id": "page_123456"
}
```

**WebSocket Alternative:**
```json
{
  "type": "unassign_agent",
  "customer_id": "cust_123",
  "page_id": "page_123456"
}
```

**Response:**
```json
{
  "message": "Agent unassigned successfully",
  "customer": {
    "customer_id": "cust_123",
    "agent_name": "",
    "agent_id": "",
    "agent_email": "",
    "assigned_at": null
  }
}
```

### 4. Send Message to Customer (Agent Only)

When an agent is assigned, only they can send messages.

**Endpoint:** `POST /api/dashboard/customers/:customerID/message`

**Request Body:**
```json
{
  "page_id": "page_123456",
  "message": "Hello, I'm here to help with your order."
}
```

**Protection:** If another agent tries to send a message:
```json
{
  "error": "Customer is already assigned to john@company.com. Only the assigned agent can send messages."
}
```

## WebSocket Events

### Real-time Broadcasts

When agent assignment changes, all connected clients receive updates:

#### Agent Assigned Event
```json
{
  "type": "agent_status_changed",
  "data": {
    "customer_id": "cust_123",
    "customer": { /* full customer object */ },
    "action": "assigned",
    "agent_id": "user_456",
    "agent_email": "john@company.com",
    "timestamp": 1705315200
  }
}
```

#### Agent Unassigned Event
```json
{
  "type": "agent_status_changed",
  "data": {
    "customer_id": "cust_123",
    "customer": { /* full customer object */ },
    "action": "unassigned",
    "timestamp": 1705315800
  }
}
```

#### Agent Requested Event
When a customer requests a human agent:
```json
{
  "type": "agent_requested",
  "data": {
    "chat_id": "cust_123",
    "customer_name": "John Doe",
    "message": "I want to speak to a human",
    "timestamp": 1705315000
  }
}
```

## Business Rules

### Assignment Rules
1. **Exclusive Assignment**: Only one agent can be assigned to a customer at a time
2. **Message Protection**: Only the assigned agent can send messages to their assigned customers
3. **Admin Override**: Admins can unassign any agent from any customer
4. **Self-Unassign**: Agents can unassign themselves from customers they're assigned to

### Stop Status Management
- When `stop=true`: Bot won't respond to the customer
- When agent is assigned: `stop` is automatically set to `true`
- When agent is unassigned: `stop` status can be manually updated

## Usage Examples

### JavaScript/TypeScript Client

```javascript
class AgentManager {
  constructor(wsConnection, apiClient) {
    this.ws = wsConnection;
    this.api = apiClient;
  }

  // Get customers waiting for agents
  async getWaitingCustomers(pageId) {
    this.ws.send(JSON.stringify({
      type: 'get_stopped_customers',
      page_id: pageId,
      limit: 20,
      skip: 0
    }));
  }

  // Assign myself to a customer
  async assignToCustomer(customerId, pageId) {
    const response = await this.api.put(
      `/api/dashboard/customers/${customerId}/agent`,
      {
        page_id: pageId,
        agent_name: this.currentUser.name
      }
    );
    return response.data;
  }

  // Release a customer
  async releaseCustomer(customerId, pageId) {
    const response = await this.api.delete(
      `/api/dashboard/customers/${customerId}/agent`,
      {
        data: { page_id: pageId }
      }
    );
    return response.data;
  }

  // Send message to assigned customer
  async sendMessage(customerId, pageId, message) {
    try {
      const response = await this.api.post(
        `/api/dashboard/customers/${customerId}/message`,
        {
          page_id: pageId,
          message: message
        }
      );
      return response.data;
    } catch (error) {
      if (error.response?.status === 403) {
        alert('This customer is assigned to another agent');
      }
      throw error;
    }
  }

  // Listen for real-time updates
  listenForUpdates() {
    this.ws.on('message', (data) => {
      const msg = JSON.parse(data);
      
      switch(msg.type) {
        case 'stopped_customers':
          this.updateCustomerList(msg.data.customers);
          break;
          
        case 'agent_status_changed':
          this.updateCustomerStatus(msg.data);
          break;
          
        case 'agent_requested':
          this.showNotification(`${msg.data.customer_name} needs help!`);
          break;
      }
    });
  }
}
```

### Usage Flow

1. **Customer Requests Agent**
   - Customer sends message like "I want to speak to a human"
   - Bot detects intent and sets `stop=true`
   - WebSocket broadcasts `agent_requested` event

2. **Agent Takes Over**
   - Agent sees notification or checks waiting customers
   - Agent assigns themselves to the customer
   - System broadcasts `agent_status_changed` event
   - Other agents see customer is now taken

3. **Agent Handles Conversation**
   - Only assigned agent can send messages
   - All messages are tracked with agent info
   - Customer sees responses from human agent

4. **Agent Completes/Releases**
   - Agent unassigns themselves when done
   - Customer can be reassigned or bot can resume
   - System broadcasts unassignment to all agents

## Security Considerations

1. **Authentication Required**: All endpoints require valid session
2. **Company Isolation**: Agents can only access customers from their company's pages
3. **Assignment Protection**: Non-assigned agents cannot send messages to assigned customers
4. **Audit Trail**: All assignments/unassignments are logged with timestamps and agent info

## Error Handling

Common error responses:

| Status | Error | Cause |
|--------|-------|-------|
| 400 | "Page ID is required" | Missing page_id in request |
| 403 | "Customer is already assigned to..." | Trying to message another agent's customer |
| 403 | "Only the assigned agent or admin can unassign" | Unauthorized unassignment attempt |
| 404 | "Customer not found" | Invalid customer ID |
| 404 | "Page not found or access denied" | Page doesn't belong to company |

## Best Practices

1. **Real-time Monitoring**: Use WebSocket for live updates
2. **Regular Checks**: Poll for waiting customers periodically
3. **Clean Handoffs**: Always unassign when done
4. **Status Updates**: Keep customer informed during handoff
5. **Fallback Handling**: Have process for unclaimed customers

## Migration Notes

For existing systems:
1. Run database migration to add agent fields to customers collection
2. Update client applications to handle agent assignment events
3. Train agents on assignment workflow
4. Configure bot to detect agent requests properly