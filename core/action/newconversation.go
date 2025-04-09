package action

import (
	"context"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const ConversationActionName = "new_conversation"

func NewConversation() *ConversationAction {
	return &ConversationAction{}
}

type ConversationAction struct{}

type ConversationActionResponse struct {
	Message string `json:"message"`
}

func (a *ConversationAction) Run(context.Context, types.ActionParams) (types.ActionResult, error) {
	return types.ActionResult{}, nil
}

func (a *ConversationAction) Plannable() bool {
	return false
}

func (a *ConversationAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        ConversationActionName,
		Description: "Use this tool to initiate a new conversation or to notify something.",
		Properties: map[string]jsonschema.Definition{
			"message": {
				Type:        jsonschema.String,
				Description: "The message to start the conversation",
			},
		},
		Required: []string{"message"},
	}
}
