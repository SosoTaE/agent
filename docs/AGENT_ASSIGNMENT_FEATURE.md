# Agent Assignment & Customer Protection Feature

## Overview

The agent assignment feature ensures that when an agent takes over a customer who has requested human assistance, that customer is exclusively assigned to that agent. This prevents multiple agents from sending conflicting messages and ensures a consistent support experience.

## Core Functionality

### 1. Exclusive Agent Assignment
- Each customer can only be assigned to ONE agent at a time
- Assignment is tracked with agent ID, email, name, and timestamp
- Other agents cannot send messages to assigned customers
- Assignment persists until explicitly released

### 2. Assignment Protection
- System prevents other agents from messaging assigned customers
- Clear error messages when attempting to message another agent's customer
- Real-time updates show assignment status to all agents
- Visual indicators show who is handling each customer

## Database Fields

The following fields in the Customer model track agent assignment:

```javascript
{
  agent_id: "unique_agent_identifier",      // ID of assigned agent
  agent_email: "agent@example.com",         // Email of assigned agent  
  agent_name: "Agent Smith",                // Name of assigned agent
  assigned_at: "2024-01-01T10:05:00Z"      // Timestamp of assignment
}
```

## Assignment Methods

### Method 1: Explicit Assignment

Agent explicitly assigns themselves to a customer using WebSocket:

```json
{
  "type": "assign_agent",
  "data": {
    "customer_id": "customer_123",
    "page_id": "page_456"
  }
}
```

**Success Response:**
```json
{
  "type": "agent_assigned",
  "data": {
    "customer": { /* full customer object with agent info */ },
    "agent_id": "agent_123",
    "agent_email": "agent@example.com",
    "agent_name": "Agent Smith"
  }
}
```

### Method 2: Auto-Assignment on First Message

When an agent sends a message to an unassigned customer, they are automatically assigned:

```json
{
  "type": "send_message",
  "customer_id": "customer_123",
  "page_id": "page_456",
  "message": "Hello, I'm here to help!"
}
```

The system will:
1. Check if customer is assigned
2. If not assigned → assign current agent
3. If assigned to current agent → send message
4. If assigned to another agent → return error

## Protection Mechanisms

### 1. Message Sending Protection

When any agent attempts to send a message, the system checks:

```go
if customer.AgentID != "" && customer.AgentID != conn.UserID {
    // Customer is assigned to another agent
    sendWebSocketError(conn, fmt.Sprintf("Customer is already assigned to %s", customer.AgentEmail))
    return
}
```

### 2. Assignment Conflict Prevention

When attempting to assign an already-assigned customer:

```go
if customer.AgentID != "" && customer.AgentID != conn.UserID {
    sendWebSocketError(conn, fmt.Sprintf("Customer is already assigned to %s", customer.AgentEmail))
    return
}
```

### 3. Visual Status Indicators

The `get_stopped_customers` response includes assignment status:

```json
{
  "type": "stopped_customers",
  "data": {
    "customers": [{
      "customer_id": "123",
      "customer_name": "John Doe",
      "agent_id": "agent_456",
      "agent_email": "other.agent@example.com",
      "agent_name": "Other Agent",
      "assigned_at": "2024-01-01T10:05:00Z",
      "is_assigned": true,              // Someone is assigned
      "is_assigned_to_me": false,        // Not me
      "time_since_last_conversation": "5 minutes ago"
    }]
  }
}
```

## Workflow Examples

### Example 1: Successful Assignment and Messaging

**Step 1: Agent A assigns customer**
```json
// Agent A WebSocket
{
  "type": "assign_agent",
  "data": {
    "customer_id": "cust_123",
    "page_id": "page_456"
  }
}
```

**Step 2: Agent A sends message (SUCCESS)**
```json
// Agent A WebSocket
{
  "type": "send_message",
  "customer_id": "cust_123",
  "page_id": "page_456",
  "message": "Hello, how can I help?"
}
```
✅ Message sent successfully

**Step 3: Agent B tries to send message (BLOCKED)**
```json
// Agent B WebSocket
{
  "type": "send_message",
  "customer_id": "cust_123",
  "page_id": "page_456",
  "message": "Hi there!"
}
```
❌ Error: "Customer is already assigned to agentA@example.com"

### Example 2: Auto-Assignment on First Message

**Step 1: Customer requests help**
- Customer: "I want to speak to a human"
- System sets `stop: true`
- No agent assigned yet

**Step 2: Agent B sends first message**
```json
{
  "type": "send_message",
  "customer_id": "cust_789",
  "page_id": "page_456",
  "message": "Hello, I'll help you with that"
}
```
✅ Agent B automatically assigned
✅ Message sent

**Step 3: Agent C tries to message**
```json
{
  "type": "send_message",
  "customer_id": "cust_789",
  "page_id": "page_456",
  "message": "Can I help?"
}
```
❌ Error: "Customer is already assigned to agentB@example.com"

### Example 3: Release and Reassignment

**Step 1: Agent A releases customer**
```json
{
  "type": "unassign_agent",
  "data": {
    "customer_id": "cust_123",
    "page_id": "page_456"
  }
}
```

**Step 2: Broadcast to all agents**
```json
{
  "type": "agent_assignment_changed",
  "data": {
    "customer_id": "cust_123",
    "action": "unassigned"
  }
}
```

**Step 3: Agent B can now assign**
```json
{
  "type": "assign_agent",
  "data": {
    "customer_id": "cust_123",
    "page_id": "page_456"
  }
}
```
✅ Agent B successfully assigned

## Real-Time Broadcasts

All agents receive real-time updates about assignment changes:

### Assignment Broadcast
```json
{
  "type": "agent_assignment_changed",
  "data": {
    "customer_id": "cust_123",
    "customer": { /* full customer object */ },
    "agent_id": "agent_123",
    "agent_email": "agent@example.com",
    "agent_name": "Agent Smith",
    "action": "assigned",
    "timestamp": 1234567890
  }
}
```

### Unassignment Broadcast
```json
{
  "type": "agent_assignment_changed",
  "data": {
    "customer_id": "cust_123",
    "customer": { /* full customer object */ },
    "action": "unassigned",
    "timestamp": 1234567890
  }
}
```

## Error Scenarios

### Error 1: Attempting to Message Assigned Customer

**Scenario:** Agent B tries to message customer assigned to Agent A

**Request:**
```json
{
  "type": "send_message",
  "customer_id": "cust_123",
  "page_id": "page_456",
  "message": "Hello"
}
```

**Error Response:**
```json
{
  "type": "error",
  "error": "Customer is already assigned to agentA@example.com"
}
```

### Error 2: Attempting to Assign Already-Assigned Customer

**Scenario:** Agent C tries to assign customer already handled by Agent A

**Request:**
```json
{
  "type": "assign_agent",
  "data": {
    "customer_id": "cust_123",
    "page_id": "page_456"
  }
}
```

**Error Response:**
```json
{
  "type": "error",
  "error": "Customer is already assigned to agentA@example.com"
}
```

### Error 3: Attempting to Unassign Another Agent's Customer

**Scenario:** Agent B tries to unassign customer assigned to Agent A

**Request:**
```json
{
  "type": "unassign_agent",
  "data": {
    "customer_id": "cust_123",
    "page_id": "page_456"
  }
}
```

**Error Response:**
```json
{
  "type": "error",
  "error": "You are not assigned to this customer"
}
```

## UI Implementation Guidelines

### 1. Customer List Display

Show assignment status clearly:

```javascript
function renderCustomer(customer) {
  if (customer.is_assigned_to_me) {
    return `✅ ${customer.customer_name} (You are handling)`;
  } else if (customer.is_assigned) {
    return `🔒 ${customer.customer_name} (${customer.agent_name} is handling)`;
  } else {
    return `🟢 ${customer.customer_name} (Available)`;
  }
}
```

### 2. Action Button States

Disable actions based on assignment:

```javascript
function renderActions(customer) {
  if (customer.is_assigned_to_me) {
    return (
      <>
        <button onClick={() => sendMessage(customer)}>Send Message</button>
        <button onClick={() => releaseCustomer(customer)}>Release</button>
      </>
    );
  } else if (customer.is_assigned) {
    return <span>Handled by {customer.agent_name}</span>;
  } else {
    return <button onClick={() => assignCustomer(customer)}>Take Over</button>;
  }
}
```

### 3. Real-Time Status Updates

Update UI when receiving broadcasts:

```javascript
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  
  if (data.type === 'agent_assignment_changed') {
    const { customer_id, action, agent_name } = data.data;
    
    if (action === 'assigned') {
      showNotification(`${agent_name} is now handling customer ${customer_id}`);
      updateCustomerStatus(customer_id, 'assigned', agent_name);
    } else {
      showNotification(`Customer ${customer_id} is now available`);
      updateCustomerStatus(customer_id, 'available');
    }
  }
};
```

## Best Practices

### 1. Always Check Assignment Status
Before attempting any action, verify assignment status:

```javascript
async function handleCustomerAction(customer) {
  // Refresh customer data first
  const latest = await getCustomerStatus(customer.customer_id);
  
  if (latest.is_assigned && !latest.is_assigned_to_me) {
    alert(`This customer is being handled by ${latest.agent_name}`);
    return;
  }
  
  // Proceed with action
}
```

### 2. Handle Conflicts Gracefully
When assignment fails, provide clear feedback:

```javascript
function handleAssignmentError(error) {
  if (error.includes('already assigned')) {
    // Refresh customer list
    loadStoppedCustomers();
    // Show non-intrusive notification
    showToast('Customer is already being handled by another agent', 'info');
  }
}
```

### 3. Release Customers When Done
Always unassign when finished helping:

```javascript
async function completeCustomerSession(customer) {
  // Send final message
  await sendMessage(customer.customer_id, 'Is there anything else I can help with?');
  
  // Wait for response or timeout
  setTimeout(() => {
    // Release the customer
    unassignCustomer(customer.customer_id, customer.page_id);
  }, 30000); // 30 seconds
}
```

### 4. Implement Visual Feedback
Show assignment status prominently:

```css
.customer-card.assigned-to-me {
  border: 2px solid #4CAF50;
  background: #E8F5E9;
}

.customer-card.assigned-to-other {
  opacity: 0.6;
  border: 2px solid #FFC107;
  background: #FFF3E0;
}

.customer-card.available {
  border: 2px solid #2196F3;
  background: #E3F2FD;
}
```

## Security Considerations

### 1. Server-Side Validation
All assignment checks happen server-side:
- Cannot bypass by modifying client code
- Agent ID verified from session, not request
- Database is single source of truth

### 2. Audit Trail
All assignments are logged:
- Agent ID, email, and name saved
- Assignment timestamp recorded
- Can track who handled each customer

### 3. Company Isolation
Agents can only see/assign their company's customers:
- Company ID verified on every request
- Page ownership validated
- Cross-company assignment impossible

## Monitoring & Analytics

### Track Assignment Metrics

```sql
-- Average time to assignment
SELECT AVG(TIMESTAMPDIFF(MINUTE, stopped_at, assigned_at)) as avg_wait_time
FROM customers
WHERE stop = true AND assigned_at IS NOT NULL;

-- Agent workload distribution
SELECT agent_name, COUNT(*) as customers_handled
FROM customers
WHERE agent_id IS NOT NULL
GROUP BY agent_name;

-- Unhandled customers
SELECT COUNT(*) as waiting_customers
FROM customers
WHERE stop = true AND agent_id IS NULL;
```

### Key Performance Indicators

1. **Response Time**: Time from `stopped_at` to `assigned_at`
2. **Handle Time**: Time from `assigned_at` to unassignment
3. **Agent Utilization**: Number of concurrent assignments per agent
4. **Abandonment Rate**: Customers who leave before assignment

## Troubleshooting

### Issue: "Already assigned" but customer appears available

**Cause:** UI not updated with latest status

**Solution:** 
1. Refresh customer list
2. Check WebSocket connection
3. Verify broadcast handling

### Issue: Cannot send message to my assigned customer

**Cause:** Assignment may have been released

**Solution:**
1. Check current assignment status
2. Re-assign if necessary
3. Verify WebSocket session is same

### Issue: Auto-assignment not working

**Cause:** Customer may not have `stop: true`

**Solution:**
1. Verify customer requested human help
2. Check `stop` flag is set
3. Ensure no existing assignment

## Future Enhancements

### Planned Features
- **Assignment History**: Track all agents who handled a customer
- **Transfer Customer**: Allow transferring customer between agents
- **Assignment Expiry**: Auto-release after inactivity period
- **Queue Management**: Automatic assignment based on availability
- **Supervisor Override**: Allow supervisors to reassign customers
- **Assignment Limits**: Maximum concurrent customers per agent

### Potential Improvements
- Add assignment reason/notes
- Track assignment duration metrics
- Implement assignment priorities
- Add customer wait time warnings
- Create assignment notifications
- Build assignment reports dashboard