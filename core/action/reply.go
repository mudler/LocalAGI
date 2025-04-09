package action

import (
	"context"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// ReplyActionName is the name of the reply action
// used by the LLM to reply to the user without
// any additional processing
const ReplyActionName = "reply"

func NewReply() *ReplyAction {
	return &ReplyAction{}
}

type ReplyAction struct{}

type ReplyResponse struct {
	Message string `json:"message"`
}

func (a *ReplyAction) Run(context.Context, types.ActionParams) (string, error) {
	return "no-op", nil
}

func (a *ReplyAction) Plannable() bool {
	return false
}

func (a *ReplyAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        ReplyActionName,
		Description: "Use this tool to reply to the user once we have all the informations we need.",
		Properties: map[string]jsonschema.Definition{
			"message": {
				Type:        jsonschema.String,
				Description: "The message to reply with",
			},
		},
		Required: []string{"message"},
	}
}
