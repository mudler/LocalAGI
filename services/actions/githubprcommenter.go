package actions

import (
	"context"
	"fmt"

	"github.com/google/go-github/v69/github"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubPRCommenter struct {
	token, repository, owner, customActionName string
	client                                     *github.Client
}

func NewGithubPRCommenter(config map[string]string) *GithubPRCommenter {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	return &GithubPRCommenter{
		client:           client,
		token:            config["token"],
		customActionName: config["customActionName"],
		repository:       config["repository"],
		owner:            config["owner"],
	}
}

func (g *GithubPRCommenter) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Repository string `json:"repository"`
		Owner      string `json:"owner"`
		PRNumber   int    `json:"pr_number"`
		Comment    string `json:"comment"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to unmarshal params: %w", err)
	}

	if g.repository != "" && g.owner != "" {
		result.Repository = g.repository
		result.Owner = g.owner
	}

	// First verify the PR exists and is in a valid state
	pr, _, err := g.client.PullRequests.Get(ctx, result.Owner, result.Repository, result.PRNumber)
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to fetch PR #%d: %w", result.PRNumber, err)
	}
	if pr == nil {
		return types.ActionResult{Result: fmt.Sprintf("Pull request #%d not found in repository %s/%s", result.PRNumber, result.Owner, result.Repository)}, nil
	}

	// Check if PR is in a state that allows comments
	if *pr.State != "open" {
		return types.ActionResult{Result: fmt.Sprintf("Pull request #%d is not open (current state: %s)", result.PRNumber, *pr.State)}, nil
	}

	if result.Comment == "" {
		return types.ActionResult{Result: "No comment provided"}, nil
	}

	// Try both PullRequests and Issues API for general comments
	var resp *github.Response

	// First try PullRequests API
	_, resp, err = g.client.PullRequests.CreateComment(ctx, result.Owner, result.Repository, result.PRNumber, &github.PullRequestComment{
		Body: &result.Comment,
	})

	// If that fails with 403, try Issues API
	if err != nil && resp != nil && resp.StatusCode == 403 {
		_, resp, err = g.client.Issues.CreateComment(ctx, result.Owner, result.Repository, result.PRNumber, &github.IssueComment{
			Body: &result.Comment,
		})
	}

	if err != nil {
		return types.ActionResult{Result: fmt.Sprintf("Error adding general comment: %s", err.Error())}, nil
	}

	return types.ActionResult{
		Result: "Successfully added general comment to pull request",
	}, nil
}

func (g *GithubPRCommenter) Definition() types.ActionDefinition {
	actionName := "comment_github_pr"
	if g.customActionName != "" {
		actionName = g.customActionName
	}
	description := "Add comments to a GitHub pull request, including line-specific feedback. Often used after reading a PR to provide a peer review."
	if g.repository != "" && g.owner != "" {
		return types.ActionDefinition{
			Name:        types.ActionDefinitionName(actionName),
			Description: description,
			Properties: map[string]jsonschema.Definition{
				"pr_number": {
					Type:        jsonschema.Number,
					Description: "The number of the pull request to comment on.",
				},
				"comment": {
					Type:        jsonschema.String,
					Description: "A general comment to add to the pull request.",
				},
			},
			Required: []string{"pr_number", "comment"},
		}
	}
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(actionName),
		Description: description,
		Properties: map[string]jsonschema.Definition{
			"pr_number": {
				Type:        jsonschema.Number,
				Description: "The number of the pull request to comment on.",
			},
			"repository": {
				Type:        jsonschema.String,
				Description: "The repository containing the pull request.",
			},
			"owner": {
				Type:        jsonschema.String,
				Description: "The owner of the repository.",
			},
			"comment": {
				Type:        jsonschema.String,
				Description: "A general comment to add to the pull request.",
			},
		},
		Required: []string{"pr_number", "repository", "owner", "comment"},
	}
}

func (a *GithubPRCommenter) Plannable() bool {
	return true
}

// GithubPRCommenterConfigMeta returns the metadata for GitHub PR Commenter action configuration fields
func GithubPRCommenterConfigMeta() []config.Field {
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
