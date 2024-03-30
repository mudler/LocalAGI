package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/mudler/local-agent-framework/llm"
	"github.com/sashabaranov/go-openai"
)

type Agent struct {
	sync.Mutex
	options          *options
	Character        Character
	client           *openai.Client
	jobQueue         chan *Job
	actionContext    *ActionContext
	context          *ActionContext
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
		jobQueue:  make(chan *Job),
		options:   options,
		client:    client,
		Character: options.character,
		context: &ActionContext{
			Context:    ctx,
			cancelFunc: cancel,
		},
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
