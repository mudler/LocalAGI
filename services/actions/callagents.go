package actions

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/mudler/LocalAGI/core/state"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func trimList(list []string) []string {
	for i, v := range list {
		list[i] = strings.TrimSpace(v)
	}
	return list
}

func NewCallAgent(config map[string]string, agentName string, pool *state.AgentPoolInternalAPI) *CallAgentAction {
	whitelist := []string{}
	blacklist := []string{}
	if v, ok := config["whitelist"]; ok {
		if strings.Contains(v, ",") {
			whitelist = trimList(strings.Split(v, ","))
		} else {
			whitelist = []string{v}
		}
	}
	if v, ok := config["blacklist"]; ok {
		if strings.Contains(v, ",") {
			blacklist = trimList(strings.Split(v, ","))
		} else {
			blacklist = []string{v}
		}
	}
	return &CallAgentAction{
		pool:      pool,
		myName:    agentName,
		whitelist: whitelist,
		blacklist: blacklist,
	}
}

type CallAgentAction struct {
	pool      *state.AgentPoolInternalAPI
	myName    string
	whitelist []string
	blacklist []string
}

func (a *CallAgentAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
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

	metadata := make(map[string]interface{})

	for _, s := range resp.State {
		for k, v := range s.Metadata {
			if existingValue, ok := metadata[k]; ok {
				switch existingValue := existingValue.(type) {
				case []string:
					switch v := v.(type) {
					case []string:
						metadata[k] = append(existingValue, v...)
					case string:
						metadata[k] = append(existingValue, v)
					}
				case string:
					switch v := v.(type) {
					case []string:
						metadata[k] = append([]string{existingValue}, v...)
					case string:
						metadata[k] = []string{existingValue, v}
					}
				}
			} else {
				metadata[k] = v
			}
		}
	}

	return types.ActionResult{Result: resp.Response, Metadata: metadata}, nil
}

func (a *CallAgentAction) isAllowedToBeCalled(agentName string) bool {
	if agentName == a.myName {
		return false
	}

	if len(a.whitelist) > 0 && len(a.blacklist) > 0 {
		return slices.Contains(a.whitelist, agentName) && !slices.Contains(a.blacklist, agentName)
	}

	if len(a.whitelist) > 0 {
		return slices.Contains(a.whitelist, agentName)
	}

	if len(a.blacklist) > 0 {
		return !slices.Contains(a.blacklist, agentName)
	}
	return true
}

func (a *CallAgentAction) Definition() types.ActionDefinition {
	allAgents := a.pool.AllAgents()

	agents := []string{}

	for _, ag := range allAgents {
		if a.isAllowedToBeCalled(ag) {
			agents = append(agents, ag)
		}
	}

	description := "Use this tool to call another agent. Available agents and their roles are:"

	for _, agent := range agents {
		agentConfig := a.pool.GetConfig(agent)
		if agentConfig == nil {
			continue
		}
		description += fmt.Sprintf("\n\t- %s: %s", agent, agentConfig.Description)
	}

	return types.ActionDefinition{
		Name:        "call_agent",
		Description: description,
		Properties: map[string]jsonschema.Definition{
			"agent_name": {
				Type:        jsonschema.String,
				Description: "The name of the agent to call.",
				Enum:        agents,
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

func CallAgentConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "whitelist",
			Label:    "Whitelist",
			Type:     config.FieldTypeText,
			Required: false,
			HelpText: "Comma-separated list of agent names to call. If not specified, all agents are allowed.",
		},
		{
			Name:     "blacklist",
			Label:    "Blacklist",
			Type:     config.FieldTypeText,
			HelpText: "Comma-separated list of agent names to exclude from the call. If not specified, all agents are allowed.",
		},
	}
}
