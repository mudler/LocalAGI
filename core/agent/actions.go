package agent

import (
	"context"
	"fmt"

	"github.com/mudler/LocalAgent/core/action"
	"github.com/mudler/LocalAgent/pkg/xlog"

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
	Run(ctx context.Context, action action.ActionParams) (action.ActionResult, error)
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
	actioName    string
}

// decision forces the agent to take one of the available actions
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
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) != 1 {
		return nil, fmt.Errorf("no choices: %d", len(resp.Choices))
	}

	msg := resp.Choices[0].Message
	if len(msg.ToolCalls) != 1 {
		return &decisionResult{message: msg.Content}, nil
	}

	params := action.ActionParams{}
	if err := params.Read(msg.ToolCalls[0].Function.Arguments); err != nil {
		return nil, err
	}

	return &decisionResult{actionParams: params, actioName: msg.ToolCalls[0].Function.Name, message: msg.Content}, nil
}

type Messages []openai.ChatCompletionMessage

func (m Messages) ToOpenAI() []openai.ChatCompletionMessage {
	return []openai.ChatCompletionMessage(m)
}

func (m Messages) String() string {
	s := ""
	for _, cc := range m {
		s += cc.Role + ": " + cc.Content + "\n"
	}
	return s
}

func (m Messages) Exist(content string) bool {
	for _, cc := range m {
		if cc.Content == content {
			return true
		}
	}
	return false
}

func (a *Agent) generateParameters(ctx context.Context, pickTemplate string, act Action, c []openai.ChatCompletionMessage, reasoning string) (*decisionResult, error) {

	stateHUD, err := renderTemplate(pickTemplate, a.prepareHUD(), a.systemInternalActions(), reasoning)
	if err != nil {
		return nil, err
	}

	// check if there is already a message with the hud in the conversation already, otherwise
	// add a message at the top with it

	conversation := c

	if !Messages(c).Exist(stateHUD) && a.options.enableHUD {
		conversation = append([]openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: stateHUD,
			},
		}, conversation...)
	}

	cc := conversation
	if a.options.forceReasoning {
		cc = append(conversation, openai.ChatCompletionMessage{
			Role:    "system",
			Content: fmt.Sprintf("The agent decided to use the tool %s with the following reasoning: %s", act.Definition().Name, reasoning),
		})
	}

	return a.decision(ctx,
		cc,
		a.systemInternalActions().ToTools(),
		openai.ToolChoice{
			Type:     openai.ToolTypeFunction,
			Function: openai.ToolFunction{Name: act.Definition().Name.String()},
		},
	)
}

func (a *Agent) systemInternalActions() Actions {
	//	defaultActions := append(a.options.userActions, action.NewReply())
	defaultActions := a.options.userActions

	if a.options.initiateConversations && a.selfEvaluationInProgress { // && self-evaluation..
		acts := append(defaultActions, action.NewConversation())
		if a.options.enableHUD {
			acts = append(acts, action.NewState())
		}
		//if a.options.canStopItself {
		//		acts = append(acts, action.NewStop())
		//	}

		return acts
	}

	if a.options.canStopItself {
		acts := append(defaultActions, action.NewStop())
		if a.options.enableHUD {
			acts = append(acts, action.NewState())
		}
		return acts
	}

	if a.options.enableHUD {
		return append(defaultActions, action.NewState())
	}

	return defaultActions
}

func (a *Agent) prepareHUD() (promptHUD *PromptHUD) {
	if !a.options.enableHUD {
		return nil
	}

	return &PromptHUD{
		Character:     a.Character,
		CurrentState:  *a.currentState,
		PermanentGoal: a.options.permanentGoal,
		ShowCharacter: a.options.showCharacter,
	}
}

// pickAction picks an action based on the conversation
func (a *Agent) pickAction(ctx context.Context, templ string, messages []openai.ChatCompletionMessage) (Action, action.ActionParams, string, error) {
	c := messages

	if !a.options.forceReasoning {
		// We also could avoid to use functions here and get just a reply from the LLM
		// and then use the reply to get the action
		thought, err := a.decision(ctx,
			messages,
			a.systemInternalActions().ToTools(),
			nil)
		if err != nil {
			return nil, nil, "", err
		}

		xlog.Debug(fmt.Sprintf("thought action Name: %v", thought.actioName))
		xlog.Debug(fmt.Sprintf("thought message: %v", thought.message))

		// Find the action
		chosenAction := a.systemInternalActions().Find(thought.actioName)
		if chosenAction == nil || thought.actioName == "" {
			xlog.Debug("no answer")

			// LLM replied with an answer?
			//fmt.Errorf("no action found for intent:" + thought.actioName)
			return nil, nil, thought.message, nil
		}
		xlog.Debug(fmt.Sprintf("chosenAction: %v", chosenAction.Definition().Name))
		return chosenAction, thought.actionParams, thought.message, nil
	}

	prompt, err := renderTemplate(templ, a.prepareHUD(), a.systemInternalActions(), "")
	if err != nil {
		return nil, nil, "", err
	}
	// Get the LLM to think on what to do
	// and have a thought
	if !Messages(c).Exist(prompt) {
		c = append([]openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: prompt,
			},
		}, c...)
	}

	// We also could avoid to use functions here and get just a reply from the LLM
	// and then use the reply to get the action
	thought, err := a.decision(ctx,
		c,
		Actions{action.NewReasoning()}.ToTools(),
		action.NewReasoning().Definition().Name)
	if err != nil {
		return nil, nil, "", err
	}
	reason := ""
	response := &action.ReasoningResponse{}
	if thought.actionParams != nil {
		if err := thought.actionParams.Unmarshal(response); err != nil {
			return nil, nil, "", err
		}
		reason = response.Reasoning
	}
	if thought.message != "" {
		reason = thought.message
	}

	// From the thought, get the action call
	// Get all the available actions IDs
	actionsID := []string{}
	for _, m := range a.systemInternalActions() {
		actionsID = append(actionsID, m.Definition().Name.String())
	}
	intentionsTools := action.NewIntention(actionsID...)

	//XXX: Why we add the reason here?
	params, err := a.decision(ctx,
		append(c, openai.ChatCompletionMessage{
			Role:    "system",
			Content: "Given the assistant thought, pick the relevant action: " + reason,
		}),
		Actions{intentionsTools}.ToTools(),
		intentionsTools.Definition().Name)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to get the action tool parameters: %v", err)
	}

	actionChoice := action.IntentResponse{}

	if params.actionParams == nil {
		return nil, nil, params.message, nil
	}

	err = params.actionParams.Unmarshal(&actionChoice)
	if err != nil {
		return nil, nil, "", err
	}

	if actionChoice.Tool == "" || actionChoice.Tool == "none" {
		return nil, nil, "", fmt.Errorf("no intent detected")
	}

	// Find the action
	chosenAction := a.systemInternalActions().Find(actionChoice.Tool)
	if chosenAction == nil {
		return nil, nil, "", fmt.Errorf("no action found for intent:" + actionChoice.Tool)
	}

	return chosenAction, nil, actionChoice.Reasoning, nil
}
