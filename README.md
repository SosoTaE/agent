# Facebook Bot with Claude AI Integration

A Go-based Facebook webhook bot that automatically responds to comments and messages using Claude AI. Supports multiple companies/pages with individual configurations stored in MongoDB.

## Features

- ✅ Facebook webhook integration for comments and messages
- ✅ Claude AI integration for intelligent responses
- ✅ Multi-company support with individual configurations
- ✅ MongoDB for data persistence
- ✅ Structured logging with slog
- ✅ Response caching and rate limiting
- ✅ Custom system prompts per company

## Architecture

```
├── cmd/
│   └── setup/          # Company setup utility
├── config/             # Configuration management
├── handlers/           # Message and comment handlers
├── models/             # Data models
├── services/           # Business logic (MongoDB, Facebook, Claude)
├── webhooks/           # Webhook routing and types
└── main.go            # Application entry point
```

## Prerequisites

- Go 1.21+
- MongoDB
- Facebook App with webhook permissions
- Claude API key

## Setup

### 1. Environment Configuration

Create a `.env` file:

```bash
cp .env .env
```

Edit `.env`:
```env
MONGO_URI=mongodb://localhost:27017
MONGO_DB_NAME=facebook_bot
WEBHOOK_VERIFY_TOKEN=your_secure_verify_token
PORT=8080
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Add a Company Configuration

Use the setup utility to add a company:

```bash
go run cmd/setup/main.go \
  -company-id="company-001" \
  -company-name="My Company" \
  -page-id="YOUR_FB_PAGE_ID" \
  -page-name="My Facebook Page" \
  -page-token="YOUR_PAGE_ACCESS_TOKEN" \
  -app-secret="YOUR_APP_SECRET" \
  -claude-key="YOUR_CLAUDE_API_KEY" \
  -claude-model="claude-3-haiku-20240307" \
  -max-tokens=1024 \
  -response-delay=2
```

Optional: Add custom system prompt:
```bash
  -system-prompt="You are a friendly customer service agent for..."
```

### 4. Run the Server

```bash
go run main.go
```

The server will start on port 8080 (or the port specified in .env).

### 5. Configure Facebook Webhook

In your Facebook App settings:

1. Go to Webhooks settings
2. Add webhook URL: `https://your-domain.com/webhook`
3. Verify token: Use the value from `WEBHOOK_VERIFY_TOKEN` in .env
4. Subscribe to these fields:
   - `messages` - For Messenger messages
   - `feed` - For page post comments

## API Endpoints

### GET /webhook
Facebook webhook verification endpoint

### POST /webhook
Receives and processes Facebook webhook events

## Data Models

### Company
- Stores company-specific configurations
- Contains Facebook pages, Claude API keys, and custom settings
- Each company can have multiple Facebook pages

### Message
- Stores incoming chat messages

### Comment
- Stores Facebook post comments

### Response
- Stores AI-generated responses for auditing

## How It Works

1. **Webhook receives event** from Facebook (comment or message)
2. **Identify company** by matching the page ID
3. **Fetch company configuration** from MongoDB
4. **Generate AI response** using Claude with company-specific settings
5. **Send response** back to Facebook (comment reply or message)
6. **Store data** in MongoDB for auditing

## MongoDB Collections

- `companies` - Company configurations
- `messages` - Chat messages
- `comments` - Post comments
- `responses` - AI responses
- `page_cache` - Cached page information

## Adding Multiple Pages to a Company

To add more pages to an existing company, update the MongoDB document directly or create an admin API endpoint.

## Security Notes

- Store sensitive tokens in environment variables
- Use HTTPS in production
- Implement rate limiting for production use
- Regularly rotate access tokens
- Monitor webhook activity

## Monitoring

The application uses structured logging (slog) with JSON output for easy parsing by log aggregation tools.

## License

MIT