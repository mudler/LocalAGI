package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/mudler/local-agent-framework/agent"
	"github.com/mudler/local-agent-framework/external"
)

type ConnectorConfig struct {
	Type   string                 `json:"type"` // e.g. Slack
	Config map[string]interface{} `json:"config"`
}

type ActionsConfig string

type AgentConfig struct {
	Connector []ConnectorConfig `json:"connector" form:"connector" `
	Actions   []ActionsConfig   `json:"actions" form:"actions"`
	// This is what needs to be part of ActionsConfig
	Model            string `json:"model" form:"model"`
	Name             string `json:"name" form:"name"`
	HUD              bool   `json:"hud" form:"hud"`
	Debug            bool   `json:"debug" form:"debug"`
	StandaloneJob    bool   `json:"standalone_job" form:"standalone_job"`
	RandomIdentity   bool   `json:"random_identity" form:"random_identity"`
	IdentityGuidance string `json:"identity_guidance" form:"identity_guidance"`
	PeriodicRuns     string `json:"periodic_runs" form:"periodic_runs"`
}

type AgentPool struct {
	file          string
	pooldir       string
	pool          AgentPoolData
	agents        map[string]*Agent
	managers      map[string]Manager
	apiURL, model string
}

type AgentPoolData map[string]AgentConfig

func NewAgentPool(model, apiURL, directory string) (*AgentPool, error) {
	// if file exists, try to load an existing pool.
	// if file does not exist, create a new pool.

	poolfile := filepath.Join(directory, "pool.json")

	if _, err := os.Stat(poolfile); err != nil {
		// file does not exist, create a new pool
		return &AgentPool{
			file:     poolfile,
			pooldir:  directory,
			apiURL:   apiURL,
			model:    model,
			agents:   make(map[string]*Agent),
			pool:     make(map[string]AgentConfig),
			managers: make(map[string]Manager),
		}, nil
	}

	poolData, err := loadPoolFromFile(poolfile)
	if err != nil {
		return nil, err
	}
	return &AgentPool{
		file:     poolfile,
		apiURL:   apiURL,
		pooldir:  directory,
		model:    model,
		agents:   make(map[string]*Agent),
		managers: make(map[string]Manager),
		pool:     *poolData,
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

func (a *AgentPool) GetManager(name string) Manager {
	return a.managers[name]
}

func (a *AgentPool) List() []string {
	var agents []string
	for agent := range a.pool {
		agents = append(agents, agent)
	}
	return agents
}

var AvailableActions = []string{"search"}

func (a *AgentConfig) availableActions() []Action {
	actions := []Action{}

	if len(a.Actions) == 0 {
		// Return search as default
		return []Action{external.NewSearch(3)}
	}
	for _, action := range a.Actions {
		fmt.Println("Set Action", action)
		switch action {
		case "search":
			actions = append(actions, external.NewSearch(3))
		}
	}

	return actions
}

func (a *AgentPool) startAgentWithConfig(name string, config *AgentConfig) error {
	manager := NewManager(5)
	model := a.model
	if config.Model != "" {
		model = config.Model
	}
	if config.PeriodicRuns == "" {
		config.PeriodicRuns = "10m"
	}
	fmt.Println("Creating agent", name)
	fmt.Println("Model", model)
	fmt.Println("API URL", a.apiURL)

	actions := config.availableActions()

	stateFile, characterFile := a.stateFiles(name)

	fmt.Println("Actions", actions)
	opts := []Option{
		WithModel(model),
		WithLLMAPIURL(a.apiURL),
		WithPeriodicRuns(config.PeriodicRuns),
		WithActions(
			actions...,
		),
		WithStateFile(stateFile),
		WithCharacterFile(characterFile),
		WithAgentReasoningCallback(func(state ActionCurrentState) bool {
			fmt.Println("Reasoning", state.Reasoning)
			manager.Send(
				NewMessage(
					fmt.Sprintf(`Thinking: %s`, htmlIfy(state.Reasoning)),
				).WithEvent("status"),
			)
			return true
		}),
		WithAgentResultCallback(func(state ActionState) {
			fmt.Println("Reasoning", state.Reasoning)

			text := fmt.Sprintf(`Reasoning: %s
			Action taken: %+v
			Parameters: %+v
			Result: %s`,
				state.Reasoning,
				state.ActionCurrentState.Action.Definition().Name,
				state.ActionCurrentState.Params,
				state.Result)
			manager.Send(
				NewMessage(
					htmlIfy(
						text,
					),
				).WithEvent("status"),
			)
		}),
	}
	if config.HUD {
		opts = append(opts, EnableHUD)
	}
	if config.Debug {
		opts = append(opts, DebugMode)
	}
	if config.StandaloneJob {
		opts = append(opts, EnableStandaloneJob)
	}
	if config.RandomIdentity {
		if config.IdentityGuidance != "" {
			opts = append(opts, WithRandomIdentity(config.IdentityGuidance))
		} else {
			opts = append(opts, WithRandomIdentity())
		}
	}

	fmt.Println("Starting agent", name)
	fmt.Printf("Config %+v\n", config)
	agent, err := New(opts...)
	if err != nil {
		return err
	}

	a.agents[name] = agent
	a.managers[name] = manager

	go func() {
		if err := agent.Run(); err != nil {
			fmt.Println("Agent stop: ", err.Error())
		}
	}()

	go func() {
		for {
			time.Sleep(1 * time.Second) // Send a message every seconds
			manager.Send(NewMessage(
				htmlIfy(agent.State().String()),
			).WithEvent("hud"))
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

func (a *AgentPool) stateFiles(name string) (string, string) {
	stateFile := filepath.Join(a.pooldir, fmt.Sprintf("%s.state.json", name))
	characterFile := filepath.Join(a.pooldir, fmt.Sprintf("%s.character.json", name))

	return stateFile, characterFile
}

func (a *AgentPool) Remove(name string) error {

	// Cleanup character and state
	stateFile, characterFile := a.stateFiles(name)

	os.Remove(stateFile)
	os.Remove(characterFile)

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
