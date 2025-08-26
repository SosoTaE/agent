# IsAssigned Field Management

## Overview

The `IsAssigned` field provides a clear boolean indicator of whether a customer is currently assigned to an agent. This field simplifies queries and provides better control over customer assignment status.

## Data Model

### Customer Model - IsAssigned Field

```go
type Customer struct {
    // ... other fields ...
    
    IsAssigned   bool       `bson:"is_assigned" json:"is_assigned"`       // Whether an agent is currently assigned
    AgentName    string     `bson:"agent_name,omitempty"`                 // Name of assigned agent
    AgentID      string     `bson:"agent_id,omitempty"`                   // User ID of assigned agent
    AgentEmail   string     `bson:"agent_email,omitempty"`                // Email of assigned agent
    AssignedAt   *time.Time `bson:"assigned_at,omitempty"`                // When agent was assigned
    
    // ... other fields ...
}
```

## Field Behavior

### Automatic Updates

The `IsAssigned` field is automatically managed:

1. **When agent is assigned**: `IsAssigned` is set to `true`
2. **When agent is unassigned**: `IsAssigned` is set to `false`
3. **When using assignment status endpoint**: Can be directly controlled

### Relationship with Agent Fields

- When `IsAssigned = true`: Agent fields (AgentID, AgentName, AgentEmail, AssignedAt) should be populated
- When `IsAssigned = false`: Agent fields are cleared/unset

## API Endpoints

### 1. Update Assignment Status

Direct control over the `IsAssigned` field.

**Endpoint:** `PUT /api/dashboard/customers/:customerID/assignment`

**Request Body:**
```json
{
  "page_id": "page_123456",
  "is_assigned": true
}
```

**Response:**
```json
{
  "message": "Assignment status updated successfully",
  "customer": {
    "customer_id": "cust_123",
    "customer_name": "John Doe",
    "is_assigned": true,
    "agent_name": "Agent Smith",
    "agent_id": "agent_456",
    "agent_email": "agent@company.com",
    "assigned_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

**Important Notes:**
- Setting `is_assigned: false` will automatically clear all agent fields
- Setting `is_assigned: true` only sets the flag (agent fields need separate assignment)

### 2. Assign Agent (Sets IsAssigned=true)

**Endpoint:** `PUT /api/dashboard/customers/:customerID/agent`

**Request Body:**
```json
{
  "page_id": "page_123456",
  "agent_name": "Agent Smith"
}
```

**Effect:** Automatically sets `IsAssigned = true` along with agent details

### 3. Unassign Agent (Sets IsAssigned=false)

**Endpoint:** `DELETE /api/dashboard/customers/:customerID/agent`

**Request Body:**
```json
{
  "page_id": "page_123456"
}
```

**Effect:** Sets `IsAssigned = false` and clears all agent fields

## WebSocket Events

When `IsAssigned` status changes, a real-time event is broadcast:

```json
{
  "type": "customer_assignment_updated",
  "data": {
    "customer_id": "cust_123",
    "customer": { /* full customer object */ },
    "is_assigned": true,
    "timestamp": 1705315200
  }
}
```

## Query Examples

### MongoDB Queries

```javascript
// Find all assigned customers
db.customers.find({ "is_assigned": true })

// Find unassigned customers who need attention
db.customers.find({ 
  "is_assigned": false,
  "stop": true 
})

// Find customers assigned to a specific agent
db.customers.find({ 
  "is_assigned": true,
  "agent_id": "agent_456" 
})

// Count assignment statistics
db.customers.aggregate([
  {
    $group: {
      _id: "$is_assigned",
      count: { $sum: 1 }
    }
  }
])
```

### API Query Examples

```javascript
// Using the field in filters (when implemented)
GET /api/dashboard/customers?is_assigned=true&page_id=page_123

// Get statistics
const assigned = await db.customers.countDocuments({ is_assigned: true });
const unassigned = await db.customers.countDocuments({ is_assigned: false });
```

## Usage in Code

### JavaScript/TypeScript Client

```javascript
class CustomerManager {
  // Check if customer is assigned
  isCustomerAssigned(customer) {
    return customer.is_assigned === true;
  }

  // Update assignment status only
  async updateAssignmentStatus(customerId, pageId, isAssigned) {
    const response = await fetch(
      `/api/dashboard/customers/${customerId}/assignment`,
      {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          page_id: pageId,
          is_assigned: isAssigned
        })
      }
    );
    return response.json();
  }

  // Quick release without full unassignment
  async quickRelease(customerId, pageId) {
    return this.updateAssignmentStatus(customerId, pageId, false);
  }

  // Mark as assigned (preparation for agent assignment)
  async markAsAssigned(customerId, pageId) {
    return this.updateAssignmentStatus(customerId, pageId, true);
  }

  // Filter customers by assignment status
  filterByAssignment(customers, assignedOnly = true) {
    return customers.filter(c => c.is_assigned === assignedOnly);
  }
}
```

### React Component Example

```jsx
function CustomerList({ customers }) {
  const assignedCustomers = customers.filter(c => c.is_assigned);
  const unassignedCustomers = customers.filter(c => !c.is_assigned);

  return (
    <div>
      <div className="assigned-section">
        <h3>Assigned Customers ({assignedCustomers.length})</h3>
        {assignedCustomers.map(customer => (
          <CustomerCard 
            key={customer.customer_id}
            customer={customer}
            showAgent={true}
          />
        ))}
      </div>

      <div className="unassigned-section">
        <h3>Available Customers ({unassignedCustomers.length})</h3>
        {unassignedCustomers.map(customer => (
          <CustomerCard 
            key={customer.customer_id}
            customer={customer}
            showAssignButton={true}
          />
        ))}
      </div>
    </div>
  );
}
```

## Use Cases

### 1. Quick Status Checks
```javascript
// Instead of checking multiple fields
if (customer.agent_id && customer.agent_id !== "") {
  // Customer is assigned
}

// Simply check
if (customer.is_assigned) {
  // Customer is assigned
}
```

### 2. Bulk Operations
```javascript
// Release all assigned customers for a page
async function releaseAllCustomers(pageId) {
  const assigned = await getCustomers({ 
    page_id: pageId, 
    is_assigned: true 
  });
  
  for (const customer of assigned) {
    await updateAssignmentStatus(customer.customer_id, pageId, false);
  }
}
```

### 3. Dashboard Statistics
```javascript
// Quick assignment stats
function getAssignmentStats(customers) {
  return {
    total: customers.length,
    assigned: customers.filter(c => c.is_assigned).length,
    unassigned: customers.filter(c => !c.is_assigned).length,
    assignmentRate: (customers.filter(c => c.is_assigned).length / customers.length) * 100
  };
}
```

### 4. Agent Workload Balancing
```javascript
// Find agents with capacity
async function getAvailableAgents() {
  const agents = await getAgents();
  const assignments = await db.customers.aggregate([
    { $match: { is_assigned: true } },
    { $group: { _id: "$agent_id", count: { $sum: 1 } } }
  ]);

  return agents.map(agent => ({
    ...agent,
    assignedCount: assignments.find(a => a._id === agent.id)?.count || 0,
    hasCapacity: (assignments.find(a => a._id === agent.id)?.count || 0) < agent.maxCustomers
  }));
}
```

## Migration for Existing Data

For existing databases without the `IsAssigned` field:

```javascript
// Set is_assigned based on existing agent_id field
// Run this once to migrate existing data

// Set true for customers with agents
db.customers.updateMany(
  { agent_id: { $exists: true, $ne: "" } },
  { $set: { is_assigned: true } }
);

// Set false for customers without agents
db.customers.updateMany(
  { $or: [
    { agent_id: { $exists: false } },
    { agent_id: "" }
  ]},
  { $set: { is_assigned: false } }
);
```

## Benefits

1. **Clearer Queries**: Simple boolean check instead of complex field validation
2. **Better Performance**: Can index on `is_assigned` for faster queries
3. **Explicit State**: No ambiguity about assignment status
4. **Easier Filtering**: Simplifies UI filtering and sorting
5. **Statistics**: Quick counts and aggregations

## Security Considerations

- Only authenticated agents can update assignment status
- Company isolation ensures agents can only modify their company's customers
- Audit logs track all assignment status changes
- WebSocket broadcasts ensure all connected clients stay synchronized

## Best Practices

1. **Always use the field**: Rely on `is_assigned` rather than checking agent fields
2. **Keep synchronized**: Ensure `is_assigned` matches actual agent assignment
3. **Index the field**: Create database index for better query performance
4. **Handle transitions**: Update UI immediately when assignment status changes
5. **Validate consistency**: Periodically check that `is_assigned` matches agent field presence