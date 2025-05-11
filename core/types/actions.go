package types

import (
	"context"
	"encoding/json"

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
