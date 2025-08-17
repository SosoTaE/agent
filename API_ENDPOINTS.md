# Facebook Bot API Documentation

## Base URL
```
http://localhost:8080
```

## Authentication
Most endpoints require authentication via session cookies. Login first to obtain a session.

---

## 1. Authentication Endpoints

### Login
**POST** `/auth/login`

Authenticates a user and creates a session.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "password123",
  "company_id": "company_test_1755424297"
}
```

**Response:**
```json
{
  "message": "Login successful",
  "user": {
    "id": "507f1f77bcf86cd799439011",
    "email": "user@example.com",
    "name": "John Doe",
    "company_id": "company_test_1755424297",
    "role": "admin"
  }
}
```

---

### Logout
**POST** `/auth/logout`

Destroys the current session.

**Response:**
```json
{
  "message": "Logout successful"
}
```

---

### Get Current User
**GET** `/auth/me`

Returns the currently authenticated user's information.

**Response:**
```json
{
  "user": {
    "id": "507f1f77bcf86cd799439011",
    "email": "user@example.com",
    "name": "John Doe",
    "company_id": "company_test_1755424297",
    "role": "admin"
  }
}
```

---

### Check Session
**GET** `/auth/check`

Checks if the current session is valid.

**Response:**
```json
{
  "authenticated": true,
  "user_id": "507f1f77bcf86cd799439011"
}
```

---

## 2. Admin Endpoints (Protected - Requires Authentication)

### Get Company Information
**GET** `/admin/company`

Returns the company information with all configured pages.

**Response:**
```json
{
  "id": "507f1f77bcf86cd799439011",
  "company_id": "company_test_1755424297",
  "company_name": "Test Company",
  "pages": [
    {
      "page_id": "461998383671026",
      "page_name": "Jigarsoft",
      "page_access_token": "EAAS0H4p...***HIDDEN***",
      "app_secret": "***HIDDEN***",
      "claude_api_key": "***HIDDEN***",
      "claude_model": "claude-3-haiku-20240307",
      "voyage_api_key": "***HIDDEN***",
      "voyage_model": "voyage-2",
      "system_prompt": "You are a helpful assistant",
      "is_active": true,
      "max_tokens": 1024
    }
  ],
  "crm_links": [],
  "is_active": true,
  "response_delay": 0,
  "default_language": "en",
  "created_at": "2025-01-19T09:51:37.215Z",
  "updated_at": "2025-01-19T09:52:57.874Z"
}
```

---

### Create New Company
**POST** `/admin/company`

Creates a new company with auto-generated ID.

**Request Body:**
```json
{
  "company_name": "New Company",
  "page_id": "123456789",
  "page_name": "My Page",
  "page_access_token": "EAAxxxxx",
  "app_secret": "secret123",
  "claude_api_key": "sk-ant-api03-xxx",
  "claude_model": "claude-3-haiku-20240307",
  "voyage_api_key": "pa--xxx",
  "voyage_model": "voyage-2",
  "system_prompt": "You are a customer service assistant",
  "max_tokens": 1024,
  "response_delay": 0,
  "default_language": "en"
}
```

**Response:**
```json
{
  "message": "Company created successfully",
  "company_id": "company_new_company_1755500000",
  "company": {
    "company_id": "company_new_company_1755500000",
    "company_name": "New Company",
    "page_id": "123456789",
    "page_name": "My Page"
  }
}
```

---

### Add Page to Company
**POST** `/admin/company/pages` *(Requires Company Admin Role)*

Adds a new Facebook page to an existing company.

**Request Body:**
```json
{
  "page_id": "987654321",
  "page_name": "Another Page",
  "page_access_token": "EAAyyyyy",
  "is_active": true
}
```

**Response:**
```json
{
  "message": "Page added successfully",
  "page": {
    "page_id": "987654321",
    "page_name": "Another Page",
    "page_access_token": "EAAyy...***HIDDEN***",
    "is_active": true
  }
}
```

---

### Add Page with Full Configuration
**POST** `/admin/company/pages/full` *(Requires Company Admin Role)*

Adds a new Facebook page with complete configuration details to an existing company.

**Request Body:**
```json
{
  "page_id": "987654321",
  "page_name": "Another Page",
  "page_access_token": "EAAyyyyy",
  "app_secret": "7c218a4a0e172e8965d4789eb7fe9c65",
  "claude_api_key": "sk-ant-api03-xxx",
  "claude_model": "claude-3-haiku-20240307",
  "voyage_api_key": "pa--xxx",
  "voyage_model": "voyage-2",
  "system_prompt": "You are a helpful customer service assistant",
  "is_active": true,
  "max_tokens": 1024
}
```

**Response:**
```json
{
  "message": "Page added successfully with full configuration",
  "page": {
    "page_id": "987654321",
    "page_name": "Another Page",
    "page_access_token": "EAAyy...***HIDDEN***",
    "app_secret": "***HIDDEN***",
    "claude_api_key": "***HIDDEN***",
    "claude_model": "claude-3-haiku-20240307",
    "voyage_api_key": "***HIDDEN***",
    "voyage_model": "voyage-2",
    "system_prompt": "You are a helpful customer service assistant",
    "is_active": true,
    "max_tokens": 1024
  }
}
```

---

### Update Page Configuration
**PUT** `/admin/company/pages/:pageID` *(Requires Company Admin Role)*

Updates the configuration of an existing page. Only provided fields will be updated.

**URL Parameters:**
- `pageID`: The Facebook page ID to update

**Request Body:** (All fields are optional)
```json
{
  "page_name": "Updated Page Name",
  "page_access_token": "EAAzzzzz",
  "app_secret": "new_secret",
  "claude_api_key": "sk-ant-api03-new",
  "claude_model": "claude-3-opus-20240229",
  "voyage_api_key": "pa--new",
  "voyage_model": "voyage-large-2",
  "system_prompt": "Updated system prompt",
  "is_active": false,
  "max_tokens": 2048
}
```

**Response:**
```json
{
  "message": "Page configuration updated successfully",
  "page_id": "987654321"
}
```

**Notes:**
- Only fields provided in the request body will be updated
- Other fields will remain unchanged
- Use `null` or omit fields to keep them unchanged
- For boolean fields like `is_active`, you must explicitly provide the value to change it

---

### Create User
**POST** `/admin/users` *(Requires Company Admin Role)*

Creates a new user for the company.

**Request Body:**
```json
{
  "username": "newuser",
  "email": "newuser@example.com",
  "password": "password123",
  "first_name": "New",
  "last_name": "User",
  "role": "viewer"
}
```

---

### Create User with Pre-hashed Password
**POST** `/admin/users/admin` *(Requires Company Admin Role)*

Creates a new user with pre-hashed password and full control over user data. This endpoint is for administrative purposes.

**Request Body:**
```json
{
  "username": "luka",
  "email": "luka@gmail.com",
  "password": "$2a$10$YJR.ZZf8/haEEOQO./I2jup4Kwm2LlktZ2ET26ti27Hm9EYW4l6pK",
  "first_name": "luka",
  "last_name": "ebralidze",
  "company_id": "company_economist_georgia_1754827518",
  "role": "company_admin",
  "is_active": true,
  "created_at": "2025-08-13T09:17:39.213Z",
  "updated_at": "2025-08-13T09:17:57.411Z",
  "last_login": "2025-08-13T09:17:57.411Z"
}
```

**Response:**
```json
{
  "message": "User created successfully",
  "user": {
    "id": "507f1f77bcf86cd799439012",
    "email": "newuser@example.com",
    "name": "New User",
    "role": "viewer"
  }
}
```

---

### Update User Role
**PUT** `/admin/users/:userID/role` *(Requires Company Admin Role)*

Updates a user's role within the company.

**Request Body:**
```json
{
  "role": "admin"
}
```

**Response:**
```json
{
  "message": "User role updated successfully",
  "user_id": "507f1f77bcf86cd799439012",
  "new_role": "admin"
}
```

---

### Get Company Users
**GET** `/admin/users`

Returns all users in the company.

**Response:**
```json
{
  "users": [
    {
      "id": "507f1f77bcf86cd799439011",
      "email": "user@example.com",
      "name": "John Doe",
      "role": "admin",
      "is_active": true
    }
  ]
}
```

---

### Get Specific User
**GET** `/admin/users/:userID`

Returns information about a specific user.

**Response:**
```json
{
  "user": {
    "id": "507f1f77bcf86cd799439011",
    "email": "user@example.com",
    "name": "John Doe",
    "role": "admin",
    "company_id": "company_test_1755424297",
    "is_active": true,
    "created_at": "2025-01-19T09:00:00Z"
  }
}
```

---

## 3. Dashboard Endpoints (Protected - Requires Authentication)

### Get Company Pages
**GET** `/api/dashboard/pages`

Returns all pages configured for the company.

**Response:**
```json
{
  "pages": [
    {
      "page_id": "461998383671026",
      "page_name": "Jigarsoft",
      "is_active": true
    }
  ]
}
```

---

### Comments Endpoints

#### Get Post Comments
**GET** `/api/dashboard/comments/post/:postID`

Returns comments for a specific post.

**Response:**
```json
{
  "post_id": "123_456",
  "comments": [
    {
      "comment_id": "789",
      "sender_id": "111",
      "sender_name": "User Name",
      "message": "Great post!",
      "created_time": "2025-01-19T10:00:00Z"
    }
  ]
}
```

---

#### Get Post Comments (Hierarchical)
**GET** `/api/dashboard/comments/post/:postID/hierarchical`

Returns comments in hierarchical structure with replies.

---

#### Get Comment with Replies
**GET** `/api/dashboard/comments/:commentID/replies`

Returns a specific comment with all its replies.

---

#### Get Threaded Comments
**GET** `/api/dashboard/comments/post/:postID/threaded`

Returns comments in threaded view format.

---

#### Get All Company Comments
**GET** `/api/dashboard/comments`

Returns all comments across all company pages.

**Query Parameters:**
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 20)
- `page_id`: Filter by specific page

---

#### Get Comment Statistics
**GET** `/api/dashboard/stats`

Returns statistics about comments.

**Response:**
```json
{
  "total_comments": 1234,
  "total_posts": 56,
  "comments_today": 78,
  "response_rate": 95.5
}
```

---

### Name Changes
**GET** `/api/dashboard/name-changes`

Returns history of customer name changes.

---

### Sender History
**GET** `/api/dashboard/sender/:senderID/history`

Returns message history for a specific sender.

---

### Messages Endpoints

#### Get All Messages
**GET** `/api/dashboard/messages`

Returns all messages with pagination and filtering.

**Query Parameters:**
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 20)
- `page_id`: Filter by page
- `sender_id`: Filter by sender
- `type`: Filter by type (message/comment)

---

#### Get User Conversations
**GET** `/api/dashboard/conversations`

Returns all conversations for the authenticated user.

---

#### Get User Messages
**GET** `/api/dashboard/messages/:senderID`

Returns all messages from a specific sender.

---

#### Get Messages by Page
**GET** `/api/dashboard/messages/page/:pageID`

Returns all messages for a specific page.

---

#### Get Page Conversations
**GET** `/api/dashboard/messages/page/:pageID/conversations`

Returns all conversations for a specific page.

---

#### Get Chat IDs
**GET** `/api/dashboard/messages/page/:pageID/chats`

Returns all chat IDs for a specific page.

---

#### Get Messages by Chat ID
**GET** `/api/dashboard/messages/chat/:chatID`

Returns all messages in a specific chat.

---

#### Get Customer Conversation
**GET** `/api/dashboard/messages/conversation/:customerID`

Returns the full conversation with a customer including bot responses.

**Response:**
```json
{
  "customer": {
    "id": "123456",
    "name": "John Doe",
    "psid": "789012"
  },
  "messages": [
    {
      "id": "msg_1",
      "sender": "customer",
      "message": "Hello",
      "timestamp": "2025-01-19T10:00:00Z"
    },
    {
      "id": "msg_2",
      "sender": "bot",
      "message": "Hi! How can I help you?",
      "timestamp": "2025-01-19T10:00:01Z"
    }
  ]
}
```

---

### Customer Management Endpoints

#### Get Customers List
**GET** `/api/dashboard/customers`

Returns list of all customers.

**Query Parameters:**
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 20)
- `page_id`: Filter by page
- `search`: Search by name or ID

**Response:**
```json
{
  "customers": [
    {
      "id": "cust_123",
      "name": "John Doe",
      "psid": "789012",
      "page_id": "461998383671026",
      "stop_bot": false,
      "last_interaction": "2025-01-19T10:00:00Z"
    }
  ],
  "total": 150,
  "page": 1,
  "limit": 20
}
```

---

#### Get Customer Details
**GET** `/api/dashboard/customers/:customerID`

Returns detailed information about a specific customer.

---

#### Search Customers
**GET** `/api/dashboard/customers/search`

Search for customers by various criteria.

**Query Parameters:**
- `q`: Search query
- `field`: Field to search (name, psid, email)

---

#### Get Customer Statistics
**GET** `/api/dashboard/customers/stats`

Returns statistics about customers.

---

#### Update Customer Stop Status
**PUT** `/api/dashboard/customers/:customerID/stop`

Updates whether the bot should respond to a customer.

**Request Body:**
```json
{
  "stop_bot": true
}
```

---

#### Toggle Customer Stop Status
**POST** `/api/dashboard/customers/:customerID/toggle-stop`

Toggles the bot stop status for a customer.

**Response:**
```json
{
  "message": "Stop status toggled",
  "customer_id": "cust_123",
  "stop_bot": true
}
```

---

#### Send Message to Customer
**POST** `/api/dashboard/customers/:customerID/message`

Sends a message to a customer via Facebook.

**Request Body:**
```json
{
  "message": "Hello, this is a message from support."
}
```

---

### Posts Endpoints

#### Get Posts List
**GET** `/api/dashboard/posts`

Returns list of Facebook posts.

---

#### Get Company Posts
**GET** `/api/dashboard/posts/company`

Returns all posts for the company's pages.

---

### WebSocket Endpoint
**GET** `/api/dashboard/ws`

WebSocket connection for real-time updates.

**Connection URL:**
```
ws://localhost:8080/api/dashboard/ws
```

**Message Format:**
```json
{
  "type": "new_message",
  "data": {
    "message_id": "msg_123",
    "sender_id": "789012",
    "content": "New message",
    "timestamp": "2025-01-19T10:00:00Z"
  }
}
```

---

## 4. Public API Endpoints (No Authentication Required)

### Comments Endpoints

#### Get Post Comments (Public)
**GET** `/api/comments/post/:postID`

Returns hierarchical comments for a post.

---

#### Get Comment with Replies (Public)
**GET** `/api/comments/:commentID`

Returns a comment with its replies.

---

#### Debug Post Comments
**GET** `/api/comments/post/:postID/debug`

Debug endpoint for viewing comment structure.

---

### Posts Endpoints

#### Get All Post IDs
**GET** `/api/posts`

Returns all post IDs in the system.

---

#### Get Paginated Posts
**GET** `/api/posts/paginated`

Returns paginated list of posts.

**Query Parameters:**
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 20)

---

#### Get Posts by Page
**GET** `/api/posts/page/:pageID`

Returns all posts for a specific Facebook page.

---

#### Get Posts Statistics
**GET** `/api/posts/stats`

Returns statistics about posts.

---

## 5. Webhook Endpoints

### Facebook Webhook Verification
**GET** `/webhook`

Facebook webhook verification endpoint.

**Query Parameters:**
- `hub.mode`: Should be "subscribe"
- `hub.verify_token`: Verification token
- `hub.challenge`: Challenge string to return

---

### Facebook Webhook Events
**POST** `/webhook`

Receives Facebook webhook events (messages, comments, etc.)

**Request Body:** (Facebook Event Format)
```json
{
  "object": "page",
  "entry": [
    {
      "id": "PAGE_ID",
      "time": 1458692752478,
      "messaging": [
        {
          "sender": {"id": "USER_ID"},
          "recipient": {"id": "PAGE_ID"},
          "timestamp": 1458692752478,
          "message": {
            "mid": "mid.1457764197618:41d102a3e1ae206a38",
            "text": "hello, world!"
          }
        }
      ]
    }
  ]
}
```

---

## 6. Health Check

### Health Status
**GET** `/health`

Returns the health status of the service.

**Response:**
```json
{
  "status": "ok",
  "service": "facebook-bot"
}
```

---

## Error Responses

All endpoints may return error responses in the following format:

```json
{
  "error": "Error message",
  "details": "Additional error details (optional)"
}
```

### Common HTTP Status Codes:
- `200 OK`: Request successful
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request parameters
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Resource not found
- `409 Conflict`: Resource already exists
- `500 Internal Server Error`: Server error

---

## Rate Limiting

The API implements rate limiting for certain endpoints. Rate limit information is provided in response headers:

- `X-RateLimit-Limit`: Maximum requests per window
- `X-RateLimit-Remaining`: Remaining requests in current window
- `X-RateLimit-Reset`: Timestamp when the rate limit resets

---

## CORS Configuration

The API allows cross-origin requests from:
- `http://localhost:5173`
- `http://localhost:3000`
- `http://localhost:5174`

Allowed methods: `GET, POST, PUT, DELETE, OPTIONS, PATCH`

---

## Session Management

Sessions expire after 24 hours. The session cookie is:
- Name: `session`
- HttpOnly: `true`
- SameSite: `Lax`
- Secure: `false` (set to `true` in production with HTTPS)