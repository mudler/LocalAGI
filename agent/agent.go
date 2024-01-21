package agent

import (
	"context"
	"fmt"

	"github.com/mudler/local-agent-framework/llm"
	"github.com/sashabaranov/go-openai"
)

type Agent struct {
	options          *options
	Character        Character
	client           *openai.Client
	jobQueue         chan *Job
	actionContext    *ActionContext
	context          *ActionContext
	availableActions []Action
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
		options:   options,
		client:    client,
		Character: options.character,
		context: &ActionContext{
			Context:    ctx,
			cancelFunc: cancel,
		},
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
func (a *Agent) Ask(text, image string) string {
	a.StopAction()
	j := NewJob(text, image)
	a.jobQueue <- j
	return j.Result.WaitResult()
}
