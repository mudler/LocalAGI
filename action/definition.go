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
	ac.cancelFunc()
}

func NewContext(ctx context.Context, cancel context.CancelFunc) *ActionContext {
	return &ActionContext{
		Context:    ctx,
		cancelFunc: cancel,
	}
}

type ActionParams map[string]string

func (ap ActionParams) Read(s string) error {
	err := json.Unmarshal([]byte(s), &ap)
	return err
}

func (ap ActionParams) String() string {
	b, _ := json.Marshal(ap)
	return string(b)
}

//type ActionDefinition openai.FunctionDefinition

type ActionDefinition struct {
	Properties  map[string]jsonschema.Definition
	Required    []string
	Name        string
	Description string
}

func (a ActionDefinition) ToFunctionDefinition() openai.FunctionDefinition {
	return openai.FunctionDefinition{
		Name:        a.Name,
		Description: a.Description,
		Parameters: jsonschema.Definition{
			Type:       jsonschema.Object,
			Properties: a.Properties,
			Required:   a.Required,
		},
	}
}
