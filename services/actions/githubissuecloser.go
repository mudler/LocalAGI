package actions

import (
	"context"
	"fmt"

	"github.com/google/go-github/v69/github"
	"github.com/mudler/LocalAgent/core/action"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubIssuesCloser struct {
	token, repository, owner, customActionName string
	context                                    context.Context
	client                                     *github.Client
}

func NewGithubIssueCloser(ctx context.Context, config map[string]string) *GithubIssuesCloser {
	client := github.NewClient(nil).WithAuthToken(config["token"])
	return &GithubIssuesCloser{
		client:           client,
		token:            config["token"],
		repository:       config["repository"],
		owner:            config["owner"],
		customActionName: config["customActionName"],
		context:          ctx,
	}
}

func (g *GithubIssuesCloser) Run(ctx context.Context, params action.ActionParams) (action.ActionResult, error) {
	result := struct {
		Repository  string `json:"repository"`
		Owner       string `json:"owner"`
		IssueNumber int    `json:"issue_number"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return action.ActionResult{}, err
	}

	if g.repository != "" {
		result.Repository = g.repository
	}

	if g.owner != "" {
		result.Owner = g.owner
	}
	// _, _, err = g.client.Issues.CreateComment(
	// 	g.context,
	// 	result.Owner, result.Repository,
	// 	result.IssueNumber, &github.IssueComment{
	// 		//Body: &result.Text,
	// 	},
	// )
	// if err != nil {
	// 	fmt.Printf("error: %v", err)

	// 	return "", err
	// }

	_, _, err = g.client.Issues.Edit(g.context, result.Owner, result.Repository, result.IssueNumber, &github.IssueRequest{
		State: github.String("closed"),
	})

	if err != nil {
		fmt.Printf("error: %v", err)

		return action.ActionResult{}, err
	}

	resultString := fmt.Sprintf("Closed issue %d in repository %s/%s", result.IssueNumber, result.Owner, result.Repository)
	if err != nil {
		resultString = fmt.Sprintf("Error closing issue %d in repository %s/%s: %v", result.IssueNumber, result.Owner, result.Repository, err)
	}
	return action.ActionResult{Result: resultString}, err
}

func (g *GithubIssuesCloser) Definition() action.ActionDefinition {
	actionName := "close_github_issue"
	if g.customActionName != "" {
		actionName = g.customActionName
	}
	if g.repository != "" && g.owner != "" {
		return action.ActionDefinition{
			Name:        action.ActionDefinitionName(actionName),
			Description: "Closes a Github issue.",
			Properties: map[string]jsonschema.Definition{
				"issue_number": {
					Type:        jsonschema.Number,
					Description: "The issue number to close",
				},
			},
			Required: []string{"issue_number"},
		}
	}

	return action.ActionDefinition{
		Name:        action.ActionDefinitionName(actionName),
		Description: "Closes a Github issue.",
		Properties: map[string]jsonschema.Definition{
			"repository": {
				Type:        jsonschema.String,
				Description: "The repository to close the issue in.",
			},
			"owner": {
				Type:        jsonschema.String,
				Description: "The owner of the repository.",
			},
			"issue_number": {
				Type:        jsonschema.Number,
				Description: "The issue number to close",
			},
		},
		Required: []string{"issue_number", "repository", "owner"},
	}
}

func (a *GithubIssuesCloser) Plannable() bool {
	return true
}
