package main

import (
	"context"
	"encoding/json"

	"github.com/mudler/local-agent-framework/xlog"

	. "github.com/mudler/local-agent-framework/agent"
	"github.com/mudler/local-agent-framework/external"
)

const (
	// Actions
	ActionSearch              = "search"
	ActionGithubIssueLabeler  = "github-issue-labeler"
	ActionGithubIssueOpener   = "github-issue-opener"
	ActionGithubIssueCloser   = "github-issue-closer"
	ActionGithubIssueSearcher = "github-issue-searcher"
	ActionScraper             = "scraper"
	ActionWikipedia           = "wikipedia"
	ActionBrowse              = "browse"
	ActionSendMail            = "send_mail"
)

var AvailableActions = []string{
	ActionSearch,
	ActionGithubIssueLabeler,
	ActionGithubIssueOpener,
	ActionGithubIssueCloser,
	ActionGithubIssueSearcher,
	ActionScraper,
	ActionBrowse,
	ActionWikipedia,
	ActionSendMail,
}

func (a *AgentConfig) availableActions(ctx context.Context) []Action {
	actions := []Action{}

	for _, action := range a.Actions {
		var config map[string]string
		if err := json.Unmarshal([]byte(action.Config), &config); err != nil {
			xlog.Info("Error unmarshalling action config", "error", err)
			continue
		}

		switch action.Name {
		case ActionSearch:
			actions = append(actions, external.NewSearch(config))
		case ActionGithubIssueLabeler:
			actions = append(actions, external.NewGithubIssueLabeler(ctx, config))
		case ActionGithubIssueOpener:
			actions = append(actions, external.NewGithubIssueOpener(ctx, config))
		case ActionGithubIssueCloser:
			actions = append(actions, external.NewGithubIssueCloser(ctx, config))
		case ActionGithubIssueSearcher:
			actions = append(actions, external.NewGithubIssueSearch(ctx, config))
		case ActionScraper:
			actions = append(actions, external.NewScraper(config))
		case ActionWikipedia:
			actions = append(actions, external.NewWikipedia(config))
		case ActionBrowse:
			actions = append(actions, external.NewBrowse(config))
		case ActionSendMail:
			actions = append(actions, external.NewSendMail(config))
		}
	}

	return actions
}
