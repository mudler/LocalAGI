package action

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
	if ac.cancelFunc == nil {
		ac.cancelFunc()
	}
}

func NewContext(ctx context.Context, cancel context.CancelFunc) *ActionContext {
	return &ActionContext{
		Context:    ctx,
		cancelFunc: cancel,
	}
}

type ActionParams map[string]interface{}

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

func (a ActionDefinition) ToFunctionDefinition() openai.FunctionDefinition {
	return openai.FunctionDefinition{
		Name:        a.Name.String(),
		Description: a.Description,
		Parameters: jsonschema.Definition{
			Type:       jsonschema.Object,
			Properties: a.Properties,
			Required:   a.Required,
		},
	}
}
