package action

import (
	"github.com/sashabaranov/go-openai/jsonschema"
)

func NewReasoning() *ReasoningAction {
	return &ReasoningAction{}
}

type ReasoningAction struct{}

type ReasoningResponse struct {
	Reasoning string `json:"reasoning"`
}

func (a *ReasoningAction) Run(ActionParams) (string, error) {
	return "no-op", nil
}

func (a *ReasoningAction) Definition() ActionDefinition {
	return ActionDefinition{
		Name:        "think",
		Description: "try to understand what's the best thing to do",
		Properties: map[string]jsonschema.Definition{
			"reasoning": {
				Type:        jsonschema.String,
				Description: "A detailed reasoning on what would you do in this situation.",
			},
		},
		Required: []string{"reasoning"},
	}
}
