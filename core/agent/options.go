package agent

import (
	"context"
	"strings"
	"time"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/sashabaranov/go-openai"
	"github.com/mudler/LocalAGI/pkg/llm"
)

type Option func(*options) error

type llmOptions struct {
	APIURL          string
	APIKey          string
	Model           string
	MultimodalModel string
}

type options struct {
	llmClient llm.LLMClient
	LLMAPI                                                                                       llmOptions
	character                                                                                    Character
	randomIdentityGuidance                                                                       string
	randomIdentity                                                                               bool
	userActions                                                                                  types.Actions
	jobFilters                                                                                   types.JobFilters
	enableHUD, standaloneJob, showCharacter, enableKB, enableSummaryMemory, enableLongTermMemory bool
	stripThinkingTags                                                                            bool

	canStopItself         bool
	initiateConversations bool
	loopDetectionSteps    int
	forceReasoning        bool
	canPlan               bool
	characterfile         string
	statefile             string
	context               context.Context
	permanentGoal         string
	timeout               string
	periodicRuns          time.Duration
	kbResults             int
	ragdb                 RAGDB

	// Evaluation settings
	maxEvaluationLoops int
	enableEvaluation   bool

	prompts []DynamicPrompt

	systemPrompt string

	// callbacks
	reasoningCallback func(types.ActionCurrentState) bool
	resultCallback    func(types.ActionState)

	conversationsPath string

	mcpServers                  []MCPServer
	mcpStdioServers             []MCPSTDIOServer
	mcpBoxURL                   string
	mcpPrepareScript            string
	newConversationsSubscribers []func(openai.ChatCompletionMessage)

	observer     Observer
	parallelJobs int

	lastMessageDuration time.Duration
}

// WithLLMClient allows injecting a custom LLM client (e.g. for testing)
func WithLLMClient(client llm.LLMClient) Option {
	return func(o *options) error {
		o.llmClient = client
		return nil
	}
}

func (o *options) SeparatedMultimodalModel() bool {
	return o.LLMAPI.MultimodalModel != "" && o.LLMAPI.Model != o.LLMAPI.MultimodalModel
}

func defaultOptions() *options {
	return &options{
		parallelJobs:       1,
		periodicRuns:       15 * time.Minute,
		loopDetectionSteps: 10,
		maxEvaluationLoops: 2,
		enableEvaluation:   false,
		LLMAPI: llmOptions{
			APIURL: "http://localhost:8080",
			Model:  "gpt-4",
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

func WithLoopDetectionSteps(steps int) Option {
	return func(o *options) error {
		o.loopDetectionSteps = steps
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

func WithNewConversationSubscriber(sub func(openai.ChatCompletionMessage)) Option {
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

func WithSystemPrompt(prompt string) Option {
	return func(o *options) error {
		o.systemPrompt = prompt
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

func WithMCPBoxURL(url string) Option {
	return func(o *options) error {
		o.mcpBoxURL = url
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
		o.userActions = actions
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
