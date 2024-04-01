package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/mudler/local-agent-framework/action"
	"github.com/sashabaranov/go-openai"
)

// Job is a request to the agent to do something
type Job struct {
	// The job is a request to the agent to do something
	// It can be a question, a command, or a request to do something
	// The agent will try to do it, and return a response
	Text              string
	Image             string // base64 encoded image
	Result            *JobResult
	reasoningCallback func(Action, action.ActionParams, string)
	resultCallback    func(Action, action.ActionParams, string, string)
}

// JobResult is the result of a job
type JobResult struct {
	sync.Mutex
	// The result of a job
	Data  []string
	Error error
	ready chan bool
}

type JobOption func(*Job)

func WithReasoningCallback(f func(Action, action.ActionParams, string)) JobOption {
	return func(r *Job) {
		r.reasoningCallback = f
	}
}

func WithResultCallback(f func(Action, action.ActionParams, string, string)) JobOption {
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

func (j *Job) Callback(a Action, p action.ActionParams, s string) {
	if j.reasoningCallback == nil {
		return
	}
	j.reasoningCallback(a, p, s)
}

func (j *Job) CallbackWithResult(a Action, p action.ActionParams, s, r string) {
	if j.resultCallback == nil {
		return
	}
	j.resultCallback(a, p, s, r)
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
func (j *JobResult) SetResult(text string) {
	j.Lock()
	defer j.Unlock()

	j.Data = append(j.Data, text)
}

// SetResult sets the result of a job
func (j *JobResult) Finish(e error) {
	j.Lock()
	defer j.Unlock()

	j.Error = e
	close(j.ready)
}

// WaitResult waits for the result of a job
func (j *JobResult) WaitResult() []string {
	<-j.ready
	j.Lock()
	defer j.Unlock()
	return j.Data
}

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

func (a *Agent) consumeJob(job *Job) {
	// Consume the job and generate a response
	a.Lock()
	// Set the action context
	ctx, cancel := context.WithCancel(context.Background())
	a.actionContext = action.NewContext(ctx, cancel)
	a.Unlock()

	if job.Image != "" {
		// TODO: Use llava to explain the image content
	}

	messages := a.currentConversation
	if job.Text != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    "user",
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
		chosenAction, reasoning, err = a.pickAction(ctx, pickActionTemplate, messages)
		if err != nil {
			fmt.Printf("error picking action: %v\n", err)
			return
		}
	}

	if chosenAction == nil || chosenAction.Definition().Name.Is(action.ReplyActionName) {
		fmt.Println("No action to do, just reply")
		job.Result.SetResult(reasoning)
		return
	}

	params, err := a.generateParameters(ctx, chosenAction, messages)
	if err != nil {
		fmt.Printf("error generating parameters: %v\n", err)
		return
	}

	job.Callback(chosenAction, params.actionParams, reasoning)

	if params.actionParams == nil {
		fmt.Println("no parameters")
		return
	}

	var result string
	for _, action := range a.options.actions {
		fmt.Println("Checking action: ", action.Definition().Name, chosenAction.Definition().Name)
		if action.Definition().Name == chosenAction.Definition().Name {
			fmt.Printf("Running action: %v\n", action.Definition().Name)
			fmt.Printf("With parameters: %v\n", params.actionParams)
			if result, err = action.Run(params.actionParams); err != nil {
				fmt.Printf("error running action: %v\n", err)
				return
			}
		}
	}
	fmt.Printf("Action run result: %v\n", result)
	job.Result.SetResult(result)
	job.CallbackWithResult(chosenAction, params.actionParams, reasoning, result)

	// calling the function
	messages = append(messages, openai.ChatCompletionMessage{
		Role: "assistant",
		FunctionCall: &openai.FunctionCall{
			Name:      chosenAction.Definition().Name.String(),
			Arguments: params.actionParams.String(),
		},
	})

	// result of calling the function
	messages = append(messages, openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    result,
		Name:       chosenAction.Definition().Name.String(),
		ToolCallID: chosenAction.Definition().Name.String(),
	})

	a.currentConversation = append(a.currentConversation, messages...)

	// given the result, we can now ask OpenAI to complete the conversation or
	// to continue using another tool given the result
	followingAction, reasoning, err := a.pickAction(ctx, reEvalTemplate, messages)
	if err != nil {
		fmt.Printf("error picking action: %v\n", err)
		return
	}

	if followingAction == nil || followingAction.Definition().Name.Is(action.ReplyActionName) {
		fmt.Println("No action to do, just reply")
	} else if !chosenAction.Definition().Name.Is(action.ReplyActionName) {
		// We need to do another action (?)
		// The agent decided to do another action
		// call ourselves again
		a.currentReasoning = reasoning
		a.nextAction = followingAction
		job.Text = ""
		a.consumeJob(job)
		return
	}

	// Generate a human-readable response
	resp, err := a.client.CreateChatCompletion(ctx,
		openai.ChatCompletionRequest{
			Model:    a.options.LLMAPI.Model,
			Messages: messages,
		},
	)
	if err != nil || len(resp.Choices) != 1 {
		fmt.Printf("2nd completion error: err:%v len(choices):%v\n", err,
			len(resp.Choices))
		return
	}

	// display OpenAI's response to the original question utilizing our function
	msg := resp.Choices[0].Message
	fmt.Printf("OpenAI answered the original request with: %v\n",
		msg.Content)

	a.currentConversation = append(a.currentConversation, msg)
	job.Result.Finish(nil)
}
