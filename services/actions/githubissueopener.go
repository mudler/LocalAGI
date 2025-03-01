package actions

import (
	"context"
	"fmt"

	"github.com/google/go-github/v61/github"
	"github.com/mudler/LocalAgent/core/action"
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

func (g *GithubIssuesOpener) Run(ctx context.Context, params action.ActionParams) (action.ActionResult, error) {
	result := struct {
		Title      string `json:"title"`
		Body       string `json:"text"`
		Repository string `json:"repository"`
		Owner      string `json:"owner"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return action.ActionResult{}, err
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

	return action.ActionResult{Result: resultString}, err
}

func (g *GithubIssuesOpener) Definition() action.ActionDefinition {
	return action.ActionDefinition{
		Name:        "create_github_issue",
		Description: "Create a new issue on a GitHub repository.",
		Properties: map[string]jsonschema.Definition{
			"text": {
				Type:        jsonschema.String,
				Description: "The text of the new issue",
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
				Description: "The repository where to create the issue.",
			},
		},
		Required: []string{"title", "text", "owner", "repository"},
	}
}
