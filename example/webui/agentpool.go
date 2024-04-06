package main

import (
	"encoding/json"
	"fmt"
	"os"

	. "github.com/mudler/local-agent-framework/agent"
)

type ConnectorConfig struct {
	Type   string                 `json:"type"` // e.g. Slack
	Config map[string]interface{} `json:"config"`
}

type ActionsConfig string

type AgentConfig struct {
	Connector []ConnectorConfig `json:"connector"`
	Actions   []ActionsConfig   `json:"actions"`
}

type AgentPool struct {
	file   string
	pool   AgentPoolData
	agents map[string]*Agent
}

type AgentPoolData map[string]AgentConfig

func NewAgentPool(file string) (*AgentPool, error) {
	// if file exists, try to load an existing pool.
	// if file does not exist, create a new pool.

	if _, err := os.Stat(file); err != nil {
		// file does not exist, create a new pool
		return &AgentPool{
			file:   file,
			agents: make(map[string]*Agent),
			pool:   make(map[string]AgentConfig),
		}, nil
	}

	poolData, err := loadPoolFromFile(file)
	if err != nil {
		return nil, err
	}
	return &AgentPool{
		file:   file,
		agents: make(map[string]*Agent),
		pool:   *poolData,
	}, nil
}

// CreateAgent adds a new agent to the pool
// and starts it.
// It also saves the state to the file.
func (a *AgentPool) CreateAgent(name string, agentConfig *AgentConfig) error {
	if _, ok := a.pool[name]; ok {
		return fmt.Errorf("agent %s already exists", name)
	}
	a.pool[name] = *agentConfig
	if err := a.Save(); err != nil {
		return err
	}

	return a.startAgentWithConfig(name, agentConfig)
}

func (a *AgentPool) startAgentWithConfig(name string, config *AgentConfig) error {

	agent, err := New(
		WithModel("hermes-2-pro-mistral"),
	)
	if err != nil {
		return err
	}

	a.agents[name] = agent

	go func() {
		if err := agent.Run(); err != nil {
			panic(err)
		}
	}()

	return nil
}

// Starts all the agents in the pool
func (a *AgentPool) StartAll() error {
	for name, config := range a.pool {
		if a.agents[name] != nil { // Agent already started
			continue
		}
		if err := a.startAgentWithConfig(name, &config); err != nil {
			return err
		}
	}
	return nil
}

func (a *AgentPool) StopAll() {
	for _, agent := range a.agents {
		agent.Stop()
	}
}

func (a *AgentPool) Stop(name string) {
	if agent, ok := a.agents[name]; ok {
		agent.Stop()
	}
}

func (a *AgentPool) Start(name string) error {
	if agent, ok := a.agents[name]; ok {
		return agent.Run()
	}
	if config, ok := a.pool[name]; ok {
		return a.startAgentWithConfig(name, &config)
	}

	return fmt.Errorf("agent %s not found", name)
}

func (a *AgentPool) Remove(name string) error {
	a.Stop(name)
	delete(a.agents, name)
	delete(a.pool, name)
	if err := a.Save(); err != nil {
		return err
	}
	return nil
}

func (a *AgentPool) Save() error {
	data, err := json.Marshal(a.pool)
	if err != nil {
		return err
	}
	return os.WriteFile(a.file, data, 0644)
}

func (a *AgentPool) GetAgent(name string) *Agent {
	return a.agents[name]
}

func loadPoolFromFile(path string) (*AgentPoolData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	poolData := &AgentPoolData{}
	err = json.Unmarshal(data, poolData)
	return poolData, err
}
