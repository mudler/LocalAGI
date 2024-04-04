package agent

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	"github.com/mudler/local-agent-framework/action"

	"github.com/sashabaranov/go-openai"
)

type ActionState struct {
	ActionCurrentState
	Result string
}

type ActionCurrentState struct {
	Action    Action
	Params    action.ActionParams
	Reasoning string
}

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
		return nil, err
	}

	msg := resp.Choices[0].Message
	if len(msg.ToolCalls) != 1 {
		return &decisionResult{message: msg.Content}, nil
	}

	params := action.ActionParams{}
	if err := params.Read(msg.ToolCalls[0].Function.Arguments); err != nil {
		return nil, err
	}

	return &decisionResult{actionParams: params}, nil
}

func (a *Agent) generateParameters(ctx context.Context, pickTemplate string, act Action, c []openai.ChatCompletionMessage, reasoning string) (*decisionResult, error) {

	// XXX: compressing conversation for generating parameters.. sucks!
	conversation, _, _, err := a.prepareConversationParse(pickTemplate, c, false, reasoning)
	if err != nil {
		return nil, err
	}

	return a.decision(ctx,
		conversation,
		a.systemActions().ToTools(),
		act.Definition().Name)
}

func (a *Agent) systemInternalActions() Actions {
	if a.options.enableHUD {
		return append(a.options.userActions, action.NewState())
	}

	return append(a.options.userActions)
}

func (a *Agent) systemActions() Actions {
	return append(a.systemInternalActions(), action.NewReply())
}

func (a *Agent) prepareHUD() PromptHUD {
	return PromptHUD{
		Character:     a.Character,
		CurrentState:  *a.currentState,
		PermanentGoal: a.options.permanentGoal,
	}
}

func (a *Agent) prepareConversationParse(templ string, messages []openai.ChatCompletionMessage, canReply bool, reasoning string) ([]openai.ChatCompletionMessage, Actions, []string, error) {
	// prepare the prompt
	prompt := bytes.NewBuffer([]byte{})

	promptTemplate, err := template.New("pickAction").Parse(templ)
	if err != nil {
		return nil, []Action{}, nil, err
	}

	actions := a.systemActions()
	if !canReply {
		actions = a.systemInternalActions()
	}

	// Get all the actions definitions
	definitions := []action.ActionDefinition{}
	for _, m := range actions {
		definitions = append(definitions, m.Definition())
	}

	var promptHUD *PromptHUD
	if a.options.enableHUD {
		h := a.prepareHUD()
		promptHUD = &h
	}

	err = promptTemplate.Execute(prompt, struct {
		HUD       *PromptHUD
		Actions   []action.ActionDefinition
		Reasoning string
		Messages  []openai.ChatCompletionMessage
	}{
		Actions:   definitions,
		Reasoning: reasoning,
		Messages:  messages,
		HUD:       promptHUD,
	})
	if err != nil {
		return nil, []Action{}, nil, err
	}

	if a.options.debugMode {
		fmt.Println("=== PROMPT START ===", prompt.String(), "=== PROMPT END ===")
	}

	// Get all the available actions IDs
	actionsID := []string{}
	for _, m := range actions {
		actionsID = append(actionsID, m.Definition().Name.String())
	}

	conversation := []openai.ChatCompletionMessage{}

	conversation = append(conversation, openai.ChatCompletionMessage{
		Role:    "user",
		Content: prompt.String(),
	})

	return conversation, actions, actionsID, nil
}

// pickAction picks an action based on the conversation
func (a *Agent) pickAction(ctx context.Context, templ string, messages []openai.ChatCompletionMessage, canReply bool) (Action, string, error) {
	conversation, actions, actionsID, err := a.prepareConversationParse(templ, messages, canReply, "")
	if err != nil {
		return nil, "", err
	}
	// Get the LLM to think on what to do
	thought, err := a.decision(ctx,
		conversation,
		Actions{action.NewReasoning()}.ToTools(),
		action.NewReasoning().Definition().Name)
	if err != nil {
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
		return nil, "", fmt.Errorf("failed to get the action tool parameters: %v", err)
	}

	actionChoice := action.IntentResponse{}

	if params.actionParams == nil {
		return nil, params.message, nil
	}

	err = params.actionParams.Unmarshal(&actionChoice)
	if err != nil {
		return nil, "", err
	}

	if actionChoice.Tool == "" || actionChoice.Tool == "none" {
		return nil, "", fmt.Errorf("no intent detected")
	}

	// Find the action
	chosenAction := actions.Find(actionChoice.Tool)
	if chosenAction == nil {
		return nil, "", fmt.Errorf("no action found for intent:" + actionChoice.Tool)
	}

	return chosenAction, actionChoice.Reasoning, nil
}
