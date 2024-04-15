package main

import (
	"context"
	"encoding/json"
	"log/slog"

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
}

func (a *AgentConfig) availableActions(ctx context.Context) []Action {
	actions := []Action{}

	for _, action := range a.Actions {
		slog.Info("Set Action", action)

		var config map[string]string
		if err := json.Unmarshal([]byte(action.Config), &config); err != nil {
			slog.Info("Error unmarshalling action config", err)
			continue
		}
		slog.Info("Config", config)

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
		}
	}

	return actions
}
