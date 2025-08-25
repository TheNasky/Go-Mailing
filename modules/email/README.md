# Email Module

A performant, MongoDB-based email queuing and delivery system for the Go Framework.

## Features

- ✅ **MongoDB-based Queue**: No Redis required - uses MongoDB for job queuing
- ✅ **Background Processing**: Asynchronous email processing with worker pools
- ✅ **Multiple Providers**: Support for SMTP and SendGrid (easily extensible)
- ✅ **Priority Queuing**: High, normal, and low priority email processing
- ✅ **Automatic Retries**: Configurable retry mechanism for failed emails
- ✅ **Rate Limiting**: Built-in rate limiting per provider
- ✅ **Real-time Status**: Track email delivery status in real-time
- ✅ **Comprehensive Logging**: Detailed logging for debugging and monitoring
- ✅ **Health Monitoring**: Built-in health checks and statistics

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP API      │    │   Email Queue   │    │  Email Workers  │
│   (REST)        │◄──►│   (MongoDB)     │◄──►│  (Background)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   MongoDB       │    │   Rate Limiter  │    │  Email Provider │
│   (Logs, Queue) │    │   (MongoDB)     │    │  (SMTP/API)     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## API Endpoints

### Send Email
```http
POST /api/v1/emails/send
Content-Type: application/json

{
  "to": "recipient@example.com",
  "subject": "Your Subject",
  "html": "<html><body>Your HTML content</body></html>",
  "from": "noreply@yourdomain.com",
  "priority": 2
}
```

**Response:**
```json
{
  "status": "success",
  "message": "Email queued successfully",
  "payload": {
    "id": "507f1f77bcf86cd799439011",
    "status": "queued",
    "message": "Email queued successfully",
    "queued_at": "2024-01-01T10:00:00Z",
    "estimated_delivery": "2024-01-01T10:05:00Z"
  }
}
```

### Get Email Status
```http
GET /api/v1/emails/{id}/status
```

**Response:**
```json
{
  "status": "success",
  "message": "Email status retrieved successfully",
  "payload": {
    "id": "507f1f77bcf86cd799439011",
    "status": "sent",
    "to": "recipient@example.com",
    "subject": "Your Subject",
    "created_at": "2024-01-01T10:00:00Z",
    "processed_at": "2024-01-01T10:00:05Z",
    "provider": "smtp",
    "provider_msg_id": "msg_1704110405123456789"
  }
}
```

### Get Statistics
```http
GET /api/v1/emails/stats
```

**Response:**
```json
{
  "status": "success",
  "message": "Statistics retrieved successfully",
  "payload": {
    "total_queued": 150,
    "total_sent": 120,
    "total_failed": 5,
    "pending_count": 20,
    "processing_count": 5,
    "queue_size": 20
  }
}
```

### Health Check
```http
GET /api/v1/emails/health
```

## Configuration

### Environment Variables

#### SMTP Configuration
```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM=noreply@yourdomain.com
SMTP_MAX_EMAILS_PER_HOUR=1000
SMTP_MAX_EMAILS_PER_DAY=10000
```

#### SendGrid Configuration (Optional)
```bash
SENDGRID_API_KEY=your-sendgrid-api-key
SENDGRID_FROM=noreply@yourdomain.com
SENDGRID_MAX_EMAILS_PER_HOUR=10000
SENDGRID_MAX_EMAILS_PER_DAY=100000
```

### Worker Configuration

The email worker can be configured with the following settings:

```go
config := &workers.WorkerConfig{
    WorkerCount:     2,                    // Number of worker goroutines
    ProcessingDelay: 100 * time.Millisecond, // Delay between job checks
    MaxRetries:      3,                    // Maximum retry attempts
    RetryDelay:      5 * time.Minute,      // Delay between retries
}
```

## Usage Examples

### Basic Email Sending

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

func main() {
    emailData := map[string]interface{}{
        "to":      "user@example.com",
        "subject": "Welcome!",
        "html":    "<h1>Welcome to our service!</h1>",
        "from":    "noreply@yourdomain.com",
        "priority": 1, // High priority
    }
    
    jsonData, _ := json.Marshal(emailData)
    
    resp, err := http.Post(
        "http://localhost:8080/api/v1/emails/send",
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    
    if err == nil && resp.StatusCode == 201 {
        println("Email queued successfully!")
    }
}
```

### Check Email Status

```go
func checkEmailStatus(emailID string) {
    resp, err := http.Get(
        "http://localhost:8080/api/v1/emails/" + emailID + "/status",
    )
    
    if err == nil {
        // Parse response to get status
        println("Email status retrieved")
    }
}
```

## Performance Characteristics

### Queue Performance
- **Enqueue**: ~1000 operations/second
- **Dequeue**: ~500 operations/second
- **Status Queries**: ~2000 operations/second

### Email Processing
- **SMTP**: ~100 emails/second (depends on provider)
- **SendGrid**: ~1000 emails/second (API limits apply)
- **Worker Pool**: Configurable (default: 2 workers)

### Database Performance
- **MongoDB Indexes**: Optimized for queue operations
- **TTL Cleanup**: Automatic cleanup of old jobs (24 hours)
- **Connection Pooling**: Efficient MongoDB connection management

## Monitoring and Debugging

### Logs
The module provides comprehensive logging:
- Email queuing events
- Worker processing status
- Provider success/failure
- Queue statistics

### Health Checks
- Worker status monitoring
- Queue health indicators
- Provider availability
- Database connectivity

### Metrics
- Queue size monitoring
- Processing rates
- Success/failure ratios
- Provider performance

## Scaling Considerations

### Horizontal Scaling
- Multiple worker instances
- Load balancer for API endpoints
- Shared MongoDB cluster

### Performance Tuning
- Adjust worker count based on load
- Optimize MongoDB indexes
- Configure provider rate limits
- Implement caching for frequently accessed data

## Troubleshooting

### Common Issues

1. **Emails not being processed**
   - Check worker status
   - Verify MongoDB connectivity
   - Check provider configuration

2. **High queue latency**
   - Increase worker count
   - Check MongoDB performance
   - Verify provider rate limits

3. **Provider failures**
   - Check provider credentials
   - Verify network connectivity
   - Check provider quotas

### Debug Mode
Enable detailed logging by setting environment variables:
```bash
LOG_ROUTE=true
LOG_BODY=true
LOG_RESPONSE=true
```

## Development

### Running Tests
```bash
go run test_email_api.go
```

### Adding New Providers
1. Implement the `EmailProvider` interface
2. Add configuration options
3. Register in the service
4. Add tests

### Local Development
1. Set up MongoDB locally
2. Configure SMTP settings
3. Run the server
4. Test with the provided test script

## License

This module is part of the Go Framework and follows the same license terms.
