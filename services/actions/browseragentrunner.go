package actions

import (
	"context"
	"fmt"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	api "github.com/mudler/LocalAGI/pkg/localoperator"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type BrowserAgentRunner struct {
	baseURL, customActionName string
	client                    *api.Client
}

func NewBrowserAgentRunner(config map[string]string, defaultURL string) *BrowserAgentRunner {
	if config["baseURL"] == "" {
		config["baseURL"] = defaultURL
	}

	client := api.NewClient(config["baseURL"])

	return &BrowserAgentRunner{
		client:           client,
		baseURL:          config["baseURL"],
		customActionName: config["customActionName"],
	}
}

func (b *BrowserAgentRunner) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	result := api.AgentRequest{}
	err := params.Unmarshal(&result)
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to unmarshal params: %w", err)
	}

	req := api.AgentRequest{
		Goal:                result.Goal,
		MaxAttempts:         result.MaxAttempts,
		MaxNoActionAttempts: result.MaxNoActionAttempts,
	}

	stateHistory, err := b.client.RunBrowserAgent(req)
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to run browser agent: %w", err)
	}

	// Format the state history into a readable string
	var historyStr string
	// for i, state := range stateHistory.States {
	// 	historyStr += fmt.Sprintf("State %d:\n", i+1)
	// 	historyStr += fmt.Sprintf("  URL: %s\n", state.CurrentURL)
	// 	historyStr += fmt.Sprintf("  Title: %s\n", state.PageTitle)
	// 	historyStr += fmt.Sprintf("  Description: %s\n\n", state.PageContentDescription)
	// }

	historyStr += fmt.Sprintf("  URL: %s\n", stateHistory.States[len(stateHistory.States)-1].CurrentURL)
	historyStr += fmt.Sprintf("  Title: %s\n", stateHistory.States[len(stateHistory.States)-1].PageTitle)
	historyStr += fmt.Sprintf("  Description: %s\n\n", stateHistory.States[len(stateHistory.States)-1].PageContentDescription)

	return types.ActionResult{
		Result:   fmt.Sprintf("Browser agent completed successfully. History:\n%s", historyStr),
		Metadata: map[string]interface{}{"browser_agent_history": stateHistory},
	}, nil
}

func (b *BrowserAgentRunner) Definition() types.ActionDefinition {
	actionName := "run_browser_agent"
	if b.customActionName != "" {
		actionName = b.customActionName
	}
	description := "Run a browser agent to achieve a specific goal, for example: 'Go to https://www.google.com and search for 'LocalAI', and tell me what's on the first page'"
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(actionName),
		Description: description,
		Properties: map[string]jsonschema.Definition{
			"goal": {
				Type:        jsonschema.String,
				Description: "The goal for the browser agent to achieve",
			},
			"max_attempts": {
				Type:        jsonschema.Number,
				Description: "Maximum number of attempts the agent can make (optional)",
			},
			"max_no_action_attempts": {
				Type:        jsonschema.Number,
				Description: "Maximum number of attempts without taking an action (optional)",
			},
		},
		Required: []string{"goal"},
	}
}

func (a *BrowserAgentRunner) Plannable() bool {
	return true
}

// BrowserAgentRunnerConfigMeta returns the metadata for Browser Agent Runner action configuration fields
func BrowserAgentRunnerConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "baseURL",
			Label:    "Base URL",
			Type:     config.FieldTypeText,
			Required: false,
			HelpText: "Base URL of the LocalOperator API",
		},
		{
			Name:     "customActionName",
			Label:    "Custom Action Name",
			Type:     config.FieldTypeText,
			HelpText: "Custom name for this action",
		},
	}
}
