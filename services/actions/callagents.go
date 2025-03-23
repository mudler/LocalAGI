package actions

import (
	"context"
	"fmt"

	"github.com/mudler/LocalAgent/core/state"
	"github.com/mudler/LocalAgent/core/types"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func NewCallAgent(config map[string]string, pool *state.AgentPool) *CallAgentAction {
	return &CallAgentAction{
		pool: pool,
	}
}

type CallAgentAction struct {
	pool *state.AgentPool
}

func (a *CallAgentAction) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		AgentName string `json:"agent_name"`
		Message   string `json:"message"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return types.ActionResult{}, err
	}

	ag := a.pool.GetAgent(result.AgentName)
	if ag == nil {
		return types.ActionResult{}, fmt.Errorf("agent '%s' not found", result.AgentName)
	}

	resp := ag.Ask(
		types.WithConversationHistory(
			[]openai.ChatCompletionMessage{
				{
					Role:    "user",
					Content: result.Message,
				},
			},
		),
	)
	if resp.Error != nil {
		return types.ActionResult{}, err
	}

	return types.ActionResult{Result: resp.Response}, nil
}

func (a *CallAgentAction) Definition() types.ActionDefinition {
	allAgents := a.pool.AllAgents()

	description := "Use this tool to call another agent. Available agents and their roles are:"

	for _, agent := range allAgents {
		agentConfig := a.pool.GetConfig(agent)
		if agentConfig == nil {
			continue
		}
		description += fmt.Sprintf("\n- %s: %s", agent, agentConfig.Description)
	}

	return types.ActionDefinition{
		Name:        "call_agent",
		Description: description,
		Properties: map[string]jsonschema.Definition{
			"agent_name": {
				Type:        jsonschema.String,
				Description: "The name of the agent to call.",
				Enum:        allAgents,
			},
			"message": {
				Type:        jsonschema.String,
				Description: "The message to send to the agent.",
			},
		},
		Required: []string{"agent_name", "message"},
	}
}

func (a *CallAgentAction) Plannable() bool {
	return true
}
