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
	Connector     []ConnectorConfig `json:"connector"`
	Actions       []ActionsConfig   `json:"actions"`
	StateFile     string            `json:"state_file"`
	CharacterFile string            `json:"character_file"`
	// This is what needs to be part of ActionsConfig

	// WithLLMAPIURL(apiModel),
	// WithModel(testModel),
	// EnableHUD,
	// DebugMode,
	// EnableStandaloneJob,
	// WithAgentReasoningCallback(func(state ActionCurrentState) bool {
	// 	sseManager.Send(
	// 		sse.NewMessage(
	// 			fmt.Sprintf(`Thinking: %s`, htmlIfy(state.Reasoning)),
	// 		).WithEvent("status"),
	// 	)
	// 	return true
	// }),
	// WithActions(external.NewSearch(3)),
	// WithAgentResultCallback(func(state ActionState) {
	// 	text := fmt.Sprintf(`Reasoning: %s
	// 	Action taken: %+v
	// 	Parameters: %+v
	// 	Result: %s`,
	// 		state.Reasoning,
	// 		state.ActionCurrentState.Action.Definition().Name,
	// 		state.ActionCurrentState.Params,
	// 		state.Result)
	// 	sseManager.Send(
	// 		sse.NewMessage(
	// 			htmlIfy(
	// 				text,
	// 			),
	// 		).WithEvent("status"),
	// 	)
	// }),
	// WithRandomIdentity(),
	// WithPeriodicRuns("10m"),

	APIURL           string `json:"api_url"`
	Model            string `json:"model"`
	HUD              bool   `json:"hud"`
	Debug            bool   `json:"debug"`
	StandaloneJob    bool   `json:"standalone_job"`
	RandomIdentity   bool   `json:"random_identity"`
	IdentityGuidance string `json:"identity_guidance"`
	PeriodicRuns     string `json:"periodic_runs"`
}

type AgentPool struct {
	file     string
	pool     AgentPoolData
	agents   map[string]*Agent
	managers map[string]Manager
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

func (a *AgentPool) startAgentWithConfig(name string, config *AgentConfig) error {
	manager := NewManager(5)
	opts := []Option{
		WithModel(config.Model),
		WithLLMAPIURL(config.APIURL),
		WithPeriodicRuns(config.PeriodicRuns),
		WithStateFile(config.StateFile),
		WithCharacterFile(config.StateFile),
		WithAgentReasoningCallback(func(state ActionCurrentState) bool {
			sseManager.Send(
				NewMessage(
					fmt.Sprintf(`Thinking: %s`, htmlIfy(state.Reasoning)),
				).WithEvent("status"),
			)
			return true
		}),
		WithAgentResultCallback(func(state ActionState) {
			text := fmt.Sprintf(`Reasoning: %s
			Action taken: %+v
			Parameters: %+v
			Result: %s`,
				state.Reasoning,
				state.ActionCurrentState.Action.Definition().Name,
				state.ActionCurrentState.Params,
				state.Result)
			sseManager.Send(
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
	agent, err := New(opts...)
	if err != nil {
		return err
	}

	a.agents[name] = agent
	a.managers[name] = manager

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
