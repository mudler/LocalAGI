package services

import (
	"encoding/json"

	"github.com/mudler/LocalAgent/pkg/config"
	"github.com/mudler/LocalAgent/pkg/xlog"
	"github.com/mudler/LocalAgent/services/connectors"

	"github.com/mudler/LocalAgent/core/state"
)

const (
	// Connectors
	ConnectorIRC          = "irc"
	ConnectorTelegram     = "telegram"
	ConnectorSlack        = "slack"
	ConnectorDiscord      = "discord"
	ConnectorGithubIssues = "github-issues"
	ConnectorGithubPRs    = "github-prs"
	ConnectorTwitter      = "twitter"
)

var AvailableConnectors = []string{
	ConnectorIRC,
	ConnectorTelegram,
	ConnectorSlack,
	ConnectorDiscord,
	ConnectorGithubIssues,
	ConnectorGithubPRs,
	ConnectorTwitter,
}

func Connectors(a *state.AgentConfig) []state.Connector {
	conns := []state.Connector{}

	for _, c := range a.Connector {
		var config map[string]string
		if err := json.Unmarshal([]byte(c.Config), &config); err != nil {
			xlog.Info("Error unmarshalling connector config", err)
			continue
		}
		switch c.Type {
		case ConnectorTelegram:
			cc, err := connectors.NewTelegramConnector(config)
			if err != nil {
				xlog.Info("Error creating telegram connector", err)
				continue
			}

			conns = append(conns, cc)
		case ConnectorSlack:
			conns = append(conns, connectors.NewSlack(config))
		case ConnectorDiscord:
			conns = append(conns, connectors.NewDiscord(config))
		case ConnectorGithubIssues:
			conns = append(conns, connectors.NewGithubIssueWatcher(config))
		case ConnectorGithubPRs:
			conns = append(conns, connectors.NewGithubPRWatcher(config))
		case ConnectorIRC:
			conns = append(conns, connectors.NewIRC(config))
		case ConnectorTwitter:
			cc, err := connectors.NewTwitterConnector(config)
			if err != nil {
				xlog.Info("Error creating twitter connector", err)
				continue
			}
			conns = append(conns, cc)
		}
	}
	return conns
}

func ConnectorsConfigMeta() []config.FieldGroup {
	return []config.FieldGroup{
		{
			Name:   "discord",
			Label:  "Discord",
			Fields: connectors.DiscordConfigMeta(),
		},
		{
			Name:   "slack",
			Label:  "Slack",
			Fields: connectors.SlackConfigMeta(),
		},
		{
			Name:   "telegram",
			Label:  "Telegram",
			Fields: connectors.TelegramConfigMeta(),
		},
		{
			Name:   "github-issues",
			Label:  "GitHub Issues",
			Fields: connectors.GithubIssueConfigMeta(),
		},
		{
			Name:   "github-prs",
			Label:  "GitHub PRs",
			Fields: connectors.GithubPRConfigMeta(),
		},
		{
			Name:   "irc",
			Label:  "IRC",
			Fields: connectors.IRCConfigMeta(),
		},
		{
			Name:   "twitter",
			Label:  "Twitter",
			Fields: connectors.TwitterConfigMeta(),
		},
	}
}
