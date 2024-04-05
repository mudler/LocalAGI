package action2

import (
	"github.com/mudler/local-agent-framework/action"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// NewIntention creates a new intention action
// The inention action is special as it tries to identify
// a tool to use and a reasoning over to use it
func NewSearch(s ...string) *SearchAction {
	return &SearchAction{tools: s}
}

type SearchAction struct {
	tools []string
}

func (a *SearchAction) Run(action.ActionParams) (string, error) {
	return "no-op", nil
}

func (a *SearchAction) Definition() action.ActionDefinition {
	return action.ActionDefinition{
		Name:        "intent",
		Description: "detect user intent",
		Properties: map[string]jsonschema.Definition{
			"reasoning": {
				Type:        jsonschema.String,
				Description: "A detailed reasoning on why you want to call this tool.",
			},
			"tool": {
				Type: jsonschema.String,
				Enum: a.tools,
			},
		},
		Required: []string{"tool", "reasoning"},
	}
}
