package agent

import (
	"bytes"
	"context"
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

func (a Actions) Find(name string) Action {
	for _, action := range a {
		if action.Definition().Name.Is(name) {
			return action
		}
	}
	return nil
}

type decisionResult struct {
	actionParams action.ActionParams
	message      string
}

// decision forces the agent to take on of the available actions
func (a *Agent) decision(
	ctx context.Context,
	conversation []openai.ChatCompletionMessage,
	tools []openai.Tool, toolchoice any) (*decisionResult, error) {

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
		fmt.Println(msg)
		return &decisionResult{message: msg.Content}, nil
	}

	params := action.ActionParams{}
	if err := params.Read(msg.ToolCalls[0].Function.Arguments); err != nil {
		fmt.Println("can't read params", err)
		return nil, err
	}

	return &decisionResult{actionParams: params}, nil
}

func (a *Agent) generateParameters(ctx context.Context, action Action, conversation []openai.ChatCompletionMessage) (*decisionResult, error) {
	return a.decision(ctx,
		conversation,
		a.options.actions.ToTools(),
		action.Definition().Name)
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

// pickAction picks an action based on the conversation
func (a *Agent) pickAction(ctx context.Context, templ string, messages []openai.ChatCompletionMessage) (Action, string, error) {
	// prepare the prompt
	prompt := bytes.NewBuffer([]byte{})

	tmpl, err := template.New("pickAction").Parse(templ)
	if err != nil {
		return nil, "", err
	}

	// Get all the actions definitions
	definitions := []action.ActionDefinition{action.NewReply().Definition()}
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
		return nil, "", err
	}

	fmt.Println("=== PROMPT START ===", prompt.String(), "=== PROMPT END ===")

	// Get all the available actions IDs
	actionsID := []string{}
	for _, m := range a.options.actions {
		actionsID = append(actionsID, m.Definition().Name.String())
	}

	conversation := []openai.ChatCompletionMessage{
		{
			Role:    "user",
			Content: prompt.String(),
		},
	}

	// Get the LLM to think on what to do
	thought, err := a.decision(ctx,
		conversation,
		Actions{action.NewReasoning()}.ToTools(),
		action.NewReasoning().Definition().Name)
	if err != nil {
		fmt.Println("failed thinking", err)
		return nil, "", err
	}
	reason := ""
	response := &action.ReasoningResponse{}
	if thought.actionParams != nil {
		if err := thought.actionParams.Unmarshal(response); err != nil {
			return nil, "", err
		}
		reason = response.Reasoning
	}
	if thought.message != "" {
		reason = thought.message
	}

	fmt.Println("---- Thought: " + reason)

	// Decode tool call
	intentionsTools := action.NewIntention(actionsID...)
	params, err := a.decision(ctx,
		append(conversation, openai.ChatCompletionMessage{
			Role:    "assistent",
			Content: reason,
		}),
		Actions{intentionsTools}.ToTools(),
		intentionsTools.Definition().Name)
	if err != nil {
		fmt.Println("failed decision", err)
		return nil, "", err
	}

	actionChoice := action.IntentResponse{}

	if params.actionParams == nil {
		return nil, params.message, nil
	}

	err = params.actionParams.Unmarshal(&actionChoice)
	if err != nil {
		return nil, "", err
	}

	fmt.Printf("Action choice: %v\n", actionChoice)

	if actionChoice.Tool == "" || actionChoice.Tool == "none" {
		return nil, "", fmt.Errorf("no intent detected")
	}

	// Find the action
	chosenAction := append(a.options.actions, action.NewReply()).Find(actionChoice.Tool)
	if chosenAction == nil {
		fmt.Println("No action found for intent: ", actionChoice.Tool)
		return nil, "", fmt.Errorf("No action found for intent:" + actionChoice.Tool)
	}

	return chosenAction, actionChoice.Reasoning, nil
}
