package agent

import (
	"sync"

	"github.com/sashabaranov/go-openai"
)

// Job is a request to the agent to do something
type Job struct {
	// The job is a request to the agent to do something
	// It can be a question, a command, or a request to do something
	// The agent will try to do it, and return a response
	Text                string
	Image               string // base64 encoded image
	Result              *JobResult
	reasoningCallback   func(ActionCurrentState) bool
	resultCallback      func(ActionState)
	conversationHistory []openai.ChatCompletionMessage
}

// JobResult is the result of a job
type JobResult struct {
	sync.Mutex
	// The result of a job
	State        []ActionState
	Conversation []openai.ChatCompletionMessage
	
	Response     string
	Error        error
	ready        chan bool
}

type JobOption func(*Job)

func WithConversationHistory(history []openai.ChatCompletionMessage) JobOption {
	return func(j *Job) {
		j.conversationHistory = history
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

func WithImage(image string) JobOption {
	return func(j *Job) {
		j.Image = image
	}
}

func WithText(text string) JobOption {
	return func(j *Job) {
		j.Text = text
	}
}

// NewJob creates a new job
// It is a request to the agent to do something
// It has a JobResult to get the result asynchronously
// To wait for a Job result, use JobResult.WaitResult()
func NewJob(opts ...JobOption) *Job {
	j := &Job{
		Result: NewJobResult(),
	}
	for _, o := range opts {
		o(j)
	}
	return j
}

// SetResult sets the result of a job
func (j *JobResult) SetResult(text ActionState) {
	j.Lock()
	defer j.Unlock()

	j.State = append(j.State, text)
}

// SetResult sets the result of a job
func (j *JobResult) Finish(e error) {
	j.Lock()
	defer j.Unlock()

	j.Error = e
	close(j.ready)
}

// SetResult sets the result of a job
func (j *JobResult) SetResponse(response string) {
	j.Lock()
	defer j.Unlock()

	j.Response = response
}

// WaitResult waits for the result of a job
func (j *JobResult) WaitResult() *JobResult {
	<-j.ready
	j.Lock()
	defer j.Unlock()
	return j
}
