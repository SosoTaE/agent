package config

import "strings"

// PropertyPromptTemplates contains customizable prompt templates for property-related responses
type PropertyPromptTemplates struct {
	// GreetingTemplate is used when customers first inquire about properties
	GreetingTemplate string

	// PropertyListingTemplate is used when presenting available properties
	PropertyListingTemplate string

	// PricingDiscussionTemplate is used when discussing prices and payment terms
	PricingDiscussionTemplate string

	// SchedulingTemplate is used when arranging viewings or meetings
	SchedulingTemplate string

	// FollowUpTemplate is used for follow-up messages
	FollowUpTemplate string
}

// DefaultPropertyPrompts returns the default prompts for property inquiries
func DefaultPropertyPrompts() PropertyPromptTemplates {
	return PropertyPromptTemplates{
		GreetingTemplate: `Thank you for your interest in our properties! I have access to our complete inventory and can help you find exactly what you're looking for.`,

		PropertyListingTemplate: `Based on your requirements, here are the available options:
{{range .Properties}}
🏠 {{.Type}} #{{.Number}}
   📍 Floor: {{.Floor}} | 🏠 Rooms: {{.Rooms}}
   📐 Area: {{.Area}} sqm
   💰 Price: ${{.Price}}
   📊 Status: {{.Status}}
{{end}}
Would you like more details about any of these properties?`,

		PricingDiscussionTemplate: `Regarding pricing and payment options:
- Base Price: ${{.Price}}
- Full Payment Discount: {{.FullPaymentPercent}}%
- Payment Plans Available: Yes
- Reservation possible with initial deposit

Would you like to discuss specific payment arrangements?`,

		SchedulingTemplate: `I'd be happy to arrange a viewing for you. Our sales team is available:
- Monday-Friday: 10:00 AM - 7:00 PM
- Saturday: 10:00 AM - 5:00 PM
- Sunday: By appointment

What time works best for you?`,

		FollowUpTemplate: `Thank you for your interest! To provide you with the most accurate and up-to-date information, could you please share:
1. Your preferred property type (apartment/parking)
2. Desired number of rooms
3. Budget range
4. Any specific floor preferences

This will help me find the perfect match for you.`,
	}
}

// PropertyResponseEnhancers contains functions to enhance property-related responses
type PropertyResponseEnhancers struct {
	// EnableEmojis adds relevant emojis to responses
	EnableEmojis bool

	// IncludeContactInfo adds contact information to responses
	IncludeContactInfo bool

	// ShowComparisons shows property comparisons when multiple options exist
	ShowComparisons bool

	// ProvideMapLinks includes location/map links when available
	ProvideMapLinks bool
}

// GetEnhancedPropertyPrompt returns an enhanced system prompt for property inquiries
func GetEnhancedPropertyPrompt(companyName string, includeRAGInstructions bool) string {
	basePrompt := `You are a professional real estate consultant for ` + companyName + `.

LANGUAGE MATCHING RULE:
You MUST respond in the SAME LANGUAGE the customer uses:
- If they write in Georgian → respond in Georgian
- If they write in English → respond in English  
- If they write in Russian → respond in Russian
- Always match the customer's language for better service

CORE RESPONSIBILITIES:
1. Property Information Expert: Provide accurate, detailed information about available properties
2. Customer Needs Analysis: Understand customer requirements and match them with suitable properties
3. Price Negotiation Guide: Explain pricing, discounts, and payment options clearly
4. Scheduling Coordinator: Arrange property viewings and meetings
5. Follow-up Specialist: Maintain engagement and guide customers through the decision process

KEY BEHAVIORS:
- Always be specific with property details (never generic)
- Quote exact prices and specifications from the database
- Highlight unique selling points of each property
- Address concerns proactively
- Create urgency when appropriate (mention limited availability)
- Build trust through transparency and accuracy`

	if includeRAGInstructions {
		basePrompt += `

DATA USAGE REQUIREMENTS:
- MANDATORY: Read and analyze ALL provided property data before responding
- MANDATORY: Base all responses on actual data, not assumptions
- MANDATORY: Include specific details (unit numbers, prices, areas) in responses
- MANDATORY: Accurately represent availability status
- If data is unavailable, explicitly state this and offer to obtain it`
	}

	basePrompt += `

RESPONSE QUALITY STANDARDS:
✅ Specific and data-driven
✅ Customer-focused and helpful
✅ Professional yet friendly
✅ Action-oriented (next steps clear)
✅ Accurate pricing and availability

❌ Generic or vague information
❌ Made-up details or prices
❌ Pushy or aggressive sales tactics
❌ Ignoring customer questions
❌ Outdated or incorrect data`

	return basePrompt
}

// GetPropertySystemPrompt generates a complete system prompt for property-related interactions
func GetPropertySystemPrompt(companyName string, customInstructions string, hasRAGData bool) string {
	prompt := GetEnhancedPropertyPrompt(companyName, hasRAGData)

	if customInstructions != "" {
		prompt += "\n\nCOMPANY-SPECIFIC INSTRUCTIONS:\n" + customInstructions
	}

	if hasRAGData {
		prompt += "\n\n🔴 REMEMBER: You MUST use the provided database information in your responses. Never make up property details, prices, or availability."
	}

	return prompt
}

// PropertyKeywords contains keywords that trigger property-specific responses
var PropertyKeywords = []string{
	// English
	"apartment", "flat", "property", "real estate", "house", "home",
	"price", "cost", "payment", "mortgage", "rent", "buy", "purchase",
	"bedroom", "room", "bathroom", "kitchen", "living room",
	"floor", "area", "square meter", "sqm", "m2",
	"available", "availability", "vacant", "occupied", "sold",
	"viewing", "visit", "tour", "show", "see",
	"location", "address", "where", "building",
	"parking", "garage", "space",

	// Georgian (examples - add more as needed)
	"ბინა", "სახლი", "ფასი", "ოთახი", "სართული",
	"პარკინგი", "გარაჟი", "ხელმისაწვდომი",

	// Questions
	"how much", "what's available", "can I see", "when can", "is there",
	"do you have", "tell me about", "show me", "I'm looking for",
	"I want", "I need", "interested in",
}

// IsPropertyInquiry checks if a message is likely about properties
func IsPropertyInquiry(message string) bool {
	lowerMessage := strings.ToLower(message)
	for _, keyword := range PropertyKeywords {
		if strings.Contains(lowerMessage, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

// GeneratePropertyResponse creates a formatted response for property inquiries
func GeneratePropertyResponse(responseType string, data interface{}) string {
	templates := DefaultPropertyPrompts()

	switch responseType {
	case "greeting":
		return templates.GreetingTemplate
	case "scheduling":
		return templates.SchedulingTemplate
	case "follow_up":
		return templates.FollowUpTemplate
	default:
		return "I'd be happy to help you with property information. Could you please tell me more about what you're looking for?"
	}
}
