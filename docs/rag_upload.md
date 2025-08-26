# RAG Document Upload

## Endpoint
`POST /api/dashboard/rag/upload`

## Request
**Content-Type:** `multipart/form-data`

### Form Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `file` | File | Yes | Text file (.txt, .md, .csv, .json), max 10MB |
| `page_id` | String | Yes | Facebook page ID |
| `channels` | String | No | Comma-separated: "facebook", "messenger" (default: both) |
| `source` | String | No | Source identifier (default: "manual_upload") |
| `metadata` | String | No | Key=value pairs, comma-separated |

## Example Request
```bash
curl -X POST https://your-api.com/api/rag/upload \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@document.txt" \
  -F "page_id=123456789" \
  -F "channels=facebook,messenger" \
  -F "metadata=category=faq,type=product"
```

## Response
**Status:** 202 Accepted
```json
{
  "message": "File received and queued for processing",
  "filename": "document.txt",
  "size": 15360,
  "source": "manual_upload",
  "page_id": "123456789",
  "channels": ["facebook", "messenger"],
  "status": "processing"
}
```

## Processing Details
- Files processed asynchronously in background
- Content split into 2000-character chunks
- Each chunk converted to embeddings
- Stored in vector database for semantic search
- Metadata preserved with each chunk