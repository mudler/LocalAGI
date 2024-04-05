package external

import (
	"fmt"

	"github.com/mudler/local-agent-framework/action"
	"github.com/sap-nocops/duckduckgogo/client"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func NewSearch(results int) *SearchAction {
	if results == 0 {
		results = 3
	}
	return &SearchAction{results: results}
}

type SearchAction struct{ results int }

func (a *SearchAction) Run(params action.ActionParams) (string, error) {
	result := struct {
		Query string `json:"query"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return "", err
	}
	ddg := client.NewDuckDuckGoSearchClient()
	res, err := ddg.SearchLimited(result.Query, a.results)
	if err != nil {
		msg := fmt.Sprintf("error: %v", err)
		fmt.Printf(msg)
		return msg, err
	}

	results := ""
	for i, r := range res {
		results += fmt.Sprintf("*********** RESULT %d\nurl:     %s\ntitle:   %s\nsnippet: %s\n", i, r.FormattedUrl, r.Title, r.Snippet)
	}

	return results, nil
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
