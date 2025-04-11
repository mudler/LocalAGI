package actions

import (
	"context"
	"fmt"

	"github.com/google/go-github/v69/github"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubPRReader struct {
	token, repository, owner, customActionName string
	showFullDiff                               bool
	client                                     *github.Client
}

func NewGithubPRReader(config map[string]string) *GithubPRReader {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	showFullDiff := false
	if config["showFullDiff"] == "true" {
		showFullDiff = true
	}

	return &GithubPRReader{
		client:           client,
		token:            config["token"],
		customActionName: config["customActionName"],
		repository:       config["repository"],
		owner:            config["owner"],
		showFullDiff:     showFullDiff,
	}
}

func (g *GithubPRReader) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Repository string `json:"repository"`
		Owner      string `json:"owner"`
		PRNumber   int    `json:"pr_number"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		return types.ActionResult{}, err
	}

	if g.repository != "" && g.owner != "" {
		result.Repository = g.repository
		result.Owner = g.owner
	}

	pr, _, err := g.client.PullRequests.Get(ctx, result.Owner, result.Repository, result.PRNumber)
	if err != nil {
		return types.ActionResult{Result: fmt.Sprintf("Error fetching pull request: %s", err.Error())}, err
	}
	if pr == nil {
		return types.ActionResult{Result: fmt.Sprintf("No pull request found")}, nil
	}

	// Get the list of changed files
	files, _, err := g.client.PullRequests.ListFiles(ctx, result.Owner, result.Repository, result.PRNumber, &github.ListOptions{})
	if err != nil {
		return types.ActionResult{Result: fmt.Sprintf("Error fetching pull request files: %s", err.Error())}, err
	}

	// Get CI status information
	ciStatus := "\n\nCI Status:\n"

	// Get PR status checks
	checkRuns, _, err := g.client.Checks.ListCheckRunsForRef(ctx, result.Owner, result.Repository, pr.GetHead().GetSHA(), &github.ListCheckRunsOptions{})
	if err == nil && checkRuns != nil {
		ciStatus += fmt.Sprintf("\nPR Status Checks:\n")
		ciStatus += fmt.Sprintf("Total Checks: %d\n", checkRuns.GetTotal())
		for _, check := range checkRuns.CheckRuns {
			ciStatus += fmt.Sprintf("- %s: %s (%s)\n",
				check.GetName(),
				check.GetConclusion(),
				check.GetStatus())
		}
	}

	// Build the file changes summary with patches
	fileChanges := "\n\nFile Changes:\n"
	for _, file := range files {
		fileChanges += fmt.Sprintf("\n--- %s\n+++ %s\n", file.GetFilename(), file.GetFilename())
		if g.showFullDiff && file.GetPatch() != "" {
			fileChanges += file.GetPatch()
		}
		fileChanges += fmt.Sprintf("\n(%d additions, %d deletions)\n", file.GetAdditions(), file.GetDeletions())
	}

	return types.ActionResult{
		Result: fmt.Sprintf(
			"Pull Request %d Repository: %s\nTitle: %s\nBody: %s\nState: %s\nBase: %s\nHead: %s%s%s",
			pr.GetNumber(),
			pr.GetBase().GetRepo().GetFullName(),
			pr.GetTitle(),
			pr.GetBody(),
			pr.GetState(),
			pr.GetBase().GetRef(),
			pr.GetHead().GetRef(),
			ciStatus,
			fileChanges)}, nil
}

func (g *GithubPRReader) Definition() types.ActionDefinition {
	actionName := "read_github_pr"
	if g.customActionName != "" {
		actionName = g.customActionName
	}
	description := "Read a GitHub pull request."
	if g.repository != "" && g.owner != "" {
		return types.ActionDefinition{
			Name:        types.ActionDefinitionName(actionName),
			Description: description,
			Properties: map[string]jsonschema.Definition{
				"pr_number": {
					Type:        jsonschema.Number,
					Description: "The number of the pull request to read.",
				},
			},
			Required: []string{"pr_number"},
		}
	}
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(actionName),
		Description: description,
		Properties: map[string]jsonschema.Definition{
			"pr_number": {
				Type:        jsonschema.Number,
				Description: "The number of the pull request to read.",
			},
			"repository": {
				Type:        jsonschema.String,
				Description: "The repository containing the pull request.",
			},
			"owner": {
				Type:        jsonschema.String,
				Description: "The owner of the repository.",
			},
		},
		Required: []string{"pr_number", "repository", "owner"},
	}
}

func (a *GithubPRReader) Plannable() bool {
	return true
}

// GithubPRReaderConfigMeta returns the metadata for GitHub PR Reader action configuration fields
func GithubPRReaderConfigMeta() []config.Field {
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
		{
			Name:     "showFullDiff",
			Label:    "Show Full Diff",
			Type:     config.FieldTypeCheckbox,
			HelpText: "Whether to show the full diff content or just the summary",
		},
	}
}
