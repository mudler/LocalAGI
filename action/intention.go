package action

import (
	"github.com/sashabaranov/go-openai/jsonschema"
)

// NewIntention creates a new intention action
// The inention action is special as it tries to identify
// a tool to use and a reasoning over to use it
func NewIntention(s ...string) *IntentAction {
	return &IntentAction{tools: s}
}

type IntentAction struct {
	tools []string
}
type IntentResponse struct {
	Tool      string `json:"tool"`
	Reasoning string `json:"reasoning"`
}

func (a *IntentAction) Run(ActionParams) (string, error) {
	return "no-op", nil
}

func (a *IntentAction) Definition() ActionDefinition {
	return ActionDefinition{
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
