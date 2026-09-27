package agent

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/mudler/LocalAGI/core/action"
	"github.com/mudler/LocalAGI/core/types"
	"golang.org/x/exp/slices"

	"github.com/mudler/xlog"

	"github.com/sashabaranov/go-openai"
)

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

// mergeLeadingSystemMessages replaces all leading system messages with a single
// system message. prefixBlocks are prepended in order (e.g. self-eval, then HUD).
// Only non-empty prefixBlocks are joined. Mid-conversation system messages are unchanged.
func (conv Messages) mergeLeadingSystemMessages(prefixBlocks ...string) Messages {
	var leading []string
	for _, s := range prefixBlocks {
		if s != "" {
			leading = append(leading, s)
		}
	}
	i := 0
	for i < len(conv) && conv[i].Role == SystemRole {
		content := conv[i].Content
		if content == "" && conv[i].MultiContent != nil {
			for _, part := range conv[i].MultiContent {
				if part.Type == openai.ChatMessagePartTypeText && part.Text != "" {
					content = part.Text
					break
				}
			}
		}
		if content != "" {
			leading = append(leading, content)
		}
		i++
	}
	if len(leading) == 0 {
		return conv
	}
	combined := strings.Join(leading, "\n\n")
	single := openai.ChatCompletionMessage{
		Role:    SystemRole,
		Content: combined,
	}
	return append([]openai.ChatCompletionMessage{single}, conv[i:]...)
}

// getAvailableActionsForJob returns available actions including user-defined ones for a specific job
func (a *Agent) getAvailableActionsForJob(job *types.Job) types.Actions {
	// Start with regular available actions
	baseActions := a.availableActions(job)

	// Add user-defined actions from the job
	userTools := job.GetUserTools()
	if len(userTools) > 0 {
		userDefinedActions := types.CreateUserDefinedActions(userTools)
		baseActions = append(baseActions, userDefinedActions...)
		xlog.Debug("Added user-defined actions", "definitions", userTools)
	}

	return baseActions
}

func (a *Agent) availableActions(j *types.Job) types.Actions {
	//	defaultActions := append(a.options.userActions, action.NewReply())

	defaultActions := slices.Clone(a.options.userActions)
	if j.Metadata["type"] == "scheduled" || (a.options.initiateConversations && a.selfEvaluationInProgress) { // && self-evaluation..
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
