# WebSocket Quick Start Guide

## Connection Setup

### 1. Connect to WebSocket

```javascript
// JavaScript/TypeScript Example
const ws = new WebSocket('wss://your-domain.com/api/dashboard/ws');

ws.onopen = () => {
  console.log('Connected to WebSocket');
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  handleWebSocketMessage(data);
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('WebSocket connection closed');
  // Implement reconnection logic here
};
```

### 2. Authentication

The WebSocket endpoint requires authentication via session cookies. Ensure you're logged in before connecting.

## Common Use Cases

### Getting Customers Who Need Help

```javascript
// Request stopped customers
function getStoppedCustomers(pageId = null) {
  const message = {
    type: 'get_stopped_customers',
    data: {
      page_id: pageId,  // Optional: filter by page
      limit: 50,
      skip: 0
    }
  };
  ws.send(JSON.stringify(message));
}

// Handle response
function handleWebSocketMessage(data) {
  switch(data.type) {
    case 'stopped_customers':
      displayCustomers(data.data.customers);
      break;
  }
}

function displayCustomers(customers) {
  customers.forEach(customer => {
    console.log(`
      Customer: ${customer.customer_name}
      Waiting: ${customer.time_since_last_conversation}
      Assigned: ${customer.is_assigned ? `Yes (${customer.agent_name})` : 'No'}
      Assigned to me: ${customer.is_assigned_to_me}
    `);
  });
}
```

### Taking Over a Customer

```javascript
// Assign yourself to a customer
function assignCustomer(customerId, pageId) {
  const message = {
    type: 'assign_agent',
    data: {
      customer_id: customerId,
      page_id: pageId
    }
  };
  ws.send(JSON.stringify(message));
}

// Handle assignment response
function handleWebSocketMessage(data) {
  switch(data.type) {
    case 'agent_assigned':
      console.log('Successfully assigned to customer');
      updateCustomerUI(data.data.customer);
      break;
    
    case 'error':
      if (data.error.includes('already assigned')) {
        alert('Customer is already being handled by another agent');
      }
      break;
  }
}
```

### Sending a Message to Customer

```javascript
// Send message (auto-assigns if not assigned)
function sendMessage(customerId, pageId, messageText) {
  const message = {
    type: 'send_message',
    customer_id: customerId,
    page_id: pageId,
    message: messageText
  };
  ws.send(JSON.stringify(message));
}

// Handle message sent confirmation
function handleWebSocketMessage(data) {
  switch(data.type) {
    case 'message_sent':
      console.log('Message sent successfully');
      addMessageToChat(data);
      break;
  }
}
```

### Releasing a Customer

```javascript
// Unassign yourself from a customer
function releaseCustomer(customerId, pageId) {
  const message = {
    type: 'unassign_agent',
    data: {
      customer_id: customerId,
      page_id: pageId
    }
  };
  ws.send(JSON.stringify(message));
}
```

### Listening for Real-Time Updates

```javascript
function handleWebSocketMessage(data) {
  switch(data.type) {
    // Customer requests human help
    case 'agent_requested':
      showNotification(`New help request from ${data.data.customer_name}`);
      refreshCustomerList();
      break;
    
    // Agent assignment changes
    case 'agent_assignment_changed':
      if (data.data.action === 'assigned') {
        updateCustomerStatus(data.data.customer_id, 'assigned', data.data.agent_name);
      } else {
        updateCustomerStatus(data.data.customer_id, 'available');
      }
      break;
    
    // Customer stop status changes
    case 'customer_stop_status_changed':
      if (data.data.stop) {
        addToWaitingList(data.data.customer);
      } else {
        removeFromWaitingList(data.data.customer.customer_id);
      }
      break;
    
    // New messages in conversations
    case 'new_message':
      if (data.data.is_human) {
        addAgentMessage(data.data);
      } else {
        addCustomerMessage(data.data);
      }
      break;
  }
}
```

## Complete Example: Customer Support Dashboard

```javascript
class CustomerSupportDashboard {
  constructor() {
    this.ws = null;
    this.customers = new Map();
    this.myUserId = null;
    this.connect();
  }

  connect() {
    this.ws = new WebSocket('wss://your-domain.com/api/dashboard/ws');
    
    this.ws.onopen = () => {
      console.log('Connected to support system');
      this.loadStoppedCustomers();
    };
    
    this.ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      this.handleMessage(data);
    };
    
    this.ws.onclose = () => {
      console.log('Disconnected - reconnecting in 3 seconds...');
      setTimeout(() => this.connect(), 3000);
    };
  }

  handleMessage(data) {
    switch(data.type) {
      case 'connected':
        this.myUserId = data.user_id;
        break;
      
      case 'stopped_customers':
        this.updateCustomerList(data.data.customers);
        break;
      
      case 'agent_assigned':
        this.onAssignmentSuccess(data.data);
        break;
      
      case 'agent_assignment_changed':
        this.onAssignmentChanged(data.data);
        break;
      
      case 'agent_requested':
        this.onNewHelpRequest(data.data);
        break;
      
      case 'error':
        this.handleError(data.error);
        break;
    }
  }

  loadStoppedCustomers() {
    this.send({
      type: 'get_stopped_customers',
      data: { limit: 100, skip: 0 }
    });
  }

  assignToMe(customerId, pageId) {
    this.send({
      type: 'assign_agent',
      data: { customer_id: customerId, page_id: pageId }
    });
  }

  sendMessage(customerId, pageId, message) {
    this.send({
      type: 'send_message',
      customer_id: customerId,
      page_id: pageId,
      message: message
    });
  }

  release(customerId, pageId) {
    this.send({
      type: 'unassign_agent',
      data: { customer_id: customerId, page_id: pageId }
    });
  }

  send(data) {
    if (this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data));
    } else {
      console.error('WebSocket not connected');
    }
  }

  updateCustomerList(customers) {
    this.customers.clear();
    customers.forEach(customer => {
      this.customers.set(customer.customer_id, customer);
      this.renderCustomer(customer);
    });
  }

  renderCustomer(customer) {
    const status = customer.is_assigned 
      ? (customer.is_assigned_to_me ? 'Mine' : `Assigned to ${customer.agent_name}`)
      : 'Available';
    
    console.log(`
      [${status}] ${customer.customer_name}
      Waiting: ${customer.time_since_last_conversation}
      Last message: ${customer.last_message}
    `);
  }

  onAssignmentSuccess(data) {
    console.log(`You are now handling ${data.customer.customer_name}`);
    this.customers.set(data.customer.customer_id, data.customer);
  }

  onAssignmentChanged(data) {
    const customer = this.customers.get(data.customer_id);
    if (customer) {
      Object.assign(customer, data.customer);
      this.renderCustomer(customer);
    }
  }

  onNewHelpRequest(data) {
    console.log(`🔔 New help request from ${data.customer_name}`);
    this.loadStoppedCustomers(); // Refresh list
  }

  handleError(error) {
    console.error('Error:', error);
    if (error.includes('already assigned')) {
      alert('This customer is already being handled by another agent');
    }
  }
}

// Initialize dashboard
const dashboard = new CustomerSupportDashboard();

// Example usage
dashboard.assignToMe('customer_123', 'page_456');
dashboard.sendMessage('customer_123', 'page_456', 'Hello, how can I help you?');
dashboard.release('customer_123', 'page_456');
```

## React Hook Example

```jsx
import { useEffect, useState, useCallback, useRef } from 'react';

function useCustomerSupport() {
  const [customers, setCustomers] = useState([]);
  const [connected, setConnected] = useState(false);
  const ws = useRef(null);

  useEffect(() => {
    connect();
    return () => {
      if (ws.current) {
        ws.current.close();
      }
    };
  }, []);

  const connect = () => {
    ws.current = new WebSocket('wss://your-domain.com/api/dashboard/ws');
    
    ws.current.onopen = () => {
      setConnected(true);
      loadCustomers();
    };
    
    ws.current.onmessage = (event) => {
      const data = JSON.parse(event.data);
      handleMessage(data);
    };
    
    ws.current.onclose = () => {
      setConnected(false);
      setTimeout(connect, 3000);
    };
  };

  const handleMessage = (data) => {
    switch(data.type) {
      case 'stopped_customers':
        setCustomers(data.data.customers);
        break;
      
      case 'agent_assignment_changed':
        setCustomers(prev => prev.map(c => 
          c.customer_id === data.data.customer_id 
            ? data.data.customer 
            : c
        ));
        break;
      
      case 'agent_requested':
        loadCustomers(); // Refresh list
        break;
    }
  };

  const loadCustomers = useCallback(() => {
    send({ type: 'get_stopped_customers', data: { limit: 100 } });
  }, []);

  const assignCustomer = useCallback((customerId, pageId) => {
    send({ type: 'assign_agent', data: { customer_id: customerId, page_id: pageId } });
  }, []);

  const sendMessage = useCallback((customerId, pageId, message) => {
    send({ type: 'send_message', customer_id: customerId, page_id: pageId, message });
  }, []);

  const releaseCustomer = useCallback((customerId, pageId) => {
    send({ type: 'unassign_agent', data: { customer_id: customerId, page_id: pageId } });
  }, []);

  const send = (data) => {
    if (ws.current?.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify(data));
    }
  };

  return {
    customers,
    connected,
    assignCustomer,
    sendMessage,
    releaseCustomer,
    refreshCustomers: loadCustomers
  };
}

// Usage in component
function CustomerList() {
  const { customers, assignCustomer, connected } = useCustomerSupport();

  if (!connected) {
    return <div>Connecting...</div>;
  }

  return (
    <div>
      {customers.map(customer => (
        <div key={customer.customer_id}>
          <h3>{customer.customer_name}</h3>
          <p>Waiting: {customer.time_since_last_conversation}</p>
          {!customer.is_assigned && (
            <button onClick={() => assignCustomer(customer.customer_id, customer.page_id)}>
              Take Over
            </button>
          )}
          {customer.is_assigned_to_me && (
            <span>You are handling this customer</span>
          )}
          {customer.is_assigned && !customer.is_assigned_to_me && (
            <span>Handled by {customer.agent_name}</span>
          )}
        </div>
      ))}
    </div>
  );
}
```

## Testing WebSocket Connection

Use this simple HTML file to test the WebSocket connection:

```html
<!DOCTYPE html>
<html>
<head>
    <title>WebSocket Test</title>
</head>
<body>
    <h2>WebSocket Customer Support Test</h2>
    <button onclick="connect()">Connect</button>
    <button onclick="getCustomers()">Get Stopped Customers</button>
    <button onclick="disconnect()">Disconnect</button>
    
    <div id="status">Disconnected</div>
    <div id="customers"></div>
    <pre id="log"></pre>

    <script>
        let ws = null;

        function connect() {
            ws = new WebSocket('wss://your-domain.com/api/dashboard/ws');
            
            ws.onopen = () => {
                document.getElementById('status').innerText = 'Connected';
                log('Connected to WebSocket');
            };
            
            ws.onmessage = (event) => {
                const data = JSON.parse(event.data);
                log('Received: ' + JSON.stringify(data, null, 2));
                
                if (data.type === 'stopped_customers') {
                    displayCustomers(data.data.customers);
                }
            };
            
            ws.onerror = (error) => {
                log('Error: ' + error);
            };
            
            ws.onclose = () => {
                document.getElementById('status').innerText = 'Disconnected';
                log('Connection closed');
            };
        }

        function getCustomers() {
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({
                    type: 'get_stopped_customers',
                    data: { limit: 50, skip: 0 }
                }));
            } else {
                alert('Not connected');
            }
        }

        function displayCustomers(customers) {
            const div = document.getElementById('customers');
            div.innerHTML = '<h3>Customers Needing Help:</h3>';
            customers.forEach(c => {
                div.innerHTML += `
                    <div style="border: 1px solid #ccc; padding: 10px; margin: 5px;">
                        <strong>${c.customer_name}</strong><br>
                        Waiting: ${c.time_since_last_conversation}<br>
                        Assigned: ${c.is_assigned ? c.agent_name : 'No'}<br>
                        <button onclick="assign('${c.customer_id}', '${c.page_id}')">
                            Take Over
                        </button>
                    </div>
                `;
            });
        }

        function assign(customerId, pageId) {
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({
                    type: 'assign_agent',
                    data: { customer_id: customerId, page_id: pageId }
                }));
            }
        }

        function disconnect() {
            if (ws) {
                ws.close();
            }
        }

        function log(message) {
            const logDiv = document.getElementById('log');
            logDiv.innerText = message + '\n' + logDiv.innerText;
        }
    </script>
</body>
</html>
```

## Troubleshooting

### Connection Issues
- Ensure you're authenticated (logged in)
- Check WebSocket URL and protocol (ws:// vs wss://)
- Verify firewall/proxy settings

### Message Not Sending
- Check WebSocket readyState before sending
- Validate message format (must be valid JSON)
- Ensure required fields are present

### Not Receiving Updates
- Verify you're in the same company as other agents
- Check for JavaScript errors in console
- Ensure event handlers are properly set up

### Assignment Conflicts
- Handle "already assigned" errors gracefully
- Refresh customer list after conflicts
- Implement optimistic UI updates with rollback