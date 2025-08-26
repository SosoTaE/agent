# Channel Separation Feature Documentation

## Overview
The Channel Separation feature enables independent configuration and management of Facebook page comments bot and Messenger bot for each Facebook page. This allows businesses to have different knowledge bases, settings, and behaviors for public Facebook comments versus private Messenger conversations.

## Architecture

### Key Components
1. **Channel-Specific Configurations**: Separate settings for Facebook and Messenger
2. **Channel-Aware Vector Storage**: Documents tagged with channel identifier
3. **Channel-Filtered RAG Retrieval**: Context retrieval filtered by channel
4. **Independent Toggle Controls**: Enable/disable each channel independently

### Data Model

#### FacebookPage Configuration
```javascript
{
  "page_id": "461998383671026",
  "page_name": "My Business Page",
  "page_access_token": "...",
  
  // Facebook Comments Configuration
  "facebook_config": {
    "is_enabled": true,              // Enable/disable Facebook comments bot
    "rag_enabled": true,              // Enable/disable RAG for Facebook
    "crm_links": [                   // CRM links specific to Facebook
      {
        "name": "Facebook Product Catalog",
        "url": "https://api.example.com/facebook-products",
        "type": "api",
        "is_active": true
      }
    ],
    "system_prompt": "You are a public-facing assistant..."  // Optional custom prompt
  },
  
  // Messenger Configuration
  "messenger_config": {
    "is_enabled": true,              // Enable/disable Messenger bot
    "rag_enabled": true,              // Enable/disable RAG for Messenger
    "crm_links": [                   // CRM links specific to Messenger
      {
        "name": "Customer Support KB",
        "url": "https://api.example.com/support-docs",
        "type": "api",
        "is_active": true
      }
    ],
    "system_prompt": "You are a private support assistant..."  // Optional custom prompt
  }
}
```

#### Vector Document with Channel
```javascript
{
  "_id": ObjectId,
  "company_id": "comp123",
  "page_id": "461998383671026",
  "channel": "facebook",        // NEW: "facebook" or "messenger"
  "content": "Document content",
  "embedding": [Float32Array],
  "source": "crm",
  "is_active": true,
  "metadata": {
    "filename": "products.pdf",
    "uploaded_by": "admin@company.com"
  }
}
```

## API Endpoints

### 1. Upload RAG Document with Channel
**Endpoint:** `POST /api/dashboard/rag/upload`

**Request:**
```bash
curl -X POST http://localhost:3000/api/dashboard/rag/upload \
  -H "Cookie: session=your_session_cookie" \
  -F "file=@document.pdf" \
  -F "page_id=461998383671026" \
  -F "channel=facebook" \
  -F "source=manual_upload"
```

**Form Fields:**
- `file` (required): Document file to upload
- `page_id` (required): Facebook page ID
- `channel` (required): **"facebook"** or **"messenger"**
- `source` (optional): Source identifier
- `metadata` (optional): Key-value pairs

**Response:**
```json
{
  "message": "File received and queued for processing",
  "filename": "document.pdf",
  "size": 245632,
  "source": "manual_upload",
  "page_id": "461998383671026",
  "channel": "facebook",
  "status": "processing"
}
```

### 2. List RAG Documents by Channel
**Endpoint:** `GET /api/dashboard/rag/documents?page_id={page_id}&channel={channel}`

**Parameters:**
- `page_id` (required): Facebook page ID
- `channel` (optional): Filter by "facebook" or "messenger"

**Response:**
```json
{
  "documents": [
    {
      "_id": "507f1f77bcf86cd799439011",
      "content": "Product catalog content...",
      "channel": "facebook",
      "source": "manual_upload",
      "is_active": true,
      "metadata": {
        "filename": "facebook_products.pdf"
      }
    },
    {
      "_id": "507f1f77bcf86cd799439012",
      "content": "Support documentation...",
      "channel": "messenger",
      "source": "manual_upload",
      "is_active": true,
      "metadata": {
        "filename": "support_docs.pdf"
      }
    }
  ],
  "facebook_count": 5,
  "messenger_count": 3,
  "total": 8
}
```

### 3. Toggle RAG Document Status
**Endpoint:** `PUT /api/dashboard/rag/document/toggle`

**Request:**
```json
{
  "company_id": "comp123",
  "page_id": "461998383671026",
  "channel": "facebook",
  "filename": "products.pdf",
  "is_active": false
}
```

### 4. Update Channel Configuration
**Endpoint:** `PUT /api/dashboard/pages/{page_id}/channel-config`

**Request:**
```json
{
  "channel": "facebook",
  "config": {
    "is_enabled": true,
    "rag_enabled": true,
    "system_prompt": "Custom prompt for Facebook comments"
  }
}
```

### 5. Toggle CRM Link for Channel
**Endpoint:** `PUT /api/dashboard/pages/{page_id}/crm-links/toggle`

**Request:**
```json
{
  "channel": "messenger",
  "crm_url": "https://api.example.com/support-docs",
  "is_active": true
}
```

## Channel Behavior

### Facebook Comments Bot
When a user comments on a Facebook post:

1. **Check Configuration**: System verifies if `facebook_config.is_enabled` is true
2. **RAG Context**: If `facebook_config.rag_enabled`:
   - Retrieves documents where `channel="facebook"`
   - Uses Facebook-specific CRM links
   - Applies Facebook-specific system prompt
3. **Response Generation**: AI generates response using Facebook context
4. **Public Visibility**: Response is posted as a public comment

### Messenger Bot
When a user sends a private message:

1. **Check Configuration**: System verifies if `messenger_config.is_enabled` is true
2. **RAG Context**: If `messenger_config.rag_enabled`:
   - Retrieves documents where `channel="messenger"`
   - Uses Messenger-specific CRM links
   - Applies Messenger-specific system prompt
3. **Response Generation**: AI generates response using Messenger context
4. **Private Response**: Response sent as private message

## Use Cases

### 1. Different Knowledge Bases
**Scenario:** Public vs Private Information

**Facebook Configuration:**
- Upload public product catalogs
- General company information
- Marketing materials
- Public FAQs

**Messenger Configuration:**
- Upload detailed support documentation
- Internal procedures
- Troubleshooting guides
- Customer-specific information

### 2. Different Response Styles
**Scenario:** Professional vs Casual Tone

**Facebook System Prompt:**
```
You are a professional brand representative responding to public comments.
Be formal, concise, and mindful that responses are visible to all.
```

**Messenger System Prompt:**
```
You are a friendly support agent in a private conversation.
Be conversational, detailed, and personalized in your responses.
```

### 3. Selective Channel Activation
**Scenario:** Gradual Rollout

**Phase 1:** Enable only Messenger bot
```json
{
  "facebook_config": { "is_enabled": false },
  "messenger_config": { "is_enabled": true }
}
```

**Phase 2:** Enable both channels
```json
{
  "facebook_config": { "is_enabled": true },
  "messenger_config": { "is_enabled": true }
}
```

### 4. Channel-Specific CRM Integration
**Scenario:** Different Data Sources

**Facebook CRM Links:**
- Product inventory API
- Pricing API
- Store locations API

**Messenger CRM Links:**
- Customer database API
- Order tracking API
- Support ticket API

## Implementation Details

### Channel Detection

#### Message Handler (Messenger)
```go
// Check if Messenger is enabled
if pageConfig.MessengerConfig != nil && !pageConfig.MessengerConfig.IsEnabled {
    slog.Info("Messenger is disabled for this page")
    return
}

// Get RAG context for Messenger channel
ragContext, err = services.GetRAGContextForChannel(
    ctx, messageText, companyID, pageID, "messenger"
)
```

#### Comment Handler (Facebook)
```go
// Check if Facebook comments are enabled
if pageConfig.FacebookConfig != nil && !pageConfig.FacebookConfig.IsEnabled {
    slog.Info("Facebook comments are disabled for this page")
    return
}

// Get RAG context for Facebook channel
ragContext, err = services.GetRAGContextForChannel(
    ctx, message, companyID, pageID, "facebook"
)
```

### Database Queries

#### Filtering by Channel
```javascript
// MongoDB query for channel-specific documents
{
  "company_id": "comp123",
  "page_id": "461998383671026",
  "channel": "facebook",    // Channel filter
  "is_active": true
}
```

#### Vector Search with Channel
```go
filter := bson.M{
    "company_id": companyID,
    "page_id":    pageID,
    "channel":    channel,  // Channel-specific search
    "is_active":  true,
}
```

## Migration Guide

### From Unified to Separated Channels

#### Step 1: Identify Current Documents
```javascript
// Find all existing documents without channel
db.vector_documents.find({
  channel: { $exists: false }
})
```

#### Step 2: Assign Default Channel
```javascript
// Assign all existing documents to both channels
db.vector_documents.updateMany(
  { channel: { $exists: false } },
  { $set: { channel: "facebook" } }  // or "messenger"
)
```

#### Step 3: Duplicate for Both Channels (Optional)
```javascript
// Create copies for the other channel
db.vector_documents.aggregate([
  { $match: { channel: "facebook" } },
  { $addFields: { channel: "messenger" } },
  { $merge: "vector_documents" }
])
```

## Configuration Examples

### E-commerce Store
```json
{
  "facebook_config": {
    "is_enabled": true,
    "rag_enabled": true,
    "system_prompt": "You are a product specialist. Focus on product features, availability, and promotions.",
    "crm_links": [
      { "name": "Product Catalog", "url": "...", "is_active": true },
      { "name": "Current Promotions", "url": "...", "is_active": true }
    ]
  },
  "messenger_config": {
    "is_enabled": true,
    "rag_enabled": true,
    "system_prompt": "You are a personal shopping assistant. Help with orders, returns, and detailed product questions.",
    "crm_links": [
      { "name": "Order System", "url": "...", "is_active": true },
      { "name": "Customer History", "url": "...", "is_active": true }
    ]
  }
}
```

### Customer Support
```json
{
  "facebook_config": {
    "is_enabled": true,
    "rag_enabled": false,  // No RAG for public comments
    "system_prompt": "Acknowledge the comment and invite to private message for support."
  },
  "messenger_config": {
    "is_enabled": true,
    "rag_enabled": true,
    "system_prompt": "Provide detailed technical support using the knowledge base.",
    "crm_links": [
      { "name": "Support KB", "url": "...", "is_active": true },
      { "name": "Ticket System", "url": "...", "is_active": true }
    ]
  }
}
```

## Best Practices

### 1. Content Segregation
- **Public (Facebook)**: General information, marketing content, public FAQs
- **Private (Messenger)**: Detailed support, sensitive information, personalized help

### 2. Document Organization
- Use clear naming conventions: `facebook_products.pdf`, `messenger_support.pdf`
- Tag documents with appropriate metadata
- Regular audits to ensure correct channel assignment

### 3. Testing Strategy
- Test each channel independently
- Verify RAG context isolation
- Confirm channel-specific behaviors

### 4. Monitoring
- Track response quality per channel
- Monitor RAG retrieval accuracy
- Log channel-specific metrics

## Troubleshooting

### Issue: Documents Appearing in Wrong Channel
**Solution:** 
1. Check document's channel field in database
2. Verify upload was done with correct channel parameter
3. Re-upload with correct channel if needed

### Issue: Channel Not Responding
**Solution:**
1. Verify `is_enabled` is true for the channel
2. Check if page configuration has channel config object
3. Review logs for "disabled for this page" messages

### Issue: Wrong Context Being Used
**Solution:**
1. Verify RAG is enabled for the channel
2. Check CRM links are active for the channel
3. Confirm documents exist for the specific channel

### Issue: Mixed Language Responses
**Solution:**
1. Ensure channel-specific system prompts include language instructions
2. Upload documents in the appropriate language per channel
3. Test with sample queries in target language

## Security Considerations

1. **Data Isolation**: Ensure sensitive information is only uploaded to appropriate channels
2. **Access Control**: Verify users have permission to manage both channels
3. **Audit Trail**: Log all channel configuration changes
4. **Content Review**: Regular audits of channel-specific content

## Performance Optimization

1. **Indexed Queries**: Ensure MongoDB indexes include channel field
2. **Caching**: Cache channel configurations to reduce database queries
3. **Lazy Loading**: Load channel-specific documents only when needed
4. **Batch Operations**: Process multiple documents for same channel together

## Future Enhancements

1. **Multiple Channels**: Support for Instagram, WhatsApp, etc.
2. **Channel Analytics**: Detailed metrics per channel
3. **A/B Testing**: Test different configurations per channel
4. **Auto-routing**: Automatically route documents to channels based on content
5. **Channel Templates**: Pre-configured templates for common use cases