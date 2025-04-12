package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mudler/LocalAGI/core/action"
	"github.com/mudler/LocalAGI/core/types"

	"github.com/mudler/LocalAGI/pkg/xlog"

	"github.com/sashabaranov/go-openai"
)

type decisionResult struct {
	actionParams types.ActionParams
	message      string
	actioName    string
}

// decision forces the agent to take one of the available actions
func (a *Agent) decision(
	ctx context.Context,
	conversation []openai.ChatCompletionMessage,
	tools []openai.Tool, toolchoice string, maxRetries int) (*decisionResult, error) {

	var choice *openai.ToolChoice

	if toolchoice != "" {
		choice = &openai.ToolChoice{
			Type:     openai.ToolTypeFunction,
			Function: openai.ToolFunction{Name: toolchoice},
		}
	}

	var lastErr error
	for attempts := 0; attempts < maxRetries; attempts++ {
		decision := openai.ChatCompletionRequest{
			Model:    a.options.LLMAPI.Model,
			Messages: conversation,
			Tools:    tools,
		}

		if choice != nil {
			decision.ToolChoice = *choice
		}

		resp, err := a.client.CreateChatCompletion(ctx, decision)
		if err != nil {
			lastErr = err
			xlog.Warn("Attempt to make a decision failed", "attempt", attempts+1, "error", err)
			continue
		}

		jsonResp, _ := json.Marshal(resp)
		xlog.Debug("Decision response", "response", string(jsonResp))

		if len(resp.Choices) != 1 {
			lastErr = fmt.Errorf("no choices: %d", len(resp.Choices))
			xlog.Warn("Attempt to make a decision failed", "attempt", attempts+1, "error", lastErr)
			continue
		}

		msg := resp.Choices[0].Message
		if len(msg.ToolCalls) != 1 {
			if err := a.saveConversation(append(conversation, msg), "decision"); err != nil {
				xlog.Error("Error saving conversation", "error", err)
			}
			return &decisionResult{message: msg.Content}, nil
		}

		params := types.ActionParams{}
		if err := params.Read(msg.ToolCalls[0].Function.Arguments); err != nil {
			lastErr = err
			xlog.Warn("Attempt to parse action parameters failed", "attempt", attempts+1, "error", err)
			continue
		}

		if err := a.saveConversation(append(conversation, msg), "decision"); err != nil {
			xlog.Error("Error saving conversation", "error", err)
		}

		return &decisionResult{actionParams: params, actioName: msg.ToolCalls[0].Function.Name, message: msg.Content}, nil
	}

	return nil, fmt.Errorf("failed to make a decision after %d attempts: %w", maxRetries, lastErr)
}

type Messages []openai.ChatCompletionMessage

func (m Messages) ToOpenAI() []openai.ChatCompletionMessage {
	return []openai.ChatCompletionMessage(m)
}

func (m Messages) RemoveIf(f func(msg openai.ChatCompletionMessage) bool) Messages {
	for i := len(m) - 1; i >= 0; i-- {
		if f(m[i]) {
			m = append(m[:i], m[i+1:]...)
		}
	}
	return m
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

func (a *Agent) generateParameters(ctx context.Context, pickTemplate string, act types.Action, c []openai.ChatCompletionMessage, reasoning string, maxAttempts int) (*decisionResult, error) {
	stateHUD, err := renderTemplate(pickTemplate, a.prepareHUD(), a.availableActions(), reasoning)
	if err != nil {
		return nil, err
	}

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

	var result *decisionResult
	var attemptErr error

	for attempts := 0; attempts < maxAttempts; attempts++ {
		result, attemptErr = a.decision(ctx,
			cc,
			a.availableActions().ToTools(),
			act.Definition().Name.String(),
			maxAttempts,
		)
		if attemptErr == nil && result.actionParams != nil {
			return result, nil
		}
		xlog.Warn("Attempt to generate parameters failed", "attempt", attempts+1, "error", attemptErr)
	}

	return nil, fmt.Errorf("failed to generate parameters after %d attempts: %w", maxAttempts, attemptErr)
}

func (a *Agent) handlePlanning(ctx context.Context, job *types.Job, chosenAction types.Action, actionParams types.ActionParams, reasoning string, pickTemplate string, conv Messages) (Messages, error) {
	// Planning: run all the actions in sequence
	if !chosenAction.Definition().Name.Is(action.PlanActionName) {
		xlog.Debug("no plan action")
		return conv, nil
	}

	xlog.Debug("[planning]...")
	planResult := action.PlanResult{}
	if err := actionParams.Unmarshal(&planResult); err != nil {
		return conv, fmt.Errorf("error unmarshalling plan result: %w", err)
	}

	stateResult := types.ActionState{
		ActionCurrentState: types.ActionCurrentState{
			Job:       job,
			Action:    chosenAction,
			Params:    actionParams,
			Reasoning: reasoning,
		},
		ActionResult: types.ActionResult{
			Result: fmt.Sprintf("planning %s, subtasks: %+v", planResult.Goal, planResult.Subtasks),
		},
	}
	job.Result.SetResult(stateResult)
	job.CallbackWithResult(stateResult)

	xlog.Info("[Planning] starts", "agent", a.Character.Name, "goal", planResult.Goal)
	for _, s := range planResult.Subtasks {
		xlog.Info("[Planning] subtask", "agent", a.Character.Name, "action", s.Action, "reasoning", s.Reasoning)
	}

	if len(planResult.Subtasks) == 0 {
		return conv, fmt.Errorf("no subtasks")
	}

	// Execute all subtasks in sequence
	for _, subtask := range planResult.Subtasks {
		xlog.Info("[subtask] Generating parameters",
			"agent", a.Character.Name,
			"action", subtask.Action,
			"reasoning", reasoning,
		)

		subTaskAction := a.availableActions().Find(subtask.Action)
		subTaskReasoning := fmt.Sprintf("%s Overall goal is: %s", subtask.Reasoning, planResult.Goal)

		params, err := a.generateParameters(ctx, pickTemplate, subTaskAction, conv, subTaskReasoning, maxRetries)
		if err != nil {
			return conv, fmt.Errorf("error generating action's parameters: %w", err)

		}
		actionParams = params.actionParams

		if !job.Callback(types.ActionCurrentState{
			Job:       job,
			Action:    subTaskAction,
			Params:    actionParams,
			Reasoning: subTaskReasoning,
		}) {
			job.Result.SetResult(types.ActionState{
				ActionCurrentState: types.ActionCurrentState{
					Job:       job,
					Action:    chosenAction,
					Params:    actionParams,
					Reasoning: subTaskReasoning,
				},
				ActionResult: types.ActionResult{
					Result: "stopped by callback",
				},
			})
			job.Result.Conversation = conv
			job.Result.Finish(nil)
			break
		}

		result, err := a.runAction(ctx, subTaskAction, actionParams)
		if err != nil {
			return conv, fmt.Errorf("error running action: %w", err)
		}

		stateResult := types.ActionState{
			ActionCurrentState: types.ActionCurrentState{
				Job:       job,
				Action:    subTaskAction,
				Params:    actionParams,
				Reasoning: subTaskReasoning,
			},
			ActionResult: result,
		}
		job.Result.SetResult(stateResult)
		job.CallbackWithResult(stateResult)
		xlog.Debug("[subtask] Action executed", "agent", a.Character.Name, "action", subTaskAction.Definition().Name, "result", result)
		conv = a.addFunctionResultToConversation(subTaskAction, actionParams, result, conv)
	}

	return conv, nil
}

func (a *Agent) availableActions() types.Actions {
	//	defaultActions := append(a.options.userActions, action.NewReply())

	addPlanAction := func(actions types.Actions) types.Actions {
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
func (a *Agent) pickAction(ctx context.Context, templ string, messages []openai.ChatCompletionMessage, maxRetries int) (types.Action, types.ActionParams, string, error) {
	c := messages

	xlog.Debug("[pickAction] picking action starts", "messages", messages)

	// Identify the goal of this conversation

	if !a.options.forceReasoning {
		xlog.Debug("not forcing reasoning")
		// We also could avoid to use functions here and get just a reply from the LLM
		// and then use the reply to get the action
		thought, err := a.decision(ctx,
			messages,
			a.availableActions().ToTools(),
			"",
			maxRetries)
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

	xlog.Debug("[pickAction] forcing reasoning")

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

	// thoughtPromptStringBuilder := strings.Builder{}
	// thoughtPromptStringBuilder.WriteString("You have to pick an action based on the conversation and the prompt. Describe the full reasoning process for your choice. Here is a list of actions: ")
	// for _, m := range a.availableActions() {
	// 	thoughtPromptStringBuilder.WriteString(
	// 		m.Definition().Name.String() + ": " + m.Definition().Description + "\n",
	// 	)
	// }

	// thoughtPromptStringBuilder.WriteString("To not use any action, respond with 'none'")

	//thoughtPromptStringBuilder.WriteString("\n\nConversation: " + Messages(c).RemoveIf(func(msg openai.ChatCompletionMessage) bool {
	//	return msg.Role == "system"
	//}).String())

	//thoughtPrompt := thoughtPromptStringBuilder.String()

	//thoughtConv := []openai.ChatCompletionMessage{}

	xlog.Debug("[pickAction] picking action", "messages", c)
	thought, err := a.askLLM(ctx,
		c,
		maxRetries,
	)
	if err != nil {
		return nil, nil, "", err
	}
	originalReasoning := thought.Content
	xlog.Debug("[pickAction] original reasoning", "originalReasoning", originalReasoning)
	// From the thought, get the action call
	// Get all the available actions IDs

	// by grammar, let's decide if we have achieved the goal
	//  1. analyze response and check if  goal is achieved

	// Extract the goal first
	params, err := a.decision(ctx,
		append(
			[]openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: "Your only task is to extract the goal from the following conversation",
				}}, messages...),
		types.Actions{action.NewGoal()}.ToTools(),
		action.NewGoal().Definition().Name.String(), maxRetries)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to get the action tool parameters: %v", err)
	}

	goalResponse := action.GoalResponse{}
	err = params.actionParams.Unmarshal(&goalResponse)
	if err != nil {
		return nil, nil, "", err
	}

	if goalResponse.Goal == "" {
		xlog.Debug("[pickAction] no goal found")
		return nil, nil, "", nil
	}

	// Check if the goal was achieved
	params, err = a.decision(ctx,
		[]openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: "You have to understand if the goal is achieved or not from the following reasoning. The goal: " + goalResponse.Goal,
			},
			{
				Role:    "user",
				Content: originalReasoning,
			}},
		types.Actions{action.NewGoal()}.ToTools(),
		action.NewGoal().Definition().Name.String(), maxRetries)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to get the action tool parameters: %v", err)
	}

	err = params.actionParams.Unmarshal(&goalResponse)
	if err != nil {
		return nil, nil, "", err
	}

	if goalResponse.Achieved {
		xlog.Debug("[pickAction] goal achieved", "goal", goalResponse.Goal)
		return nil, nil, "", nil
	}

	// if the goal is not achieved, pick an action
	xlog.Debug("[pickAction] goal not achieved", "goal", goalResponse.Goal)

	xlog.Debug("[pickAction] thought", "conv", c, "originalReasoning", originalReasoning)

	actionsID := []string{"reply"}
	for _, m := range a.availableActions() {
		actionsID = append(actionsID, m.Definition().Name.String())
	}

	xlog.Debug("[pickAction] actionsID", "actionsID", actionsID)

	intentionsTools := action.NewIntention(actionsID...)
	// TODO: FORCE to select ana ction here
	// NOTE: we do not give the full conversation here to pick the action
	// to avoid hallucinations

	// Extract an action
	params, err = a.decision(ctx,
		[]openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: prompt,
			},
			{
				Role:    "system",
				Content: "Extract an action to perform from the following reasoning: ",
			},
			{
				Role:    "user",
				Content: originalReasoning,
			}},
		types.Actions{intentionsTools}.ToTools(),
		intentionsTools.Definition().Name.String(), maxRetries)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to get the action tool parameters: %v", err)
	}

	if params.actionParams == nil {
		xlog.Debug("[pickAction] no action params found")
		return nil, nil, params.message, nil
	}

	actionChoice := action.IntentResponse{}
	err = params.actionParams.Unmarshal(&actionChoice)
	if err != nil {
		return nil, nil, "", err
	}

	if actionChoice.Tool == "" || actionChoice.Tool == "reply" {
		xlog.Debug("[pickAction] no action found, replying")
		return nil, nil, "", nil
	}

	chosenAction := a.availableActions().Find(actionChoice.Tool)

	xlog.Debug("[pickAction] chosenAction", "chosenAction", chosenAction, "actionName", actionChoice.Tool)

	// // Let's double check if the action is correct by asking the LLM to judge it

	// if chosenAction!= nil {
	// 	promptString:= "Given the following goal and thoughts, is the action correct? \n\n"
	// 	promptString+= fmt.Sprintf("Goal: %s\n", goalResponse.Goal)
	// 	promptString+= fmt.Sprintf("Thoughts: %s\n", originalReasoning)
	// 	promptString+= fmt.Sprintf("Action: %s\n", chosenAction.Definition().Name.String())
	// 	promptString+= fmt.Sprintf("Action description: %s\n", chosenAction.Definition().Description)
	// 	promptString+= fmt.Sprintf("Action parameters: %s\n", params.actionParams)

	// }

	return chosenAction, nil, originalReasoning, nil
}
