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

type GithubRepositorySearchFiles struct {
	token, repository, owner, customActionName string
	client                                     *github.Client
}

func NewGithubRepositorySearchFiles(config map[string]string) *GithubRepositorySearchFiles {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	return &GithubRepositorySearchFiles{
		client:           client,
		token:            config["token"],
		repository:       config["repository"],
		owner:            config["owner"],
		customActionName: config["customActionName"],
	}
}

func (g *GithubRepositorySearchFiles) searchFilesRecursively(ctx context.Context, path string, owner string, repository string, searchPattern string) (string, error) {
	var result strings.Builder

	// Get content at the current path
	_, directoryContent, _, err := g.client.Repositories.GetContents(ctx, owner, repository, path, nil)
	if err != nil {
		return "", fmt.Errorf("error getting content at path %s: %w", path, err)
	}

	// Process each item in the directory
	for _, item := range directoryContent {
		if item.GetType() == "dir" {
			// Recursively search in subdirectories
			subContent, err := g.searchFilesRecursively(ctx, item.GetPath(), owner, repository, searchPattern)
			if err != nil {
				return "", err
			}
			result.WriteString(subContent)
		} else if item.GetType() == "file" {
			// Check if file name matches the search pattern
			if strings.Contains(strings.ToLower(item.GetName()), strings.ToLower(searchPattern)) {
				// Get file content
				fileContent, _, _, err := g.client.Repositories.GetContents(ctx, owner, repository, item.GetPath(), nil)
				if err != nil {
					return "", fmt.Errorf("error getting file content for %s: %w", item.GetPath(), err)
				}

				content, err := fileContent.GetContent()
				if err != nil {
					return "", fmt.Errorf("error decoding content for %s: %w", item.GetPath(), err)
				}

				// Add file content to result with clear markers
				result.WriteString(fmt.Sprintf("\n--- START FILE: %s ---\n", item.GetPath()))
				result.WriteString(content)
				result.WriteString(fmt.Sprintf("\n--- END FILE: %s ---\n", item.GetPath()))
			}
		}
	}

	return result.String(), nil
}

func (g *GithubRepositorySearchFiles) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Repository    string `json:"repository"`
		Owner         string `json:"owner"`
		Path          string `json:"path,omitempty"`
		SearchPattern string `json:"searchPattern"`
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

	content, err := g.searchFilesRecursively(ctx, result.Path, result.Owner, result.Repository, result.SearchPattern)
	if err != nil {
		return types.ActionResult{}, err
	}

	return types.ActionResult{Result: content}, nil
}

func (g *GithubRepositorySearchFiles) Definition() types.ActionDefinition {
	actionName := "search_github_repository_files"
	if g.customActionName != "" {
		actionName = g.customActionName
	}
	description := "Search for files in a GitHub repository and return their content"
	if g.repository != "" && g.owner != "" {
		return types.ActionDefinition{
			Name:        types.ActionDefinitionName(actionName),
			Description: description,
			Properties: map[string]jsonschema.Definition{
				"path": {
					Type:        jsonschema.String,
					Description: "Optional path to start searching from (defaults to repository root)",
				},
				"searchPattern": {
					Type:        jsonschema.String,
					Description: "Pattern to search for in file names (case-insensitive)",
				},
			},
			Required: []string{"searchPattern"},
		}
	}
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(actionName),
		Description: description,
		Properties: map[string]jsonschema.Definition{
			"path": {
				Type:        jsonschema.String,
				Description: "Optional path to start searching from (defaults to repository root)",
			},
			"repository": {
				Type:        jsonschema.String,
				Description: "The repository to search in",
			},
			"owner": {
				Type:        jsonschema.String,
				Description: "The owner of the repository",
			},
			"searchPattern": {
				Type:        jsonschema.String,
				Description: "Pattern to search for in file names (case-insensitive)",
			},
		},
		Required: []string{"repository", "owner", "searchPattern"},
	}
}

func (a *GithubRepositorySearchFiles) Plannable() bool {
	return true
}

// GithubRepositorySearchFilesConfigMeta returns the metadata for GitHub Repository Search Files action configuration fields
func GithubRepositorySearchFilesConfigMeta() []config.Field {
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
