package types

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
)

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

	pastActions         []*ActionRequest
	nextAction          *Action
	nextActionParams    *ActionParams
	nextActionReasoning string

	context context.Context
	cancel  context.CancelFunc

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

func (j *Job) SetNextAction(action *Action, params *ActionParams, reasoning string) {
	j.nextAction = action
	j.nextActionParams = params
	j.nextActionReasoning = reasoning
}

func (j *Job) AddPastAction(action Action, params *ActionParams) {
	j.pastActions = append(j.pastActions, &ActionRequest{
		Action: action,
		Params: params,
	})
}

func (j *Job) GetPastActions() []*ActionRequest {
	return j.pastActions
}

func (j *Job) GetNextAction() (*Action, *ActionParams, string) {
	return j.nextAction, j.nextActionParams, j.nextActionReasoning
}

func (j *Job) HasNextAction() bool {
	return j.nextAction != nil
}

func (j *Job) ResetNextAction() {
	j.nextAction = nil
	j.nextActionParams = nil
	j.nextActionReasoning = ""
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

	var ctx context.Context
	if j.context == nil {
		ctx = context.Background()
	} else {
		ctx = j.context
	}

	context, cancel := context.WithCancel(ctx)
	j.context = context
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
