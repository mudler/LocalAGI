package actions

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/google/go-github/v69/github"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubPRReviewer struct {
	token, repository, owner, customActionName string
	client                                     *github.Client
}

func NewGithubPRReviewer(config map[string]string) *GithubPRReviewer {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	return &GithubPRReviewer{
		client:           client,
		token:            config["token"],
		customActionName: config["customActionName"],
		repository:       config["repository"],
		owner:            config["owner"],
	}
}

func (g *GithubPRReviewer) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Repository    string `json:"repository"`
		Owner         string `json:"owner"`
		PRNumber      int    `json:"pr_number"`
		ReviewComment string `json:"review_comment"`
		ReviewAction  string `json:"review_action"` // APPROVE, REQUEST_CHANGES, or COMMENT
		Comments      []struct {
			File      string `json:"file"`
			Line      int    `json:"line"`
			Comment   string `json:"comment"`
			StartLine int    `json:"start_line,omitempty"`
		} `json:"comments"`
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

	// Check if PR is in a state that allows reviews
	if *pr.State != "open" {
		return types.ActionResult{Result: fmt.Sprintf("Pull request #%d is not open (current state: %s)", result.PRNumber, *pr.State)}, nil
	}

	// Get the list of changed files to verify the files exist in the PR
	files, _, err := g.client.PullRequests.ListFiles(ctx, result.Owner, result.Repository, result.PRNumber, &github.ListOptions{})
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to list PR files: %w", err)
	}

	// Create a map of valid files
	validFiles := make(map[string]bool)
	for _, file := range files {
		if *file.Status != "deleted" {
			validFiles[*file.Filename] = true
		}
	}

	// Process each comment
	var reviewComments []*github.DraftReviewComment
	for _, comment := range result.Comments {
		// Check if file exists in PR
		if !validFiles[comment.File] {
			continue
		}

		reviewComment := &github.DraftReviewComment{
			Path: &comment.File,
			Line: &comment.Line,
			Body: &comment.Comment,
		}

		// Set start line if provided
		if comment.StartLine > 0 {
			reviewComment.StartLine = &comment.StartLine
		}

		reviewComments = append(reviewComments, reviewComment)
	}

	// Create the review
	review := &github.PullRequestReviewRequest{
		Event:    &result.ReviewAction,
		Body:     &result.ReviewComment,
		Comments: reviewComments,
	}

	xlog.Debug("[githubprreviewer] review", "review", review)

	// Submit the review
	_, resp, err := g.client.PullRequests.CreateReview(ctx, result.Owner, result.Repository, result.PRNumber, review)
	if err != nil {
		errorDetails := fmt.Sprintf("Error submitting review: %s", err.Error())
		if resp != nil {
			errorDetails += fmt.Sprintf("\nResponse Status: %s", resp.Status)
			if resp.Body != nil {
				body, _ := io.ReadAll(resp.Body)
				errorDetails += fmt.Sprintf("\nResponse Body: %s", string(body))
			}
		}
		return types.ActionResult{Result: errorDetails}, err
	}

	actionResult := fmt.Sprintf(
		"Pull request https://github.com/%s/%s/pull/%d reviewed successfully with status: %s",
		result.Owner,
		result.Repository,
		result.PRNumber,
		strings.ToLower(result.ReviewAction),
	)

	return types.ActionResult{Result: actionResult}, nil
}

func (g *GithubPRReviewer) Definition() types.ActionDefinition {
	actionName := "review_github_pr"
	if g.customActionName != "" {
		actionName = g.customActionName
	}
	description := "Review a GitHub pull request by approving, requesting changes, or commenting."
	if g.repository != "" && g.owner != "" {
		return types.ActionDefinition{
			Name:        types.ActionDefinitionName(actionName),
			Description: description,
			Properties: map[string]jsonschema.Definition{
				"pr_number": {
					Type:        jsonschema.Number,
					Description: "The number of the pull request to review.",
				},
				"review_comment": {
					Type:        jsonschema.String,
					Description: "The main review comment to add to the pull request.",
				},
				"review_action": {
					Type:        jsonschema.String,
					Description: "The type of review to submit (APPROVE, REQUEST_CHANGES, or COMMENT).",
					Enum:        []string{"APPROVE", "REQUEST_CHANGES", "COMMENT"},
				},
				"comments": {
					Type: jsonschema.Array,
					Items: &jsonschema.Definition{
						Type: jsonschema.Object,
						Properties: map[string]jsonschema.Definition{
							"file": {
								Type:        jsonschema.String,
								Description: "The file to comment on.",
							},
							"line": {
								Type:        jsonschema.Number,
								Description: "The line number to comment on.",
							},
							"comment": {
								Type:        jsonschema.String,
								Description: "The comment text.",
							},
							"start_line": {
								Type:        jsonschema.Number,
								Description: "Optional start line for multi-line comments.",
							},
						},
						Required: []string{"file", "line", "comment"},
					},
					Description: "Array of line-specific comments to add to the review.",
				},
			},
			Required: []string{"pr_number", "review_action"},
		}
	}
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(actionName),
		Description: description,
		Properties: map[string]jsonschema.Definition{
			"pr_number": {
				Type:        jsonschema.Number,
				Description: "The number of the pull request to review.",
			},
			"repository": {
				Type:        jsonschema.String,
				Description: "The repository containing the pull request.",
			},
			"owner": {
				Type:        jsonschema.String,
				Description: "The owner of the repository.",
			},
			"review_comment": {
				Type:        jsonschema.String,
				Description: "The main review comment to add to the pull request.",
			},
			"review_action": {
				Type:        jsonschema.String,
				Description: "The type of review to submit (APPROVE, REQUEST_CHANGES, or COMMENT).",
				Enum:        []string{"APPROVE", "REQUEST_CHANGES", "COMMENT"},
			},
			"comments": {
				Type: jsonschema.Array,
				Items: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"file": {
							Type:        jsonschema.String,
							Description: "The file to comment on.",
						},
						"line": {
							Type:        jsonschema.Number,
							Description: "The line number to comment on.",
						},
						"comment": {
							Type:        jsonschema.String,
							Description: "The comment text.",
						},
						"start_line": {
							Type:        jsonschema.Number,
							Description: "Optional start line for multi-line comments.",
						},
					},
					Required: []string{"file", "line", "comment"},
				},
				Description: "Array of line-specific comments to add to the review.",
			},
		},
		Required: []string{"pr_number", "repository", "owner", "review_action"},
	}
}

func (a *GithubPRReviewer) Plannable() bool {
	return true
}

// GithubPRReviewerConfigMeta returns the metadata for GitHub PR Reviewer action configuration fields
func GithubPRReviewerConfigMeta() []config.Field {
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
