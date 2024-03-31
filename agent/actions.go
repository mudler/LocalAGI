package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/mudler/local-agent-framework/action"

	"github.com/sashabaranov/go-openai"
)

// Actions is something the agent can do
type Action interface {
	Run(action.ActionParams) (string, error)
	Definition() action.ActionDefinition
}

type Actions []Action

func (a Actions) ToTools() []openai.Tool {
	tools := []openai.Tool{}
	for _, action := range a {
		tools = append(tools, openai.Tool{
			Type:     openai.ToolTypeFunction,
			Function: action.Definition().ToFunctionDefinition(),
		})
	}
	return tools
}

func (a *Agent) decision(ctx context.Context, conversation []openai.ChatCompletionMessage, tools []openai.Tool, toolchoice any) (action.ActionParams, error) {
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

	params := action.ActionParams{}
	if err := params.Read(msg.ToolCalls[0].Function.Arguments); err != nil {
		fmt.Println("can't read params", err)

		return nil, err
	}

	return params, nil
}

func (a *Agent) generateParameters(ctx context.Context, action Action, conversation []openai.ChatCompletionMessage) (action.ActionParams, error) {
	return a.decision(ctx, conversation, a.options.actions.ToTools(), action.Definition().Name)
}

const pickActionTemplate = `You can take any of the following tools: 

{{range .Actions}}{{.Name}}: {{.Description}}{{end}}

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
	definitions := []action.ActionDefinition{}
	for _, m := range a.options.actions {
		definitions = append(definitions, m.Definition())
	}
	err = tmpl.Execute(prompt, struct {
		Actions  []action.ActionDefinition
		Messages []openai.ChatCompletionMessage
	}{
		Actions:  definitions,
		Messages: messages,
	})
	if err != nil {
		return nil, err
	}

	fmt.Println(prompt.String())

	actionsID := []string{}
	for _, m := range a.options.actions {
		actionsID = append(actionsID, m.Definition().Name)
	}
	intentionsTools := action.NewIntention(actionsID...)

	conversation := []openai.ChatCompletionMessage{
		{
			Role:    "user",
			Content: prompt.String(),
		},
	}

	params, err := a.decision(ctx,
		conversation,
		Actions{intentionsTools}.ToTools(),
		intentionsTools.Definition().Name)
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
		if a.Definition().Name == actionChoice.Intent {
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
