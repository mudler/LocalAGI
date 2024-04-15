package external

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mudler/local-agent-framework/action"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/tmc/langchaingo/tools/duckduckgo"
)

func NewSearch(config map[string]string) *SearchAction {
	results := config["results"]
	intResult := 1

	// decode int from string
	if results != "" {
		_, err := fmt.Sscanf(results, "%d", &intResult)
		if err != nil {
			fmt.Printf("error: %v", err)
		}
	}

	slog.Info("Search action with results: ", "results", intResult)
	return &SearchAction{results: intResult}
}

type SearchAction struct{ results int }

func (a *SearchAction) Run(ctx context.Context, params action.ActionParams) (string, error) {
	result := struct {
		Query string `json:"query"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return "", err
	}
	ddg, err := duckduckgo.New(a.results, "LocalAgent")
	if err != nil {
		fmt.Printf("error: %v", err)

		return "", err
	}
	return ddg.Call(ctx, result.Query)
}

func (a *SearchAction) Definition() action.ActionDefinition {
	return action.ActionDefinition{
		Name:        "search_internet",
		Description: "Search the internet for something.",
		Properties: map[string]jsonschema.Definition{
			"query": {
				Type:        jsonschema.String,
				Description: "The query to search for.",
			},
		},
		Required: []string{"query"},
	}
}
