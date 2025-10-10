package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mudler/LocalAGI/core/action"
	"github.com/mudler/LocalAGI/core/types"

	"github.com/mudler/LocalAGI/pkg/xlog"

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

	decision := openai.ChatCompletionRequest{
		Model:    a.options.LLMAPI.Model,
		Messages: conversation,
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
		conv = a.addFunctionResultToConversation(job.GetContext(), subTaskAction, actionParams, result, conv)
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
