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

const pickActionTemplate = `You can take any of the following tools: 

{{range .Actions -}}
- {{.Name}}: {{.Description }}
{{ end }}
To answer back to the user, use the "reply" tool.
Given the text below, decide which action to take and explain the detailed reasoning behind it. For answering without picking a choice, reply with 'none'.

{{range .Messages -}}
{{.Role}}{{if .FunctionCall}}(tool_call){{.FunctionCall}}{{end}}: {{if .FunctionCall}}{{.FunctionCall}}{{else if .ToolCalls -}}{{range .ToolCalls -}}{{.Name}} called with {{.Arguments}}{{end}}{{ else }}{{.Content -}}{{end}}
{{end}}
`

const reEvalTemplate = `You can take any of the following tools: 

{{range .Actions -}}
- {{.Name}}: {{.Description }}
{{ end }}
To answer back to the user, use the "reply" tool.
Given the text below, decide which action to take and explain the detailed reasoning behind it. For answering without picking a choice, reply with 'none'.

{{range .Messages -}}
{{.Role}}{{if .FunctionCall}}(tool_call){{.FunctionCall}}{{end}}: {{if .FunctionCall}}{{.FunctionCall}}{{else if .ToolCalls -}}{{range .ToolCalls -}}{{.Name}} called with {{.Arguments}}{{end}}{{ else }}{{.Content -}}{{end}}
{{end}}

We already have called tools. Evaluate the current situation and decide if we need to execute other tools or answer back with a result.`

const (
	UserRole      = "user"
	AssistantRole = "assistant"
	SystemRole    = "system"
)

type Agent struct {
	sync.Mutex
	options                *options
	Character              Character
	client                 *openai.Client
	jobQueue, selfJobQueue chan *Job
	actionContext          *action.ActionContext
	context                *action.ActionContext
	availableActions       []Action

	currentReasoning    string
	nextAction          Action
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
		selfJobQueue:     make(chan *Job),
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

// StopAction stops the current action
// if any. Can be called before adding a new job.
func (a *Agent) StopAction() {
	a.Lock()
	defer a.Unlock()
	if a.actionContext != nil {
		a.actionContext.Cancel()
	}
}

// Ask is a pre-emptive, blocking call that returns the response as soon as it's ready.
// It discards any other computation.
func (a *Agent) Ask(opts ...JobOption) []ActionState {
	a.StopAction()
	j := NewJob(opts...)
	//	fmt.Println("Job created", text)
	a.jobQueue <- j

	return j.Result.WaitResult()
}

func (a *Agent) CurrentConversation() []openai.ChatCompletionMessage {
	a.Lock()
	defer a.Unlock()
	return a.currentConversation
}

func (a *Agent) ResetConversation() {
	a.Lock()
	defer a.Unlock()
	a.currentConversation = []openai.ChatCompletionMessage{}
}

var ErrContextCanceled = fmt.Errorf("context canceled")

func (a *Agent) Stop() {
	a.Lock()
	defer a.Unlock()
	a.context.Cancel()
}

func (a *Agent) consumeJob(job *Job, role string) {
	// Consume the job and generate a response
	a.Lock()
	// Set the action context
	ctx, cancel := context.WithCancel(context.Background())
	a.actionContext = action.NewContext(ctx, cancel)
	a.Unlock()

	if job.Image != "" {
		// TODO: Use llava to explain the image content

	}

	if job.Text != "" {
		a.currentConversation = append(a.currentConversation, openai.ChatCompletionMessage{
			Role:    role,
			Content: job.Text,
		})
	}

	// choose an action first
	var chosenAction Action
	var reasoning string

	if a.currentReasoning != "" && a.nextAction != nil {
		// if we are being re-evaluated, we already have the action
		// and the reasoning. Consume it here and reset it
		chosenAction = a.nextAction
		reasoning = a.currentReasoning
		a.currentReasoning = ""
		a.nextAction = nil
	} else {
		var err error
		chosenAction, reasoning, err = a.pickAction(ctx, pickActionTemplate, a.currentConversation)
		if err != nil {
			job.Result.Finish(err)
			return
		}
	}

	if chosenAction == nil || chosenAction.Definition().Name.Is(action.ReplyActionName) {
		job.Result.SetResult(ActionState{ActionCurrentState{nil, nil, "No action to do, just reply"}, ""})
		job.Result.Finish(nil)
		return
	}

	params, err := a.generateParameters(ctx, chosenAction, a.currentConversation)
	if err != nil {
		job.Result.Finish(err)
		return
	}

	if !job.Callback(ActionCurrentState{chosenAction, params.actionParams, reasoning}) {
		job.Result.SetResult(ActionState{ActionCurrentState{chosenAction, params.actionParams, reasoning}, "stopped by callback"})
		job.Result.Finish(nil)
		return
	}

	if params.actionParams == nil {
		job.Result.Finish(fmt.Errorf("no parameters"))
		return
	}

	var result string
	for _, action := range a.options.actions {
		if action.Definition().Name == chosenAction.Definition().Name {
			if result, err = action.Run(params.actionParams); err != nil {
				job.Result.Finish(fmt.Errorf("error running action: %w", err))
				return
			}
		}
	}

	stateResult := ActionState{ActionCurrentState{chosenAction, params.actionParams, reasoning}, result}
	job.Result.SetResult(stateResult)
	job.CallbackWithResult(stateResult)

	// calling the function
	a.currentConversation = append(a.currentConversation, openai.ChatCompletionMessage{
		Role: "assistant",
		FunctionCall: &openai.FunctionCall{
			Name:      chosenAction.Definition().Name.String(),
			Arguments: params.actionParams.String(),
		},
	})

	// result of calling the function
	a.currentConversation = append(a.currentConversation, openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    result,
		Name:       chosenAction.Definition().Name.String(),
		ToolCallID: chosenAction.Definition().Name.String(),
	})

	//a.currentConversation = append(a.currentConversation, messages...)
	//a.currentConversation = messages

	// given the result, we can now ask OpenAI to complete the conversation or
	// to continue using another tool given the result
	followingAction, reasoning, err := a.pickAction(ctx, reEvalTemplate, a.currentConversation)
	if err != nil {
		job.Result.Finish(fmt.Errorf("error picking action: %w", err))
		return
	}

	if followingAction != nil &&
		!followingAction.Definition().Name.Is(action.ReplyActionName) &&
		!chosenAction.Definition().Name.Is(action.ReplyActionName) {
		// We need to do another action (?)
		// The agent decided to do another action
		// call ourselves again
		a.currentReasoning = reasoning
		a.nextAction = followingAction
		job.Text = ""
		a.consumeJob(job, role)
		return
	}

	// Generate a human-readable response
	resp, err := a.client.CreateChatCompletion(ctx,
		openai.ChatCompletionRequest{
			Model:    a.options.LLMAPI.Model,
			Messages: a.currentConversation,
		},
	)

	if err != nil {
		job.Result.Finish(err)
		return
	}

	if len(resp.Choices) != 1 {
		job.Result.Finish(fmt.Errorf("no enough choices: %w", err))
		return
	}

	// display OpenAI's response to the original question utilizing our function
	msg := resp.Choices[0].Message

	a.currentConversation = append(a.currentConversation, msg)
	job.Result.Finish(nil)
}

func (a *Agent) periodicallyRun() {
	a.consumeJob(NewJob(WithText("What should I do next?")), SystemRole)
	// TODO: decide to do something on its own with the conversation result
	// before clearing it out

	// Clear the conversation
	//	a.currentConversation = []openai.ChatCompletionMessage{}
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

	todoTimer := time.NewTicker(1 * time.Minute)
	for {
		select {
		case job := <-a.selfJobQueue:

			// XXX: is it needed?
			a.consumeJob(job, SystemRole)
		case job := <-a.jobQueue:

			// Consume the job and generate a response
			// TODO: Give a short-term memory to the agent
			a.consumeJob(job, UserRole)
		case <-a.context.Done():
			// Agent has been canceled, return error
			return ErrContextCanceled
		case <-todoTimer.C:
			a.periodicallyRun()
		}
	}
}
