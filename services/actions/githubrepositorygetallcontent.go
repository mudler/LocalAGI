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

type GithubRepositoryGetAllContent struct {
	token, repository, owner, customActionName string
	client                                     *github.Client
}

func NewGithubRepositoryGetAllContent(config map[string]string) *GithubRepositoryGetAllContent {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	return &GithubRepositoryGetAllContent{
		client:           client,
		token:            config["token"],
		repository:       config["repository"],
		owner:            config["owner"],
		customActionName: config["customActionName"],
	}
}

func (g *GithubRepositoryGetAllContent) getContentRecursively(ctx context.Context, path string, owner string, repository string) (string, error) {
	var result strings.Builder

	// Get content at the current path
	_, directoryContent, _, err := g.client.Repositories.GetContents(ctx, owner, repository, path, nil)
	if err != nil {
		return "", fmt.Errorf("error getting content at path %s: %w", path, err)
	}

	// Process each item in the directory
	for _, item := range directoryContent {
		if item.GetType() == "dir" {
			// Recursively get content for subdirectories
			subContent, err := g.getContentRecursively(ctx, item.GetPath(), owner, repository)
			if err != nil {
				return "", err
			}
			result.WriteString(subContent)
		} else if item.GetType() == "file" {
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

	return result.String(), nil
}

func (g *GithubRepositoryGetAllContent) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
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

	content, err := g.getContentRecursively(ctx, result.Path, result.Owner, result.Repository)
	if err != nil {
		return types.ActionResult{}, err
	}

	return types.ActionResult{Result: content}, nil
}

func (g *GithubRepositoryGetAllContent) Definition() types.ActionDefinition {
	actionName := "get_all_github_repository_content"
	if g.customActionName != "" {
		actionName = g.customActionName
	}
	description := "Get all content of a GitHub repository recursively"
	if g.repository != "" && g.owner != "" {
		return types.ActionDefinition{
			Name:        types.ActionDefinitionName(actionName),
			Description: description,
			Properties: map[string]jsonschema.Definition{
				"path": {
					Type:        jsonschema.String,
					Description: "Optional path to start from (defaults to repository root)",
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
				Description: "Optional path to start from (defaults to repository root)",
			},
			"repository": {
				Type:        jsonschema.String,
				Description: "The repository to get content from",
			},
			"owner": {
				Type:        jsonschema.String,
				Description: "The owner of the repository",
			},
		},
		Required: []string{"repository", "owner"},
	}
}

func (a *GithubRepositoryGetAllContent) Plannable() bool {
	return true
}

// GithubRepositoryGetAllContentConfigMeta returns the metadata for GitHub Repository Get All Content action configuration fields
func GithubRepositoryGetAllContentConfigMeta() []config.Field {
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
