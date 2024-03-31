package action

import (
	"github.com/sashabaranov/go-openai/jsonschema"
)

func NewIntention(s ...string) *IntentAction {
	return &IntentAction{tools: s}
}

type IntentAction struct {
	tools []string
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
