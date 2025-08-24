# Facebook Bot Management System

A comprehensive bot management platform for Facebook and Messenger interactions with advanced CRM integration, RAG (Retrieval-Augmented Generation) capabilities, and multi-company support.

## Overview

This system provides a complete solution for managing Facebook pages, handling customer interactions through both Facebook comments and Messenger, and integrating with external CRM systems. It includes advanced AI capabilities through Claude integration and supports document-based knowledge augmentation.

## Core Features

### 🏢 Multi-Company & Multi-Page Management
- **Company Management**: Support for multiple companies with isolated data
- **Page Management**: Each company can manage multiple Facebook pages
- **User Role System**: Admin and regular user roles with different permissions
- **Channel-Specific Configuration**: Separate settings for Facebook comments and Messenger

### 🔐 Authentication & Security
- **Session-Based Authentication**: Secure login/logout with session management
- **Role-Based Access Control**: Company admins have elevated privileges
- **Session Cleanup**: Automatic cleanup of expired sessions
- **Secure Password Handling**: Bcrypt password hashing

### 💬 Message & Comment Management
- **Facebook Comments**: 
  - Hierarchical comment tracking with parent-child relationships
  - Threaded view support
  - Real-time comment monitoring
  - Post-level comment aggregation
  
- **Messenger Integration**:
  - Full conversation tracking
  - Customer message history
  - Agent assignment system
  - Stop/resume bot functionality per customer

### 🤖 AI Integration (Claude)
- **Intelligent Responses**: Claude AI integration for automated responses
- **Context-Aware Conversations**: Maintains conversation context
- **Channel-Specific AI Behavior**: Different AI configurations for Facebook vs Messenger
- **Debug Mode**: Optional Claude debug mode for development

### 📊 CRM Integration
- **Multi-Channel CRM Links**: Separate CRM configurations for Facebook and Messenger
- **Dynamic Data Fetching**: Automatic CRM data synchronization
- **API Integration**: Support for external CRM APIs with authentication
- **Scheduled Updates**: Background scheduler for CRM data updates
- **CRM Link Management**:
  - Add/remove CRM endpoints
  - Toggle active/inactive status
  - Channel-specific CRM configurations
  - Automatic data refresh

### 🔍 RAG (Retrieval-Augmented Generation)
- **Document Upload**: Support for text, markdown, CSV, and JSON files
- **Multi-Channel Documents**: Documents can be active for specific channels
- **Embedding Storage**: Vector database integration for semantic search
- **Chunk Management**: Automatic document chunking for large files
- **Document Control**:
  - Toggle documents on/off
  - Channel-specific document activation
  - Metadata tracking (uploader, timestamp)
  - Filename-based grouping

### 👥 Customer Management
- **Customer Profiles**: Comprehensive customer data tracking
- **Stop List Management**: Block/unblock customers from bot interactions
- **Agent Assignment**: Assign human agents to specific customers
- **Conversation History**: Full message history per customer
- **Search & Filter**: Advanced customer search capabilities
- **Statistics**: Customer interaction analytics

### 📈 Analytics & Monitoring
- **Comment Statistics**: Track comment volumes and patterns
- **Message Analytics**: Monitor message flow and response rates
- **Name Change Tracking**: Detect and log customer name changes
- **Real-time Updates**: WebSocket support for live updates
- **Health Monitoring**: System health check endpoints

### 🔄 Real-Time Features
- **WebSocket Support**: Live updates for dashboard
- **Background Processing**: Asynchronous document processing
- **Session Management**: Real-time session validation
- **Event Streaming**: Live comment and message notifications

## Technical Architecture

### Backend Technologies
- **Language**: Go (Golang)
- **Web Framework**: Fiber v2
- **Database**: MongoDB
- **Vector Database**: For RAG embeddings
- **Authentication**: Session-based with bcrypt
- **Real-time**: WebSocket support
- **AI Integration**: Claude AI API

### Project Structure
```
├── config/             # Configuration management
├── handlers/           # HTTP request handlers
│   ├── auth_handler.go       # Authentication endpoints
│   ├── comment_handler.go    # Comment management
│   ├── crm_handler.go        # CRM integration
│   ├── customer_handler.go   # Customer management
│   ├── message_handler.go    # Message handling
│   ├── rag_upload_handler.go # RAG document upload
│   └── websocket_handler.go  # WebSocket connections
├── middleware/         # Middleware functions
│   ├── auth.go               # Authentication middleware
│   └── page_auth.go          # Page-level authorization
├── models/             # Data models
│   ├── company.go            # Company structure
│   ├── customer.go           # Customer profiles
│   ├── session.go            # Session management
│   └── user.go               # User accounts
├── services/           # Business logic
│   ├── claude.go             # Claude AI integration
│   ├── crm_service.go        # CRM operations
│   ├── customer.go           # Customer service
│   ├── embedding_search.go   # RAG search
│   ├── mongodb.go            # Database operations
│   └── vectordb.go           # Vector database
├── webhooks/           # Facebook webhook handling
├── docs/               # Feature documentation
└── main.go            # Application entry point
```

### Key Services
- **MongoDB Service**: Database operations and indexing
- **CRM Service**: External CRM integration and data sync
- **Vector Service**: Document embedding and search
- **Claude Service**: AI response generation
- **Session Service**: Authentication and session management
- **Customer Service**: Customer data management

### Middleware
- **CORS**: Cross-origin resource sharing
- **Authentication**: Session validation
- **Authorization**: Role-based access control
- **Logging**: Structured JSON logging with slog
- **Recovery**: Panic recovery
- **Rate Limiting**: API rate limiting (if configured)

### Background Jobs
- **Session Cleanup**: Removes expired sessions
- **CRM Scheduler**: Periodic CRM data updates
- **Document Processing**: Asynchronous RAG document processing

## Prerequisites

- Go 1.19+
- MongoDB 4.4+
- Vector database (configured)
- Facebook App with webhook permissions
- Claude API key

## Configuration

### Environment Variables

Create a `.env` file:

```bash
cp .env.example .env
```

Edit `.env`:
```env
# MongoDB
MONGO_URI=mongodb://localhost:27017
DATABASE_NAME=facebook_bot

# Server
PORT=8080

# Claude AI
CLAUDE_API_KEY=your_claude_api_key
DEBUG_CLAUDE=false  # Set to true for debug mode

# Session
SESSION_SECRET=your_session_secret
SESSION_TIMEOUT=24h

# Facebook
FACEBOOK_APP_ID=your_app_id
FACEBOOK_APP_SECRET=your_app_secret
WEBHOOK_VERIFY_TOKEN=your_secure_verify_token
```

### Installation

```bash
# Clone repository
git clone [repository-url]

# Install dependencies
go mod download

# Set up environment variables
cp .env.example .env
# Edit .env with your configuration

# Run database migrations/setup
go run scripts/setup.go

# Start server
go run main.go
```

### Database Collections
- `companies` - Company and page configurations
- `users` - User accounts and roles
- `sessions` - Active user sessions
- `customers` - Customer profiles and settings
- `messages` - Messenger conversations
- `comments` - Facebook comments
- `posts` - Facebook posts
- `vector_documents` - RAG document embeddings
- `crm_links` - CRM integration endpoints
- `name_changes` - Customer name change history

### Production Deployment
- Use systemd or similar for process management
- Configure reverse proxy (nginx/caddy)
- Set up SSL/TLS certificates
- Configure monitoring and logging
- Set up backup strategies for MongoDB

### Configure Facebook Webhook

In your Facebook App settings:

1. Go to Webhooks settings
2. Add webhook URL: `https://your-domain.com/webhook`
3. Verify token: Use the value from `WEBHOOK_VERIFY_TOKEN` in .env
4. Subscribe to these fields:
   - `messages` - For Messenger messages
   - `feed` - For page post comments

## API Endpoints

### Authentication
- `POST /auth/login` - User login
- `POST /auth/logout` - User logout
- `GET /auth/me` - Get current user
- `GET /auth/check` - Check session validity

### Admin Management
- `GET /admin/company` - Get company information
- `POST /admin/company` - Create new company (super admin)
- `POST /admin/company/pages` - Add page to company
- `POST /admin/company/pages/full` - Add page with full configuration
- `PUT /admin/company/pages/:pageID` - Update page configuration
- `POST /admin/users` - Create new user
- `POST /admin/users/admin` - Admin user creation
- `PUT /admin/users/:userID/role` - Update user role
- `GET /admin/users` - List company users
- `GET /admin/users/:userID` - Get specific user

### Dashboard APIs

#### Pages & Posts
- `GET /api/dashboard/pages` - Get company pages
- `GET /api/dashboard/posts` - Get posts list
- `GET /api/dashboard/posts/company` - Get company posts

#### Comments
- `GET /api/dashboard/comments` - Get all comments
- `GET /api/dashboard/comments/post/:postID` - Get post comments
- `GET /api/dashboard/comments/post/:postID/hierarchical` - Hierarchical view
- `GET /api/dashboard/comments/post/:postID/threaded` - Threaded view
- `GET /api/dashboard/comments/:commentID/replies` - Get comment replies
- `GET /api/dashboard/stats` - Comment statistics
- `GET /api/dashboard/name-changes` - Track name changes

#### Messages
- `GET /api/dashboard/messages` - Get all messages
- `GET /api/dashboard/messages/:senderID` - Get user messages
- `GET /api/dashboard/messages/page/:pageID` - Page messages
- `GET /api/dashboard/messages/page/:pageID/conversations` - Page conversations
- `GET /api/dashboard/messages/chat/:chatID` - Chat messages
- `GET /api/dashboard/messages/conversation/:customerID` - Customer conversation
- `GET /api/dashboard/conversations` - User conversations
- `GET /api/dashboard/sender/:senderID/history` - Sender history

#### Customers
- `GET /api/dashboard/customers` - List customers
- `GET /api/dashboard/customers/search` - Search customers
- `GET /api/dashboard/customers/stats` - Customer statistics
- `GET /api/dashboard/customers/:customerID` - Customer details
- `PUT /api/dashboard/customers/:customerID/stop` - Update stop status
- `POST /api/dashboard/customers/:customerID/toggle-stop` - Toggle stop status
- `POST /api/dashboard/customers/:customerID/message` - Send message
- `PUT /api/dashboard/customers/:customerID/agent` - Assign agent
- `DELETE /api/dashboard/customers/:customerID/agent` - Remove agent
- `PUT /api/dashboard/customers/:customerID/assignment` - Update assignment

#### CRM Management
- `GET /api/dashboard/crm-links` - Get CRM links for channel
- `POST /api/dashboard/crm-links` - Add new CRM link
- `PUT /api/dashboard/crm-links/status` - Update CRM link status
- `DELETE /api/dashboard/crm-links` - Delete CRM link
- `GET /api/dashboard/crm-links/all` - Get all CRM data
- `GET /api/dashboard/pages/:pageID/crm-links` - Page CRM links
- `PUT /api/dashboard/pages/:pageID/crm-links/toggle` - Toggle CRM link
- `PUT /api/dashboard/pages/:pageID/crm-links` - Update CRM link

#### RAG Document Management
- `POST /api/dashboard/rag/upload` - Upload RAG document
- `DELETE /api/dashboard/rag/document` - Delete document
- `GET /api/dashboard/rag/documents` - List documents
- `PUT /api/dashboard/rag/document/toggle` - Toggle document status
- `PUT /api/dashboard/rag/document/channels` - Update document channels
- `GET /api/dashboard/rag/documents/details` - Document details
- `PUT /api/dashboard/rag/document/toggle-by-id` - Toggle by ID
- `GET /api/dashboard/rag/documents/by-filename` - Get by filename
- `GET /api/dashboard/rag/documents/all` - Get all documents

#### WebSocket
- `GET /api/dashboard/ws` - WebSocket connection for real-time updates

### Public APIs
- `GET /api/comments/post/:postID` - Public comment endpoint
- `GET /api/comments/:commentID` - Single comment with replies
- `GET /api/posts` - All post IDs
- `GET /api/posts/paginated` - Paginated posts
- `GET /api/posts/page/:pageID` - Posts by page
- `GET /api/posts/stats` - Post statistics
- `GET /health` - Health check

### Facebook Webhooks
- `GET /webhook` - Facebook webhook verification
- `POST /webhook` - Receives Facebook webhook events

## API Usage Examples

### Login
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password"}'
```

### Upload RAG Document
```bash
curl -X POST http://localhost:8080/api/dashboard/rag/upload \
  -H "Authorization: Bearer [token]" \
  -F "file=@document.txt" \
  -F "page_id=123456" \
  -F "channels=facebook,messenger"
```

### Add CRM Link
```bash
curl -X POST http://localhost:8080/api/dashboard/crm-links \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer [token]" \
  -d '{
    "page_id": "123456",
    "channels": ["facebook", "messenger"],
    "name": "Customer Database",
    "url": "https://api.crm.com/customers",
    "type": "api",
    "api_key": "crm_api_key",
    "is_active": true
  }'
```

### Toggle Customer Stop Status
```bash
curl -X POST http://localhost:8080/api/dashboard/customers/123/toggle-stop \
  -H "Authorization: Bearer [token]"
```

### Assign Agent to Customer
```bash
curl -X PUT http://localhost:8080/api/dashboard/customers/123/agent \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer [token]" \
  -d '{"agent_name":"John Doe"}'
```

## Security Features

1. **Authentication**
   - Secure password hashing with bcrypt
   - Session-based authentication
   - Automatic session expiration

2. **Authorization**
   - Role-based access control (Admin/User)
   - Company-level data isolation
   - Page-level permissions

3. **Data Protection**
   - MongoDB connection security
   - API key protection for external services
   - CORS configuration

4. **Input Validation**
   - Request body validation
   - File upload size limits (10MB max)
   - Content type verification

## Monitoring & Maintenance

### Health Checks
- `/health` endpoint for service monitoring
- Database connection monitoring
- Background job status tracking

### Logging
- Structured JSON logging with slog
- Log levels: Debug, Info, Warn, Error
- Request/response logging
- Error tracking

### Performance
- Database indexing on key fields
- Connection pooling
- Async processing for heavy operations
- Caching strategies for frequently accessed data

## Support & Documentation

For additional documentation and support:
- Check inline code documentation
- Review API endpoint handlers for detailed parameters
- Monitor application logs for debugging
- See `/docs` folder for feature-specific documentation:
  - `AGENT_ASSIGNMENT.md` - Agent assignment feature
  - `CRM_LINK_MANAGEMENT.md` - CRM integration guide
  - `RAG_FEATURE_DOCUMENTATION.md` - RAG system details
  - `WEBSOCKET_QUICKSTART.md` - WebSocket usage guide
  - And more...

## License

MIT