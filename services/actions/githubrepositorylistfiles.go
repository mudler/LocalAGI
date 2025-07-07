package actions

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v69/github"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubRepositoryListFiles struct {
	token, repository, owner, customActionName string
	client                                     *github.Client
}

func NewGithubRepositoryListFiles(config map[string]string) *GithubRepositoryListFiles {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	return &GithubRepositoryListFiles{
		client:           client,
		token:            config["token"],
		repository:       config["repository"],
		owner:            config["owner"],
		customActionName: config["customActionName"],
	}
}

func (g *GithubRepositoryListFiles) listFilesRecursively(ctx context.Context, path string, owner string, repository string) ([]string, error) {
	var files []string

	// Get content at the current path
	_, directoryContent, _, err := g.client.Repositories.GetContents(ctx, owner, repository, path, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting content at path %s: %w", path, err)
	}

	// Process each item in the directory
	for _, item := range directoryContent {
		if item.GetType() == "dir" {
			// Recursively list files in subdirectories
			subFiles, err := g.listFilesRecursively(ctx, item.GetPath(), owner, repository)
			if err != nil {
				return nil, err
			}
			files = append(files, subFiles...)
		} else if item.GetType() == "file" {
			// Add file path to the list
			files = append(files, item.GetPath())
		}
	}

	return files, nil
}

func (g *GithubRepositoryListFiles) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Repository string `json:"repository"`
		Owner      string `json:"owner"`
		Path       string `json:"path,omitempty"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to unmarshal params: %w", err)
	}

	if g.repository != "" && g.owner != "" {
		result.Repository = g.repository
		result.Owner = g.owner
	}

	// Start from root if no path specified
	if result.Path == "" {
		result.Path = "."
	}

	files, err := g.listFilesRecursively(ctx, result.Path, result.Owner, result.Repository)
	if err != nil {
		return types.ActionResult{}, err
	}

	// Join all file paths with newlines for better readability
	content := strings.Join(files, "\n")
	return types.ActionResult{Result: content}, nil
}

func (g *GithubRepositoryListFiles) Definition() types.ActionDefinition {
	actionName := "list_github_repository_files"
	if g.customActionName != "" {
		actionName = g.customActionName
	}
	description := "List all files in a GitHub repository"
	if g.repository != "" && g.owner != "" {
		return types.ActionDefinition{
			Name:        types.ActionDefinitionName(actionName),
			Description: description,
			Properties: map[string]jsonschema.Definition{
				"path": {
					Type:        jsonschema.String,
					Description: "Optional path to start listing from (defaults to repository root)",
				},
			},
		}
	}
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(actionName),
		Description: description,
		Properties: map[string]jsonschema.Definition{
			"path": {
				Type:        jsonschema.String,
				Description: "Optional path to start listing from (defaults to repository root)",
			},
			"repository": {
				Type:        jsonschema.String,
				Description: "The repository to list files from",
			},
			"owner": {
				Type:        jsonschema.String,
				Description: "The owner of the repository",
			},
		},
		Required: []string{"repository", "owner"},
	}
}

func (a *GithubRepositoryListFiles) Plannable() bool {
	return true
}

// GithubRepositoryListFilesConfigMeta returns the metadata for GitHub Repository List Files action configuration fields
func GithubRepositoryListFilesConfigMeta() []config.Field {
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
