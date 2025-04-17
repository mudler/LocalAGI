package actions

import (
	"context"
	"fmt"

	"github.com/google/go-github/v69/github"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubPRCreator struct {
	token, repository, owner, customActionName, defaultBranch string
	client                                                    *github.Client
}

func NewGithubPRCreator(config map[string]string) *GithubPRCreator {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	defaultBranch := config["defaultBranch"]
	if defaultBranch == "" {
		defaultBranch = "main" // Default to "main" if not specified
	}

	return &GithubPRCreator{
		client:           client,
		token:            config["token"],
		repository:       config["repository"],
		owner:            config["owner"],
		customActionName: config["customActionName"],
		defaultBranch:    defaultBranch,
	}
}

func (g *GithubPRCreator) createOrUpdateBranch(ctx context.Context, branchName string, owner string, repository string) error {
	// Get the latest commit SHA from the default branch
	ref, _, err := g.client.Git.GetRef(ctx, owner, repository, "refs/heads/"+g.defaultBranch)
	if err != nil {
		return fmt.Errorf("failed to get reference for default branch %s: %w", g.defaultBranch, err)
	}

	// Try to get the branch if it exists
	_, resp, err := g.client.Git.GetRef(ctx, owner, repository, "refs/heads/"+branchName)
	if err != nil {
		if resp == nil {
			return fmt.Errorf("failed to check branch existence: %w", err)
		}

		// If branch doesn't exist (404), create it
		if resp.StatusCode == 404 {
			newRef := &github.Reference{
				Ref:    github.String("refs/heads/" + branchName),
				Object: &github.GitObject{SHA: ref.Object.SHA},
			}
			_, _, err = g.client.Git.CreateRef(ctx, owner, repository, newRef)
			if err != nil {
				return fmt.Errorf("failed to create branch: %w", err)
			}
			return nil
		}

		// For other errors, return the error
		return fmt.Errorf("failed to check branch existence: %w", err)
	}

	// Branch exists, update it to the latest commit
	updateRef := &github.Reference{
		Ref:    github.String("refs/heads/" + branchName),
		Object: &github.GitObject{SHA: ref.Object.SHA},
	}
	_, _, err = g.client.Git.UpdateRef(ctx, owner, repository, updateRef, true)
	if err != nil {
		return fmt.Errorf("failed to update branch: %w", err)
	}

	return nil
}

func (g *GithubPRCreator) createOrUpdateFile(ctx context.Context, branch string, filePath string, content string, message string, owner string, repository string) error {
	// Get the current file content if it exists
	var sha *string
	fileContent, _, _, err := g.client.Repositories.GetContents(ctx, owner, repository, filePath, &github.RepositoryContentGetOptions{
		Ref: branch,
	})
	if err == nil && fileContent != nil {
		sha = fileContent.SHA
	}

	// Create or update the file
	_, _, err = g.client.Repositories.CreateFile(ctx, owner, repository, filePath, &github.RepositoryContentFileOptions{
		Message: &message,
		Content: []byte(content),
		Branch:  &branch,
		SHA:     sha,
	})
	if err != nil {
		return fmt.Errorf("failed to create/update file: %w", err)
	}

	return nil
}

func (g *GithubPRCreator) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Repository string `json:"repository"`
		Owner      string `json:"owner"`
		Branch     string `json:"branch"`
		Title      string `json:"title"`
		Body       string `json:"body"`
		BaseBranch string `json:"base_branch"`
		Files      []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		} `json:"files"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to unmarshal params: %w", err)
	}

	if g.repository != "" && g.owner != "" {
		result.Repository = g.repository
		result.Owner = g.owner
	}

	if result.BaseBranch == "" {
		result.BaseBranch = g.defaultBranch
	}

	// Create or update branch
	err = g.createOrUpdateBranch(ctx, result.Branch, result.Owner, result.Repository)
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to create/update branch: %w", err)
	}

	// Create or update files
	for _, file := range result.Files {
		err = g.createOrUpdateFile(ctx, result.Branch, file.Path, file.Content, fmt.Sprintf("Update %s", file.Path), result.Owner, result.Repository)
		if err != nil {
			return types.ActionResult{}, fmt.Errorf("failed to update file %s: %w", file.Path, err)
		}
	}

	// Check if PR already exists for this branch
	prs, _, err := g.client.PullRequests.List(ctx, result.Owner, result.Repository, &github.PullRequestListOptions{
		State: "open",
		Head:  fmt.Sprintf("%s:%s", result.Owner, result.Branch),
	})
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to list pull requests: %w", err)
	}

	if len(prs) > 0 {
		// Update existing PR
		pr := prs[0]
		update := &github.PullRequest{
			Title: &result.Title,
			Body:  &result.Body,
		}
		updatedPR, _, err := g.client.PullRequests.Edit(ctx, result.Owner, result.Repository, pr.GetNumber(), update)
		if err != nil {
			return types.ActionResult{}, fmt.Errorf("failed to update pull request: %w", err)
		}
		return types.ActionResult{
			Result: fmt.Sprintf("Updated pull request #%d: %s", updatedPR.GetNumber(), updatedPR.GetHTMLURL()),
		}, nil
	}

	// Create new pull request
	newPR := &github.NewPullRequest{
		Title: &result.Title,
		Body:  &result.Body,
		Head:  &result.Branch,
		Base:  &result.BaseBranch,
	}

	createdPR, _, err := g.client.PullRequests.Create(ctx, result.Owner, result.Repository, newPR)
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to create pull request: %w", err)
	}

	return types.ActionResult{
		Result: fmt.Sprintf("Created pull request #%d: %s", createdPR.GetNumber(), createdPR.GetHTMLURL()),
	}, nil
}

func (g *GithubPRCreator) Definition() types.ActionDefinition {
	actionName := "create_github_pr"
	if g.customActionName != "" {
		actionName = g.customActionName
	}
	description := "Create a GitHub pull request with file changes"
	if g.repository != "" && g.owner != "" && g.defaultBranch != "" {
		return types.ActionDefinition{
			Name:        types.ActionDefinitionName(actionName),
			Description: description,
			Properties: map[string]jsonschema.Definition{
				"branch": {
					Type:        jsonschema.String,
					Description: "The name of the new branch to create",
				},
				"title": {
					Type:        jsonschema.String,
					Description: "The title of the pull request",
				},
				"body": {
					Type:        jsonschema.String,
					Description: "The body/description of the pull request",
				},
				"files": {
					Type: jsonschema.Array,
					Items: &jsonschema.Definition{
						Type: jsonschema.Object,
						Properties: map[string]jsonschema.Definition{
							"path": {
								Type:        jsonschema.String,
								Description: "The path of the file to create/update",
							},
							"content": {
								Type:        jsonschema.String,
								Description: "The content of the file",
							},
						},
						Required: []string{"path", "content"},
					},
					Description: "Array of files to create or update",
				},
			},
			Required: []string{"branch", "title", "files"},
		}
	}
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(actionName),
		Description: description,
		Properties: map[string]jsonschema.Definition{
			"branch": {
				Type:        jsonschema.String,
				Description: "The name of the new branch to create",
			},
			"title": {
				Type:        jsonschema.String,
				Description: "The title of the pull request",
			},
			"body": {
				Type:        jsonschema.String,
				Description: "The body/description of the pull request",
			},
			"base_branch": {
				Type:        jsonschema.String,
				Description: "The base branch to merge into (defaults to configured default branch)",
			},
			"files": {
				Type: jsonschema.Array,
				Items: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"path": {
							Type:        jsonschema.String,
							Description: "The path of the file to create/update",
						},
						"content": {
							Type:        jsonschema.String,
							Description: "The content of the file",
						},
					},
					Required: []string{"path", "content"},
				},
				Description: "Array of files to create or update",
			},
			"repository": {
				Type:        jsonschema.String,
				Description: "The repository to create the pull request in",
			},
			"owner": {
				Type:        jsonschema.String,
				Description: "The owner of the repository",
			},
		},
		Required: []string{"branch", "title", "files", "repository", "owner"},
	}
}

func (a *GithubPRCreator) Plannable() bool {
	return true
}

// GithubPRCreatorConfigMeta returns the metadata for GitHub PR Creator action configuration fields
func GithubPRCreatorConfigMeta() []config.Field {
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
			Name:     "defaultBranch",
			Label:    "Default Branch",
			Type:     config.FieldTypeText,
			Required: false,
			HelpText: "Default branch to create PRs against (defaults to main)",
		},
	}
}
