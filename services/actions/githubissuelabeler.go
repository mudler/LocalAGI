package actions

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v69/github"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubIssuesLabeler struct {
	token, repository, owner, customActionName string
	availableLabels                            []string
	client                                     *github.Client
}

func NewGithubIssueLabeler(config map[string]string) *GithubIssuesLabeler {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	// Get available labels
	availableLabels := []string{"bug", "enhancement"}

	if config["availableLabels"] != "" {
		availableLabels = strings.Split(config["availableLabels"], ",")
	}

	return &GithubIssuesLabeler{
		client:           client,
		token:            config["token"],
		customActionName: config["customActionName"],
		repository:       config["repository"],
		owner:            config["owner"],
		availableLabels:  availableLabels,
	}
}

func (g *GithubIssuesLabeler) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Repository  string `json:"repository"`
		Owner       string `json:"owner"`
		Label       string `json:"label"`
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

	labels, _, err := g.client.Issues.AddLabelsToIssue(ctx, result.Owner, result.Repository, result.IssueNumber, []string{result.Label})
	//labelsNames := []string{}
	for _, l := range labels {
		xlog.Info("Label added", "label", l.Name)
		//labelsNames = append(labelsNames, l.GetName())
	}

	resultString := fmt.Sprintf("Added label '%s' to issue %d in repository %s/%s", result.Label, result.IssueNumber, result.Owner, result.Repository)
	if err != nil {
		resultString = fmt.Sprintf("Error adding label '%s' to issue %d in repository %s/%s: %v", result.Label, result.IssueNumber, result.Owner, result.Repository, err)
	}
	return types.ActionResult{Result: resultString}, err
}

func (g *GithubIssuesLabeler) Definition() types.ActionDefinition {
	actionName := "add_label_to_github_issue"
	if g.customActionName != "" {
		actionName = g.customActionName
	}
	if g.repository != "" && g.owner != "" {
		return types.ActionDefinition{
			Name:        types.ActionDefinitionName(actionName),
			Description: "Add a label to a Github issue. You might want to assign labels to issues to categorize them.",
			Properties: map[string]jsonschema.Definition{
				"issue_number": {
					Type:        jsonschema.Number,
					Description: "The number of the issue to add the label to.",
				},
				"label": {
					Type:        jsonschema.String,
					Description: "The label to add to the issue.",
					Enum:        g.availableLabels,
				},
			},
			Required: []string{"issue_number", "label"},
		}
	}
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(actionName),
		Description: "Add a label to a Github issue. You might want to assign labels to issues to categorize them.",
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
			"label": {
				Type:        jsonschema.String,
				Description: "The label to add to the issue.",
				Enum:        g.availableLabels,
			},
		},
		Required: []string{"issue_number", "repository", "owner", "label"},
	}
}

func (a *GithubIssuesLabeler) Plannable() bool {
	return true
}

// GithubIssueLabelerConfigMeta returns the metadata for GitHub Issue Labeler action configuration fields
func GithubIssueLabelerConfigMeta() []config.Field {
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
			Name:         "availableLabels",
			Label:        "Available Labels",
			Type:         config.FieldTypeText,
			HelpText:     "Comma-separated list of available labels",
			DefaultValue: "bug,enhancement",
		},
		{
			Name:     "customActionName",
			Label:    "Custom Action Name",
			Type:     config.FieldTypeText,
			HelpText: "Custom name for this action",
		},
	}
}
