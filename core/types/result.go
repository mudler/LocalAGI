package types

import (
	"context"
	"sync"

	"github.com/mudler/cogito"
	"github.com/sashabaranov/go-openai"
)

// JobResult is the result of a job
type JobResult struct {
	sync.Mutex
	// The result of a job
	State        []ActionState
	Plans        []cogito.PlanStatus
	Conversation []openai.ChatCompletionMessage

	Finalizers []func([]openai.ChatCompletionMessage)

	Response string
	Error    error
	ready    chan bool
}

// SetResult sets the result of a job
func (j *JobResult) SetResult(text ActionState) {
	j.Lock()
	defer j.Unlock()

	j.State = append(j.State, text)
}

// Finish marks the job as done and closes the ready channel.
func (j *JobResult) Finish(e error) {
	j.Lock()
	j.Error = e
	j.Unlock()

	close(j.ready)

	for _, f := range j.Finalizers {
		f(j.Conversation)
	}
	j.Finalizers = []func([]openai.ChatCompletionMessage){}
}

// AddFinalizer adds a finalizer to the job result
func (j *JobResult) AddFinalizer(f func([]openai.ChatCompletionMessage)) {
	j.Lock()
	defer j.Unlock()

	j.Finalizers = append(j.Finalizers, f)
}

// SetResult sets the result of a job
func (j *JobResult) SetResponse(response string) {
	j.Lock()
	defer j.Unlock()

	j.Response = response
}

// WaitResult waits for the result of a job
func (j *JobResult) WaitResult(ctx context.Context) (*JobResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-j.ready:
	}
	j.Lock()
	defer j.Unlock()
	return j, nil
}
