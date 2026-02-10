package types

import (
	"github.com/sashabaranov/go-openai"
)

// ConversationMessage represents a message with associated metadata
// Used when the agent initiates new conversations to preserve context
// such as generated images, files, or URLs
type ConversationMessage struct {
	Message  openai.ChatCompletionMessage
	Metadata map[string]interface{}
}

// NewConversationMessage creates a new ConversationMessage with the given message
func NewConversationMessage(msg openai.ChatCompletionMessage) *ConversationMessage {
	return &ConversationMessage{
		Message:  msg,
		Metadata: make(map[string]interface{}),
	}
}

// WithMetadata adds metadata to the conversation message
func (c *ConversationMessage) WithMetadata(metadata map[string]interface{}) *ConversationMessage {
	c.Metadata = metadata
	return c
}
