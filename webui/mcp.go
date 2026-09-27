package webui

import (
	"context"
	"fmt"
	"net/http"

	fiber "github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/state"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/services"
)

// mcpServerName identifies LocalAGI to MCP clients.
const mcpServerName = "LocalAGI"

// agentNameArgs is the input of every tool addressing a single agent.
type agentNameArgs struct {
	Name string `json:"name" jsonschema:"name of the agent"`
}

// statusResult is returned by tools that only report success.
type statusResult struct {
	Status string `json:"status"`
}

// agentSummary is one entry of list_agents.
type agentSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Model       string `json:"model" jsonschema:"model configured for this agent, empty when it uses the instance default"`
	Paused      bool   `json:"paused"`
	Running     bool   `json:"running" jsonschema:"whether the agent is loaded in the pool; a stopped instance still keeps its configuration"`
}

type listAgentsResult struct {
	Agents []agentSummary `json:"agents"`
}

// updateAgentArgs targets an agent by name and replaces its configuration.
type updateAgentArgs struct {
	Name   string            `json:"name" jsonschema:"name of the agent to update"`
	Config state.AgentConfig `json:"config" jsonschema:"the new configuration, replacing the current one entirely"`
}

type emptyArgs struct{}

// registerMCPRoutes mounts the MCP endpoint. It is registered after the API key
// middleware, so MCP clients authenticate with the same bearer token as the
// REST API.
//
// The handler runs stateless and answers with application/json rather than an
// event stream: every call is a self-contained request/response, which is what
// the agent management tools need and what survives the fasthttp adaptor.
func (a *App) registerMCPRoutes(pool *state.AgentPool, webapp *fiber.App) {
	srv := a.newMCPServer(pool)
	handler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return srv },
		&mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true},
	)
	webapp.All("/mcp", adaptor.HTTPHandler(handler))
}

// newMCPServer builds the MCP server exposing agent management over the
// Model Context Protocol. The tools mirror the agent REST API.
func (a *App) newMCPServer(pool *state.AgentPool) *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{Name: mcpServerName, Version: "v1"}, nil)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_agents",
		Description: "List the agents configured in LocalAGI, with their model and current state.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args emptyArgs) (*mcp.CallToolResult, listAgentsResult, error) {
		result := listAgentsResult{Agents: []agentSummary{}}
		for _, name := range pool.AllAgents() {
			summary := agentSummary{Name: name}
			if cfg := pool.GetConfig(name); cfg != nil {
				summary.Description = cfg.Description
				summary.Model = cfg.Model
			}
			if agent := pool.GetAgent(name); agent != nil {
				summary.Running = true
				summary.Paused = agent.Paused()
			}
			result.Agents = append(result.Agents, summary)
		}
		return nil, result, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_agent_config",
		Description: "Get the full configuration of an agent.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args agentNameArgs) (*mcp.CallToolResult, state.AgentConfig, error) {
		cfg := pool.GetConfig(args.Name)
		if cfg == nil {
			return nil, state.AgentConfig{}, errAgentNotFound(args.Name)
		}
		return nil, normalizeAgentConfig(*cfg), nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "create_agent",
		Description: "Create a new agent and start it. Only the name is required; every other field falls back to the instance default. Call get_agent_config_schema first to discover the available connectors, actions, dynamic prompts and filters.",
		InputSchema: agentConfigSchema("name"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args state.AgentConfig) (*mcp.CallToolResult, statusResult, error) {
		if args.Name == "" {
			return nil, statusResult{}, fmt.Errorf("name is required")
		}
		if err := pool.CreateAgent(args.Name, &args); err != nil {
			return nil, statusResult{}, err
		}
		return nil, statusResult{Status: "ok"}, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "update_agent_config",
		Description: "Replace the configuration of an existing agent and restart it. The configuration replaces the current one in full, so read it with get_agent_config first and send it back with your changes applied.",
		InputSchema: updateAgentSchema(),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args updateAgentArgs) (*mcp.CallToolResult, statusResult, error) {
		if pool.GetConfig(args.Name) == nil {
			return nil, statusResult{}, errAgentNotFound(args.Name)
		}
		if err := pool.RecreateAgent(args.Name, &args.Config); err != nil {
			return nil, statusResult{}, err
		}
		return nil, statusResult{Status: "ok"}, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "delete_agent",
		Description: "Delete an agent and remove it from the pool. This also discards its state and character files.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args agentNameArgs) (*mcp.CallToolResult, statusResult, error) {
		if pool.GetConfig(args.Name) == nil {
			return nil, statusResult{}, errAgentNotFound(args.Name)
		}
		if err := pool.Remove(args.Name); err != nil {
			return nil, statusResult{}, err
		}
		return nil, statusResult{Status: "ok"}, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "pause_agent",
		Description: "Pause a running agent. It keeps its configuration and stops processing jobs until it is started again.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args agentNameArgs) (*mcp.CallToolResult, statusResult, error) {
		agent := pool.GetAgent(args.Name)
		if agent == nil {
			return nil, statusResult{}, errAgentNotFound(args.Name)
		}
		agent.Pause()
		return nil, statusResult{Status: "ok"}, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "start_agent",
		Description: "Resume a paused agent.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args agentNameArgs) (*mcp.CallToolResult, statusResult, error) {
		agent := pool.GetAgent(args.Name)
		if agent == nil {
			return nil, statusResult{}, errAgentNotFound(args.Name)
		}
		agent.Resume()
		return nil, statusResult{Status: "ok"}, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_agent_config_schema",
		Description: "Describe the agent configuration fields, and the connectors, actions, dynamic prompts and filters available on this LocalAGI instance.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args emptyArgs) (*mcp.CallToolResult, state.AgentConfigMeta, error) {
		meta := state.NewAgentConfigMeta(
			services.ActionsConfigMeta(a.config.CustomActionsDir),
			services.ConnectorsConfigMeta(),
			services.DynamicPromptsConfigMeta(a.config.CustomActionsDir),
			services.FiltersConfigMeta(),
		)
		return nil, normalizeConfigMeta(meta), nil
	})

	return srv
}

// normalizeConfigMeta replaces nil slices with empty ones. The MCP SDK
// validates tool output against the schema reflected from the returned type,
// where a nil slice marshals to null and fails the "array" check.
func normalizeConfigMeta(meta state.AgentConfigMeta) state.AgentConfigMeta {
	meta.Fields = normalizeFields(meta.Fields)
	meta.MCPServers = normalizeFields(meta.MCPServers)
	meta.Filters = normalizeFieldGroups(meta.Filters)
	meta.Connectors = normalizeFieldGroups(meta.Connectors)
	meta.Actions = normalizeFieldGroups(meta.Actions)
	meta.DynamicPrompts = normalizeFieldGroups(meta.DynamicPrompts)
	return meta
}

func normalizeFields(fields []config.Field) []config.Field {
	if fields == nil {
		return []config.Field{}
	}
	return fields
}

func normalizeFieldGroups(groups []config.FieldGroup) []config.FieldGroup {
	out := make([]config.FieldGroup, 0, len(groups))
	for _, g := range groups {
		g.Fields = normalizeFields(g.Fields)
		out = append(out, g)
	}
	return out
}

// mustSchemaFor reflects the JSON schema of T, or panics. The schemas are built
// once at server construction, so a failure here is a programming error.
func mustSchemaFor[T any]() *jsonschema.Schema {
	schema, err := jsonschema.For[T](nil)
	if err != nil {
		panic(fmt.Sprintf("reflecting JSON schema: %v", err))
	}
	return schema
}

// agentConfigSchema is the reflected schema of state.AgentConfig with only the
// listed properties required. Reflection marks every field without omitempty as
// required, which would force a client to send the whole configuration; every
// field other than the name is in fact optional.
func agentConfigSchema(required ...string) *jsonschema.Schema {
	schema := mustSchemaFor[state.AgentConfig]()
	schema.Required = required
	return schema
}

// updateAgentSchema describes update_agent_config: the target agent name, plus
// a full replacement configuration whose fields are all optional.
func updateAgentSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"name": {
				Type:        "string",
				Description: "name of the agent to update",
			},
			"config": withDescription(
				agentConfigSchema(),
				"the new configuration, replacing the current one entirely",
			),
		},
		Required: []string{"name", "config"},
	}
}

func withDescription(schema *jsonschema.Schema, description string) *jsonschema.Schema {
	schema.Description = description
	return schema
}

// normalizeAgentConfig replaces nil slices with empty ones so the configuration
// validates against its own reflected schema, where a nil slice marshals to
// null instead of an array.
func normalizeAgentConfig(cfg state.AgentConfig) state.AgentConfig {
	if cfg.Connector == nil {
		cfg.Connector = []state.ConnectorConfig{}
	}
	if cfg.Actions == nil {
		cfg.Actions = []state.ActionsConfig{}
	}
	if cfg.DynamicPrompts == nil {
		cfg.DynamicPrompts = []state.DynamicPromptsConfig{}
	}
	if cfg.Filters == nil {
		cfg.Filters = []state.FiltersConfig{}
	}
	if cfg.MCPServers == nil {
		cfg.MCPServers = []agent.MCPServer{}
	}
	if cfg.MCPSTDIOServers == nil {
		cfg.MCPSTDIOServers = []agent.MCPSTDIOServer{}
	}
	for i, srv := range cfg.MCPSTDIOServers {
		if srv.Args == nil {
			cfg.MCPSTDIOServers[i].Args = []string{}
		}
		if srv.Env == nil {
			cfg.MCPSTDIOServers[i].Env = []string{}
		}
	}
	return cfg
}

// errAgentNotFound is returned to the client as a tool error, so the model can
// see it and correct itself.
func errAgentNotFound(name string) error {
	return fmt.Errorf("agent %q not found", name)
}
