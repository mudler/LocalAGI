package actions

import (
	"context"
	"fmt"

	"github.com/mudler/LocalAgent/core/types"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/tmc/langchaingo/tools/wikipedia"
)

func NewWikipedia(config map[string]string) *WikipediaAction {
	return &WikipediaAction{}
}

type WikipediaAction struct{}

func (a *WikipediaAction) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Query string `json:"query"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return types.ActionResult{}, err
	}
	wiki := wikipedia.New("LocalAgent")
	res, err := wiki.Call(ctx, result.Query)
	if err != nil {
		fmt.Printf("error: %v", err)

		return types.ActionResult{}, err
	}
	return types.ActionResult{Result: res}, nil
}

func (a *WikipediaAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
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

func (a *WikipediaAction) Plannable() bool {
	return true
}
