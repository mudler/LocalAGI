package webui

import (
	"encoding/json"

	"github.com/mudler/local-agent-framework/pkg/xlog"
	"github.com/mudler/local-agent-framework/services/connectors"

	"github.com/mudler/local-agent-framework/core/state"
)

const (
	// Connectors
	ConnectorTelegram     = "telegram"
	ConnectorSlack        = "slack"
	ConnectorDiscord      = "discord"
	ConnectorGithubIssues = "github-issues"
	ConnectorGithubPRs    = "github-prs"
)

var AvailableConnectors = []string{
	ConnectorTelegram,
	ConnectorSlack,
	ConnectorDiscord,
	ConnectorGithubIssues,
	ConnectorGithubPRs,
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
		}
	}
	return conns
}
