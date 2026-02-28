package types

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/mudler/cogito"
	"github.com/sashabaranov/go-openai"
)

// MetadataKeyConversationID is the job metadata key for per-conversation identity.
// When set (e.g. "slack:CHANNEL_ID", "telegram:CHAT_ID"), the agent may cancel the
// currently running job for that conversation before enqueueing a new one.
const MetadataKeyConversationID = "conversation_id"

// Job is a request to the agent to do something
type Job struct {
	// The job is a request to the agent to do something
	// It can be a question, a command, or a request to do something
	// The agent will try to do it, and return a response
	Result              *JobResult
	ReasoningCallback   func(ActionCurrentState) bool
	ResultCallback      func(ActionState)
	ConversationHistory []openai.ChatCompletionMessage
	UUID                string
	Metadata            map[string]interface{}
	DoneFilter          bool

	// Tools available for this job
	BuiltinTools []ActionDefinition // Built-in tools like web search
	UserTools    []ActionDefinition // User-defined function tools
	ToolChoice   string

	context  context.Context
	fragment *cogito.Fragment
	cancel   context.CancelFunc

	Obs *Observable
}

type ActionRequest struct {
	Action Action
	Params *ActionParams
}

type JobOption func(*Job)

func WithConversationHistory(history []openai.ChatCompletionMessage) JobOption {
	return func(j *Job) {
		j.ConversationHistory = history
	}
}

func WithBuiltinTools(tools []ActionDefinition) JobOption {
	return func(j *Job) {
		j.BuiltinTools = tools
	}
}

func WithUserTools(tools []ActionDefinition) JobOption {
	return func(j *Job) {
		j.UserTools = tools
	}
}

func WithToolChoice(choice string) JobOption {
	return func(j *Job) {
		j.ToolChoice = choice
	}
}

func WithReasoningCallback(f func(ActionCurrentState) bool) JobOption {
	return func(r *Job) {
		r.ReasoningCallback = f
	}
}

func WithResultCallback(f func(ActionState)) JobOption {
	return func(r *Job) {
		r.ResultCallback = f
	}
}

func WithMetadata(metadata map[string]any) JobOption {
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
	if j.ReasoningCallback == nil {
		return true
	}
	return j.ReasoningCallback(stateResult)
}

func (j *Job) CallbackWithResult(stateResult ActionState) {
	if j.ResultCallback == nil {
		return
	}
	j.ResultCallback(stateResult)
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
		Result:              NewJobResult(),
		UUID:                uuid.New().String(),
		Metadata:            make(map[string]interface{}),
		context:             context.Background(),
		ConversationHistory: []openai.ChatCompletionMessage{},
	}

	for _, opt := range opts {
		opt(j)
	}

	// Store the original request if it exists in the conversation history

	ctx, cancel := context.WithCancel(j.context)
	j.context = ctx
	j.cancel = cancel

	return j
}

func WithUUID(uuid string) JobOption {
	return func(j *Job) {
		j.UUID = uuid
	}
}

func WithContext(ctx context.Context) JobOption {
	return func(j *Job) {
		j.context = ctx
	}
}

func (j *Job) Cancel() {
	j.cancel()
}

func (j *Job) GetContext() context.Context {
	return j.context
}

func WithObservable(obs *Observable) JobOption {
	return func(j *Job) {
		j.Obs = obs
	}
}

// GetEvaluationLoop returns the current evaluation loop count
func (j *Job) GetEvaluationLoop() int {
	if j.Metadata == nil {
		j.Metadata = make(map[string]interface{})
	}
	if loop, ok := j.Metadata["evaluation_loop"].(int); ok {
		return loop
	}
	return 0
}

// IncrementEvaluationLoop increments the evaluation loop count
func (j *Job) IncrementEvaluationLoop() {
	if j.Metadata == nil {
		j.Metadata = make(map[string]interface{})
	}
	currentLoop := j.GetEvaluationLoop()
	j.Metadata["evaluation_loop"] = currentLoop + 1
}

// GetBuiltinTools returns the builtin tools for this job
func (j *Job) GetBuiltinTools() []ActionDefinition {
	return j.BuiltinTools
}

// GetUserTools returns the user tools for this job
func (j *Job) GetUserTools() []ActionDefinition {
	return j.UserTools
}

// GetAllTools returns all tools (builtin + user) for this job
func (j *Job) GetAllTools() []ActionDefinition {
	allTools := make([]ActionDefinition, 0, len(j.BuiltinTools)+len(j.UserTools))
	allTools = append(allTools, j.BuiltinTools...)
	allTools = append(allTools, j.UserTools...)
	return allTools
}
