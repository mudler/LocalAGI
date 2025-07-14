package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mudler/LocalAGI/core/action"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/db"
	models "github.com/mudler/LocalAGI/dbmodels"

	"github.com/mudler/LocalAGI/pkg/utils"
	"github.com/mudler/LocalAGI/pkg/xlog"

	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const parameterReasoningPrompt = `You are tasked with generating the optimal parameters for the action "%s". The action requires the following parameters:
%s

Your task is to:
1. Generate the best possible values for each required parameter
2. If the parameter requires code, provide complete, working code
3. If the parameter requires text or documentation, provide comprehensive, well-structured content
4. Ensure all parameters are complete and ready to be used

Focus on quality and completeness. Do not explain your reasoning or analyze the action's purpose - just provide the best possible parameter values.`

type decisionResult struct {
	actionParams types.ActionParams
	message      string
	actionName   string
}

// decision forces the agent to take one of the available actions
func (a *Agent) decision(
	job *types.Job,
	conversation []openai.ChatCompletionMessage,
	tools []openai.Tool, toolchoice string, maxRetries int) (*decisionResult, error) {

	var choice *openai.ToolChoice

	if toolchoice != "" {
		choice = &openai.ToolChoice{
			Type:     openai.ToolTypeFunction,
			Function: openai.ToolFunction{Name: toolchoice},
		}
	}

	// Add guidance for single tool call preference when no specific tool is chosen
	enhancedConversation := conversation
	if toolchoice == "" && !a.options.forceReasoning {
		singleToolGuidance := `IMPORTANT: When responding to user requests, prefer using a SINGLE tool call that can handle the entire request when possible.

Examples:
- For "weather in Paris and Boston" → use one search like "current weather in Paris and Boston"
- For "tell me about X and Y" → use one search like "information about X and Y"
- For multiple related items → combine them into one comprehensive query

Choose the most appropriate single tool call to fulfill the user's complete request.`

		enhancedConversation = append([]openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: singleToolGuidance,
			},
		}, conversation...)
	}

	decision := openai.ChatCompletionRequest{
		Model:    a.options.LLMAPI.Model,
		Messages: enhancedConversation,
		Tools:    tools,
	}

	if choice != nil {
		decision.ToolChoice = *choice
	}

	var obs *types.Observable
	if job.Obs != nil {
		obs = a.observer.NewObservable()
		obs.Name = "decision"
		obs.ParentID = job.Obs.ID
		obs.Icon = "brain"
		obs.Creation = &types.Creation{
			ChatCompletionRequest: &decision,
		}
		a.observer.Update(*obs)
	}

	var lastErr error
	for attempts := 0; attempts < maxRetries; attempts++ {
		resp, err := a.client.CreateChatCompletion(job.GetContext(), decision)
		if err != nil {
			lastErr = err
			xlog.Warn("Attempt to make a decision failed", "attempt", attempts+1, "error", err)

			if obs != nil {
				obs.Progress = append(obs.Progress, types.Progress{
					Error: err.Error(),
				})
				a.observer.Update(*obs)
			}

			continue
		}

		if len(resp.Choices) == 1 && (resp.Choices[0].Message.Content != "" || len(resp.Choices[0].Message.ToolCalls) > 0) {
			// Track usage after successful API call
			usage := utils.GetOpenRouterUsage(resp.ID)
			llmUsage := &models.LLMUsage{
				ID:               uuid.New(),
				UserID:           a.options.userID,
				AgentID:          a.options.agentID,
				Model:            a.options.LLMAPI.Model,
				PromptTokens:     usage.PromptTokens,
				CompletionTokens: usage.CompletionTokens,
				TotalTokens:      usage.TotalTokens,
				Cost:             usage.Cost,
				RequestType:      "chat",
				GenID:            resp.ID,
				CreatedAt:        time.Now(),
			}
			if err := db.DB.Create(llmUsage).Error; err != nil {
				xlog.Error("Error tracking LLM usage", "error", err)
			}
		}

		jsonResp, _ := json.Marshal(resp)
		xlog.Debug("Decision response", "response", string(jsonResp))

		if obs != nil {
			obs.AddProgress(types.Progress{
				ChatCompletionResponse: &resp,
			})
		}

		if len(resp.Choices) != 1 {
			lastErr = fmt.Errorf("no choices: %d", len(resp.Choices))
			xlog.Warn("Attempt to make a decision failed", "attempt", attempts+1, "error", lastErr)

			if obs != nil {
				obs.Progress[len(obs.Progress)-1].Error = lastErr.Error()
				a.observer.Update(*obs)
			}

			continue
		}

		msg := resp.Choices[0].Message
		if len(msg.ToolCalls) != 1 {
			if err := a.saveConversation(append(conversation, msg), "decision"); err != nil {
				xlog.Error("Error saving conversation", "error", err)
			}

			if obs != nil {
				obs.MakeLastProgressCompletion()
				a.observer.Update(*obs)
			}

			return &decisionResult{message: msg.Content}, nil
		}

		params := types.ActionParams{}
		if err := params.Read(msg.ToolCalls[0].Function.Arguments); err != nil {
			lastErr = err
			xlog.Warn("Attempt to parse action parameters failed", "attempt", attempts+1, "error", err)

			if obs != nil {
				obs.Progress[len(obs.Progress)-1].Error = lastErr.Error()
				a.observer.Update(*obs)
			}

			continue
		}

		if err := a.saveConversation(append(conversation, msg), "decision"); err != nil {
			xlog.Error("Error saving conversation", "error", err)
		}

		if obs != nil {
			obs.MakeLastProgressCompletion()
			a.observer.Update(*obs)
		}

		return &decisionResult{actionParams: params, actionName: msg.ToolCalls[0].Function.Name, message: msg.Content}, nil
	}

	// Instead of returning an error, return a message explaining the failure
	failureMessage := fmt.Sprintf("I apologize, but I'm having difficulty making a decision")
	return &decisionResult{message: failureMessage}, nil
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
		s += cc.Role + ": "

		// Handle messages with tool calls
		if len(cc.ToolCalls) > 0 {
			s += "[Tool Calls] "
			for _, tc := range cc.ToolCalls {
				s += fmt.Sprintf("%s(%s) ", tc.Function.Name, tc.Function.Arguments)
			}
		}

		// Handle tool messages with their ID
		if cc.Role == openai.ChatMessageRoleTool {
			s += fmt.Sprintf("[Tool ID: %s] ", cc.ToolCallID)
		}

		s += cc.Content + "\n"
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
	xlog.Debug("Getting latest user message", "messages", m)
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

func (a *Agent) generateParameters(job *types.Job, pickTemplate string, act types.Action, c []openai.ChatCompletionMessage, reasoning string, maxAttempts int) (*decisionResult, error) {
	if act == nil {
		return nil, fmt.Errorf("action is nil")
	}
	if len(act.Definition().Properties) > 0 {
		xlog.Debug("Action has properties", "action", act.Definition().Name, "properties", act.Definition().Properties)
	} else {
		xlog.Debug("Action has no properties", "action", act.Definition().Name)
		return &decisionResult{actionParams: types.ActionParams{}}, nil
	}

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
		// First, get the LLM to reason about optimal parameter usage
		parameterReasoningPrompt := fmt.Sprintf(parameterReasoningPrompt,
			act.Definition().Name,
			formatProperties(act.Definition().Properties))

		// Get initial reasoning about parameters using askLLM
		paramReasoningMsg, err := a.askLLM(job.GetContext(),
			append(conversation, openai.ChatCompletionMessage{
				Role:    "system",
				Content: parameterReasoningPrompt,
			}),
			maxAttempts,
		)
		if err != nil {
			xlog.Warn("Failed to get parameter reasoning", "error", err)
		}

		// Combine original reasoning with parameter-specific reasoning
		enhancedReasoning := reasoning
		if paramReasoningMsg.Content != "" {
			enhancedReasoning = fmt.Sprintf("%s\n\nParameter Analysis:\n%s", reasoning, paramReasoningMsg.Content)
		}

		cc = append(conversation, openai.ChatCompletionMessage{
			Role:    "system",
			Content: fmt.Sprintf("The agent decided to use the tool %s with the following reasoning: %s", act.Definition().Name, enhancedReasoning),
		})
	}

	var result *decisionResult
	var attemptErr error

	for attempts := 0; attempts < maxAttempts; attempts++ {
		result, attemptErr = a.decision(job,
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

// Helper function to format properties for the prompt
func formatProperties(props map[string]jsonschema.Definition) string {
	var result strings.Builder
	for name, prop := range props {
		result.WriteString(fmt.Sprintf("- %s: %s\n", name, prop.Description))
	}
	return result.String()
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
		if subTaskAction == nil {
			xlog.Error("Action not found: %s", subtask.Action)
			return conv, fmt.Errorf("action %s not found", subtask.Action)
		}

		subTaskReasoning := fmt.Sprintf("%s Overall goal is: %s", subtask.Reasoning, planResult.Goal)

		params, err := a.generateParameters(job, pickTemplate, subTaskAction, conv, subTaskReasoning, maxRetries)
		if err != nil {
			xlog.Error("error generating action's parameters", "error", err)
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

		result, err := a.runAction(job, subTaskAction, actionParams)
		if err != nil {
			xlog.Error("error running action", "error", err)
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

// getAvailableActionsForJob returns available actions including user-defined ones for a specific job
func (a *Agent) getAvailableActionsForJob(job *types.Job) types.Actions {
	// Start with regular available actions
	baseActions := a.availableActions()

	// Add user-defined actions from the job
	userTools := job.GetUserTools()
	if len(userTools) > 0 {
		userDefinedActions := types.CreateUserDefinedActions(userTools)
		baseActions = append(baseActions, userDefinedActions...)
		xlog.Debug("Added user-defined actions", "definitions", userTools)
	}

	return baseActions
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
func (a *Agent) pickAction(job *types.Job, templ string, messages []openai.ChatCompletionMessage, maxRetries int) (types.Action, types.ActionParams, string, error) {
	c := messages

	xlog.Debug("[pickAction] picking action starts", "messages", messages)

	// Get available actions including user-defined ones
	availableActions := a.getAvailableActionsForJob(job)

	// Identify the goal of this conversation

	if !a.options.forceReasoning || job.ToolChoice != "" {
		xlog.Debug("not forcing reasoning", "forceReasoning", a.options.forceReasoning, "ToolChoice", job.ToolChoice)
		// We also could avoid to use functions here and get just a reply from the LLM
		// and then use the reply to get the action
		thought, err := a.decision(job,
			messages,
			availableActions.ToTools(),
			job.ToolChoice,
			maxRetries)
		if err != nil {
			return nil, nil, "", err
		}

		xlog.Debug("thought action Name", "actionName", thought.actionName)
		xlog.Debug("thought message", "message", thought.message)

		// Find the action
		chosenAction := availableActions.Find(thought.actionName)
		if chosenAction == nil || thought.actionName == "" {
			xlog.Debug("no answer")

			// LLM replied with an answer?
			//fmt.Errorf("no action found for intent:" + thought.actioName)
			return nil, nil, thought.message, nil
		}
		xlog.Debug(fmt.Sprintf("chosenAction: %v", chosenAction.Definition().Name))
		return chosenAction, thought.actionParams, thought.message, nil
	}

	// Force the LLM to think and we extract a "reasoning" to pick a specific action and with which parameters
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

	// Create a detailed prompt for reasoning that includes available actions and their properties
	reasoningPrompt := "Analyze the current situation and determine the best course of action. Consider the following:\n\n"
	reasoningPrompt += "Available Actions:\n"
	for _, act := range a.availableActions() {
		reasoningPrompt += fmt.Sprintf("- %s: %s\n", act.Definition().Name, act.Definition().Description)
		if len(act.Definition().Properties) > 0 {
			reasoningPrompt += "  Properties:\n"
			for name, prop := range act.Definition().Properties {
				reasoningPrompt += fmt.Sprintf("  - %s: %s\n", name, prop.Description)
			}
		}
		reasoningPrompt += "\n"
	}
	reasoningPrompt += "\nProvide a detailed reasoning about what action would be most appropriate in this situation and why. You can also just reply with a simple message by choosing the 'reply' or 'answer' action."

	// Get reasoning using askLLM
	reasoningMsg, err := a.askLLM(job.GetContext(),
		append(c, openai.ChatCompletionMessage{
			Role:    "system",
			Content: reasoningPrompt,
		}),
		maxRetries)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to get reasoning: %w", err)
	}

	originalReasoning := reasoningMsg.Content

	xlog.Debug("[pickAction] picking action", "messages", c)

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
	params, err := a.decision(job,
		append(c, openai.ChatCompletionMessage{
			Role:    "system",
			Content: "Pick the relevant action given the following reasoning: " + originalReasoning,
		}),
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
