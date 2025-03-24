package types

import (
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
)

// Job is a request to the agent to do something
type Job struct {
	// The job is a request to the agent to do something
	// It can be a question, a command, or a request to do something
	// The agent will try to do it, and return a response
	Result              *JobResult
	reasoningCallback   func(ActionCurrentState) bool
	resultCallback      func(ActionState)
	ConversationHistory []openai.ChatCompletionMessage
	UUID                string
	Metadata            map[string]interface{}
}

// JobResult is the result of a job
type JobResult struct {
	sync.Mutex
	// The result of a job
	State        []ActionState
	Conversation []openai.ChatCompletionMessage

	Response string
	Error    error
	ready    chan bool
}

type JobOption func(*Job)

func WithConversationHistory(history []openai.ChatCompletionMessage) JobOption {
	return func(j *Job) {
		j.ConversationHistory = history
	}
}

func WithReasoningCallback(f func(ActionCurrentState) bool) JobOption {
	return func(r *Job) {
		r.reasoningCallback = f
	}
}

func WithResultCallback(f func(ActionState)) JobOption {
	return func(r *Job) {
		r.resultCallback = f
	}
}

func WithMetadata(metadata map[string]interface{}) JobOption {
	return func(j *Job) {
		j.Metadata = metadata
	}
}

// NewJobResult creates a new job result
func NewJobResult() *JobResult {
	r := &JobResult{
		ready: make(chan bool),
	}
	return r
}

func (j *Job) Callback(stateResult ActionCurrentState) bool {
	if j.reasoningCallback == nil {
		return true
	}
	return j.reasoningCallback(stateResult)
}

func (j *Job) CallbackWithResult(stateResult ActionState) {
	if j.resultCallback == nil {
		return
	}
	j.resultCallback(stateResult)
}

func WithTextImage(text, image string) JobOption {
	return func(j *Job) {
		j.ConversationHistory = append(j.ConversationHistory, openai.ChatCompletionMessage{
			Role: "user",
			MultiContent: []openai.ChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: text,
				},
				{
					Type:     openai.ChatMessagePartTypeImageURL,
					ImageURL: &openai.ChatMessageImageURL{URL: image},
				},
			},
		})
	}
}

func WithText(text string) JobOption {
	return func(j *Job) {
		j.ConversationHistory = append(j.ConversationHistory, openai.ChatCompletionMessage{
			Role:    "user",
			Content: text,
		})
	}
}

func newUUID() string {
	// Generate UUID with google/uuid
	// https://pkg.go.dev/github.com/google/uuid

	// Generate a Version 4 UUID
	u, err := uuid.NewRandom()
	if err != nil {
		log.Fatalf("failed to generate UUID: %v", err)
	}

	return u.String()
}

// NewJob creates a new job
// It is a request to the agent to do something
// It has a JobResult to get the result asynchronously
// To wait for a Job result, use JobResult.WaitResult()
func NewJob(opts ...JobOption) *Job {
	j := &Job{
		Result: NewJobResult(),
		UUID:   newUUID(),
	}
	for _, o := range opts {
		o(j)
	}

	return j
}

func WithUUID(uuid string) JobOption {
	return func(j *Job) {
		j.UUID = uuid
	}
}
