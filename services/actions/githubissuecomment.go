package actions

import (
	"context"
	"fmt"

	"github.com/google/go-github/v69/github"
	"github.com/mudler/LocalAgent/core/action"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubIssuesCommenter struct {
	token, repository, owner, customActionName string
	context                                    context.Context
	client                                     *github.Client
}

func NewGithubIssueCommenter(ctx context.Context, config map[string]string) *GithubIssuesCommenter {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	return &GithubIssuesCommenter{
		client:           client,
		token:            config["token"],
		customActionName: config["customActionName"],
		repository:       config["repository"],
		owner:            config["owner"],
		context:          ctx,
	}
}

func (g *GithubIssuesCommenter) Run(ctx context.Context, params action.ActionParams) (action.ActionResult, error) {
	result := struct {
		Repository  string `json:"repository"`
		Owner       string `json:"owner"`
		Comment     string `json:"comment"`
		IssueNumber int    `json:"issue_number"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		return action.ActionResult{}, err
	}

	if g.repository != "" && g.owner != "" {
		result.Repository = g.repository
		result.Owner = g.owner
	}

	_, _, err = g.client.Issues.CreateComment(g.context, result.Owner, result.Repository, result.IssueNumber,
		&github.IssueComment{
			Body: &result.Comment,
		})
	resultString := fmt.Sprintf("Added comment to issue %d in repository %s/%s", result.IssueNumber, result.Owner, result.Repository)
	if err != nil {
		resultString = fmt.Sprintf("Error adding comment to issue %d in repository %s/%s: %v", result.IssueNumber, result.Owner, result.Repository, err)
	}
	return action.ActionResult{Result: resultString}, err
}

func (g *GithubIssuesCommenter) Definition() action.ActionDefinition {
	actionName := "add_comment_to_github_issue"
	if g.customActionName != "" {
		actionName = g.customActionName
	}
	description := "Add a comment to a Github issue to a repository."
	if g.repository != "" && g.owner != "" {
		return action.ActionDefinition{
			Name:        action.ActionDefinitionName(actionName),
			Description: description,
			Properties: map[string]jsonschema.Definition{
				"issue_number": {
					Type:        jsonschema.Number,
					Description: "The number of the issue to add the label to.",
				},
				"comment": {
					Type:        jsonschema.String,
					Description: "The comment to add to the issue.",
				},
			},
			Required: []string{"issue_number", "comment"},
		}
	}
	return action.ActionDefinition{
		Name:        action.ActionDefinitionName(actionName),
		Description: description,
		Properties: map[string]jsonschema.Definition{
			"issue_number": {
				Type:        jsonschema.Number,
				Description: "The number of the issue to add the label to.",
			},
			"repository": {
				Type:        jsonschema.String,
				Description: "The repository to add the label to.",
			},
			"owner": {
				Type:        jsonschema.String,
				Description: "The owner of the repository.",
			},
			"comment": {
				Type:        jsonschema.String,
				Description: "The comment to add to the issue.",
			},
		},
		Required: []string{"issue_number", "repository", "owner", "comment"},
	}
}
