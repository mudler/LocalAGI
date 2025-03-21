package action

import (
	"context"

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

func (a *ConversationAction) Run(context.Context, ActionParams) (ActionResult, error) {
	return ActionResult{}, nil
}

func (a *ConversationAction) Plannable() bool {
	return false
}

func (a *ConversationAction) Definition() ActionDefinition {
	return ActionDefinition{
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
