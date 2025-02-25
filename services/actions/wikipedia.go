package actions

import (
	"context"
	"fmt"

	"github.com/mudler/local-agent-framework/core/action"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/tmc/langchaingo/tools/wikipedia"
)

func NewWikipedia(config map[string]string) *WikipediaAction {
	return &WikipediaAction{}
}

type WikipediaAction struct{}

func (a *WikipediaAction) Run(ctx context.Context, params action.ActionParams) (string, error) {
	result := struct {
		Query string `json:"query"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return "", err
	}
	wiki := wikipedia.New("LocalAgent")
	return wiki.Call(ctx, result.Query)
}

func (a *WikipediaAction) Definition() action.ActionDefinition {
	return action.ActionDefinition{
		Name:        "wikipedia",
		Description: "Find wikipedia pages using the wikipedia api",
		Properties: map[string]jsonschema.Definition{
			"query": {
				Type:        jsonschema.String,
				Description: "The website URL.",
			},
		},
		Required: []string{"query"},
	}
}
