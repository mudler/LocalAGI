package external

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v61/github"
	"github.com/mudler/local-agent-framework/action"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubIssuesLabeler struct {
	token           string
	availableLabels []string
	context         context.Context
	client          *github.Client
}

func NewGithubIssueLabeler(ctx context.Context, config map[string]string) *GithubIssuesLabeler {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	// Get available labels
	availableLabels := []string{"bug", "enhancement"}

	if config["availableLabels"] != "" {
		availableLabels = strings.Split(config["availableLabels"], ",")
	}

	return &GithubIssuesLabeler{
		client:          client,
		token:           config["token"],
		context:         ctx,
		availableLabels: availableLabels,
	}
}

func (g *GithubIssuesLabeler) Run(params action.ActionParams) (string, error) {
	result := struct {
		Repository  string `json:"repository"`
		Owner       string `json:"owner"`
		Label       string `json:"label"`
		IssueNumber int    `json:"issue_number"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return "", err
	}

	labels, _, err := g.client.Issues.AddLabelsToIssue(g.context, result.Owner, result.Repository, result.IssueNumber, []string{result.Label})
	//labelsNames := []string{}
	for _, l := range labels {
		fmt.Println("Label added:", l.Name)
		//labelsNames = append(labelsNames, l.GetName())
	}

	resultString := fmt.Sprintf("Added label '%s' to issue %d in repository %s/%s", result.Label, result.IssueNumber, result.Owner, result.Repository)
	if err != nil {
		resultString = fmt.Sprintf("Error adding label '%s' to issue %d in repository %s/%s: %v", result.Label, result.IssueNumber, result.Owner, result.Repository, err)
	}
	return resultString, err
}

func (g *GithubIssuesLabeler) Definition() action.ActionDefinition {
	return action.ActionDefinition{
		Name:        "add_label_to_github_issue",
		Description: "Add a label to a Github issue.",
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
