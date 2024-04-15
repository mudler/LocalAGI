package external

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/go-github/v61/github"
	"github.com/mudler/local-agent-framework/action"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubIssueSearch struct {
	token   string
	context context.Context
	client  *github.Client
}

func NewGithubIssueSearch(ctx context.Context, config map[string]string) *GithubIssueSearch {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	return &GithubIssueSearch{
		client:  client,
		token:   config["token"],
		context: ctx,
	}
}

func (g *GithubIssueSearch) Run(ctx context.Context, params action.ActionParams) (string, error) {
	result := struct {
		Query      string `json:"query"`
		Repository string `json:"repository"`
		Owner      string `json:"owner"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return "", err
	}

	query := fmt.Sprintf("%s in:%s user:%s", result.Query, result.Repository, result.Owner)
	resultString := ""
	issues, _, err := g.client.Search.Issues(g.context, query, &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 5},
		Order:       "desc",
		//Sort:        "created",
	})
	if err != nil {
		resultString = fmt.Sprintf("Error listing issues: %v", err)
	}
	for _, i := range issues.Issues {
		slog.Info("Issue found:", i.GetTitle())
		resultString += fmt.Sprintf("Issue found: %s\n", i.GetTitle())
		resultString += fmt.Sprintf("URL: %s\n", i.GetHTMLURL())
		//	resultString += fmt.Sprintf("Body: %s\n", i.GetBody())
	}

	return resultString, err
}

func (g *GithubIssueSearch) Definition() action.ActionDefinition {
	return action.ActionDefinition{
		Name:        "search_github_issue",
		Description: "Search between github issues",
		Properties: map[string]jsonschema.Definition{
			"query": {
				Type:        jsonschema.String,
				Description: "The text to search for",
			},
			"repository": {
				Type:        jsonschema.String,
				Description: "The repository to search in",
			},
			"owner": {
				Type:        jsonschema.String,
				Description: "The owner of the repository",
			},
		},
		Required: []string{"text", "repository", "owner"},
	}
}
