package webhooks

// WebhookEvent represents the main webhook payload from Facebook
type WebhookEvent struct {
	Object string  `json:"object"`
	Entry  []Entry `json:"entry"`
}

// Entry represents a page entry in the webhook
type Entry struct {
	ID        string      `json:"id"`
	Time      int64       `json:"time"`
	Messaging []Messaging `json:"messaging,omitempty"`
	Changes   []Change    `json:"changes,omitempty"`
}

// Messaging represents a messaging event
type Messaging struct {
	Sender    User     `json:"sender"`
	Recipient User     `json:"recipient"`
	Timestamp int64    `json:"timestamp"`
	Message   *Message `json:"message,omitempty"`
}

// User represents a Facebook user
type User struct {
	ID string `json:"id"`
}

// Message represents a message
type Message struct {
	MID         string       `json:"mid"`
	Text        string       `json:"text"`
	QuickReply  *QuickReply  `json:"quick_reply,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// QuickReply represents a quick reply
type QuickReply struct {
	Payload string `json:"payload"`
}

// Attachment represents a message attachment
type Attachment struct {
	Type    string  `json:"type"`
	Payload Payload `json:"payload"`
}

// Payload represents attachment payload
type Payload struct {
	URL string `json:"url"`
}

// Change represents a feed change event
type Change struct {
	Field string      `json:"field"`
	Value ChangeValue `json:"value"`
}

// ChangeValue represents the value of a change
type ChangeValue struct {
	Item        string `json:"item"`
	CommentID   string `json:"comment_id"`
	PostID      string `json:"post_id"`
	ParentID    string `json:"parent_id"`
	SenderID    string `json:"sender_id"`
	SenderName  string `json:"sender_name"`
	Message     string `json:"message"`
	CreatedTime int64  `json:"created_time"` // Unix timestamp from Facebook
}
