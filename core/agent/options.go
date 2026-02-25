package agent

import (
	"context"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/mudler/LocalAGI/core/types"
)

type Option func(*options) error

// ConversationStorageMode defines how conversations are stored in the knowledge base
type ConversationStorageMode string

const (
	// StoreUserOnly stores only user messages (default)
	StoreUserOnly ConversationStorageMode = "user_only"
	// StoreUserAndAssistant stores both user and assistant messages separately
	StoreUserAndAssistant ConversationStorageMode = "user_and_assistant"
	// StoreWholeConversation stores the entire conversation as a single block
	StoreWholeConversation ConversationStorageMode = "whole_conversation"
)

type llmOptions struct {
	APIURL                string
	APIKey                string
	Model                 string
	MultimodalModel       string
	ReviewerModel         string
	TranscriptionModel    string
	TranscriptionLanguage string
	TTSModel              string
}

type options struct {
	LLMAPI                                                                                       llmOptions
	character                                                                                    Character
	randomIdentityGuidance                                                                       string
	randomIdentity                                                                               bool
	userActions                                                                                  types.Actions
	jobFilters                                                                                   types.JobFilters
	enableHUD, standaloneJob, showCharacter, enableKB, enableSummaryMemory, enableLongTermMemory bool
	stripThinkingTags                                                                            bool
	kbAutoSearch                                                                                 bool
	conversationStorageMode                                                                      ConversationStorageMode

	canStopItself         bool
	initiateConversations bool
	forceReasoning        bool
	forceReasoningTool    bool
	enableGuidedTools     bool
	canPlan               bool
	disableSinkState      bool
	characterfile         string
	statefile             string
	schedulerStorePath    string // Path to scheduler JSON storage file
	context               context.Context
	permanentGoal         string
	timeout               string
	periodicRuns          time.Duration
	schedulerPollInterval time.Duration
	kbResults             int
	ragdb                 RAGDB

	// Evaluation settings
	maxEvaluationLoops int
	loopDetection      int
	enableEvaluation   bool

	prompts []DynamicPrompt

	systemPrompt           string
	innerMonologueTemplate string
	skillPromptTemplate    string
	schedulerTaskTemplate  string

	// callbacks
	reasoningCallback func(types.ActionCurrentState) bool
	resultCallback    func(types.ActionState)

	conversationsPath string

	mcpServers                  []MCPServer
	mcpStdioServers             []MCPSTDIOServer
	mcpPrepareScript            string
	extraMCPSessions            []*mcp.ClientSession
	newConversationsSubscribers []func(*types.ConversationMessage)

	observer     Observer
	parallelJobs int

	lastMessageDuration time.Duration

	// cancelPreviousOnNewMessage: when true (or nil), Enqueue cancels the running job for the same conversation_id. When false, jobs are queued.
	cancelPreviousOnNewMessage *bool

	// maxAttempts: on ExecuteTools failure, retry up to this many times before surfacing the error to the user (1 = no retries).
	maxAttempts int
}

func (o *options) SeparatedMultimodalModel() bool {
	return o.LLMAPI.MultimodalModel != "" && o.LLMAPI.Model != o.LLMAPI.MultimodalModel
}

func defaultOptions() *options {
	return &options{
		parallelJobs:            1,
		maxAttempts:             1,
		periodicRuns:            15 * time.Minute,
		schedulerPollInterval:   30 * time.Second,
		maxEvaluationLoops:      2,
		enableEvaluation:        false,
		kbAutoSearch:            true,          // Default to true to maintain backward compatibility
		conversationStorageMode: StoreUserOnly, // Default to user-only for backward compatibility
		LLMAPI: llmOptions{
			APIURL:                "http://localhost:8080",
			Model:                 "gpt-4",
			TranscriptionModel:    "whisper-1",
			TranscriptionLanguage: "",
			TTSModel:              "tts-1",
		},
		character: Character{
			Name:       "",
			Age:        "",
			Occupation: "",
			Hobbies:    []string{},
			MusicTaste: []string{},
		},
	}
}

func newOptions(opts ...Option) (*options, error) {
	options := defaultOptions()
	for _, o := range opts {
		if err := o(options); err != nil {
			return nil, err
		}
	}
	return options, nil
}

var EnableHUD = func(o *options) error {
	o.enableHUD = true
	return nil
}

var EnableForceReasoning = func(o *options) error {
	o.forceReasoning = true
	return nil
}

var EnableGuidedTools = func(o *options) error {
	o.enableGuidedTools = true
	return nil
}

var EnableForceReasoningTool = func(o *options) error {
	o.forceReasoningTool = true
	return nil
}

var EnableKnowledgeBase = func(o *options) error {
	o.enableKB = true
	o.kbResults = 5
	return nil
}

var CanStopItself = func(o *options) error {
	o.canStopItself = true
	return nil
}

func WithTimeout(timeout string) Option {
	return func(o *options) error {
		o.timeout = timeout
		return nil
	}
}

func WithConversationsPath(path string) Option {
	return func(o *options) error {
		o.conversationsPath = path
		return nil
	}
}

func EnableKnowledgeBaseWithResults(results int) Option {
	return func(o *options) error {
		o.enableKB = true
		o.kbResults = results
		return nil
	}
}

func WithLastMessageDuration(duration string) Option {
	return func(o *options) error {
		d, err := time.ParseDuration(duration)
		if err != nil {
			d = types.DefaultLastMessageDuration
		}
		o.lastMessageDuration = d
		return nil
	}
}

func WithParallelJobs(jobs int) Option {
	return func(o *options) error {
		o.parallelJobs = jobs
		return nil
	}
}

// WithCancelPreviousOnNewMessage sets whether a new job with the same conversation_id cancels the currently running job (true) or is queued (false). Nil/default means true.
func WithCancelPreviousOnNewMessage(cancel bool) Option {
	return func(o *options) error {
		o.cancelPreviousOnNewMessage = &cancel
		return nil
	}
}

// WithMaxAttempts sets how many times to attempt execution on failure before surfacing the error to the user (1 = no retries).
func WithMaxAttempts(attempts int) Option {
	return func(o *options) error {
		o.maxAttempts = attempts
		return nil
	}
}

func WithLoopDetection(loops int) Option {
	return func(o *options) error {
		o.loopDetection = loops
		return nil
	}
}

func WithNewConversationSubscriber(sub func(*types.ConversationMessage)) Option {
	return func(o *options) error {
		o.newConversationsSubscribers = append(o.newConversationsSubscribers, sub)
		return nil
	}
}

var EnableInitiateConversations = func(o *options) error {
	o.initiateConversations = true
	return nil
}

var EnablePlanning = func(o *options) error {
	o.canPlan = true
	return nil
}

var DisableSinkState = func(o *options) error {
	o.disableSinkState = true
	return nil
}

var WithPlanReviewerLLM = func(model string) Option {
	return func(o *options) error {
		o.LLMAPI.ReviewerModel = model
		return nil
	}
}

// EnableStandaloneJob is an option to enable the agent
// to run jobs in the background automatically
var EnableStandaloneJob = func(o *options) error {
	o.standaloneJob = true
	return nil
}

var EnablePersonality = func(o *options) error {
	o.showCharacter = true
	return nil
}

var EnableSummaryMemory = func(o *options) error {
	o.enableSummaryMemory = true
	return nil
}

var EnableLongTermMemory = func(o *options) error {
	o.enableLongTermMemory = true
	return nil
}

func WithRAGDB(db RAGDB) Option {
	return func(o *options) error {
		o.ragdb = db
		return nil
	}
}

// WithConversationStorageMode sets how conversations are stored in the knowledge base
func WithConversationStorageMode(mode ConversationStorageMode) Option {
	return func(o *options) error {
		switch mode {
		case StoreUserOnly, StoreUserAndAssistant, StoreWholeConversation:
			o.conversationStorageMode = mode
		default:
			o.conversationStorageMode = StoreUserOnly
		}
		return nil
	}
}

func WithSystemPrompt(prompt string) Option {
	return func(o *options) error {
		o.systemPrompt = prompt
		return nil
	}
}

// WithInnerMonologueTemplate sets the prompt used for periodic/standalone runs. If empty, the default template is used.
func WithInnerMonologueTemplate(template string) Option {
	return func(o *options) error {
		o.innerMonologueTemplate = template
		return nil
	}
}

// WithSkillPromptTemplate sets the template for rendering skills in the prompt. If empty, the default template is used.
func WithSkillPromptTemplate(template string) Option {
	return func(o *options) error {
		o.skillPromptTemplate = template
		return nil
	}
}

func WithMCPServers(servers ...MCPServer) Option {
	return func(o *options) error {
		o.mcpServers = servers
		return nil
	}
}

func WithMCPSTDIOServers(servers ...MCPSTDIOServer) Option {
	return func(o *options) error {
		o.mcpStdioServers = servers
		return nil
	}
}

func WithMCPPrepareScript(script string) Option {
	return func(o *options) error {
		o.mcpPrepareScript = script
		return nil
	}
}

func WithLLMAPIURL(url string) Option {
	return func(o *options) error {
		o.LLMAPI.APIURL = url
		return nil
	}
}

func WithStateFile(path string) Option {
	return func(o *options) error {
		o.statefile = path
		return nil
	}
}

func WithCharacterFile(path string) Option {
	return func(o *options) error {
		o.characterfile = path
		return nil
	}
}

// WithPrompts adds additional block prompts to the agent
// to be rendered internally in the conversation
// when processing the conversation to the LLM
func WithPrompts(prompts ...DynamicPrompt) Option {
	return func(o *options) error {
		o.prompts = prompts
		return nil
	}
}

// WithMCPSession adds a pre-connected MCP client session (e.g. in-process skills MCP) to the agent.
func WithMCPSession(session *mcp.ClientSession) Option {
	return func(o *options) error {
		o.extraMCPSessions = append(o.extraMCPSessions, session)
		return nil
	}
}

// WithDynamicPrompts is a helper function to create dynamic prompts
// Dynamic prompts contains golang code which is executed dynamically
// // to render a prompt to the LLM
// func WithDynamicPrompts(prompts ...map[string]string) Option {
// 	return func(o *options) error {
// 		for _, p := range prompts {
// 			prompt, err := NewDynamicPrompt(p, "")
// 			if err != nil {
// 				return err
// 			}
// 			o.prompts = append(o.prompts, prompt)
// 		}
// 		return nil
// 	}
// }

func WithLLMAPIKey(key string) Option {
	return func(o *options) error {
		o.LLMAPI.APIKey = key
		return nil
	}
}

func WithMultimodalModel(model string) Option {
	return func(o *options) error {
		o.LLMAPI.MultimodalModel = model
		return nil
	}
}

func WithPermanentGoal(goal string) Option {
	return func(o *options) error {
		o.permanentGoal = goal
		return nil
	}
}

func WithPeriodicRuns(duration string) Option {
	return func(o *options) error {
		t, err := time.ParseDuration(duration)
		if err != nil {
			o.periodicRuns, _ = time.ParseDuration("10m")
		}
		o.periodicRuns = t
		return nil
	}
}

func WithSchedulerPollInterval(duration string) Option {
	return func(o *options) error {
		t, err := time.ParseDuration(duration)
		if err != nil {
			o.schedulerPollInterval = 30 * time.Second
			return nil
		}
		o.schedulerPollInterval = t
		return nil
	}
}

func WithContext(ctx context.Context) Option {
	return func(o *options) error {
		o.context = ctx
		return nil
	}
}

func WithAgentReasoningCallback(cb func(types.ActionCurrentState) bool) Option {
	return func(o *options) error {
		o.reasoningCallback = cb
		return nil
	}
}

func WithAgentResultCallback(cb func(types.ActionState)) Option {
	return func(o *options) error {
		o.resultCallback = cb
		return nil
	}
}

func WithModel(model string) Option {
	return func(o *options) error {
		o.LLMAPI.Model = model
		return nil
	}
}

func WithCharacter(c Character) Option {
	return func(o *options) error {
		o.character = c
		return nil
	}
}

func FromFile(path string) Option {
	return func(o *options) error {
		c, err := Load(path)
		if err != nil {
			return err
		}
		o.character = *c
		return nil
	}
}

func WithRandomIdentity(guidance ...string) Option {
	return func(o *options) error {
		o.randomIdentityGuidance = strings.Join(guidance, "")
		o.randomIdentity = true
		o.showCharacter = true
		return nil
	}
}

func WithActions(actions ...types.Action) Option {
	return func(o *options) error {
		o.userActions = append(o.userActions, actions...)
		return nil
	}
}

func WithJobFilters(filters ...types.JobFilter) Option {
	return func(o *options) error {
		o.jobFilters = filters
		return nil
	}
}

func WithObserver(observer Observer) Option {
	return func(o *options) error {
		o.observer = observer
		return nil
	}
}

var EnableStripThinkingTags = func(o *options) error {
	o.stripThinkingTags = true
	return nil
}

func WithMaxEvaluationLoops(loops int) Option {
	return func(o *options) error {
		o.maxEvaluationLoops = loops
		return nil
	}
}

func EnableEvaluation() Option {
	return func(o *options) error {
		o.enableEvaluation = true
		return nil
	}
}

func WithTranscriptionModel(model string) Option {
	return func(o *options) error {
		o.LLMAPI.TranscriptionModel = model
		return nil
	}
}

func WithTranscriptionLanguage(language string) Option {
	return func(o *options) error {
		o.LLMAPI.TranscriptionLanguage = language
		return nil
	}
}

func WithTTSModel(model string) Option {
	return func(o *options) error {
		o.LLMAPI.TTSModel = model
		return nil
	}
}

func WithKBAutoSearch(enabled bool) Option {
	return func(o *options) error {
		o.kbAutoSearch = enabled
		return nil
	}
}

// WithSchedulerStorePath sets the path for the scheduler's JSON storage file
func WithSchedulerStorePath(path string) Option {
	return func(o *options) error {
		o.schedulerStorePath = path
		return nil
	}
}

// WithSchedulerTaskTemplate sets the prompt used for scheduled/recurring tasks run by the scheduler.
// If empty, the default inner monologue template is used with the task injected.
func WithSchedulerTaskTemplate(template string) Option {
	return func(o *options) error {
		o.schedulerTaskTemplate = template
		return nil
	}
}
