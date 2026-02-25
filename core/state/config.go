package state

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
)

// parseIntField parses an integer field that may be received as either a number or a string
func parseIntField(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return 0
}

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
	Connector        []ConnectorConfig      `json:"connectors" form:"connectors" `
	Actions          []ActionsConfig        `json:"actions" form:"actions"`
	DynamicPrompts   []DynamicPromptsConfig `json:"dynamic_prompts" form:"dynamic_prompts"`
	MCPServers       []agent.MCPServer      `json:"mcp_servers" form:"mcp_servers"`
	MCPSTDIOServers  []agent.MCPSTDIOServer `json:"mcp_stdio_servers" form:"mcp_stdio_servers"`
	MCPPrepareScript string                 `json:"mcp_prepare_script" form:"mcp_prepare_script"`
	Filters          []FiltersConfig        `json:"filters" form:"filters"`

	Description string `json:"description" form:"description"`

	Model                 string `json:"model" form:"model"`
	MultimodalModel       string `json:"multimodal_model" form:"multimodal_model"`
	TranscriptionModel    string `json:"transcription_model" form:"transcription_model"`
	TranscriptionLanguage string `json:"transcription_language" form:"transcription_language"`
	TTSModel              string `json:"tts_model" form:"tts_model"`
	APIURL                string `json:"api_url" form:"api_url"`
	APIKey                string `json:"api_key" form:"api_key"`
	LocalRAGURL           string `json:"local_rag_url" form:"local_rag_url"`
	LocalRAGAPIKey        string `json:"local_rag_api_key" form:"local_rag_api_key"`
	LastMessageDuration   string `json:"last_message_duration" form:"last_message_duration"`

	Name                       string `json:"name" form:"name"`
	HUD                        bool   `json:"hud" form:"hud"`
	StandaloneJob              bool   `json:"standalone_job" form:"standalone_job"`
	RandomIdentity             bool   `json:"random_identity" form:"random_identity"`
	InitiateConversations      bool   `json:"initiate_conversations" form:"initiate_conversations"`
	CanPlan                    bool   `json:"enable_planning" form:"enable_planning"`
	PlanReviewerModel          string `json:"plan_reviewer_model" form:"plan_reviewer_model"`
	DisableSinkState           bool   `json:"disable_sink_state" form:"disable_sink_state"`
	IdentityGuidance           string `json:"identity_guidance" form:"identity_guidance"`
	PeriodicRuns               string `json:"periodic_runs" form:"periodic_runs"`
	SchedulerPollInterval      string `json:"scheduler_poll_interval" form:"scheduler_poll_interval"`
	SchedulerTaskTemplate   string `json:"scheduler_task_template" form:"scheduler_task_template"`
	PermanentGoal              string `json:"permanent_goal" form:"permanent_goal"`
	EnableKnowledgeBase        bool   `json:"enable_kb" form:"enable_kb"`
	EnableKBCompaction         bool   `json:"enable_kb_compaction" form:"enable_kb_compaction"`
	KBCompactionInterval       string `json:"kb_compaction_interval" form:"kb_compaction_interval"`
	KBCompactionSummarize      bool   `json:"kb_compaction_summarize" form:"kb_compaction_summarize"`
	KBAutoSearch               bool   `json:"kb_auto_search" form:"kb_auto_search"`
	KBAsTools                  bool   `json:"kb_as_tools" form:"kb_as_tools"`
	EnableReasoning            bool   `json:"enable_reasoning" form:"enable_reasoning"`
	EnableForceReasoningTool   bool   `json:"enable_reasoning_tool" form:"enable_reasoning_tool"`
	EnableGuidedTools          bool   `json:"enable_guided_tools" form:"enable_guided_tools"`
	EnableSkills               bool   `json:"enable_skills" form:"enable_skills"`
	KnowledgeBaseResults       int    `json:"kb_results" form:"kb_results"`
	CanStopItself              bool   `json:"can_stop_itself" form:"can_stop_itself"`
	SystemPrompt               string `json:"system_prompt" form:"system_prompt"`
	SkillsPrompt               string `json:"skills_prompt" form:"skills_prompt"`
	InnerMonologueTemplate     string `json:"inner_monologue_template" form:"inner_monologue_template"`
	LongTermMemory             bool   `json:"long_term_memory" form:"long_term_memory"`
	SummaryLongTermMemory      bool   `json:"summary_long_term_memory" form:"summary_long_term_memory"`
	ConversationStorageMode    string `json:"conversation_storage_mode" form:"conversation_storage_mode"`
	ParallelJobs               int    `json:"parallel_jobs" form:"parallel_jobs"`
	CancelPreviousOnNewMessage *bool  `json:"cancel_previous_on_new_message" form:"cancel_previous_on_new_message"`
	StripThinkingTags          bool   `json:"strip_thinking_tags" form:"strip_thinking_tags"`
	EnableEvaluation           bool   `json:"enable_evaluation" form:"enable_evaluation"`
	MaxEvaluationLoops         int    `json:"max_evaluation_loops" form:"max_evaluation_loops"`
	MaxAttempts                int    `json:"max_attempts" form:"max_attempts"`
	LoopDetection              int    `json:"loop_detection" form:"loop_detection"`
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
				Type:         "text",
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
				Name:         "transcription_model",
				Label:        "Transcription Model",
				Type:         "text",
				DefaultValue: "",
				Tags:         config.Tags{Section: "ModelSettings"},
			},
			{
				Name:         "transcription_language",
				Label:        "Transcription Language",
				Type:         "text",
				DefaultValue: "",
				Tags:         config.Tags{Section: "ModelSettings"},
			},
			{
				Name:         "tts_model",
				Label:        "TTS Model",
				Type:         "text",
				DefaultValue: "",
				Tags:         config.Tags{Section: "ModelSettings"},
			},
			{
				Name:         "plan_reviewer_model",
				Label:        "Plan Reviewer Model",
				Type:         "text",
				DefaultValue: "",
				Tags:         config.Tags{Section: "ModelSettings"},
			},
			{
				Name:         "api_url",
				Label:        "API URL",
				Type:         "text",
				DefaultValue: "",
				Tags:         config.Tags{Section: "ModelSettings"},
			},
			{
				Name:         "api_key",
				Label:        "API Key",
				Type:         "password",
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
				Name:         "enable_kb_compaction",
				Label:        "Enable KB Compaction",
				Type:         "checkbox",
				DefaultValue: false,
				HelpText:     "Periodically group collection entries by date (daily/weekly/monthly), optionally summarize or concatenate, then store and remove originals",
				Tags:         config.Tags{Section: "MemorySettings"},
			},
			{
				Name:         "kb_compaction_interval",
				Label:        "KB Compaction Interval",
				Type:         "text",
				DefaultValue: "daily",
				Placeholder:  "daily, weekly, monthly",
				HelpText:     "Compaction window: daily, weekly, or monthly",
				Tags:         config.Tags{Section: "MemorySettings"},
			},
			{
				Name:         "kb_compaction_summarize",
				Label:        "KB Compaction Summarize",
				Type:         "checkbox",
				DefaultValue: true,
				HelpText:     "When enabled, summarize grouped content via LLM; when disabled, store concatenated content only (no LLM call)",
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
				Name:         "kb_auto_search",
				Label:        "KB Auto Search",
				Type:         "checkbox",
				DefaultValue: true,
				HelpText:     "Automatically search knowledge base when a user message is received",
				Tags:         config.Tags{Section: "MemorySettings"},
			},
			{
				Name:         "kb_as_tools",
				Label:        "KB As Tools",
				Type:         "checkbox",
				DefaultValue: false,
				HelpText:     "Inject knowledge base search and add actions as tools, allowing the agent to access its memory without manual configuration",
				Tags:         config.Tags{Section: "MemorySettings"},
			},
			{
				Name:         "conversation_storage_mode",
				Label:        "Conversation Storage Mode",
				Type:         "select",
				DefaultValue: "user_only",
				Options: []config.FieldOption{
					{Value: "user_only", Label: "User Messages Only"},
					{Value: "user_and_assistant", Label: "User and Assistant Messages"},
					{Value: "whole_conversation", Label: "Whole Conversation as Block"},
				},
				HelpText: "Controls what gets stored in the knowledge base: only user messages, user and assistant messages separately, or the entire conversation as a single block",
				Tags:     config.Tags{Section: "MemorySettings"},
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
				Name:         "skills_prompt",
				Label:        "Skills Prompt",
				Type:         "textarea",
				DefaultValue: "",
				HelpText:     "Optional instructions for using skills. Used when Enable Skills is on. If empty, default instructions are used.",
				Tags:         config.Tags{Section: "PromptsGoals"},
			},
			{
				Name:         "inner_monologue_template",
				Label:        "Inner Monologue Template",
				Type:         "textarea",
				DefaultValue: "",
				HelpText:     "Prompt used for periodic/standalone runs when the agent evaluates what to do next. If empty, the default autonomous agent instructions are used.",
				Tags:         config.Tags{Section: "PromptsGoals"},
			},
				{
					Name:         "scheduler_task_template",
					Label:        "Scheduler Task Template",
					Type:         "textarea",
					DefaultValue: "",
					HelpText:     "Template for scheduled/recurring tasks. Use {{.Task}} to reference the task. Example: \"Execute: {{.Task}}\". If empty, the default inner monologue template is used with the task injected.",
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
				Name:         "cancel_previous_on_new_message",
				Label:        "Cancel previous message on new message",
				Type:         "checkbox",
				DefaultValue: true,
				HelpText:     "When a new message arrives for the same conversation, cancel the currently running job and start the new one. If disabled, new messages are queued.",
				Tags:         config.Tags{Section: "AdvancedSettings"},
			},
			{
				Name:         "loop_detection",
				Label:        "Loop Detection",
				Type:         "number",
				DefaultValue: 5,
				Min:          1,
				Step:         1,
				HelpText:     "Number of messages to check for loop detection. If a message is the same as the previous message, the job is cancelled.",
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
				Name:         "scheduler_poll_interval",
				Label:        "Scheduler Poll Interval",
				Type:         "text",
				DefaultValue: "30s",
				Placeholder:  "30s",
				HelpText:     "Duration for polling the scheduler for planned tasks",
				Tags:         config.Tags{Section: "AdvancedSettings"},
			},
			{
				Name:         "enable_reasoning",
				Label:        "Enable Reasoning",
				Type:         "checkbox",
				DefaultValue: false,
				HelpText:     "Enable agent to explain its reasoning process",
				Tags:         config.Tags{Section: "AdvancedSettings"},
			},
			{
				Name:         "enable_reasoning_tool",
				Label:        "Enable Reasoning for tools",
				Type:         "checkbox",
				DefaultValue: true,
				HelpText:     "Enable agent to reason more on tools",
				Tags:         config.Tags{Section: "AdvancedSettings"},
			},
			{
				Name:         "enable_guided_tools",
				Label:        "Enable Guided Tools",
				Type:         "checkbox",
				DefaultValue: false,
				HelpText:     "Filter tools through guidance using their descriptions; creates virtual guidelines when none exist",
				Tags:         config.Tags{Section: "AdvancedSettings"},
			},
			{
				Name:         "enable_skills",
				Label:        "Enable Skills",
				Type:         "checkbox",
				DefaultValue: false,
				HelpText:     "Inject available skills into the agent and expose skill tools (list, read, search, resources) via MCP",
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
				Name:         "disable_sink_state",
				Label:        "Disable Sink State",
				Type:         "checkbox",
				DefaultValue: false,
				HelpText:     "Disable the sink state of the agent",
				Tags:         config.Tags{Section: "AdvancedSettings"},
			},
			{
				Name:         "mcp_stdio_servers",
				Label:        "MCP STDIO Servers",
				Type:         "textarea",
				DefaultValue: "",
				HelpText:     "JSON configuration for MCP STDIO servers",
				Tags:         config.Tags{Section: "MCP"},
			},
			{
				Name:         "mcp_prepare_script",
				Label:        "MCP Prepare Script",
				Type:         "textarea",
				DefaultValue: "",
				HelpText:     "Script to prepare for running MCP servers",
				Tags:         config.Tags{Section: "MCP"},
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
				Name:         "max_attempts",
				Label:        "Max Attempts",
				Type:         "number",
				DefaultValue: 1,
				Min:          1,
				Step:         1,
				HelpText:     "Number of attempts on failure before surfacing the error to the user (1 = no retries)",
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
			{
				Name:     "token",
				Label:    "API Key",
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

// UnmarshalJSON implements json.Unmarshaler for AgentConfig
func (a *AgentConfig) UnmarshalJSON(data []byte) error {
	// Create a temporary type to avoid infinite recursion
	type Alias AgentConfig
	aux := &struct {
		*Alias
		MCPSTDIOServersConfig interface{} `json:"mcp_stdio_servers"`
		MaxEvaluationLoops    interface{} `json:"max_evaluation_loops"`
		MaxAttempts            interface{} `json:"max_attempts"`
		ParallelJobs           interface{} `json:"parallel_jobs"`
		KnowledgeBaseResults  interface{} `json:"kb_results"`
	}{
		Alias: (*Alias)(a),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Parse integer fields that may come as strings
	a.MaxEvaluationLoops = parseIntField(aux.MaxEvaluationLoops)
	a.MaxAttempts = parseIntField(aux.MaxAttempts)
	a.ParallelJobs = parseIntField(aux.ParallelJobs)
	a.KnowledgeBaseResults = parseIntField(aux.KnowledgeBaseResults)
	a.LoopDetection = parseIntField(aux.LoopDetection)

	// Handle MCP STDIO servers configuration
	if aux.MCPSTDIOServersConfig != nil {
		switch v := aux.MCPSTDIOServersConfig.(type) {
		case string:
			// Parse string configuration
			var mcpConfig struct {
				MCPServers map[string]struct {
					Command string            `json:"command"`
					Args    []string          `json:"args"`
					Env     map[string]string `json:"env"`
				} `json:"mcpServers"`
			}

			if err := json.Unmarshal([]byte(v), &mcpConfig); err != nil {
				return fmt.Errorf("failed to parse MCP STDIO servers configuration: %w", err)
			}

			a.MCPSTDIOServers = make([]agent.MCPSTDIOServer, 0, len(mcpConfig.MCPServers))
			for name, server := range mcpConfig.MCPServers {
				// Convert env map to slice of "KEY=VALUE" strings
				envSlice := make([]string, 0, len(server.Env))
				for k, v := range server.Env {
					envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
				}

				a.MCPSTDIOServers = append(a.MCPSTDIOServers, agent.MCPSTDIOServer{
					Name: name,
					Cmd:  server.Command,
					Args: server.Args,
					Env:  envSlice,
				})
			}
		case []interface{}:
			// Parse array configuration
			a.MCPSTDIOServers = make([]agent.MCPSTDIOServer, 0, len(v))
			for _, server := range v {
				serverMap, ok := server.(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid server configuration format")
				}

				name, _ := serverMap["name"].(string)
				cmd, _ := serverMap["cmd"].(string)
				args := make([]string, 0)
				if argsInterface, ok := serverMap["args"].([]interface{}); ok {
					for _, arg := range argsInterface {
						if argStr, ok := arg.(string); ok {
							args = append(args, argStr)
						}
					}
				}

				env := make([]string, 0)
				if envInterface, ok := serverMap["env"].([]interface{}); ok {
					for _, e := range envInterface {
						if envStr, ok := e.(string); ok {
							env = append(env, envStr)
						}
					}
				}

				a.MCPSTDIOServers = append(a.MCPSTDIOServers, agent.MCPSTDIOServer{
					Name: name,
					Cmd:  cmd,
					Args: args,
					Env:  env,
				})
			}
		}
	}

	return nil
}

// MarshalJSON implements json.Marshaler for AgentConfig
func (a *AgentConfig) MarshalJSON() ([]byte, error) {
	// Create a temporary type to avoid infinite recursion
	type Alias AgentConfig
	aux := &struct {
		*Alias
		MCPSTDIOServersConfig string `json:"mcp_stdio_servers,omitempty"`
	}{
		Alias: (*Alias)(a),
	}

	// Convert MCPSTDIOServers back to the expected JSON format
	if len(a.MCPSTDIOServers) > 0 {
		mcpConfig := struct {
			MCPServers map[string]struct {
				Command string            `json:"command"`
				Args    []string          `json:"args"`
				Env     map[string]string `json:"env"`
			} `json:"mcpServers"`
		}{
			MCPServers: make(map[string]struct {
				Command string            `json:"command"`
				Args    []string          `json:"args"`
				Env     map[string]string `json:"env"`
			}),
		}

		// Convert each MCPSTDIOServer to the expected format
		for i, server := range a.MCPSTDIOServers {
			// Convert env slice back to map
			envMap := make(map[string]string)
			for _, env := range server.Env {
				if parts := strings.SplitN(env, "=", 2); len(parts) == 2 {
					envMap[parts[0]] = parts[1]
				}
			}

			key := server.Name
			if key == "" {
				key = fmt.Sprintf("server%d", i)
			}
			mcpConfig.MCPServers[key] = struct {
				Command string            `json:"command"`
				Args    []string          `json:"args"`
				Env     map[string]string `json:"env"`
			}{
				Command: server.Cmd,
				Args:    server.Args,
				Env:     envMap,
			}
		}

		// Marshal the MCP config to JSON string
		mcpConfigJSON, err := json.Marshal(mcpConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal MCP STDIO servers configuration: %w", err)
		}
		aux.MCPSTDIOServersConfig = string(mcpConfigJSON)
	}

	return json.Marshal(aux)
}
