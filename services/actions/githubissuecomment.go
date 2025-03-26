package actions

import (
	"context"
	"fmt"

	"github.com/google/go-github/v69/github"
	"github.com/mudler/LocalAgent/core/types"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubIssuesCommenter struct {
	token, repository, owner, customActionName string
	client                                     *github.Client
}

func NewGithubIssueCommenter(config map[string]string) *GithubIssuesCommenter {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	return &GithubIssuesCommenter{
		client:           client,
		token:            config["token"],
		customActionName: config["customActionName"],
		repository:       config["repository"],
		owner:            config["owner"],
	}
}

func (g *GithubIssuesCommenter) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Repository  string `json:"repository"`
		Owner       string `json:"owner"`
		Comment     string `json:"comment"`
		IssueNumber int    `json:"issue_number"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		return types.ActionResult{}, err
	}

	if g.repository != "" && g.owner != "" {
		result.Repository = g.repository
		result.Owner = g.owner
	}

	_, _, err = g.client.Issues.CreateComment(ctx, result.Owner, result.Repository, result.IssueNumber,
		&github.IssueComment{
			Body: &result.Comment,
		})
	resultString := fmt.Sprintf("Added comment to issue %d in repository %s/%s", result.IssueNumber, result.Owner, result.Repository)
	if err != nil {
		resultString = fmt.Sprintf("Error adding comment to issue %d in repository %s/%s: %v", result.IssueNumber, result.Owner, result.Repository, err)
	}
	return types.ActionResult{Result: resultString}, err
}

func (g *GithubIssuesCommenter) Definition() types.ActionDefinition {
	actionName := "add_comment_to_github_issue"
	if g.customActionName != "" {
		actionName = g.customActionName
	}
	description := "Add a comment to a Github issue to a repository."
	if g.repository != "" && g.owner != "" {
		return types.ActionDefinition{
			Name:        types.ActionDefinitionName(actionName),
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
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(actionName),
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

func (a *GithubIssuesCommenter) Plannable() bool {
	return true
}
