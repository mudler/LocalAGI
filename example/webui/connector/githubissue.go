package connector

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/go-github/v61/github"
	"github.com/mudler/local-agent-framework/agent"
	"github.com/sashabaranov/go-openai"
)

type GithubIssues struct {
	token        string
	repository   string
	owner        string
	agent        *agent.Agent
	pollInterval time.Duration
	client       *github.Client
}

func NewGithubIssueWatcher(config map[string]string) *GithubIssues {
	client := github.NewClient(nil).WithAuthToken(config["token"])
	interval, err := time.ParseDuration(config["pollInterval"])
	if err != nil {
		interval = 1 * time.Minute
	}

	return &GithubIssues{
		client:       client,
		token:        config["token"],
		repository:   config["repository"],
		owner:        config["owner"],
		pollInterval: interval,
	}
}

func (g *GithubIssues) AgentResultCallback() func(state agent.ActionState) {
	return func(state agent.ActionState) {
		// Send the result to the bot
	}
}

func (g *GithubIssues) AgentReasoningCallback() func(state agent.ActionCurrentState) bool {
	return func(state agent.ActionCurrentState) bool {
		// Send the reasoning to the bot
		return true
	}
}

func (g *GithubIssues) Start(a *agent.Agent) {
	// Start the connector
	g.agent = a

	go func() {
		ticker := time.NewTicker(g.pollInterval)
		for {
			select {
			case <-ticker.C:
				slog.Info("Looking into github issues...")
				g.issuesService()
			case <-a.Context().Done():
				return
			}
		}
	}()
}

func (g *GithubIssues) issuesService() {
	user, _, err := g.client.Users.Get(g.agent.Context(), "")
	if err != nil {
		fmt.Printf("\nerror: %v\n", err)
		return
	}

	issues, _, err := g.client.Issues.ListByRepo(
		g.agent.Context(),
		g.owner,
		g.repository,
		&github.IssueListByRepoOptions{})
	if err != nil {
		slog.Info("Error listing issues", err)
	}
	for _, issue := range issues {
		// Do something with the issue
		if issue.IsPullRequest() {
			continue
		}
		labels := []string{}
		for _, l := range issue.Labels {
			labels = append(labels, l.GetName())
		}

		// Get user that opened the issue
		userNameLogin := issue.GetUser().Login
		userName := ""
		if userNameLogin != nil {
			userName = *userNameLogin
		}

		if userName == user.GetLogin() {
			slog.Info("Ignoring issue opened by the bot")
			continue
		}
		messages := []openai.ChatCompletionMessage{
			{
				Role: "system",
				Content: fmt.Sprintf(
					`This is a conversation with an user ("%s") that opened a Github issue with title "%s" in the repository "%s" owned by "%s". The issue is the issue number %d. Current labels: %+v`, userName, issue.GetTitle(), g.repository, g.owner, issue.GetNumber(), labels),
			},
			{
				Role:    "user",
				Content: issue.GetBody(),
			},
		}

		comments, _, _ := g.client.Issues.ListComments(g.agent.Context(), g.owner, g.repository, issue.GetNumber(),
			&github.IssueListCommentsOptions{})

		mustAnswer := false
		botAnsweredAlready := false
		for i, comment := range comments {
			role := "user"
			if comment.GetUser().GetLogin() == user.GetLogin() {
				botAnsweredAlready = true
				role = "assistant"
			}
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    role,
				Content: comment.GetBody(),
			})

			// if last comment is from the user and mentions the bot username, we must answer
			if comment.User.GetName() != user.GetLogin() && len(comments)-1 == i {
				if strings.Contains(comment.GetBody(), fmt.Sprintf("@%s", user.GetLogin())) {
					slog.Info("Bot was mentioned in the last comment")
					mustAnswer = true
				}
			}
		}

		if len(comments) == 0 || !botAnsweredAlready {
			// if no comments, or bot didn't answer yet, we must answer
			slog.Info("No comments, or bot didn't answer yet")
			slog.Info("Comments:", len(comments))
			slog.Info("Bot answered already", botAnsweredAlready)
			mustAnswer = true
		}

		if !mustAnswer {
			continue
		}

		go func() {
			res := g.agent.Ask(
				agent.WithConversationHistory(messages),
			)
			if res.Error != nil {
				slog.Error("Error asking", "error", res.Error)
				return
			}

			_, _, err := g.client.Issues.CreateComment(
				g.agent.Context(),
				g.owner, g.repository,
				issue.GetNumber(), &github.IssueComment{
					Body: github.String(res.Response),
				},
			)
			if err != nil {
				slog.Error("Error creating comment", "error", err)
			}
		}()
	}
}
