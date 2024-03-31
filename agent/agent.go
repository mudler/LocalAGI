package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mudler/local-agent-framework/action"
	"github.com/mudler/local-agent-framework/llm"
	"github.com/sashabaranov/go-openai"
)

type Agent struct {
	sync.Mutex
	options          *options
	Character        Character
	client           *openai.Client
	jobQueue         chan *Job
	actionContext    *action.ActionContext
	context          *action.ActionContext
	availableActions []Action

	currentConversation []openai.ChatCompletionMessage
}

func New(opts ...Option) (*Agent, error) {
	options, err := newOptions(opts...)
	if err != nil {
		if err != nil {
			err = fmt.Errorf("failed to set options: %v", err)
		}
		return nil, err
	}

	client := llm.NewClient(options.LLMAPI.APIKey, options.LLMAPI.APIURL)

	c := context.Background()
	if options.context != nil {
		c = options.context
	}

	ctx, cancel := context.WithCancel(c)
	a := &Agent{
		jobQueue:         make(chan *Job),
		options:          options,
		client:           client,
		Character:        options.character,
		context:          action.NewContext(ctx, cancel),
		availableActions: options.actions,
	}

	if a.options.randomIdentity {
		if err = a.generateIdentity(a.options.randomIdentityGuidance); err != nil {
			return a, fmt.Errorf("failed to generate identity: %v", err)
		}
	}

	return a, nil
}

// Ask is a pre-emptive, blocking call that returns the response as soon as it's ready.
// It discards any other computation.
func (a *Agent) Ask(text, image string) []string {
	//a.StopAction()
	j := NewJob(text, image)
	fmt.Println("Job created", text)
	a.jobQueue <- j
	fmt.Println("Waiting for result")

	return j.Result.WaitResult()
}

var ErrContextCanceled = fmt.Errorf("context canceled")

func (a *Agent) Stop() {
	a.Lock()
	defer a.Unlock()
	a.context.Cancel()
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

	fmt.Println("Agent is running")
	clearConvTimer := time.NewTicker(1 * time.Minute)
	for {
		fmt.Println("Agent loop")

		select {
		case job := <-a.jobQueue:
			fmt.Println("job from the queue")

			// Consume the job and generate a response
			// TODO: Give a short-term memory to the agent
			a.consumeJob(job)
		case <-a.context.Done():
			fmt.Println("Context canceled, agent is stopping...")

			// Agent has been canceled, return error
			return ErrContextCanceled
		case <-clearConvTimer.C:
			fmt.Println("Removing chat history...")

			// TODO: decide to do something on its own with the conversation result
			// before clearing it out

			// Clear the conversation
			a.currentConversation = []openai.ChatCompletionMessage{}
		}
	}
}

// StopAction stops the current action
// if any. Can be called before adding a new job.
func (a *Agent) StopAction() {
	a.Lock()
	defer a.Unlock()
	if a.actionContext != nil {
		a.actionContext.Cancel()
	}
}
