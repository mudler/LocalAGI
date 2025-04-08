package connectors_test

import (
	"time"

	"github.com/mudler/LocalAGI/services/connectors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sashabaranov/go-openai"
)

var _ = Describe("ConversationTracker", func() {
	var (
		tracker  *connectors.ConversationTracker[string]
		duration time.Duration
	)

	BeforeEach(func() {
		duration = 1 * time.Second
		tracker = connectors.NewConversationTracker[string](duration)
	})

	It("should initialize with empty conversations", func() {
		Expect(tracker.GetConversation("test")).To(BeEmpty())
	})

	It("should add a message and retrieve it", func() {
		message := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: "Hello",
		}
		tracker.AddMessage("test", message)
		conv := tracker.GetConversation("test")
		Expect(conv).To(HaveLen(1))
		Expect(conv[0]).To(Equal(message))
	})

	It("should clear the conversation after the duration", func() {
		message := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: "Hello",
		}
		tracker.AddMessage("test", message)
		time.Sleep(2 * time.Second)
		conv := tracker.GetConversation("test")
		Expect(conv).To(BeEmpty())
	})

	It("should keep the conversation within the duration", func() {
		message := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: "Hello",
		}
		tracker.AddMessage("test", message)
		time.Sleep(500 * time.Millisecond) // Half the duration
		conv := tracker.GetConversation("test")
		Expect(conv).To(HaveLen(1))
		Expect(conv[0]).To(Equal(message))
	})

	It("should handle multiple keys and clear old conversations", func() {
		message1 := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: "Hello 1",
		}
		message2 := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: "Hello 2",
		}

		tracker.AddMessage("key1", message1)
		tracker.AddMessage("key2", message2)

		time.Sleep(2 * time.Second)

		conv1 := tracker.GetConversation("key1")
		conv2 := tracker.GetConversation("key2")

		Expect(conv1).To(BeEmpty())
		Expect(conv2).To(BeEmpty())
	})

	It("should handle different key types", func() {
		trackerInt := connectors.NewConversationTracker[int](duration)
		trackerInt64 := connectors.NewConversationTracker[int64](duration)

		message := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: "Hello",
		}

		trackerInt.AddMessage(1, message)
		trackerInt64.AddMessage(int64(1), message)

		Expect(trackerInt.GetConversation(1)).To(HaveLen(1))
		Expect(trackerInt64.GetConversation(int64(1))).To(HaveLen(1))
	})

	It("should cleanup other conversations if older", func() {
		message := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: "Hello",
		}
		tracker.AddMessage("key1", message)
		tracker.AddMessage("key2", message)
		time.Sleep(2 * time.Second)
		tracker.GetConversation("key3")
		Expect(tracker.GetConversation("key1")).To(BeEmpty())
		Expect(tracker.GetConversation("key2")).To(BeEmpty())
	})
})
