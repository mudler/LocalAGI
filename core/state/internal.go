package state

import (
	. "github.com/mudler/LocalAGI/core/agent"
)

type AgentPoolInternalAPI struct {
	*AgentPool
}

func (a *AgentPool) InternalAPI() *AgentPoolInternalAPI {
	return &AgentPoolInternalAPI{a}
}

func (a *AgentPoolInternalAPI) GetAgent(name string) *Agent {
	return a.agents[name]
}

func (a *AgentPoolInternalAPI) AllAgents() []string {
	var agents []string
	for agent := range a.agents {
		agents = append(agents, agent)
	}
	return agents
}

func (a *AgentPoolInternalAPI) GetConfig(name string) *AgentConfig {
	agent, exists := a.pool[name]
	if !exists {
		return nil
	}
	return &agent
}
