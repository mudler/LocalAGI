package state

import (
	"encoding/json"

	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
)

type ConnectorConfig struct {
	Type   string `json:"type"` // e.g. Slack
	Config string `json:"config"`
}

type ActionsConfig struct {
	Name   string `json:"name"` // e.g. search
	Config string `json:"config"`
}

type DynamicPromptsConfig struct {
	Type   string `json:"type"`
	Config string `json:"config"`
}

func (d DynamicPromptsConfig) ToMap() map[string]string {
	config := map[string]string{}
	json.Unmarshal([]byte(d.Config), &config)
	return config
}

type FiltersConfig struct {
	Type   string `json:"type"`
	Config string `json:"config"`
}

type AgentConfig struct {
	Connector      []ConnectorConfig      `json:"connectors" form:"connectors" `
	Actions        []ActionsConfig        `json:"actions" form:"actions"`
	DynamicPrompts []DynamicPromptsConfig `json:"dynamic_prompts" form:"dynamic_prompts"`
	MCPServers     []agent.MCPServer      `json:"mcp_servers" form:"mcp_servers"`
	Filters        []FiltersConfig        `json:"filters" form:"filters"`

	Description string `json:"description" form:"description"`

	Model           string `json:"model" form:"model"`
	MultimodalModel string `json:"multimodal_model" form:"multimodal_model"`
	LocalRAGURL     string `json:"local_rag_url" form:"local_rag_url"`
	LocalRAGAPIKey  string `json:"local_rag_api_key" form:"local_rag_api_key"`

	Name                  string `json:"name" form:"name"`
	HUD                   bool   `json:"hud" form:"hud"`
	StandaloneJob         bool   `json:"standalone_job" form:"standalone_job"`
	RandomIdentity        bool   `json:"random_identity" form:"random_identity"`
	InitiateConversations bool   `json:"initiate_conversations" form:"initiate_conversations"`
	CanPlan               bool   `json:"enable_planning" form:"enable_planning"`
	IdentityGuidance      string `json:"identity_guidance" form:"identity_guidance"`
	PeriodicRuns          string `json:"periodic_runs" form:"periodic_runs"`
	PermanentGoal         string `json:"permanent_goal" form:"permanent_goal"`
	EnableKnowledgeBase   bool   `json:"enable_kb" form:"enable_kb"`
	EnableReasoning       bool   `json:"enable_reasoning" form:"enable_reasoning"`
	KnowledgeBaseResults  int    `json:"kb_results" form:"kb_results"`
	LoopDetectionSteps    int    `json:"loop_detection_steps" form:"loop_detection_steps"`
	CanStopItself         bool   `json:"can_stop_itself" form:"can_stop_itself"`
	SystemPrompt          string `json:"system_prompt" form:"system_prompt"`
	LongTermMemory        bool   `json:"long_term_memory" form:"long_term_memory"`
	SummaryLongTermMemory bool   `json:"summary_long_term_memory" form:"summary_long_term_memory"`
	ParallelJobs          int    `json:"parallel_jobs" form:"parallel_jobs"`
	StripThinkingTags     bool   `json:"strip_thinking_tags" form:"strip_thinking_tags"`
	EnableEvaluation      bool   `json:"enable_evaluation" form:"enable_evaluation"`
	MaxEvaluationLoops    int    `json:"max_evaluation_loops" form:"max_evaluation_loops"`
}

type AgentConfigMeta struct {
	Filters        []config.FieldGroup
	Fields         []config.Field
	Connectors     []config.FieldGroup
	Actions        []config.FieldGroup
	DynamicPrompts []config.FieldGroup
	MCPServers     []config.Field
}

func NewAgentConfigMeta(
	actionsConfig []config.FieldGroup,
	connectorsConfig []config.FieldGroup,
	dynamicPromptsConfig []config.FieldGroup,
	filtersConfig []config.FieldGroup,
) AgentConfigMeta {
	return AgentConfigMeta{
		Fields: []config.Field{
			{
				Name:         "name",
				Label:        "Name",
				Type:         "text",
				DefaultValue: "",
				Required:     true,
				Tags:         config.Tags{Section: "BasicInfo"},
			},
			{
				Name:         "description",
				Label:        "Description",
				Type:         "textarea",
				DefaultValue: "",
				Tags:         config.Tags{Section: "BasicInfo"},
			},
			{
				Name:         "identity_guidance",
				Label:        "Identity Guidance",
				Type:         "textarea",
				DefaultValue: "",
				Tags:         config.Tags{Section: "BasicInfo"},
			},
			{
				Name:         "random_identity",
				Label:        "Random Identity",
				Type:         "checkbox",
				DefaultValue: false,
				Tags:         config.Tags{Section: "BasicInfo"},
			},
			{
				Name:         "hud",
				Label:        "HUD",
				Type:         "checkbox",
				DefaultValue: false,
				Tags:         config.Tags{Section: "BasicInfo"},
			},
			{
				Name:         "model",
				Label:        "Model",
				Type:         "select",
				DefaultValue: "",
				Tags:         config.Tags{Section: "ModelSettings"},
			},
			{
				Name:         "multimodal_model",
				Label:        "Multimodal Model",
				Type:         "text",
				DefaultValue: "",
				Tags:         config.Tags{Section: "ModelSettings"},
			},
			{
				Name:         "local_rag_url",
				Label:        "Local RAG URL",
				Type:         "text",
				DefaultValue: "",
				Tags:         config.Tags{Section: "ModelSettings"},
			},
			{
				Name:         "local_rag_api_key",
				Label:        "Local RAG API Key",
				Type:         "password",
				DefaultValue: "",
				Tags:         config.Tags{Section: "ModelSettings"},
			},
			{
				Name:         "enable_kb",
				Label:        "Enable Knowledge Base",
				Type:         "checkbox",
				DefaultValue: false,
				Tags:         config.Tags{Section: "MemorySettings"},
			},
			{
				Name:         "kb_results",
				Label:        "Knowledge Base Results",
				Type:         "number",
				DefaultValue: 5,
				Min:          1,
				Step:         1,
				Tags:         config.Tags{Section: "MemorySettings"},
			},
			{
				Name:         "long_term_memory",
				Label:        "Long Term Memory",
				Type:         "checkbox",
				DefaultValue: false,
				Tags:         config.Tags{Section: "MemorySettings"},
			},
			{
				Name:         "summary_long_term_memory",
				Label:        "Summary Long Term Memory",
				Type:         "checkbox",
				DefaultValue: false,
				Tags:         config.Tags{Section: "MemorySettings"},
			},
			{
				Name:         "system_prompt",
				Label:        "System Prompt",
				Type:         "textarea",
				DefaultValue: "",
				HelpText:     "Instructions that define the agent's behavior and capabilities",
				Tags:         config.Tags{Section: "PromptsGoals"},
			},
			{
				Name:         "permanent_goal",
				Label:        "Permanent Goal",
				Type:         "textarea",
				DefaultValue: "",
				HelpText:     "Long-term objective for the agent to pursue",
				Tags:         config.Tags{Section: "PromptsGoals"},
			},
			{
				Name:         "standalone_job",
				Label:        "Standalone Job",
				Type:         "checkbox",
				DefaultValue: false,
				HelpText:     "Run as a standalone job without user interaction",
				Tags:         config.Tags{Section: "AdvancedSettings"},
			},
			{
				Name:         "initiate_conversations",
				Label:        "Initiate Conversations",
				Type:         "checkbox",
				DefaultValue: false,
				HelpText:     "Allow agent to start conversations on its own",
				Tags:         config.Tags{Section: "AdvancedSettings"},
			},
			{
				Name:         "enable_planning",
				Label:        "Enable Planning",
				Type:         "checkbox",
				DefaultValue: false,
				HelpText:     "Enable agent to create and execute plans",
				Tags:         config.Tags{Section: "AdvancedSettings"},
			},
			{
				Name:         "can_stop_itself",
				Label:        "Can Stop Itself",
				Type:         "checkbox",
				DefaultValue: false,
				HelpText:     "Allow agent to terminate its own execution",
				Tags:         config.Tags{Section: "AdvancedSettings"},
			},
			{
				Name:         "periodic_runs",
				Label:        "Periodic Runs",
				Type:         "text",
				DefaultValue: "",
				Placeholder:  "10m",
				HelpText:     "Duration for scheduling periodic agent runs",
				Tags:         config.Tags{Section: "AdvancedSettings"},
			},
			{
				Name:         "enable_reasoning",
				Label:        "Enable Reasoning",
				Type:         "checkbox",
				DefaultValue: true,
				HelpText:     "Enable agent to explain its reasoning process",
				Tags:         config.Tags{Section: "AdvancedSettings"},
			},
			{
				Name:         "loop_detection_steps",
				Label:        "Max Loop Detection Steps",
				Type:         "number",
				DefaultValue: 5,
				Min:          1,
				Step:         1,
				Tags:         config.Tags{Section: "AdvancedSettings"},
			},
			{
				Name:         "parallel_jobs",
				Label:        "Parallel Jobs",
				Type:         "number",
				DefaultValue: 5,
				Min:          1,
				Step:         1,
				HelpText:     "Number of concurrent tasks that can run in parallel",
				Tags:         config.Tags{Section: "AdvancedSettings"},
			},
			{
				Name:         "strip_thinking_tags",
				Label:        "Strip Thinking Tags",
				Type:         "checkbox",
				DefaultValue: false,
				HelpText:     "Remove content between <thinking></thinking> and <think></think> tags from agent responses",
				Tags:         config.Tags{Section: "ModelSettings"},
			},
			{
				Name:         "enable_evaluation",
				Label:        "Enable Evaluation",
				Type:         "checkbox",
				DefaultValue: false,
				HelpText:     "Enable automatic evaluation of agent responses to ensure they meet user requirements",
				Tags:         config.Tags{Section: "AdvancedSettings"},
			},
			{
				Name:         "max_evaluation_loops",
				Label:        "Max Evaluation Loops",
				Type:         "number",
				DefaultValue: 2,
				Min:          1,
				Step:         1,
				HelpText:     "Maximum number of evaluation loops to perform when addressing gaps in responses",
				Tags:         config.Tags{Section: "AdvancedSettings"},
			},
			{
				Name:         "last_message_duration",
				Label:        "Last Message Duration",
				Type:         "text",
				DefaultValue: "5m",
				HelpText:     "Duration for the last message to be considered in the conversation",
				Tags:         config.Tags{Section: "AdvancedSettings"},
			},
		},
		MCPServers: []config.Field{
			{
				Name:     "url",
				Label:    "URL",
				Type:     config.FieldTypeText,
				Required: true,
			},
		},
		DynamicPrompts: dynamicPromptsConfig,
		Connectors:     connectorsConfig,
		Actions:        actionsConfig,
		Filters:        filtersConfig,
	}
}

type Connector interface {
	AgentResultCallback() func(state types.ActionState)
	AgentReasoningCallback() func(state types.ActionCurrentState) bool
	Start(a *agent.Agent)
}
