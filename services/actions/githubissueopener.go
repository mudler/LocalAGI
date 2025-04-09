package actions

import (
	"context"
	"fmt"

	"github.com/google/go-github/v69/github"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubIssuesOpener struct {
	token, repository, owner, customActionName string
	client                                     *github.Client
}

func NewGithubIssueOpener(config map[string]string) *GithubIssuesOpener {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	return &GithubIssuesOpener{
		client:           client,
		token:            config["token"],
		repository:       config["repository"],
		owner:            config["owner"],
		customActionName: config["customActionName"],
	}
}

func (g *GithubIssuesOpener) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Title      string `json:"title"`
		Body       string `json:"text"`
		Repository string `json:"repository"`
		Owner      string `json:"owner"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return types.ActionResult{}, err
	}

	if g.repository != "" && g.owner != "" {
		result.Repository = g.repository
		result.Owner = g.owner
	}

	issue := &github.IssueRequest{
		Title: &result.Title,
		Body:  &result.Body,
	}

	resultString := ""
	createdIssue, _, err := g.client.Issues.Create(ctx, result.Owner, result.Repository, issue)
	if err != nil {
		resultString = fmt.Sprintf("Error creating issue: %v", err)
	} else {
		resultString = fmt.Sprintf("Created issue %d in repository %s/%s: %s", createdIssue.GetNumber(), result.Owner, result.Repository, createdIssue.GetURL())
	}

	return types.ActionResult{Result: resultString}, err
}

func (g *GithubIssuesOpener) Definition() types.ActionDefinition {
	actionName := "create_github_issue"
	if g.customActionName != "" {
		actionName = g.customActionName
	}
	if g.repository != "" && g.owner != "" {
		return types.ActionDefinition{
			Name:        types.ActionDefinitionName(actionName),
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
			},
			Required: []string{"title", "text"},
		}
	}
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(actionName),
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

func (a *GithubIssuesOpener) Plannable() bool {
	return true
}

// GithubIssueOpenerConfigMeta returns the metadata for GitHub Issue Opener action configuration fields
func GithubIssueOpenerConfigMeta() []config.Field {
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
			Required: true,
			HelpText: "GitHub repository name",
		},
		{
			Name:     "owner",
			Label:    "Owner",
			Type:     config.FieldTypeText,
			Required: true,
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
