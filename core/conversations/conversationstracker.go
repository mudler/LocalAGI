package conversations

import (
	"fmt"
	"sync"
	"time"

	"github.com/mudler/xlog"
	"github.com/sashabaranov/go-openai"
)

type TrackerKey interface{ ~int | ~int64 | ~string }

type ConversationTracker[K TrackerKey] struct {
	convMutex           sync.Mutex
	currentconversation map[K][]openai.ChatCompletionMessage
	lastMessageTime     map[K]time.Time
	lastMessageDuration time.Duration
}

func NewConversationTracker[K TrackerKey](lastMessageDuration time.Duration) *ConversationTracker[K] {
	return &ConversationTracker[K]{
		lastMessageDuration: lastMessageDuration,
		currentconversation: map[K][]openai.ChatCompletionMessage{},
		lastMessageTime:     map[K]time.Time{},
	}
}

func (c *ConversationTracker[K]) GetConversation(key K) []openai.ChatCompletionMessage {
	// Lock the conversation mutex to update the conversation history
	c.convMutex.Lock()
	defer c.convMutex.Unlock()

	// Clear up the conversation if the last message was sent more than lastMessageDuration ago
	currentConv := []openai.ChatCompletionMessage{}
	lastMessageTime := c.lastMessageTime[key]
	if lastMessageTime.IsZero() {
		lastMessageTime = time.Now()
	}
	if lastMessageTime.Add(c.lastMessageDuration).Before(time.Now()) {
		currentConv = []openai.ChatCompletionMessage{}
		c.lastMessageTime[key] = time.Now()
		xlog.Debug("Conversation history does not exist for", "key", fmt.Sprintf("%v", key))
	} else {
		xlog.Debug("Conversation history exists for", "key", fmt.Sprintf("%v", key))
		currentConv = append(currentConv, c.currentconversation[key]...)
	}

	// cleanup other conversations if older
	for k := range c.currentconversation {
		lastMessage, exists := c.lastMessageTime[k]
		if !exists {
			delete(c.currentconversation, k)
			delete(c.lastMessageTime, k)
			continue
		}
		if lastMessage.Add(c.lastMessageDuration).Before(time.Now()) {
			xlog.Debug("Cleaning up conversation for", k)
			delete(c.currentconversation, k)
			delete(c.lastMessageTime, k)
		}
	}

	return currentConv

}

func (c *ConversationTracker[K]) AddMessage(key K, message openai.ChatCompletionMessage) {
	// Lock the conversation mutex to update the conversation history
	c.convMutex.Lock()
	defer c.convMutex.Unlock()

	c.currentconversation[key] = append(c.currentconversation[key], message)
	c.lastMessageTime[key] = time.Now()
}

func (c *ConversationTracker[K]) SetConversation(key K, messages []openai.ChatCompletionMessage) {
	// Lock the conversation mutex to update the conversation history
	c.convMutex.Lock()
	defer c.convMutex.Unlock()

	c.currentconversation[key] = messages
	c.lastMessageTime[key] = time.Now()
}
