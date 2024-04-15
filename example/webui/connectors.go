package main

import (
	"encoding/json"
	"log/slog"

	. "github.com/mudler/local-agent-framework/agent"

	"github.com/mudler/local-agent-framework/example/webui/connector"
)

const (
	// Connectors
	ConnectorTelegram     = "telegram"
	ConnectorSlack        = "slack"
	ConnectorDiscord      = "discord"
	ConnectorGithubIssues = "github-issues"
)

type Connector interface {
	AgentResultCallback() func(state ActionState)
	AgentReasoningCallback() func(state ActionCurrentState) bool
	Start(a *Agent)
}

var AvailableConnectors = []string{
	ConnectorTelegram,
	ConnectorSlack,
	ConnectorDiscord,
	ConnectorGithubIssues,
}

func (a *AgentConfig) availableConnectors() []Connector {
	connectors := []Connector{}

	for _, c := range a.Connector {
		slog.Info("Set Connector", c)

		var config map[string]string
		if err := json.Unmarshal([]byte(c.Config), &config); err != nil {
			slog.Info("Error unmarshalling connector config", err)
			continue
		}
		slog.Info("Config", config)

		switch c.Type {
		case ConnectorTelegram:
			cc, err := connector.NewTelegramConnector(config)
			if err != nil {
				slog.Info("Error creating telegram connector", err)
				continue
			}

			connectors = append(connectors, cc)
		case ConnectorSlack:
			connectors = append(connectors, connector.NewSlack(config))
		case ConnectorDiscord:
			connectors = append(connectors, connector.NewDiscord(config))
		case ConnectorGithubIssues:
			connectors = append(connectors, connector.NewGithubIssueWatcher(config))
		}
	}
	return connectors
}
