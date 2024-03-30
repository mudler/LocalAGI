package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mudler/local-agent-framework/llm"
	"github.com/sashabaranov/go-openai"
)

type ActionContext struct {
	context.Context
	cancelFunc context.CancelFunc
}

type ActionParams map[string]string

func (ap ActionParams) Read(s string) error {
	err := json.Unmarshal([]byte(s), &ap)
	return err
}

type ActionDefinition openai.FunctionDefinition

func (a ActionDefinition) FD() openai.FunctionDefinition {
	return openai.FunctionDefinition(a)
}

// Actions is something the agent can do
type Action interface {
	ID() string
	Run(ActionParams) (string, error)
	Definition() ActionDefinition
}

var ErrContextCanceled = fmt.Errorf("context canceled")

func (a *Agent) Stop() {
	a.Lock()
	defer a.Unlock()
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
		a.actionContext.cancelFunc()
	}
}

func (a *Agent) consumeJob(job *Job) {

	// Consume the job and generate a response
	a.Lock()
	// Set the action context
	ctx, cancel := context.WithCancel(context.Background())
	a.actionContext = &ActionContext{
		Context:    ctx,
		cancelFunc: cancel,
	}
	a.Unlock()

	if job.Image != "" {
		// TODO: Use llava to explain the image content
	}

	if job.Text == "" {
		fmt.Println("no text!")
		return
	}

	actionChoice := struct {
		Choice string `json:"choice"`
	}{}

	llm.GenerateJSON(ctx, a.client, a.options.LLMAPI.Model, , &actionChoice)

	// https://github.com/sashabaranov/go-openai/blob/0925563e86c2fdc5011310aa616ba493989cfe0a/examples/completion-with-tool/main.go#L16
	actions := a.options.actions
	tools := []openai.Tool{}

	messages := a.currentConversation
	if job.Text != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    "user",
			Content: job.Text,
		})
	}

	for _, action := range actions {
		tools = append(tools, openai.Tool{
			Type:     openai.ToolTypeFunction,
			Function: action.Definition().FD(),
		})
	}

	decision := openai.ChatCompletionRequest{
		Model:    a.options.LLMAPI.Model,
		Messages: messages,
		Tools:    tools,
	}
	resp, err := a.client.CreateChatCompletion(ctx, decision)
	if err != nil || len(resp.Choices) != 1 {
		fmt.Printf("Completion error: err:%v len(choices):%v\n", err,
			len(resp.Choices))
		return
	}

	msg := resp.Choices[0].Message
	if len(msg.ToolCalls) != 1 {
		fmt.Printf("Completion error: len(toolcalls): %v\n", len(msg.ToolCalls))
		return
	}

	// simulate calling the function & responding to OpenAI
	messages = append(messages, msg)
	fmt.Printf("OpenAI called us back wanting to invoke our function '%v' with params '%v'\n",
		msg.ToolCalls[0].Function.Name, msg.ToolCalls[0].Function.Arguments)

	params := ActionParams{}
	if err := params.Read(msg.ToolCalls[0].Function.Arguments); err != nil {
		fmt.Printf("error unmarshalling arguments: %v\n", err)
		return
	}

	var result string
	for _, action := range actions {
		fmt.Println("Checking action: ", action.ID())
		fmt.Println("Checking action: ", msg.ToolCalls[0].Function.Name)
		if action.ID() == msg.ToolCalls[0].Function.Name {
			fmt.Printf("Running action: %v\n", action.ID())
			if result, err = action.Run(params); err != nil {
				fmt.Printf("error running action: %v\n", err)
				return
			}
		}
	}
	fmt.Printf("Action run result: %v\n", result)

	// simulate calling the function
	messages = append(messages, openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    result,
		Name:       msg.ToolCalls[0].Function.Name,
		ToolCallID: msg.ToolCalls[0].ID,
	})

	resp, err = a.client.CreateChatCompletion(ctx,
		openai.ChatCompletionRequest{
			Model:    a.options.LLMAPI.Model,
			Messages: messages,
			Tools:    tools,
		},
	)
	if err != nil || len(resp.Choices) != 1 {
		fmt.Printf("2nd completion error: err:%v len(choices):%v\n", err,
			len(resp.Choices))
		return
	}

	// display OpenAI's response to the original question utilizing our function
	msg = resp.Choices[0].Message
	fmt.Printf("OpenAI answered the original request with: %v\n",
		msg.Content)

	messages = append(messages, msg)
	a.currentConversation = append(a.currentConversation, messages...)

	if len(msg.ToolCalls) != 0 {
		fmt.Printf("OpenAI wants to call again functions: %v\n", msg)
		// wants to call again an action (?)
		job.Text = "" // Call the job with the current conversation
		job.Result.SetResult(result)
		a.jobQueue <- job
		return
	}

	// perform the action (if any)
	// or reply with a result
	// if there is an action...
	job.Result.SetResult(result)
	job.Result.Finish()
}
