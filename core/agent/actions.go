package agent

import (
	"encoding/json"
	"os"

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

// Helper function to format properties for the prompt
func formatProperties(props map[string]jsonschema.Definition) string {
	var result strings.Builder
	for name, prop := range props {
		result.WriteString(fmt.Sprintf("- %s: %s\n", name, prop.Description))
	}
	return result.String()
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

	defaultActions := append(a.mcpActions, a.options.userActions...)

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
