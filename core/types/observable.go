package types

import (
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/sashabaranov/go-openai"
)

type Creation struct {
	ChatCompletionRequest *openai.ChatCompletionRequest `json:"chat_completion_request,omitempty"`
	FunctionDefinition    *openai.FunctionDefinition    `json:"function_definition,omitempty"`
	FunctionParams        ActionParams                  `json:"function_params,omitempty"`
}

type Progress struct {
	Error                  string                         `json:"error,omitempty"`
	ChatCompletionResponse *openai.ChatCompletionResponse `json:"chat_completion_response,omitempty"`
	ActionResult           string                         `json:"action_result,omitempty"`
	AgentState             *AgentInternalState            `json:"agent_state"`
}

type Completion struct {
	Error                  string                         `json:"error,omitempty"`
	ChatCompletionResponse *openai.ChatCompletionResponse `json:"chat_completion_response,omitempty"`
	Conversation           []openai.ChatCompletionMessage `json:"conversation,omitempty"`
	ActionResult           string                         `json:"action_result,omitempty"`
	AgentState             *AgentInternalState            `json:"agent_state"`
}

type Observable struct {
	ID       int32  `json:"id"`
	ParentID int32  `json:"parent_id,omitempty"`
	Agent    string `json:"agent"`
	Name     string `json:"name"`
	Icon     string `json:"icon"`

	Creation   *Creation   `json:"creation,omitempty"`
	Progress   []Progress  `json:"progress,omitempty"`
	Completion *Completion `json:"completion,omitempty"`
}

func (o *Observable) AddProgress(p Progress) {
	if o.Progress == nil {
		o.Progress = make([]Progress, 0)
	}
	o.Progress = append(o.Progress, p)
}

func (o *Observable) MakeLastProgressCompletion() {
	if len(o.Progress) == 0 {
		xlog.Error("Observable completed without any progress", "id", o.ID, "name", o.Name)
		return
	}
	p := o.Progress[len(o.Progress)-1]
	o.Progress = o.Progress[:len(o.Progress)-1]
	o.Completion = &Completion{
		Error:                  p.Error,
		ChatCompletionResponse: p.ChatCompletionResponse,
		ActionResult:           p.ActionResult,
		AgentState:             p.AgentState,
	}
}
