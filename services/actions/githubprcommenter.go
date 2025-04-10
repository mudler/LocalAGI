package actions

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v69/github"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubPRCommenter struct {
	token, repository, owner, customActionName string
	client                                     *github.Client
}

var (
	patchRegex = regexp.MustCompile(`^@@.*\d [\+\-](\d+),?(\d+)?.+?@@`)
)

type commitFileInfo struct {
	FileName  string
	hunkInfos []*hunkInfo
	sha       string
}

type hunkInfo struct {
	hunkStart int
	hunkEnd   int
}

func (hi hunkInfo) isLineInHunk(line int) bool {
	return line >= hi.hunkStart && line <= hi.hunkEnd
}

func (cfi *commitFileInfo) getHunkInfo(line int) *hunkInfo {
	for _, hunkInfo := range cfi.hunkInfos {
		if hunkInfo.isLineInHunk(line) {
			return hunkInfo
		}
	}
	return nil
}

func (cfi *commitFileInfo) isLineInChange(line int) bool {
	return cfi.getHunkInfo(line) != nil
}

func (cfi commitFileInfo) calculatePosition(line int) *int {
	hi := cfi.getHunkInfo(line)
	if hi == nil {
		return nil
	}
	position := line - hi.hunkStart
	return &position
}

func parseHunkPositions(patch, filename string) ([]*hunkInfo, error) {
	hunkInfos := make([]*hunkInfo, 0)
	if patch != "" {
		groups := patchRegex.FindAllStringSubmatch(patch, -1)
		if len(groups) < 1 {
			return hunkInfos, fmt.Errorf("the patch details for [%s] could not be resolved", filename)
		}
		for _, patchGroup := range groups {
			endPos := 2
			if len(patchGroup) > 2 && patchGroup[2] == "" {
				endPos = 1
			}

			hunkStart, err := strconv.Atoi(patchGroup[1])
			if err != nil {
				hunkStart = -1
			}
			hunkEnd, err := strconv.Atoi(patchGroup[endPos])
			if err != nil {
				hunkEnd = -1
			}
			hunkInfos = append(hunkInfos, &hunkInfo{
				hunkStart: hunkStart,
				hunkEnd:   hunkEnd,
			})
		}
	}
	return hunkInfos, nil
}

func getCommitInfo(file *github.CommitFile) (*commitFileInfo, error) {
	patch := file.GetPatch()
	hunkInfos, err := parseHunkPositions(patch, *file.Filename)
	if err != nil {
		return nil, err
	}

	sha := file.GetSHA()
	if sha == "" {
		return nil, fmt.Errorf("the sha details for [%s] could not be resolved", *file.Filename)
	}

	return &commitFileInfo{
		FileName:  *file.Filename,
		hunkInfos: hunkInfos,
		sha:       sha,
	}, nil
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

func (g *GithubPRCommenter) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Repository     string `json:"repository"`
		Owner          string `json:"owner"`
		PRNumber       int    `json:"pr_number"`
		GeneralComment string `json:"general_comment"`
		Comments       []struct {
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

	// Check if PR is in a state that allows comments
	if *pr.State != "open" {
		return types.ActionResult{Result: fmt.Sprintf("Pull request #%d is not open (current state: %s)", result.PRNumber, *pr.State)}, nil
	}

	// Get the list of changed files to verify the files exist in the PR
	files, _, err := g.client.PullRequests.ListFiles(ctx, result.Owner, result.Repository, result.PRNumber, &github.ListOptions{})
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to list PR files: %w", err)
	}

	// Create a map of valid files with their commit info
	validFiles := make(map[string]*commitFileInfo)
	for _, file := range files {
		if *file.Status != "deleted" {
			info, err := getCommitInfo(file)
			if err != nil {
				continue
			}
			validFiles[*file.Filename] = info
		}
	}

	// Process each comment
	var results []string
	for _, comment := range result.Comments {
		// Check if file exists in PR
		fileInfo, exists := validFiles[comment.File]
		if !exists {
			availableFiles := make([]string, 0, len(validFiles))
			for f := range validFiles {
				availableFiles = append(availableFiles, f)
			}
			results = append(results, fmt.Sprintf("Error: File %s not found in PR #%d. Available files: %v",
				comment.File, result.PRNumber, availableFiles))
			continue
		}

		// Check if line is in a changed hunk
		if !fileInfo.isLineInChange(comment.Line) {
			results = append(results, fmt.Sprintf("Error: Line %d is not in a changed hunk in file %s",
				comment.Line, comment.File))
			continue
		}

		// Calculate position
		position := fileInfo.calculatePosition(comment.Line)
		if position == nil {
			results = append(results, fmt.Sprintf("Error: Could not calculate position for line %d in file %s",
				comment.Line, comment.File))
			continue
		}

		// Create the review comment
		reviewComment := &github.PullRequestComment{
			Path:     &comment.File,
			Line:     &comment.Line,
			Body:     &comment.Comment,
			Position: position,
			CommitID: &fileInfo.sha,
		}

		// Set start line if provided
		if comment.StartLine > 0 {
			reviewComment.StartLine = &comment.StartLine
		}

		// Create the comment with retries
		var resp *github.Response
		for i := 0; i < 6; i++ {
			_, resp, err = g.client.PullRequests.CreateComment(ctx, result.Owner, result.Repository, result.PRNumber, reviewComment)
			if err == nil {
				break
			}
			if resp != nil && resp.StatusCode == 422 {
				// Rate limit hit, wait and retry
				retrySeconds := i * i
				time.Sleep(time.Second * time.Duration(retrySeconds))
				continue
			}
			break
		}

		if err != nil {
			errorDetails := fmt.Sprintf("Error commenting on file %s, line %d: %s", comment.File, comment.Line, err.Error())
			if resp != nil {
				errorDetails += fmt.Sprintf("\nResponse Status: %s", resp.Status)
				if resp.Body != nil {
					body, _ := io.ReadAll(resp.Body)
					errorDetails += fmt.Sprintf("\nResponse Body: %s", string(body))
				}
			}
			results = append(results, errorDetails)
			continue
		}

		results = append(results, fmt.Sprintf("Successfully commented on file %s, line %d", comment.File, comment.Line))
	}

	if result.GeneralComment != "" {
		// Try both PullRequests and Issues API for general comments
		var resp *github.Response
		var err error

		// First try PullRequests API
		_, resp, err = g.client.PullRequests.CreateComment(ctx, result.Owner, result.Repository, result.PRNumber, &github.PullRequestComment{
			Body: &result.GeneralComment,
		})

		// If that fails with 403, try Issues API
		if err != nil && resp != nil && resp.StatusCode == 403 {
			_, resp, err = g.client.Issues.CreateComment(ctx, result.Owner, result.Repository, result.PRNumber, &github.IssueComment{
				Body: &result.GeneralComment,
			})
		}

		if err != nil {
			errorDetails := fmt.Sprintf("Error adding general comment: %s", err.Error())
			if resp != nil {
				errorDetails += fmt.Sprintf("\nResponse Status: %s", resp.Status)
				if resp.Body != nil {
					body, _ := io.ReadAll(resp.Body)
					errorDetails += fmt.Sprintf("\nResponse Body: %s", string(body))
				}
			}
			results = append(results, errorDetails)
		} else {
			results = append(results, "Successfully added general comment to pull request")
		}
	}

	return types.ActionResult{
		Result: strings.Join(results, "\n"),
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
				"general_comment": {
					Type:        jsonschema.String,
					Description: "A general comment to add to the pull request.",
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
					Description: "Array of comments to add to the pull request.",
				},
			},
			Required: []string{"pr_number", "comments"},
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
			"general_comment": {
				Type:        jsonschema.String,
				Description: "A general comment to add to the pull request.",
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
				Description: "Array of comments to add to the pull request.",
			},
		},
		Required: []string{"pr_number", "repository", "owner", "comments"},
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
