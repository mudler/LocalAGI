package types

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type ActionContext struct {
	context.Context
	cancelFunc context.CancelFunc
}

func (ac *ActionContext) Cancel() {
	if ac.cancelFunc != nil {
		ac.cancelFunc()
	}
}

func NewActionContext(ctx context.Context, cancel context.CancelFunc) *ActionContext {
	return &ActionContext{
		Context:    ctx,
		cancelFunc: cancel,
	}
}

type ActionParams map[string]interface{}

type ActionResult struct {
	Job      *Job
	Result   string
	Metadata map[string]interface{}
}

func (ap ActionParams) Read(s string) error {
	err := json.Unmarshal([]byte(s), &ap)
	return err
}

func (ap ActionParams) String() string {
	b, _ := json.Marshal(ap)
	return string(b)
}

func (ap ActionParams) Unmarshal(v interface{}) error {
	b, err := json.Marshal(ap)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return err
	}
	return nil
}

//type ActionDefinition openai.FunctionDefinition

type ActionDefinition struct {
	Properties  map[string]jsonschema.Definition
	Required    []string
	Name        ActionDefinitionName
	Description string
}

type ActionDefinitionName string

func (a ActionDefinitionName) Is(name string) bool {
	return string(a) == name
}

func (a ActionDefinitionName) String() string {
	return string(a)
}

func (a ActionDefinition) ToFunctionDefinition() *openai.FunctionDefinition {
	return &openai.FunctionDefinition{
		Name:        a.Name.String(),
		Description: a.Description,
		Parameters: jsonschema.Definition{
			Type:       jsonschema.Object,
			Properties: a.Properties,
			Required:   a.Required,
		},
	}
}

// Actions is something the agent can do
type Action interface {
	Run(ctx context.Context, sharedState *AgentSharedState, action ActionParams) (ActionResult, error)
	Definition() ActionDefinition
	Plannable() bool
}

// UserDefinedChecker interface to identify user-defined actions
type UserDefinedChecker interface {
	IsUserDefined() bool
}

// BaseAction provides default implementation for Action interface
// Embed this in action implementations to get the default IsUserDefined behavior
type BaseAction struct{}

func (b *BaseAction) IsUserDefined() bool {
	return false // Regular actions are not user-defined
}

// IsActionUserDefined checks if an action is user-defined
func IsActionUserDefined(action Action) bool {
	if checker, ok := action.(UserDefinedChecker); ok {
		return checker.IsUserDefined()
	}
	return false // Actions without UserDefinedChecker are not user-defined
}

// UserDefinedAction represents a user-defined function tool
type UserDefinedAction struct {
	ActionDef *ActionDefinition
}

func (u *UserDefinedAction) Run(ctx context.Context, sharedState *AgentSharedState, action ActionParams) (ActionResult, error) {
	// User-defined actions should not be executed directly
	return ActionResult{}, fmt.Errorf("user-defined action '%s' cannot be executed by agent", u.ActionDef.Name)
}

func (u *UserDefinedAction) Definition() ActionDefinition {
	return *u.ActionDef
}

func (u *UserDefinedAction) Plannable() bool {
	return true // User-defined actions are plannable
}

func (u *UserDefinedAction) IsUserDefined() bool {
	return true
}

// CreateUserDefinedActions converts user tools to UserDefinedAction instances
func CreateUserDefinedActions(userTools []ActionDefinition) []Action {
	var actions []Action
	for _, tool := range userTools {
			actions = append(actions, &UserDefinedAction{
				ActionDef: &tool,
			})
	}
	return actions
}

type Actions []Action

func (a Actions) ToTools() []openai.Tool {
	tools := []openai.Tool{}
	for _, action := range a {
		tools = append(tools, openai.Tool{
			Type:     openai.ToolTypeFunction,
			Function: action.Definition().ToFunctionDefinition(),
		})
	}
	return tools
}

func (a Actions) Find(name string) Action {
	for _, action := range a {
		if action.Definition().Name.Is(name) {
			return action
		}
	}
	return nil
}

type ActionState struct {
	ActionCurrentState
	ActionResult
}

type ActionCurrentState struct {
	Job       *Job
	Action    Action
	Params    ActionParams
	Reasoning string
}
