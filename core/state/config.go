package state

import (
	"encoding/json"

	"github.com/mudler/LocalAgent/core/agent"
)

type ConnectorConfig struct {
	Type   string `json:"type"` // e.g. Slack
	Config string `json:"config"`
}

type ActionsConfig struct {
	Name   string `json:"name"` // e.g. search
	Config string `json:"config"`
}

type PromptBlocksConfig struct {
	Type   string `json:"type"`
	Config string `json:"config"`
}

func (d PromptBlocksConfig) ToMap() map[string]string {
	config := map[string]string{}
	json.Unmarshal([]byte(d.Config), &config)
	return config
}

type AgentConfig struct {
	Connector    []ConnectorConfig    `json:"connectors" form:"connectors" `
	Actions      []ActionsConfig      `json:"actions" form:"actions"`
	PromptBlocks []PromptBlocksConfig `json:"promptblocks" form:"promptblocks"`
	MCPServers   []agent.MCPServer    `json:"mcp_servers" form:"mcp_servers"`

	Description string `json:"description" form:"description"`
	// This is what needs to be part of ActionsConfig
	Model           string `json:"model" form:"model"`
	MultimodalModel string `json:"multimodal_model" form:"multimodal_model"`
	APIURL          string `json:"api_url" form:"api_url"`
	APIKey          string `json:"api_key" form:"api_key"`
	LocalRAGURL     string `json:"local_rag_url" form:"local_rag_url"`
	LocalRAGAPIKey  string `json:"local_rag_api_key" form:"local_rag_api_key"`

	Name                  string `json:"name" form:"name"`
	HUD                   bool   `json:"hud" form:"hud"`
	StandaloneJob         bool   `json:"standalone_job" form:"standalone_job"`
	RandomIdentity        bool   `json:"random_identity" form:"random_identity"`
	InitiateConversations bool   `json:"initiate_conversations" form:"initiate_conversations"`
	IdentityGuidance      string `json:"identity_guidance" form:"identity_guidance"`
	PeriodicRuns          string `json:"periodic_runs" form:"periodic_runs"`
	PermanentGoal         string `json:"permanent_goal" form:"permanent_goal"`
	EnableKnowledgeBase   bool   `json:"enable_kb" form:"enable_kb"`
	EnableReasoning       bool   `json:"enable_reasoning" form:"enable_reasoning"`
	KnowledgeBaseResults  int    `json:"kb_results" form:"kb_results"`
	CanStopItself         bool   `json:"can_stop_itself" form:"can_stop_itself"`
	SystemPrompt          string `json:"system_prompt" form:"system_prompt"`
	LongTermMemory        bool   `json:"long_term_memory" form:"long_term_memory"`
	SummaryLongTermMemory bool   `json:"summary_long_term_memory" form:"summary_long_term_memory"`
}

type Connector interface {
	AgentResultCallback() func(state agent.ActionState)
	AgentReasoningCallback() func(state agent.ActionCurrentState) bool
	Start(a *agent.Agent)
}
