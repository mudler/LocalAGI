package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mudler/LocalAgent/core/action"
	"github.com/mudler/LocalAgent/pkg/xlog"

	"github.com/sashabaranov/go-openai"
)

type ActionState struct {
	ActionCurrentState
	action.ActionResult
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
	Plannable() bool
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

	if err := a.saveConversation(append(conversation, msg), "decision"); err != nil {
		xlog.Error("Error saving conversation", "error", err)
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

func (m Messages) RemoveLastUserMessage() Messages {
	if len(m) == 0 {
		return m
	}

	for i := len(m) - 1; i >= 0; i-- {
		if m[i].Role == UserRole {
			return append(m[:i], m[i+1:]...)
		}
	}

	return m
}

func (m Messages) Save(path string) error {
	content, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err := f.Write(content); err != nil {
		return err
	}

	return nil
}

func (m Messages) GetLatestUserMessage() *openai.ChatCompletionMessage {
	for i := len(m) - 1; i >= 0; i-- {
		msg := m[i]
		if msg.Role == UserRole {
			return &msg
		}
	}

	return nil
}

func (m Messages) IsLastMessageFromRole(role string) bool {
	if len(m) == 0 {
		return false
	}

	return m[len(m)-1].Role == role
}

func (a *Agent) generateParameters(ctx context.Context, pickTemplate string, act Action, c []openai.ChatCompletionMessage, reasoning string) (*decisionResult, error) {

	stateHUD, err := renderTemplate(pickTemplate, a.prepareHUD(), a.availableActions(), reasoning)
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
		a.availableActions().ToTools(),
		openai.ToolChoice{
			Type:     openai.ToolTypeFunction,
			Function: openai.ToolFunction{Name: act.Definition().Name.String()},
		},
	)
}

func (a *Agent) handlePlanning(ctx context.Context, job *Job, chosenAction Action, actionParams action.ActionParams, reasoning string, pickTemplate string) error {
	// Planning: run all the actions in sequence
	if !chosenAction.Definition().Name.Is(action.PlanActionName) {
		xlog.Debug("no plan action")
		return nil
	}

	xlog.Debug("[planning]...")
	planResult := action.PlanResult{}
	if err := actionParams.Unmarshal(&planResult); err != nil {
		return fmt.Errorf("error unmarshalling plan result: %w", err)
	}

	xlog.Info("[Planning] starts", "agent", a.Character.Name, "goal", planResult.Goal)
	for _, s := range planResult.Subtasks {
		xlog.Info("[Planning] subtask", "agent", a.Character.Name, "action", s.Action, "reasoning", s.Reasoning)
	}

	if len(planResult.Subtasks) == 0 {
		return fmt.Errorf("no subtasks")
	}

	// Execute all subtasks in sequence
	for _, subtask := range planResult.Subtasks {
		xlog.Info("[subtask] Generating parameters",
			"agent", a.Character.Name,
			"action", subtask.Action,
			"reasoning", reasoning,
		)

		action := a.availableActions().Find(subtask.Action)

		params, err := a.generateParameters(ctx, pickTemplate, action, a.currentConversation, fmt.Sprintf("%s, overall goal is: %s", subtask.Reasoning, planResult.Goal))
		if err != nil {
			return fmt.Errorf("error generating action's parameters: %w", err)

		}
		actionParams = params.actionParams

		result, err := a.runAction(action, actionParams)
		if err != nil {
			return fmt.Errorf("error running action: %w", err)
		}

		stateResult := ActionState{ActionCurrentState{action, actionParams, subtask.Reasoning}, result}
		job.Result.SetResult(stateResult)
		job.CallbackWithResult(stateResult)
		xlog.Debug("[subtask] Action executed", "agent", a.Character.Name, "action", action.Definition().Name, "result", result)
		a.addFunctionResultToConversation(action, actionParams, result)
	}

	return nil
}

func (a *Agent) availableActions() Actions {
	//	defaultActions := append(a.options.userActions, action.NewReply())

	addPlanAction := func(actions Actions) Actions {
		if !a.options.canPlan {
			return actions
		}
		plannablesActions := []string{}
		for _, a := range actions {
			if a.Plannable() {
				plannablesActions = append(plannablesActions, a.Definition().Name.String())
			}
		}
		planAction := action.NewPlan(plannablesActions)
		actions = append(actions, planAction)
		return actions
	}

	defaultActions := append(a.mcpActions, a.options.userActions...)

	if a.options.initiateConversations && a.selfEvaluationInProgress { // && self-evaluation..
		acts := append(defaultActions, action.NewConversation())
		if a.options.enableHUD {
			acts = append(acts, action.NewState())
		}
		//if a.options.canStopItself {
		//		acts = append(acts, action.NewStop())
		//	}

		return addPlanAction(acts)
	}

	if a.options.canStopItself {
		acts := append(defaultActions, action.NewStop())
		if a.options.enableHUD {
			acts = append(acts, action.NewState())
		}
		return addPlanAction(acts)
	}

	if a.options.enableHUD {
		return addPlanAction(append(defaultActions, action.NewState()))
	}

	return addPlanAction(defaultActions)
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
			a.availableActions().ToTools(),
			nil)
		if err != nil {
			return nil, nil, "", err
		}

		xlog.Debug(fmt.Sprintf("thought action Name: %v", thought.actioName))
		xlog.Debug(fmt.Sprintf("thought message: %v", thought.message))

		// Find the action
		chosenAction := a.availableActions().Find(thought.actioName)
		if chosenAction == nil || thought.actioName == "" {
			xlog.Debug("no answer")

			// LLM replied with an answer?
			//fmt.Errorf("no action found for intent:" + thought.actioName)
			return nil, nil, thought.message, nil
		}
		xlog.Debug(fmt.Sprintf("chosenAction: %v", chosenAction.Definition().Name))
		return chosenAction, thought.actionParams, thought.message, nil
	}

	prompt, err := renderTemplate(templ, a.prepareHUD(), a.availableActions(), "")
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
	for _, m := range a.availableActions() {
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
	chosenAction := a.availableActions().Find(actionChoice.Tool)
	if chosenAction == nil {
		return nil, nil, "", fmt.Errorf("no action found for intent:" + actionChoice.Tool)
	}

	return chosenAction, nil, actionChoice.Reasoning, nil
}
