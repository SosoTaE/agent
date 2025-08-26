# Customer Management API Documentation

## Overview

This document describes the customer management features including agent assignment, stop status tracking, and real-time WebSocket updates for handling customers who request human assistance.

## Customer Model

The customer model tracks Facebook users who interact with your pages and includes agent assignment information.

### Customer Fields

| Field | Type | Description |
|-------|------|-------------|
| `customer_id` | string | Facebook user ID |
| `customer_name` | string | Full name of the customer |
| `first_name` | string | Customer's first name |
| `last_name` | string | Customer's last name |
| `page_id` | string | Facebook page ID they're messaging |
| `page_name` | string | Name of the Facebook page |
| `company_id` | string | Company that owns the page |
| `message_count` | int | Total messages sent by customer |
| `last_message` | string | Last message text from customer |
| `last_seen` | timestamp | Last interaction time |
| `first_seen` | timestamp | First interaction time |
| `stop` | boolean | Whether customer wants to talk to real person |
| `stopped_at` | timestamp | When customer requested real person |
| `agent_name` | string | Name of the assigned agent |
| `agent_id` | string | ID of the assigned agent |
| `agent_email` | string | Email of the assigned agent |
| `assigned_at` | timestamp | When agent was assigned |
| `created_at` | timestamp | Customer record creation time |
| `updated_at` | timestamp | Last update time (used to track time since last conversation) |

## REST API Endpoints

### Get Customers
```
GET /api/dashboard/customers
```

Query Parameters:
- `page_id` (optional): Filter by specific page
- `page` (default: 1): Page number for pagination
- `limit` (default: 50, max: 200): Items per page

Response includes `agent_name`, `agent_id`, `agent_email`, and `updated_at` fields.

### Update Agent Name
```
PUT /api/dashboard/customers/:customerID/agent
```

Request Body:
```json
{
  "page_id": "page_id",
  "agent_name": "Agent Name"
}
```

Response:
```json
{
  "message": "Agent name updated successfully",
  "customer": { /* full customer object */ }
}
```

### Update Customer Stop Status
```
PUT /api/dashboard/customers/:customerID/stop
```

Request Body:
```json
{
  "page_id": "page_id",
  "stop": true
}
```

Response:
```json
{
  "message": "Customer status updated successfully",
  "customer": { /* full customer object */ }
}
```

### Toggle Customer Stop Status
```
POST /api/dashboard/customers/:customerID/toggle-stop
```

Request Body:
```json
{
  "page_id": "page_id"
}
```

Response:
```json
{
  "message": "Customer status toggled successfully",
  "customer": { /* full customer object */ },
  "previous_stop": false,
  "current_stop": true
}
```

## WebSocket API

The WebSocket endpoint is available at `/api/dashboard/ws` and requires authentication.

### Connection

Upon successful connection, you'll receive:
```json
{
  "type": "connected",
  "message": "WebSocket connection established",
  "user_id": "unique_user_id"
}
```

### Message Types

#### 1. Get Stopped Customers

Request customers who want to talk to a real person.

**Request:**
```json
{
  "type": "get_stopped_customers",
  "data": {
    "page_id": "optional_page_id",
    "limit": 50,
    "skip": 0
  }
}
```

**Response:**
```json
{
  "type": "stopped_customers",
  "data": {
    "customers": [
      {
        "customer_id": "123456",
        "customer_name": "John Doe",
        "page_id": "page_123",
        "page_name": "My Business Page",
        "stop": true,
        "stopped_at": "2024-01-01T10:00:00Z",
        "agent_id": "agent_123",
        "agent_email": "agent@example.com",
        "agent_name": "Agent Smith",
        "assigned_at": "2024-01-01T10:05:00Z",
        "updated_at": "2024-01-01T10:30:00Z",
        "time_since_last_conversation": "5 minutes ago",
        "minutes_since_last_conversation": 5,
        "is_assigned": true,
        "is_assigned_to_me": false
      }
    ],
    "pagination": {
      "total": 10,
      "limit": 50,
      "skip": 0,
      "has_more": false
    },
    "page_id": "page_123"
  },
  "timestamp": 1234567890
}
```

**Additional Fields in Response:**
- `time_since_last_conversation`: Human-readable time since `updated_at`
- `minutes_since_last_conversation`: Numeric minutes since `updated_at`
- `is_assigned`: Boolean indicating if any agent is assigned
- `is_assigned_to_me`: Boolean indicating if current user is assigned

#### 2. Assign Agent to Customer

Assign yourself to handle a customer who requested human assistance.

**Request:**
```json
{
  "type": "assign_agent",
  "data": {
    "customer_id": "customer_123",
    "page_id": "page_123"
  }
}
```

**Success Response:**
```json
{
  "type": "agent_assigned",
  "data": {
    "customer": { /* full customer object */ },
    "agent_id": "agent_123",
    "agent_email": "agent@example.com",
    "agent_name": "Agent Smith"
  },
  "timestamp": 1234567890
}
```

**Error Cases:**
- Customer not found
- Customer has not requested human assistance (`stop: false`)
- Customer already assigned to another agent
- Page access denied

#### 3. Unassign Agent from Customer

Release a customer assignment.

**Request:**
```json
{
  "type": "unassign_agent",
  "data": {
    "customer_id": "customer_123",
    "page_id": "page_123"
  }
}
```

**Success Response:**
```json
{
  "type": "agent_unassigned",
  "data": {
    "customer": { /* full customer object */ }
  },
  "timestamp": 1234567890
}
```

**Error Cases:**
- Customer not found
- You are not assigned to this customer
- Page access denied

#### 4. Send Message to Customer

Send a message to a customer via Facebook Messenger.

**Request:**
```json
{
  "type": "send_message",
  "customer_id": "customer_123",
  "page_id": "page_123",
  "message": "Hello, how can I help you today?"
}
```

**Success Response:**
```json
{
  "type": "message_sent",
  "customer_id": "customer_123",
  "page_id": "page_123",
  "message": "Hello, how can I help you today?",
  "timestamp": 1234567890
}
```

**Note:** If the customer is not assigned to any agent, sending a message will automatically assign you to that customer.

### Real-Time Broadcast Events

All connected WebSocket clients in the same company receive these broadcast events:

#### 1. Customer Stop Status Changed

Broadcast when a customer's stop status changes (requests or cancels human assistance).

```json
{
  "type": "customer_stop_status_changed",
  "data": {
    "customer": { /* full customer object */ },
    "stop": true,
    "timestamp": 1234567890
  }
}
```

#### 2. Agent Assignment Changed

Broadcast when an agent is assigned or unassigned to/from a customer.

```json
{
  "type": "agent_assignment_changed",
  "data": {
    "customer_id": "customer_123",
    "customer": { /* full customer object */ },
    "agent_id": "agent_123",
    "agent_email": "agent@example.com",
    "agent_name": "Agent Smith",
    "action": "assigned",  // or "unassigned"
    "timestamp": 1234567890
  }
}
```

#### 3. Agent Requested

Broadcast when a customer requests to speak with a real person.

```json
{
  "type": "agent_requested",
  "data": {
    "chat_id": "customer_123",
    "customer_name": "John Doe",
    "message": "I want to speak to a human",
    "timestamp": 1234567890
  }
}
```

#### 4. New Message

Broadcast when messages are sent/received.

```json
{
  "type": "new_message",
  "data": {
    "chat_id": "customer_123",
    "sender_id": "page_123",
    "recipient_id": "customer_123",
    "message": "Message text",
    "is_bot": false,
    "is_human": true,
    "agent_id": "agent_123",
    "agent_email": "agent@example.com",
    "agent_name": "Agent Smith",
    "timestamp": 1234567890
  }
}
```

### Error Responses

WebSocket errors follow this format:

```json
{
  "type": "error",
  "error": "Error message description"
}
```

## Workflow Examples

### 1. Customer Requests Human Assistance

1. Customer sends a message indicating they want human help
2. System sets `stop: true` and `stopped_at` timestamp
3. WebSocket broadcasts `agent_requested` and `customer_stop_status_changed` events
4. All connected agents see the customer in their stopped customers list

### 2. Agent Takes Over Customer

1. Agent sends `assign_agent` WebSocket message
2. System updates customer with agent details and `assigned_at` timestamp
3. WebSocket broadcasts `agent_assignment_changed` event
4. Other agents see the customer is now assigned
5. Agent can now send messages to the customer

### 3. Auto-Assignment on Message

1. Agent sends message to unassigned customer via `send_message`
2. System automatically assigns the agent to the customer
3. WebSocket broadcasts `agent_assignment_changed` event
4. Message is sent to customer via Facebook Messenger
5. Other agents see the customer is now assigned

### 4. Agent Releases Customer

1. Agent sends `unassign_agent` WebSocket message
2. System removes agent assignment fields
3. WebSocket broadcasts `agent_assignment_changed` event with `action: "unassigned"`
4. Customer becomes available for other agents

## Best Practices

### Agent Assignment

1. **Check Assignment Status**: Always check if a customer is already assigned before attempting to take over
2. **Auto-Assignment**: Sending a message automatically assigns you to the customer
3. **Release When Done**: Unassign yourself when you're finished helping a customer
4. **Monitor Broadcasts**: Listen for `agent_assignment_changed` events to update your UI

### Time Tracking

1. **Updated At**: The `updated_at` field tracks the last conversation time
2. **Time Since**: Use `time_since_last_conversation` for display
3. **Priority**: Sort by `minutes_since_last_conversation` to prioritize waiting customers

### Error Handling

1. **Conflict Detection**: Handle "already assigned" errors gracefully
2. **Permission Checks**: Ensure page ownership before operations
3. **Connection Recovery**: Implement reconnection logic for WebSocket disconnections

## Implementation Notes

### Database Indexes

The following indexes are created for optimal performance:
- Compound index on `(customer_id, page_id)` for unique customers per page
- Index on `company_id` for company-wide queries
- Index on `last_seen` for sorting
- Text index on name fields for searching

### Security

- All operations require authentication
- Page ownership is validated for all operations
- Agent information is tracked for audit purposes
- WebSocket connections are authenticated via session

### Performance Considerations

- Pagination is implemented with default limit of 50
- Maximum limit is 200 items per request
- WebSocket broadcasts are scoped to company level
- Real-time updates use efficient broadcast mechanisms

## Error Codes

| Error Message | Description | Resolution |
|--------------|-------------|------------|
| "Customer not found" | Customer ID doesn't exist | Verify customer ID and page ID |
| "Customer has not requested human assistance" | Attempting to message customer with `stop: false` | Wait for customer to request help |
| "Customer is already assigned to [email]" | Another agent has taken the customer | Choose a different customer |
| "You are not assigned to this customer" | Attempting to unassign when not assigned | Only assigned agent can unassign |
| "Page not found or access denied" | Invalid page ID or no permission | Verify page ownership |
| "Failed to assign agent" | Database error during assignment | Retry the operation |

## Migration Guide

If upgrading from a previous version:

1. **Database Migration**: New fields added to customer collection:
   - `agent_name` (string)
   - `agent_id` (string) 
   - `agent_email` (string)
   - `assigned_at` (timestamp)

2. **WebSocket Updates**: Implement handlers for new message types:
   - `get_stopped_customers`
   - `assign_agent`
   - `unassign_agent`
   - `agent_assignment_changed` (broadcast)
   - `customer_stop_status_changed` (broadcast)

3. **UI Updates**: Update customer displays to show:
   - Assignment status indicators
   - Time since last conversation
   - Agent information when assigned

## Support

For issues or questions about the customer management API:
1. Check error messages and this documentation
2. Review WebSocket connection logs
3. Ensure proper authentication and permissions
4. Contact system administrator for database issues