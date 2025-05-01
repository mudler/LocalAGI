package actions

import (
	"context"
	"fmt"

	"github.com/google/go-github/v69/github"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubIssueEditor struct {
	token, repository, owner, customActionName string
	client                                     *github.Client
}

func NewGithubIssueEditor(config map[string]string) *GithubIssueEditor {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	return &GithubIssueEditor{
		client:           client,
		token:            config["token"],
		customActionName: config["customActionName"],
		repository:       config["repository"],
		owner:            config["owner"],
	}
}

func (g *GithubIssueEditor) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Repository  string `json:"repository"`
		Owner       string `json:"owner"`
		Description string `json:"description"`
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

	_, _, err = g.client.Issues.Edit(ctx, result.Owner, result.Repository, result.IssueNumber,
		&github.IssueRequest{
			Body: &result.Description,
		})
	resultString := fmt.Sprintf("Updated description for issue %d in repository %s/%s", result.IssueNumber, result.Owner, result.Repository)
	if err != nil {
		resultString = fmt.Sprintf("Error updating description for issue %d in repository %s/%s: %v", result.IssueNumber, result.Owner, result.Repository, err)
	}
	return types.ActionResult{Result: resultString}, err
}

func (g *GithubIssueEditor) Definition() types.ActionDefinition {
	actionName := "edit_github_issue"
	if g.customActionName != "" {
		actionName = g.customActionName
	}
	description := "Edit the description of a Github issue in a repository."
	if g.repository != "" && g.owner != "" {
		return types.ActionDefinition{
			Name:        types.ActionDefinitionName(actionName),
			Description: description,
			Properties: map[string]jsonschema.Definition{
				"issue_number": {
					Type:        jsonschema.Number,
					Description: "The number of the issue to edit.",
				},
				"description": {
					Type:        jsonschema.String,
					Description: "The new description for the issue.",
				},
			},
			Required: []string{"issue_number", "description"},
		}
	}
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(actionName),
		Description: description,
		Properties: map[string]jsonschema.Definition{
			"issue_number": {
				Type:        jsonschema.Number,
				Description: "The number of the issue to edit.",
			},
			"repository": {
				Type:        jsonschema.String,
				Description: "The repository containing the issue.",
			},
			"owner": {
				Type:        jsonschema.String,
				Description: "The owner of the repository.",
			},
			"description": {
				Type:        jsonschema.String,
				Description: "The new description for the issue.",
			},
		},
		Required: []string{"issue_number", "repository", "owner", "description"},
	}
}

func (a *GithubIssueEditor) Plannable() bool {
	return true
}

// GithubIssueEditorConfigMeta returns the metadata for GitHub Issue Editor action configuration fields
func GithubIssueEditorConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "token",
			Label:    "GitHub Token",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "GitHub API token with repository access",
		},
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     config.FieldTypeText,
			Required: false,
			HelpText: "GitHub repository name",
		},
		{
			Name:     "owner",
			Label:    "Owner",
			Type:     config.FieldTypeText,
			Required: false,
			HelpText: "GitHub repository owner",
		},
		{
			Name:     "customActionName",
			Label:    "Custom Action Name",
			Type:     config.FieldTypeText,
			HelpText: "Custom name for this action",
		},
	}
}
