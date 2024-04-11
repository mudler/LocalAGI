package external

import (
	"context"
	"fmt"

	"github.com/google/go-github/v61/github"
	"github.com/mudler/local-agent-framework/action"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubIssuesOpener struct {
	token   string
	context context.Context
	client  *github.Client
}

func NewGithubIssueOpener(ctx context.Context, config map[string]string) *GithubIssuesOpener {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	return &GithubIssuesOpener{
		client:  client,
		token:   config["token"],
		context: ctx,
	}
}

func (g *GithubIssuesOpener) Run(params action.ActionParams) (string, error) {
	result := struct {
		Title      string `json:"title"`
		Body       string `json:"body"`
		Repository string `json:"repository"`
		Owner      string `json:"owner"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return "", err
	}

	issue := &github.IssueRequest{
		Title: &result.Title,
		Body:  &result.Body,
	}

	resultString := ""
	createdIssue, _, err := g.client.Issues.Create(g.context, result.Owner, result.Repository, issue)
	if err != nil {
		resultString = fmt.Sprintf("Error creating issue: %v", err)
	} else {
		resultString = fmt.Sprintf("Created issue %d in repository %s/%s", createdIssue.GetNumber(), result.Owner, result.Repository)
	}

	return resultString, err
}

func (g *GithubIssuesOpener) Definition() action.ActionDefinition {
	return action.ActionDefinition{
		Name:        "create_github_issue",
		Description: "Create a new issue on a GitHub repository.",
		Properties: map[string]jsonschema.Definition{
			"body": {
				Type:        jsonschema.String,
				Description: "The number of the issue to add the label to.",
			},
			"title": {
				Type:        jsonschema.String,
				Description: "The title of the issue.",
			},
			"owner": {
				Type:        jsonschema.String,
				Description: "The owner of the repository.",
			},
			"repository": {
				Type:        jsonschema.String,
				Description: "The repository to create the issue in.",
			},
		},
		Required: []string{"title", "body", "owner", "repository"},
	}
}
