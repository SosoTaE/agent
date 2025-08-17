# Facebook Bot API Documentation

## Overview
This is a Facebook Messenger bot and comment management system built with Go and Fiber framework. It provides APIs for managing Facebook page interactions, comments, messages, user authentication, and real-time messaging via WebSocket.

## Base URL
```
http://localhost:8080
```

## Authentication
The API uses session-based authentication with cookies. Most endpoints require authentication except for public API endpoints and webhooks.

### Session Management
- Sessions are managed using cookies
- Authentication state is stored in the session
- Company and role information is attached to authenticated sessions

## API Endpoints

### Health Check

#### GET /health
Check service health status.

**Response:**
```json
{
  "status": "ok",
  "service": "facebook-bot"
}
```

---

## Authentication Endpoints

### POST /auth/login
Authenticate a user and create a session.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "password123",
  "company_id": "company_id"
}
```

**Response:**
```json
{
  "message": "Login successful",
  "user": {
    "id": "user_id",
    "email": "user@example.com",
    "username": "username",
    "first_name": "John",
    "last_name": "Doe",
    "company_id": "company_id",
    "role": "company_admin",
    "is_active": true,
    "created_at": "2024-01-01T00:00:00Z",
    "last_login": "2024-01-01T00:00:00Z"
  }
}
```

**Error Responses:**
- `400`: Invalid request body or missing required fields
- `401`: Invalid credentials or account disabled

### POST /auth/logout
Logout the current user and destroy the session.

**Response:**
```json
{
  "message": "Logged out successfully"
}
```

### GET /auth/me
Get current authenticated user information. Requires authentication.

**Response:**
```json
{
  "id": "user_id",
  "email": "user@example.com",
  "username": "username",
  "first_name": "John",
  "last_name": "Doe",
  "company_id": "company_id",
  "role": "company_admin",
  "is_active": true
}
```

**Error Response:**
- `401`: Not authenticated

### GET /auth/check
Check if the current session is valid.

**Response:**
```json
{
  "authenticated": true,
  "user_id": "user_id",
  "username": "username",
  "email": "user@example.com",
  "company_id": "company_id",
  "role": "company_admin"
}
```

---

## Webhook Endpoints

### GET /webhook/
Facebook webhook verification endpoint.

**Query Parameters:**
- `hub.mode`: Should be "subscribe"
- `hub.verify_token`: Verification token matching server config
- `hub.challenge`: Challenge string to echo back

**Response:**
Returns the challenge string if verification succeeds.

### POST /webhook/
Receive Facebook webhook events.

**Request Body:**
Facebook webhook event payload containing messaging or comment events.

**Response:**
```
EVENT_RECEIVED
```

---

## Admin Endpoints (Protected - Requires Authentication)

### GET /admin/company
Get company information for the authenticated user.

**Response:**
```json
{
  "id": "company_id",
  "name": "Company Name",
  "pages": [
    {
      "page_id": "123456",
      "page_name": "Page Name",
      "is_active": true,
      "added_at": "2024-01-01T00:00:00Z"
    }
  ],
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```
Note: Access tokens are hidden in responses for security.

### POST /admin/company/pages
Add a Facebook page to the company (Company Admin only).

**Request Body:**
```json
{
  "page_id": "123456",
  "page_name": "Page Name",
  "page_access_token": "page_access_token",
  "is_active": true
}
```

**Response:**
```json
{
  "message": "Page added successfully",
  "page": {
    "page_id": "123456",
    "page_name": "Page Name",
    "is_active": true
  }
}
```

### POST /admin/users
Create a new user (Company Admin only).

**Request Body:**
```json
{
  "username": "newuser",
  "email": "newuser@example.com",
  "password": "password123",
  "first_name": "John",
  "last_name": "Doe",
  "role": "agent"
}
```

**Valid Roles:**
- `company_admin`
- `bot_manager`
- `human_agent`
- `analyst`
- `viewer`

**Response:**
```json
{
  "message": "User created successfully",
  "user": {
    "id": "user_id",
    "username": "newuser",
    "email": "newuser@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "role": "agent",
    "company_id": "company_id",
    "is_active": true,
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

### POST /admin/users/admin
Create a new user with pre-hashed password and full control over user data (Company Admin only).
This endpoint is for administrative purposes when you need to create users with specific data including pre-hashed passwords.

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

**Note:** 
- The password field must contain a bcrypt-hashed password (starting with `$2a$`)
- All timestamps are optional and will be set to current time if not provided
- Company ID must be specified explicitly

**Response:**
```json
{
  "message": "User created successfully",
  "user": {
    "id": "user_id",
    "username": "luka",
    "email": "luka@gmail.com",
    "first_name": "luka",
    "last_name": "ebralidze",
    "company_id": "company_economist_georgia_1754827518",
    "role": "company_admin",
    "is_active": true,
    "created_at": "2025-08-13T09:17:39.213Z",
    "updated_at": "2025-08-13T09:17:57.411Z",
    "last_login": "2025-08-13T09:17:57.411Z"
  }
}
```

### PUT /admin/users/:userID/role
Update user role (Company Admin only).

**Request Body:**
```json
{
  "role": "bot_manager"
}
```

**Response:**
```json
{
  "message": "User role updated successfully",
  "user_id": "user_id",
  "new_role": "bot_manager"
}
```

### GET /admin/users
Get all users in the company.

**Response:**
```json
{
  "company_id": "company_id",
  "total_users": 10,
  "users_by_role": {
    "company_admin": [
      {
        "id": "user_id",
        "username": "admin",
        "email": "admin@example.com",
        "first_name": "Admin",
        "last_name": "User",
        "role": "company_admin",
        "is_active": true,
        "created_at": "2024-01-01T00:00:00Z"
      }
    ],
    "agent": [...]
  }
}
```

### GET /admin/users/:userID
Get specific user details.

**Response:**
```json
{
  "id": "user_id",
  "username": "username",
  "email": "user@example.com",
  "first_name": "John",
  "last_name": "Doe",
  "role": "agent",
  "company_id": "company_id",
  "is_active": true,
  "created_at": "2024-01-01T00:00:00Z",
  "last_login": "2024-01-01T00:00:00Z"
}
```

---

## Dashboard API Endpoints (Protected - Requires Authentication)

### GET /api/dashboard/pages
Get all pages registered to the company.

**Response:**
```json
{
  "pages": [
    {
      "page_id": "123456",
      "page_name": "Page Name",
      "is_active": true
    }
  ],
  "company": "Company Name"
}
```

### GET /api/dashboard/comments/post/:postID
Get comments for a specific post.

**Response:**
```json
{
  "post_id": "post_id",
  "post_content": "Post content text",
  "comments": [
    {
      "comment": {
        "id": "comment_id",
        "comment_id": "facebook_comment_id",
        "sender_id": "sender_id",
        "sender_name": "Sender Name",
        "message": "Comment text",
        "timestamp": "2024-01-01T00:00:00Z"
      },
      "replies": [...],
      "bot_reply": {
        "message": "Bot response",
        "timestamp": "2024-01-01T00:00:00Z"
      }
    }
  ],
  "total_count": 50,
  "bot_replies": 25,
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### GET /api/dashboard/comments/post/:postID/hierarchical
Get comments in hierarchical structure with nested replies.

**Response:**
Same structure as public `/api/comments/post/:postID` with `hierarchical=true`

### GET /api/dashboard/comments/:commentID/replies
Get a specific comment with all its replies.

**Response:**
Same structure as public `/api/comments/:commentID`

### GET /api/dashboard/comments/post/:postID/threaded
Get comments in threaded view format.

**Response:**
```json
{
  "post_id": "post_id",
  "threads": [
    {
      "parent": {
        "comment_id": "comment_id",
        "sender_name": "Sender Name",
        "message": "Parent comment",
        "timestamp": "2024-01-01T00:00:00Z"
      },
      "replies": [
        {
          "comment_id": "reply_id",
          "sender_name": "Sender Name",
          "message": "Reply text",
          "timestamp": "2024-01-01T00:00:00Z"
        }
      ],
      "reply_count": 2
    }
  ]
}
```

### GET /api/dashboard/comments
Get all comments for company pages.

**Query Parameters:**
- `page_id`: **Required** - Facebook page ID
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 100, max: 500)
- `filter`: "all", "unanswered", "bot" (default: "all")
- `sort`: "newest" or "oldest" (default: "newest")

**Response:**
```json
{
  "comments": [
    {
      "id": "comment_id",
      "comment_id": "facebook_comment_id",
      "post_id": "post_id",
      "sender_id": "sender_id",
      "sender_name": "Sender Name",
      "message": "Comment text",
      "is_reply": false,
      "is_bot": false,
      "timestamp": "2024-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 100,
    "total": 200,
    "total_pages": 2,
    "has_more": true
  },
  "page_id": "123456",
  "page_name": "Page Name",
  "company": "Company Name"
}
```

### GET /api/dashboard/stats
Get comment statistics for company pages.

**Query Parameters:**
- `page_id`: Optional - Filter stats for specific page

**Response:**
```json
{
  "total_comments": 500,
  "bot_replies": 250,
  "unanswered": 50,
  "recent_24h": 20,
  "response_rate": 83.33,
  "pages_monitored": 3,
  "page_id": "123456"
}
```

### GET /api/dashboard/name-changes
Get history of user name changes.

**Query Parameters:**
- `page_id`: Optional - Filter by page ID
- `limit`: Number of results (default: 50)

**Response:**
```json
{
  "changes": [
    {
      "id": "change_id",
      "sender_id": "sender_id",
      "old_name": "Old Name",
      "new_name": "New Name",
      "page_id": "123456",
      "source_type": "comment",
      "detected_at": "2024-01-01T00:00:00Z"
    }
  ],
  "page_id": "123456",
  "company": "Company Name",
  "retrieved": 10
}
```

### GET /api/dashboard/sender/:senderID/history
Get complete history for a specific sender.

**Query Parameters:**
- `page_id`: **Required** - Facebook page ID

**Response:**
```json
{
  "history": {
    "sender_id": "sender_id",
    "current_name": "Current Name",
    "interactions": [
      {
        "type": "comment",
        "content": "Comment text",
        "timestamp": "2024-01-01T00:00:00Z",
        "post_id": "post_id"
      }
    ],
    "total_interactions": 50
  },
  "changes": [
    {
      "old_name": "Old Name",
      "new_name": "New Name",
      "detected_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### GET /api/dashboard/conversations
Get user conversations (chat messages).

**Query Parameters:**
- `page_id`: Optional - Filter by page ID

**Response:**
```json
{
  "users": [
    {
      "sender_id": "sender_id",
      "sender_name": "Sender Name",
      "first_name": "John",
      "last_name": "Doe",
      "last_message": "Last message text",
      "last_message_time": "2024-01-01T00:00:00Z",
      "message_count": 10,
      "page_id": "123456",
      "page_name": "Page Name"
    }
  ],
  "total": 25
}
```

### GET /api/dashboard/messages/page/:pageID
Get all messages for a specific Facebook page.

**URL Parameters:**
- `pageID`: Required - Facebook page ID

**Query Parameters:**
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 50, max: 200)

**Response:**
```json
{
  "page_id": "123456",
  "page_name": "Page Name",
  "messages": [
    {
      "id": "message_id",
      "type": "chat",
      "sender_id": "sender_id",
      "sender_name": "Sender Name",
      "first_name": "John",
      "last_name": "Doe",
      "recipient_id": "page_id",
      "page_id": "123456",
      "page_name": "Page Name",
      "message": "Message text",
      "is_bot": false,
      "timestamp": "2024-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 500,
    "total_pages": 10,
    "has_more": true
  }
}
```

**Notes:**
- Returns ALL messages for the specified page (both sent and received)
- Includes customer messages and bot replies
- Verifies that the page belongs to the authenticated user's company
- Messages are sorted by timestamp in descending order (newest first)

### GET /api/dashboard/messages/page/:pageID/chats
Get all unique chat IDs for a specific page (optimized for chat list).

**URL Parameters:**
- `pageID`: Required - Facebook page ID

**Query Parameters:**
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 50, max: 200)

**Response:**
```json
{
  "page_id": "123456",
  "page_name": "Page Name",
  "chats": [
    {
      "chat_id": "customer_123",
      "customer_name": "John Doe",
      "first_name": "John",
      "last_name": "Doe",
      "last_message": "Thanks for your help!",
      "last_timestamp": "2024-01-01T15:30:00Z",
      "last_sender_id": "customer_123",
      "last_is_bot": false,
      "message_count": 25
    },
    {
      "chat_id": "customer_456",
      "customer_name": "Jane Smith",
      "first_name": "Jane",
      "last_name": "Smith",
      "last_message": "I'll check that out, thank you.",
      "last_timestamp": "2024-01-01T14:00:00Z",
      "last_sender_id": "123456",
      "last_is_bot": true,
      "message_count": 10
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 150,
    "total_pages": 3,
    "has_more": true
  }
}
```

**Notes:**
- Returns unique chats grouped by chat_id
- Includes customer info from customers collection
- Shows last message metadata for preview
- Sorted by last message timestamp (newest first)
- Perfect for building a chat inbox/list view

### GET /api/dashboard/messages/chat/:chatID
Get all messages for a specific chat ID.

**URL Parameters:**
- `chatID`: Required - Chat ID (customer ID)

**Query Parameters:**
- `page_id`: **Required** - Facebook page ID (for security verification)
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 50, max: 200)

**Response:**
```json
{
  "chat_id": "customer_123",
  "customer_name": "John Doe",
  "page_id": "123456",
  "page_name": "Page Name",
  "messages": [
    {
      "id": "message_id_1",
      "type": "chat",
      "chat_id": "customer_123",
      "sender_id": "customer_123",
      "sender_name": "John Doe",
      "recipient_id": "123456",
      "page_id": "123456",
      "page_name": "Page Name",
      "message": "Hello, I need help",
      "is_bot": false,
      "timestamp": "2024-01-01T10:00:00Z"
    },
    {
      "id": "message_id_2",
      "type": "chat",
      "chat_id": "customer_123",
      "sender_id": "123456",
      "recipient_id": "customer_123",
      "page_id": "123456",
      "page_name": "Page Name",
      "message": "Hello! How can I assist you today?",
      "is_bot": true,
      "timestamp": "2024-01-01T10:00:05Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 25,
    "total_pages": 1,
    "has_more": false
  }
}
```

**Notes:**
- Returns all messages for a specific chat_id
- Messages sorted in ascending order (oldest first) for natural chat flow
- Requires page_id for security verification
- Includes customer name from customers collection

### GET /api/dashboard/messages/page/:pageID/conversations
Get all unique conversations for a specific page (legacy endpoint, use /chats instead).

**URL Parameters:**
- `pageID`: Required - Facebook page ID

**Query Parameters:**
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 20, max: 100)

**Response:**
Similar to `/chats` endpoint but with different structure for backward compatibility.

### GET /api/dashboard/messages/conversation/:customerID
Get a customer's conversation with the bot to reconstruct chat in dashboard.

**URL Parameters:**
- `customerID`: Required - Facebook customer/user ID

**Query Parameters:**
- `page_id`: **Required** - Facebook page ID
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 50, max: 200)

**Response:**
```json
{
  "customer_id": "customer_123",
  "customer_name": "John Doe",
  "page_id": "123456",
  "page_name": "Page Name",
  "conversation": [
    {
      "id": "message_id_1",
      "type": "customer",
      "message": "Hello, I need help",
      "timestamp": "2024-01-01T10:00:00Z",
      "is_bot": false,
      "sender_id": "customer_123",
      "recipient_id": "123456",
      "sender_name": "John Doe"
    },
    {
      "id": "message_id_2",
      "type": "bot",
      "message": "Hello! How can I assist you today?",
      "timestamp": "2024-01-01T10:00:05Z",
      "is_bot": true,
      "sender_id": "123456",
      "recipient_id": "customer_123",
      "sender_name": "Page Name"
    },
    {
      "id": "message_id_3",
      "type": "customer",
      "message": "I want to know about your products",
      "timestamp": "2024-01-01T10:01:00Z",
      "is_bot": false,
      "sender_id": "customer_123",
      "recipient_id": "123456",
      "sender_name": "John Doe"
    },
    {
      "id": "message_id_4",
      "type": "bot",
      "message": "Sure! We have a wide range of products...",
      "timestamp": "2024-01-01T10:01:10Z",
      "is_bot": true,
      "sender_id": "123456",
      "recipient_id": "customer_123",
      "sender_name": "Page Name"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 25,
    "total_pages": 1,
    "has_more": false
  }
}
```

**Notes:**
- Returns the complete conversation between a customer and the bot/page
- Each message in the conversation array includes a `type` field: "customer" or "bot"
- Perfect for reconstructing chat UI in dashboard
- Requires both customerID and page_id to fetch the specific conversation
- Messages are sorted by timestamp in descending order (newest first)
- Verifies page ownership before returning data

## Customer Management Endpoints

### GET /api/dashboard/customers
Get list of customers who have sent messages.

**Query Parameters:**
- `page_id`: Optional - Filter by specific page ID
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 50, max: 200)

**Response:**
```json
{
  "company_id": "company_123",
  "customers": [
    {
      "id": "mongodb_id",
      "customer_id": "facebook_user_id",
      "customer_name": "John Doe",
      "first_name": "John",
      "last_name": "Doe",
      "page_id": "123456",
      "page_name": "Page Name",
      "company_id": "company_123",
      "message_count": 25,
      "last_message": "Thanks for your help!",
      "last_seen": "2024-01-01T15:30:00Z",
      "first_seen": "2024-01-01T10:00:00Z",
      "created_at": "2024-01-01T10:00:00Z",
      "updated_at": "2024-01-01T15:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 150,
    "total_pages": 3,
    "has_more": true
  }
}
```

**Notes:**
- Returns customers for the entire company or filtered by page
- Sorted by last_seen (most recent first)
- Includes message count and last message preview

### GET /api/dashboard/customers/:customerID
Get details for a specific customer.

**URL Parameters:**
- `customerID`: Required - Facebook customer ID

**Query Parameters:**
- `page_id`: **Required** - Facebook page ID

**Response:**
```json
{
  "customer": {
    "id": "mongodb_id",
    "customer_id": "facebook_user_id",
    "customer_name": "John Doe",
    "first_name": "John",
    "last_name": "Doe",
    "page_id": "123456",
    "page_name": "Page Name",
    "company_id": "company_123",
    "message_count": 25,
    "last_message": "Thanks for your help!",
    "last_seen": "2024-01-01T15:30:00Z",
    "first_seen": "2024-01-01T10:00:00Z",
    "created_at": "2024-01-01T10:00:00Z",
    "updated_at": "2024-01-01T15:30:00Z"
  }
}
```

### GET /api/dashboard/customers/search
Search for customers by name or ID.

**Query Parameters:**
- `q`: **Required** - Search query (searches name and customer ID)
- `limit`: Maximum results (default: 20, max: 100)

**Response:**
```json
{
  "query": "john",
  "customers": [
    {
      "customer_id": "customer_123",
      "customer_name": "John Doe",
      "page_name": "Page Name",
      "message_count": 25,
      "last_seen": "2024-01-01T15:30:00Z"
    }
  ],
  "count": 5
}
```

**Notes:**
- Searches across customer_name, first_name, last_name
- Exact match for customer_id
- Case-insensitive partial matching for names

### GET /api/dashboard/customers/stats
Get customer statistics for the company.

**Response:**
```json
{
  "total_customers": 500,
  "active_customers": 150,  // Last 30 days
  "new_customers": 25,      // Last 7 days
  "top_customers": [
    {
      "customer_id": "customer_123",
      "customer_name": "John Doe",
      "page_name": "Page Name",
      "message_count": 150,
      "last_seen": "2024-01-01T15:30:00Z"
    }
  ]
}
```

**Notes:**
- Active customers: Interacted in last 30 days
- New customers: First message in last 7 days
- Top customers: Top 5 by message count

---

### GET /api/dashboard/messages/:senderID
Get messages from a specific sender.

**Query Parameters:**
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 50, max: 200)
- `page_id`: Optional - Filter by page ID

**Response:**
```json
{
  "messages": [
    {
      "id": "message_id",
      "type": "chat",
      "sender_id": "sender_id",
      "sender_name": "Sender Name",
      "message": "Message text",
      "is_bot": false,
      "page_id": "123456",
      "page_name": "Page Name",
      "timestamp": "2024-01-01T00:00:00Z"
    }
  ],
  "user": {
    "sender_id": "sender_id",
    "sender_name": "Sender Name",
    "first_name": "John",
    "last_name": "Doe"
  },
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 100,
    "total_pages": 2,
    "has_more": true
  }
}
```

### GET /api/dashboard/posts
Get list of posts for company pages.

**Query Parameters:**
- `page_id`: Optional - Filter by page ID
- `limit`: Items per page (default: 100, max: 500)

**Response:**
```json
{
  "posts": [
    {
      "post_id": "post_id",
      "page_id": "page_id",
      "page_name": "Page Name",
      "content": "Post content",
      "comment_count": 25,
      "bot_replies": 10,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 50,
  "limit": 100,
  "filter": "page_id or all"
}
```

### GET /api/dashboard/posts/company
Get all post IDs for the company.

**Query Parameters:**
- `company_id`: Optional if in session, required otherwise
- `page_id`: Optional - Filter by specific page

**Response:**
```json
{
  "company_id": "company_id",
  "posts": [
    {
      "post_id": "post_id",
      "page_id": "page_id",
      "page_name": "Page Name",
      "comment_count": 25,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 50,
  "filter": "all or page_id"
}
```

---

## Public API Endpoints (No Authentication Required)

### GET /api/comments/post/:postID
Get hierarchical comments for a post (public).

**Query Parameters:**
- `hierarchical`: "true" or "false" (default: "true")
- `company_id`: Optional - Filter by company
- `page_id`: Optional - Filter by page

**Response (hierarchical=true):**
```json
{
  "post_id": "post_id",
  "comments": [
    {
      "comment_id": "facebook_comment_id",
      "sender_id": "sender_id",
      "sender_name": "Sender Name",
      "message": "Comment text",
      "timestamp": "2024-01-01T00:00:00Z",
      "replies": [
        {
          "comment_id": "reply_id",
          "sender_id": "sender_id",
          "sender_name": "Sender Name",
          "message": "Reply text",
          "is_reply": true,
          "timestamp": "2024-01-01T00:00:00Z"
        }
      ]
    }
  ],
  "total": 50
}
```

### GET /api/comments/:commentID
Get a specific comment with replies (public).

**Response:**
```json
{
  "comment": {
    "id": "comment_id",
    "comment_id": "facebook_comment_id",
    "post_id": "post_id",
    "sender_id": "sender_id",
    "sender_name": "Sender Name",
    "message": "Comment text",
    "timestamp": "2024-01-01T00:00:00Z"
  },
  "replies": [...],
  "reply_count": 5
}
```

### GET /api/comments/post/:postID/debug
Debug endpoint for post comments.

**Response:**
```json
{
  "post_id": "post_id",
  "structure": {
    "total_comments": 50,
    "top_level_comments": 20,
    "replies": 30,
    "orphaned_replies": 0
  },
  "hierarchy_tree": [...],
  "debug_info": {
    "database": "mongodb",
    "collection": "comments"
  }
}
```

### GET /api/posts
Get all post IDs in the system.

**Query Parameters:**
- `detailed`: "true" or "false" (default: "false")

**Response (detailed=false):**
```json
{
  "total": 500,
  "post_ids": ["post_id_1", "post_id_2", ...]
}
```

**Response (detailed=true):**
```json
{
  "total": 500,
  "posts": [
    {
      "post_id": "post_id",
      "page_id": "page_id",
      "page_name": "Page Name",
      "comment_count": 25,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### GET /api/posts/paginated
Get paginated list of posts.

**Query Parameters:**
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 20, max: 100)

**Response:**
```json
{
  "posts": [
    {
      "post_id": "post_id",
      "page_id": "page_id",
      "page_name": "Page Name",
      "comment_count": 25,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 500,
    "total_pages": 25,
    "has_more": true
  }
}
```

### GET /api/posts/page/:pageID
Get posts for a specific Facebook page.

**Response:**
```json
{
  "page_id": "page_id",
  "total": 25,
  "posts": [
    {
      "post_id": "post_id",
      "comment_count": 10,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### GET /api/posts/stats
Get overall posts statistics.

**Response:**
```json
{
  "statistics": {
    "total_posts": 500,
    "total_comments": 5000,
    "average_comments_per_post": 10,
    "pages_count": 5
  },
  "most_active_post": {
    "post_id": "post_id",
    "page_id": "page_id",
    "page_name": "Page Name",
    "comment_count": 150
  },
  "page_distribution": {
    "Page Name 1": 100,
    "Page Name 2": 150
  },
  "recent_posts": [
    {
      "post_id": "post_id",
      "page_name": "Page Name",
      "comment_count": 25,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

---

## Error Responses

All endpoints follow a consistent error response format:

```json
{
  "error": "Error message description"
}
```

### Common HTTP Status Codes

- `200 OK`: Request successful
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request parameters
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error

---

## Middleware

### Authentication Middleware
- `RequireAuth`: Validates session and authentication status
- `RequireCompanyAdmin`: Requires company admin role
- `RequireRole`: Validates specific user roles
- `RequirePermission`: Validates specific permissions

### Page Access Middleware
- `ValidatePageAccess`: Ensures user has access to requested Facebook page
- `ValidatePostAccess`: Ensures user has access to requested post
- `ExtractCompanyPages`: Extracts and stores company page IDs in context

---

## Rate Limiting

The API implements rate limiting for certain endpoints to prevent abuse. Rate limit information is provided in response headers:

- `X-RateLimit-Limit`: Maximum requests allowed
- `X-RateLimit-Remaining`: Remaining requests in current window
- `X-RateLimit-Reset`: Time when rate limit resets

---

## CORS Configuration

The API allows cross-origin requests from:
- `http://localhost:5173`
- `http://localhost:3000`
- `http://localhost:5174`

Allowed methods: GET, POST, PUT, DELETE, OPTIONS, PATCH
Credentials are allowed for authenticated requests.

---

## Environment Variables

Required environment variables:
- `PORT`: Server port (default: 8080)
- `MONGO_URI`: MongoDB connection string
- `DATABASE_NAME`: MongoDB database name
- `VERIFY_TOKEN`: Facebook webhook verification token
- `PAGE_ACCESS_TOKEN`: Facebook page access token
- `ANTHROPIC_API_KEY`: Claude AI API key
- `DEBUG_CLAUDE`: Enable Claude debug mode (true/false)

---

## Data Models

### User
```json
{
  "id": "string",
  "username": "string",
  "email": "string",
  "password_hash": "string",
  "first_name": "string",
  "last_name": "string",
  "role": "company_admin|bot_manager|human_agent|analyst|viewer",
  "company_id": "string",
  "api_key": "string",
  "is_active": "boolean",
  "created_at": "datetime",
  "updated_at": "datetime",
  "last_login": "datetime"
}
```

### Company
```json
{
  "id": "string",
  "name": "string",
  "pages": [
    {
      "page_id": "string",
      "page_name": "string",
      "page_access_token": "string",
      "is_active": "boolean",
      "added_at": "datetime",
      "claude_model": "string",
      "system_prompt": "string"
    }
  ],
  "voyage_config": {
    "api_key": "string",
    "model": "voyage-2|voyage-large-2|voyage-code-2"
  },
  "created_at": "datetime",
  "updated_at": "datetime"
}
```

### Message
```json
{
  "id": "string",
  "type": "chat|crm_data|crm_response",
  "chat_id": "string",  // Always the customer ID for grouping conversations
  "sender_id": "string",
  "sender_name": "string",
  "first_name": "string",
  "last_name": "string",
  "recipient_id": "string",
  "page_id": "string",
  "page_name": "string",
  "message": "string",
  "processed_data": "object",
  "is_bot": "boolean",
  "timestamp": "datetime",
  "updated_at": "datetime"
}
```

**Important Notes about chat_id:**
- The `chat_id` field is ALWAYS set to the customer's ID, regardless of who sent the message
- For customer messages: `chat_id = sender_id` (the customer)
- For bot/page messages: `chat_id = recipient_id` (still the customer)
- This makes it easy to query all messages in a conversation using a single field

### Comment
```json
{
  "id": "string",
  "comment_id": "string",
  "post_id": "string",
  "parent_id": "string",
  "sender_id": "string",
  "sender_name": "string",
  "first_name": "string",
  "last_name": "string",
  "page_id": "string",
  "page_name": "string",
  "message": "string",
  "post_content": "string",
  "is_reply": "boolean",
  "is_bot": "boolean",
  "replies": [],
  "timestamp": "datetime",
  "updated_at": "datetime"
}
```

### Customer
```json
{
  "id": "string",
  "customer_id": "string",  // Facebook user ID
  "customer_name": "string",
  "first_name": "string",
  "last_name": "string",
  "page_id": "string",      // The page they're messaging
  "page_name": "string",
  "company_id": "string",    // Company that owns the page
  "message_count": "number",
  "last_message": "string",
  "last_seen": "datetime",
  "first_seen": "datetime",
  "created_at": "datetime",
  "updated_at": "datetime"
}
```

**Customer Collection Notes:**
- One record per customer per page (unique on customer_id + page_id)
- Automatically created/updated when customer sends a message
- Tracks message count and last interaction
- Used for customer list views and analytics

### NameHistory
```json
{
  "id": "string",
  "sender_id": "string",
  "old_name": "string",
  "new_name": "string",
  "page_id": "string",
  "source_type": "comment|message",
  "source_id": "string",
  "detected_at": "datetime"
}
```

---

## User Roles and Permissions

### Company Admin (company_admin)
- Full access to all company resources
- Can manage users and pages
- Can view all data
- Can update company settings

### Bot Manager (bot_manager)
- Can manage bot settings and responses
- Can view all conversations and comments
- Can configure AI prompts
- Cannot manage users

### Human Agent (human_agent)
- Can view and respond to messages and comments
- Can view company data
- Can handle escalated conversations
- Cannot manage users or pages

### Analyst (analyst)
- Read-only access to all data
- Can view statistics and reports
- Can export data for analysis
- Cannot send responses or manage resources

### Viewer (viewer)
- Read-only access to company data
- Cannot send responses
- Cannot manage resources
- Limited access to sensitive information

---

## Additional Endpoints

### GET /api/dashboard/chat/messages
Get chat messages with filtering options.

**Query Parameters:**
- `page_id`: Optional - Filter by page ID
- `sender_id`: Optional - Filter by sender ID
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 50, max: 200)

**Response:**
```json
{
  "messages": [
    {
      "id": "message_id",
      "type": "chat",
      "sender_id": "sender_id",
      "sender_name": "Sender Name",
      "message": "Message text",
      "is_bot": false,
      "page_id": "123456",
      "page_name": "Page Name",
      "timestamp": "2024-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 200,
    "total_pages": 4,
    "has_more": true
  }
}
```

### GET /api/dashboard/chat/conversations
Get chat conversations summary.

**Query Parameters:**
- `page_id`: Optional - Filter by page ID

**Response:**
```json
{
  "conversations": [
    {
      "sender_id": "sender_id",
      "sender_name": "Sender Name",
      "last_message": "Last message text",
      "last_message_time": "2024-01-01T00:00:00Z",
      "message_count": 25,
      "page_id": "123456",
      "page_name": "Page Name"
    }
  ],
  "total": 10
}
```

### GET /api/dashboard/chat/:senderID/history
Get chat history between user and page.

**URL Parameters:**
- `senderID`: Required - Sender's Facebook ID

**Query Parameters:**
- `page_id`: **Required** - Facebook page ID
- `limit`: Number of messages (default: 50)

**Response:**
```json
{
  "sender": {
    "id": "sender_id",
    "name": "Sender Name"
  },
  "messages": [
    {
      "id": "message_id",
      "message": "Message text",
      "is_bot": false,
      "timestamp": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 50
}
```

---

## WebSocket Real-time Messaging

### WebSocket Connection
Connect to WebSocket for real-time message updates.

**Endpoint:** `GET /dashboard/ws`  
**Authentication:** Required (JWT token in cookie)

#### Connection Flow:
1. Client connects with authentication
2. Server sends connection confirmation with user_id
3. Client can subscribe to page updates
4. Server broadcasts new messages in real-time

#### WebSocket Message Types:

##### Incoming Messages (Client → Server):

**1. Ping/Keep-alive:**
```json
{
  "type": "ping"
}
```

**2. Subscribe to Page:**
```json
{
  "type": "subscribe",
  "page_id": "page123"
}
```

**3. Send Message to Customer (requires customer.stop = true):**
```json
{
  "type": "send_message",
  "customer_id": "customer123",
  "page_id": "page123",
  "message": "Hello, how can I help you?"
}
```

##### Outgoing Messages (Server → Client):

**1. Connection Established:**
```json
{
  "type": "connected",
  "message": "WebSocket connection established",
  "user_id": "uuid"
}
```

**2. New Message Broadcast:**
```json
{
  "type": "new_message",
  "page_id": "page123",
  "data": {
    "chat_id": "customer123",
    "sender_id": "customer123",
    "sender_name": "John Doe",
    "recipient_id": "page123",
    "message": "I need help",
    "is_bot": false,
    "is_human": false,
    "agent_id": null,
    "agent_email": null,
    "agent_name": null,
    "timestamp": 1234567890
  },
  "timestamp": 1234567890
}
```

**3. Message Sent Confirmation:**
```json
{
  "type": "message_sent",
  "customer_id": "customer123",
  "page_id": "page123",
  "message": "Your message text",
  "timestamp": 1234567890
}
```

**4. Error:**
```json
{
  "type": "error",
  "error": "Error message description"
}
```

### Message Model Updates
Messages now include agent tracking for human responses:

```json
{
  "is_human": true,        // true if sent by human agent
  "agent_id": "user123",   // ID of agent who sent message
  "agent_email": "agent@example.com",  // Agent's email
  "agent_name": "John Doe" // Agent's name
}
```

---

## Customer Management

### Customer Model
The customer model includes fields for tracking human assistance requests:

```json
{
  "customer_id": "facebook_user_id",
  "customer_name": "John Doe",
  "page_id": "page123",
  "company_id": "company_id",
  "stop": false,           // true if customer wants real person
  "stopped_at": null,      // timestamp when customer requested human help
  "message_count": 42,
  "last_message": "I need help",
  "last_seen": "2024-01-01T00:00:00Z"
}
```

### Human Handoff System

The system implements a human handoff mechanism where:
1. **Bot detects intent** - When a customer expresses desire to talk to a human (e.g., "I want to speak to a real person"), the bot automatically sets `stop=true`
2. **Bot stops responding** - When `stop=true`, the bot will NOT process or respond to customer messages
3. **Agent notification** - Dashboard shows customers with `stop=true` who need human assistance
4. **Agent messaging** - Agents can ONLY send messages to customers where `stop=true`
5. **Message logging** - All customer messages are still saved and broadcast via WebSocket for monitoring
6. **Resume bot** - Setting `stop=false` returns the customer to bot-only responses

**Important Behavior:**
- When `stop=true`: Customer messages are saved but NOT processed by the bot
- When `stop=false`: Customer messages are processed normally by the bot
- Agents cannot send messages when `stop=false`

### PUT /dashboard/customers/:customerID/stop
Update customer's stop status (manually control human assistance mode).

**URL Parameters:**
- `customerID`: Required - Customer's Facebook ID

**Request Body:**
```json
{
  "page_id": "page123",
  "stop": true  // true = human mode, false = bot mode
}
```

**Response:**
```json
{
  "message": "Customer status updated successfully",
  "customer": {
    "id": "mongo_id",
    "customer_id": "facebook_id",
    "customer_name": "John Doe",
    "stop": true,
    "stopped_at": "2024-01-01T00:00:00Z"
  }
}
```

**Use Cases:**
- Manually escalate customer to human agent
- Return customer to bot after human interaction
- Override automatic detection

### POST /dashboard/customers/:customerID/toggle-stop
Toggle customer's stop status (quick switch between bot/human mode).

**URL Parameters:**
- `customerID`: Required - Customer's Facebook ID

**Request Body:**
```json
{
  "page_id": "page123"
}
```

**Response (Success):**
```json
{
  "message": "Customer status toggled successfully",
  "customer": {
    "id": "mongo_id",
    "customer_id": "facebook_id",
    "customer_name": "John Doe",
    "stop": true,
    "stopped_at": "2024-01-01T00:00:00Z"
  },
  "previous_stop": false,
  "current_stop": true
}
```

**Response (Error - Customer Not Found):**
```json
{
  "error": "Customer not found"
}
```

### POST /dashboard/customers/:customerID/message
Send a message from dashboard to a customer.

**⚠️ IMPORTANT:** Customer MUST have `stop=true` to receive messages from agents. If `stop=false`, the endpoint will return an error.

**URL Parameters:**
- `customerID`: Required - Customer's Facebook ID

**Request Body:**
```json
{
  "page_id": "page123",
  "message": "Hello! I'm here to help with your question."
}
```

**Response (Success - stop=true):**
```json
{
  "message": "Message sent successfully",
  "data": {
    "customer_id": "customer123",
    "page_id": "page123",
    "message": "Hello! I'm here to help with your question.",
    "agent": "agent@example.com",
    "timestamp": 1234567890
  }
}
```

**Response (Error - stop=false):**
```json
{
  "error": "Customer has not requested human assistance. Set stop=true first."
}
```

**Response (Error - Invalid Request):**
```json
{
  "error": "Page ID and message are required"
}
```

**Process Flow:**
1. Validates customer has `stop=true`
2. Sends message via Facebook Messenger API
3. Saves message to database with agent tracking:
   - `is_human`: true
   - `agent_id`: Session user ID
   - `agent_email`: Agent's email
   - `agent_name`: Agent's full name
4. Broadcasts message via WebSocket to all dashboard users

### GET /dashboard/customers/stats
Get customer statistics for the company.

**Query Parameters:**
- `page_id`: Optional - Filter stats by specific page

**Response:**
```json
{
  "total_customers": 150,
  "active_customers": 45,        // Active in last 30 days
  "new_customers": 12,           // Created in last 7 days
  "stopped_customers": 5,        // Customers with stop=true
  "top_customers": [
    {
      "customer_id": "123",
      "customer_name": "John Doe",
      "message_count": 50,
      "last_seen": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### Human Handoff Workflow

```mermaid
graph LR
    A[Customer Message] --> B{Check stop field}
    B -->|stop=true| C[Skip Bot Processing]
    B -->|stop=false| D{Contains "want human"?}
    C --> E[Save Message Only]
    E --> F[Broadcast to Dashboard]
    F --> G[Agent Responds]
    D -->|Yes| H[Set stop=true]
    D -->|No| I[Bot Response]
    H --> C
    G --> J[Customer Responds]
    J --> B
    G --> K[Agent Completes]
    K --> L[Set stop=false]
    L --> I
```

**Message Flow Details:**
1. Every incoming customer message checks the `stop` field first
2. If `stop=true`, bot processing is completely skipped
3. Messages are always saved and broadcast for dashboard monitoring
4. Only human agents can respond when `stop=true`
5. Setting `stop=false` re-enables bot processing

### Message Model Updates
Messages include agent tracking for human responses:

```json
{
  "type": "chat",
  "chat_id": "customer_id",
  "sender_id": "page_id or customer_id",
  "message": "Message text",
  "is_bot": false,
  "is_human": true,              // true if sent by human agent
  "agent_id": "user123",         // ID of agent who sent message
  "agent_email": "agent@example.com",  // Agent's email
  "agent_name": "John Agent",   // Agent's name
  "timestamp": "2024-01-01T00:00:00Z"
}
```

---

## Internal Webhook Handlers

### Message Handler
Processes incoming Facebook Messenger messages.
- Saves message to database
- Gets AI response from Claude
- Sends reply back to user
- Tracks conversation history

### Comment Handler
Processes Facebook comment notifications.
- Saves comment to database
- Checks for duplicate processing
- Gets AI response from Claude
- Posts reply to comment
- Tracks name changes

---

## Notes

1. All timestamps are in ISO 8601 format
2. Pagination is available on most list endpoints
3. The API uses MongoDB for data storage
4. Facebook webhook events are processed asynchronously
5. The bot integrates with Claude AI for generating responses
6. Session cookies are used for authentication (not token-based)
7. Rate limiting is implemented for certain endpoints
8. Company data isolation is enforced across all endpoints