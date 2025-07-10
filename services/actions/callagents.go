package actions

import (
	"context"
	"fmt"
	"strings"

	"github.com/mudler/LocalAGI/core/state"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/db"
	models "github.com/mudler/LocalAGI/dbmodels"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func trimList(list []string) []string {
	var result []string
	for _, v := range list {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func NewCallAgent(config map[string]string, agentName string, pool *state.AgentPoolInternalAPI) *CallAgentAction {
	whitelist := []string{}
	blacklist := []string{}
	if v, ok := config["whitelist"]; ok && strings.TrimSpace(v) != "" {
		if strings.Contains(v, ",") {
			whitelist = trimList(strings.Split(v, ","))
		} else {
			whitelist = []string{strings.TrimSpace(v)}
		}
	}
	if v, ok := config["blacklist"]; ok && strings.TrimSpace(v) != "" {
		if strings.Contains(v, ",") {
			blacklist = trimList(strings.Split(v, ","))
		} else {
			blacklist = []string{strings.TrimSpace(v)}
		}
	}

	// Convert agent ID to human-readable name
	myName := agentName // fallback to ID if DB lookup fails
	userID := pool.GetUserID()
	var myAgent models.Agent
	err := db.DB.Where("ID = ? AND UserId = ? AND archive = false", agentName, userID).First(&myAgent).Error
	if err == nil {
		myName = myAgent.Name
	}

	return &CallAgentAction{
		pool:      pool,
		myName:    myName,
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

	// Check if the agent is allowed to be called (whitelist/blacklist)
	if !a.isAllowedToBeCalled(result.AgentName) {
		return types.ActionResult{}, fmt.Errorf("agent '%s' is not allowed to be called (blocked by whitelist/blacklist)", result.AgentName)
	}

	// Query database to find agent by name and user ID
	userID := a.pool.GetUserID()
	var dbAgent models.Agent
	err = db.DB.Where("Name = ? AND UserId = ? AND archive = false", result.AgentName, userID).First(&dbAgent).Error
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("agent '%s' not found", result.AgentName)
	}

	// Use the agent ID from database to get the agent from pool
	agentID := dbAgent.ID.String()
	ag := a.pool.GetAgent(agentID)
	if ag == nil {
		return types.ActionResult{}, fmt.Errorf("agent '%s' (ID: %s) not found in pool", result.AgentName, agentID)
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

// containsCaseInsensitive checks if a slice contains a string (case-insensitive)
func containsCaseInsensitive(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

func (a *CallAgentAction) isAllowedToBeCalled(agentName string) bool {
	// Prevent self-calling (case-insensitive)
	if strings.EqualFold(agentName, a.myName) {
		return false
	}

	fmt.Printf("isAllowedToBeCalled: %v %v %v %v", agentName, a.myName, a.whitelist, a.blacklist)

	if len(a.whitelist) > 0 && len(a.blacklist) > 0 {
		return containsCaseInsensitive(a.whitelist, agentName) && !containsCaseInsensitive(a.blacklist, agentName)
	}

	if len(a.whitelist) > 0 {
		return containsCaseInsensitive(a.whitelist, agentName)
	}

	if len(a.blacklist) > 0 {
		return !containsCaseInsensitive(a.blacklist, agentName)
	}
	return true
}

func (a *CallAgentAction) Definition() types.ActionDefinition {
	// Query database to get all agents for this user
	userID := a.pool.GetUserID()
	var dbAgents []models.Agent
	err := db.DB.Where("UserId = ? AND archive = false", userID).Find(&dbAgents).Error
	if err != nil {
		// If database query fails, return empty agent list
		return types.ActionDefinition{
			Name:        "call_agent",
			Description: "Use this tool to call another agent. No agents available.",
			Properties: map[string]jsonschema.Definition{
				"agent_name": {
					Type:        jsonschema.String,
					Description: "The name of the agent to call.",
					Enum:        []string{},
				},
				"message": {
					Type:        jsonschema.String,
					Description: "The message to send to the agent.",
				},
			},
			Required: []string{"agent_name", "message"},
		}
	}

	agents := []string{}
	description := "Use this tool to call another agent. Available agents and their roles are:"

	for _, dbAgent := range dbAgents {
		// Check if this agent is allowed to be called
		if a.isAllowedToBeCalled(dbAgent.Name) {
			agents = append(agents, dbAgent.Name)

			// Try to get agent config for description
			agentConfig := a.pool.GetConfig(dbAgent.ID.String())
			if agentConfig != nil && agentConfig.Description != "" {
				description += fmt.Sprintf("\n\t- %s: %s", dbAgent.Name, agentConfig.Description)
			} else {
				description += fmt.Sprintf("\n\t- %s", dbAgent.Name)
			}
		}
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
