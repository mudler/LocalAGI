package agent

import (
	"context"
	"errors"

	"github.com/mudler/cogito"
	"github.com/sashabaranov/go-openai"

	"github.com/mudler/LocalAGI/core/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type recordingRAGDB struct {
	stored []string
}

func (r *recordingRAGDB) Store(s string) error {
	r.stored = append(r.stored, s)
	return nil
}
func (r *recordingRAGDB) Reset() error                         { return nil }
func (r *recordingRAGDB) Search(string, int) ([]string, error) { return nil, nil }
func (r *recordingRAGDB) Count() int                           { return len(r.stored) }

type failingLLM struct{}

func (failingLLM) Ask(context.Context, cogito.Fragment) (cogito.Fragment, error) {
	return cogito.Fragment{}, errors.New("summary backend unavailable")
}

func (failingLLM) CreateChatCompletion(context.Context, openai.ChatCompletionRequest) (cogito.LLMReply, cogito.LLMUsage, error) {
	return cogito.LLMReply{}, cogito.LLMUsage{}, errors.New("summary backend unavailable")
}

var _ = Describe("saveCurrentConversation", func() {
	conv := Messages{
		{Role: "user", Content: "remember that my favourite colour is green"},
		{Role: "assistant", Content: "noted"},
	}

	newMemoryAgent := func(opts *options) *Agent {
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		return &Agent{
			options:   opts,
			Character: Character{Name: "memory-test"},
			context:   types.NewActionContext(ctx, cancel),
		}
	}

	It("does not panic when long term memory is enabled without a RAG DB", func() {
		a := newMemoryAgent(&options{enableLongTermMemory: true})
		Expect(func() { a.saveCurrentConversation(conv) }).NotTo(Panic())
	})

	It("does not panic when summary memory is enabled without a RAG DB", func() {
		// llm is nil on purpose: with no RAG DB the summary must not be requested at all.
		a := newMemoryAgent(&options{enableSummaryMemory: true})
		Expect(func() { a.saveCurrentConversation(conv) }).NotTo(Panic())
	})

	It("does not panic or store anything when the summary request fails", func() {
		db := &recordingRAGDB{}
		a := newMemoryAgent(&options{enableSummaryMemory: true, ragdb: db})
		a.llm = failingLLM{}
		Expect(func() { a.saveCurrentConversation(conv) }).NotTo(Panic())
		Expect(db.stored).To(BeEmpty())
	})

	It("still stores user messages when a RAG DB is configured", func() {
		db := &recordingRAGDB{}
		a := newMemoryAgent(&options{enableLongTermMemory: true, ragdb: db})
		a.saveCurrentConversation(conv)
		Expect(db.stored).To(Equal([]string{"remember that my favourite colour is green"}))
	})
})
