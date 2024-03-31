package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"time"

	//"github.com/mudler/local-agent-framework/llm"

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

func (ap ActionParams) String() string {
	b, _ := json.Marshal(ap)
	return string(b)
}

type ActionDefinition openai.FunctionDefinition

func (a ActionDefinition) FD() openai.FunctionDefinition {
	return openai.FunctionDefinition(a)
}

// Actions is something the agent can do
type Action interface {
	ID() string
	Description() string
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

func (a *Agent) decision(ctx context.Context, conversation []openai.ChatCompletionMessage, tools []openai.Tool, toolchoice any) (ActionParams, error) {
	decision := openai.ChatCompletionRequest{
		Model:      a.options.LLMAPI.Model,
		Messages:   conversation,
		Tools:      tools,
		ToolChoice: toolchoice,
	}
	resp, err := a.client.CreateChatCompletion(ctx, decision)
	if err != nil || len(resp.Choices) != 1 {
		fmt.Println("no choices", err)

		return nil, err
	}

	msg := resp.Choices[0].Message
	if len(msg.ToolCalls) != 1 {
		return nil, fmt.Errorf("len(toolcalls): %v", len(msg.ToolCalls))
	}

	params := ActionParams{}
	if err := params.Read(msg.ToolCalls[0].Function.Arguments); err != nil {
		fmt.Println("can't read params", err)

		return nil, err
	}

	return params, nil
}

type Actions []Action

func (a Actions) ToTools() []openai.Tool {
	tools := []openai.Tool{}
	for _, action := range a {
		tools = append(tools, openai.Tool{
			Type:     openai.ToolTypeFunction,
			Function: action.Definition().FD(),
		})
	}
	return tools
}

func (a *Agent) generateParameters(ctx context.Context, action Action, conversation []openai.ChatCompletionMessage) (ActionParams, error) {
	return a.decision(ctx, conversation, a.options.actions.ToTools(), action.ID())
}

const pickActionTemplate = `You can take any of the following tools: 

{{range .Actions}}{{.ID}}: {{.Description}}{{end}}

or none. Given the text below, decide which action to take and explain the reasoning behind it. For answering without picking a choice, reply with 'none'.

{{range .Messages}}{{.Content}}{{end}}
`

func (a *Agent) pickAction(ctx context.Context, messages []openai.ChatCompletionMessage) (Action, error) {
	actionChoice := struct {
		Intent    string `json:"tool"`
		Reasoning string `json:"reasoning"`
	}{}

	prompt := bytes.NewBuffer([]byte{})
	tmpl, err := template.New("pickAction").Parse(pickActionTemplate)
	if err != nil {
		return nil, err
	}

	err = tmpl.Execute(prompt, struct {
		Actions  []Action
		Messages []openai.ChatCompletionMessage
	}{
		Actions:  a.options.actions,
		Messages: messages,
	})
	if err != nil {
		return nil, err
	}

	fmt.Println(prompt.String())

	actionsID := []string{}
	for _, m := range a.options.actions {
		actionsID = append(actionsID, m.ID())
	}
	intentionsTools := NewIntention(actionsID...)

	conversation := []openai.ChatCompletionMessage{
		{
			Role:    "user",
			Content: prompt.String(),
		},
	}

	params, err := a.decision(ctx, conversation, Actions{intentionsTools}.ToTools(), intentionsTools.ID())
	if err != nil {
		fmt.Println("failed decision", err)
		return nil, err
	}

	dat, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(dat, &actionChoice)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Action choice: %v\n", actionChoice)
	if actionChoice.Intent == "" || actionChoice.Intent == "none" {
		return nil, fmt.Errorf("no intent detected")
	}

	// Find the action
	var action Action
	for _, a := range a.options.actions {
		if a.ID() == actionChoice.Intent {
			action = a
			break
		}
	}

	if action == nil {
		fmt.Println("No action found for intent: ", actionChoice.Intent)
		return nil, fmt.Errorf("No action found for intent:" + actionChoice.Intent)
	}

	return action, nil
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

	messages := a.currentConversation
	if job.Text != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    "user",
			Content: job.Text,
		})
	}

	chosenAction, err := a.pickAction(ctx, messages)
	if err != nil {
		fmt.Printf("error picking action: %v\n", err)
		return
	}
	params, err := a.generateParameters(ctx, chosenAction, messages)
	if err != nil {
		fmt.Printf("error generating parameters: %v\n", err)
		return
	}

	var result string
	for _, action := range a.options.actions {
		fmt.Println("Checking action: ", action.ID(), chosenAction.ID())
		if action.ID() == chosenAction.ID() {
			fmt.Printf("Running action: %v\n", action.ID())
			if result, err = action.Run(params); err != nil {
				fmt.Printf("error running action: %v\n", err)
				return
			}
		}
	}
	fmt.Printf("Action run result: %v\n", result)

	// calling the function
	messages = append(messages, openai.ChatCompletionMessage{
		Role: "assistant",
		FunctionCall: &openai.FunctionCall{
			Name:      chosenAction.ID(),
			Arguments: params.String(),
		},
	})

	// result of calling the function
	messages = append(messages, openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    result,
		Name:       chosenAction.ID(),
		ToolCallID: chosenAction.ID(),
	})

	resp, err := a.client.CreateChatCompletion(ctx,
		openai.ChatCompletionRequest{
			Model:    a.options.LLMAPI.Model,
			Messages: messages,
			//		Tools:    tools,
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
