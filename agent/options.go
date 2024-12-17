package agent

import (
	"context"
	"strings"
	"time"
)

type Option func(*options) error
type llmOptions struct {
	APIURL string
	APIKey string
	Model  string
}

type options struct {
	LLMAPI                                                                                       llmOptions
	character                                                                                    Character
	randomIdentityGuidance                                                                       string
	randomIdentity                                                                               bool
	userActions                                                                                  Actions
	enableHUD, standaloneJob, showCharacter, enableKB, enableSummaryMemory, enableLongTermMemory bool

	canStopItself         bool
	initiateConversations bool
	forceReasoning        bool
	characterfile         string
	statefile             string
	context               context.Context
	permanentGoal         string
	timeout               string
	periodicRuns          time.Duration
	kbResults             int
	ragdb                 RAGDB

	prompts []PromptBlock

	systemPrompt string

	// callbacks
	reasoningCallback func(ActionCurrentState) bool
	resultCallback    func(ActionState)
}

type PromptBlock interface {
	Render(a *Agent) string
	Role() string
}

func defaultOptions() *options {
	return &options{
		periodicRuns: 15 * time.Minute,
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

func EnableKnowledgeBaseWithResults(results int) Option {
	return func(o *options) error {
		o.enableKB = true
		o.kbResults = results
		return nil
	}
}

var EnableInitiateConversations = func(o *options) error {
	o.initiateConversations = true
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
func WithPrompts(prompts ...PromptBlock) Option {
	return func(o *options) error {
		o.prompts = prompts
		return nil
	}
}

func WithLLMAPIKey(key string) Option {
	return func(o *options) error {
		o.LLMAPI.APIKey = key
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

func WithAgentReasoningCallback(cb func(ActionCurrentState) bool) Option {
	return func(o *options) error {
		o.reasoningCallback = cb
		return nil
	}
}

func WithAgentResultCallback(cb func(ActionState)) Option {
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

func WithActions(actions ...Action) Option {
	return func(o *options) error {
		o.userActions = actions
		return nil
	}
}
