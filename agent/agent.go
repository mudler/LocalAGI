package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
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
	options       *options
	Character     Character
	client        *openai.Client
	jobQueue      chan *Job
	actionContext *action.ActionContext
	context       *action.ActionContext

	currentReasoning    string
	currentState        *action.StateResult
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
		jobQueue:     make(chan *Job),
		options:      options,
		client:       client,
		Character:    options.character,
		currentState: &action.StateResult{},
		context:      action.NewContext(ctx, cancel),
	}

	if a.options.randomIdentity {
		if err = a.generateIdentity(a.options.randomIdentityGuidance); err != nil {
			return a, fmt.Errorf("failed to generate identity: %v", err)
		}
	}

	if a.options.statefile != "" {
		if _, err := os.Stat(a.options.statefile); err == nil {
			if err = a.LoadState(a.options.statefile); err != nil {
				return a, fmt.Errorf("failed to load state: %v", err)
			}
		}
	}

	if a.options.characterfile != "" {
		if _, err := os.Stat(a.options.characterfile); err == nil {
			// if there is a file, load the character back
			if err = a.LoadCharacter(a.options.characterfile); err != nil {
				return a, fmt.Errorf("failed to load character: %v", err)
			}
		} else {
			// otherwise save it for next time
			if err = a.SaveCharacter(a.options.characterfile); err != nil {
				return a, fmt.Errorf("failed to save character: %v", err)
			}
		}
	}

	if a.options.debugMode {
		fmt.Println("=== Agent in Debug mode ===")
		fmt.Println(a.Character.String())
		fmt.Println(a.State().String())
		fmt.Println("Permanent goal: ", a.options.permanentGoal)
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
func (a *Agent) Ask(opts ...JobOption) *JobResult {
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

func (a *Agent) runAction(chosenAction Action, decisionResult *decisionResult) (result string, err error) {
	for _, action := range a.systemActions() {
		if action.Definition().Name == chosenAction.Definition().Name {
			if result, err = action.Run(decisionResult.actionParams); err != nil {
				return "", fmt.Errorf("error running action: %w", err)
			}
		}
	}

	if a.options.debugMode {
		fmt.Println("Action", chosenAction.Definition().Name)
		fmt.Println("Result", result)
	}

	if chosenAction.Definition().Name.Is(action.StateActionName) {
		// We need to store the result in the state
		state := action.StateResult{}

		err = decisionResult.actionParams.Unmarshal(&state)
		if err != nil {
			return "", fmt.Errorf("error unmarshalling state of the agent: %w", err)
		}
		// update the current state with the one we just got from the action
		a.currentState = &state

		// update the state file
		if a.options.statefile != "" {
			if err := a.SaveState(a.options.statefile); err != nil {
				return "", err
			}
		}
	}

	return result, nil
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

	if chosenAction == nil {
		//job.Result.SetResult(ActionState{ActionCurrentState{nil, nil, "No action to do, just reply"}, ""})
		job.Result.Finish(fmt.Errorf("no action to do"))
		return
	}

	params, err := a.generateParameters(ctx, chosenAction, a.currentConversation)
	if err != nil {
		job.Result.Finish(fmt.Errorf("error generating action's parameters: %w", err))
		return
	}

	if params.actionParams == nil {
		job.Result.Finish(fmt.Errorf("no parameters"))
		return
	}

	if !job.Callback(ActionCurrentState{chosenAction, params.actionParams, reasoning}) {
		job.Result.SetResult(ActionState{ActionCurrentState{chosenAction, params.actionParams, reasoning}, "stopped by callback"})
		job.Result.Finish(nil)
		return
	}

	// If we don't have to reply , run the action!
	if !chosenAction.Definition().Name.Is(action.ReplyActionName) {
		result, err := a.runAction(chosenAction, params)
		if err != nil {
			job.Result.Finish(fmt.Errorf("error running action: %w", err))
			return
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
	// Here the LLM could decide to store some part of the conversation too in the memory
	evaluateMemory := NewJob(
		WithText(
			`Evaluate the current conversation and decide if we need to store some relevant informations from it`,
		))
	a.consumeJob(evaluateMemory, SystemRole)

	a.ResetConversation()

	// Here we go in a loop of
	// - asking the agent to do something
	// - evaluating the result
	// - asking the agent to do something else based on the result

	whatNext := NewJob(WithText("What should I do next?"))
	a.consumeJob(whatNext, SystemRole)

	doWork := NewJob(WithText("Try to fullfill our goals automatically"))
	a.consumeJob(doWork, SystemRole)

	results := []string{}
	for _, v := range doWork.Result.State {
		results = append(results, v.Result)
	}

	a.ResetConversation()

	// Here the LLM could decide to do something based on the result of our automatic action
	evaluateAction := NewJob(
		WithText(
			`Evaluate the current situation and decide if we need to execute other tools (for instance to store results into permanent, or short memory).
			We have done the following actions:
			` + strings.Join(results, "\n"),
		))
	a.consumeJob(evaluateAction, SystemRole)

	a.ResetConversation()
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
		case job := <-a.jobQueue:
			// Consume the job and generate a response
			// TODO: Give a short-term memory to the agent
			a.consumeJob(job, UserRole)
		case <-a.context.Done():
			// Agent has been canceled, return error
			return ErrContextCanceled
		case <-todoTimer.C:
			if !a.options.standaloneJob {
				continue
			}
			a.periodicallyRun()
		}
	}
}
