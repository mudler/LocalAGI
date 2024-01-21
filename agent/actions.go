package agent

import (
	"context"
	"fmt"

	"github.com/mudler/local-agent-framework/llm"
)

type ActionContext struct {
	context.Context
	cancelFunc context.CancelFunc
}

// Actions is something the agent can do
type Action interface {
	Description() string
	ID() string
	Run(map[string]string) error
}

var ErrContextCanceled = fmt.Errorf("context canceled")

func (a *Agent) Stop() {
	a.context.cancelFunc()
}

func (a *Agent) Run() error {
	// The agent run does two things:
	// picks up requests from a queue
	// and generates a response/perform actions

	// It is also preemptive.
	// That is, it can interrupt the current action
	// if another one comes in.

	// If there is no action, periodically evaluate if it has to do something on its own.

	// Expose a REST API to interact with the agent to ask it things

	for {
		select {
		case job := <-a.jobQueue:
			// Consume the job and generate a response
			a.consumeJob(job)
		case <-a.context.Done():
			// Agent has been canceled, return error
			return ErrContextCanceled
		}
	}
}

// StopAction stops the current action
// if any. Can be called before adding a new job.
func (a *Agent) StopAction() {
	if a.actionContext != nil {
		a.actionContext.cancelFunc()
	}
}

func (a *Agent) consumeJob(job *Job) {
	// Consume the job and generate a response
	// Implement your logic here

	// Set the action context
	ctx, cancel := context.WithCancel(context.Background())
	a.actionContext = &ActionContext{
		Context:    ctx,
		cancelFunc: cancel,
	}

	if job.Image != "" {
		// TODO: Use llava to explain the image content
	}

	if job.Text == "" {
		fmt.Println("no text!")
		return
	}

	decision := struct {
		Action string `json:"action"`
	}{
		Action: "generate_identity",
	}

	llm.GenerateJSON(ctx, a.client, a.options.LLMAPI.Model,
		"decide which action to take give the",
		&decision)

	// perform the action (if any)
	// or reply with a result

	// if there is an action...
	job.Result.SetResult("I don't know how to do that yet.")

}
