package agent

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"time"

	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/rs/zerolog/log"

	"github.com/sashabaranov/go-openai/jsonschema"
)

var _ types.Action = &mcpAction{}

type MCPServer struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

type MCPSTDIOServer struct {
	Args []string `json:"args"`
	Env  []string `json:"env"`
	Cmd  string   `json:"cmd"`
}

type mcpAction struct {
	mcpClient       *mcp.ClientSession
	inputSchema     ToolInputSchema
	toolName        string
	toolDescription string
}

func (a *mcpAction) Plannable() bool {
	return true
}

func (m *mcpAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	p := &mcp.CallToolParams{
		Name:      m.toolName,
		Arguments: params,
	}
	resp, err := m.mcpClient.CallTool(ctx, p)
	if err != nil {
		xlog.Error("Failed to call tool", "error", err.Error())
		return types.ActionResult{}, err
	}

	xlog.Debug("MCP response", "response", resp)

	if resp.IsError {
		log.Error().Msgf("tool failed")
		return types.ActionResult{}, errors.New("tool failed")
	}

	result := ""
	for _, c := range resp.Content {
		result += c.(*mcp.TextContent).Text
	}

	return types.ActionResult{Result: result}, nil
}

func (m *mcpAction) Definition() types.ActionDefinition {
	props := map[string]jsonschema.Definition{}
	dat, err := json.Marshal(m.inputSchema.Properties)
	if err != nil {
		xlog.Error("Failed to marshal input schema", "error", err.Error())
	}
	json.Unmarshal(dat, &props)

	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(m.toolName),
		Description: m.toolDescription,
		Required:    m.inputSchema.Required,
		//Properties:  ,
		Properties: props,
	}
}

type ToolInputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

func (a *Agent) addTools(client *mcp.ClientSession) (types.Actions, error) {

	var generatedActions types.Actions

	tools, err := client.ListTools(a.context, nil)
	if err != nil {
		xlog.Error("Failed to list tools", "error", err.Error())
		return nil, err
	}

	for _, t := range tools.Tools {
		desc := ""
		if t.Description != "" {
			desc = t.Description
		}

		xlog.Debug("Tool", "name", t.Name, "description", desc)

		dat, err := json.Marshal(t.InputSchema)
		if err != nil {
			xlog.Error("Failed to marshal input schema", "error", err.Error())
		}

		xlog.Debug("Input schema", "tool", t.Name, "schema", string(dat))

		// XXX: This is a wild guess, to verify (data types might be incompatible)
		var inputSchema ToolInputSchema
		err = json.Unmarshal(dat, &inputSchema)
		if err != nil {
			xlog.Error("Failed to unmarshal input schema", "error", err.Error())
		}

		// Create a new action with Client + tool
		generatedActions = append(generatedActions, &mcpAction{
			mcpClient:       client,
			toolName:        t.Name,
			inputSchema:     inputSchema,
			toolDescription: desc,
		})
	}

	return generatedActions, nil

}

// bearerTokenRoundTripper is a custom roundtripper that injects a bearer token
// into HTTP requests
type bearerTokenRoundTripper struct {
	token string
	base  http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface
func (rt *bearerTokenRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.token != "" {
		req.Header.Set("Authorization", "Bearer "+rt.token)
	}
	return rt.base.RoundTrip(req)
}

// newBearerTokenRoundTripper creates a new roundtripper that injects the given token
func newBearerTokenRoundTripper(token string, base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &bearerTokenRoundTripper{
		token: token,
		base:  base,
	}
}

func (a *Agent) initMCPActions() error {

	a.mcpActions = nil
	var err error

	generatedActions := types.Actions{}
	client := mcp.NewClient(&mcp.Implementation{Name: "LocalAI", Version: "v1.0.0"}, nil)

	// Connect to a server over stdin/stdout.

	// MCP HTTP Servers
	for _, mcpServer := range a.options.mcpServers {
		// Create HTTP client with custom roundtripper for bearer token injection
		httpclient := &http.Client{
			Timeout:   360 * time.Second,
			Transport: newBearerTokenRoundTripper(mcpServer.Token, http.DefaultTransport),
		}

		transport := &mcp.SSEClientTransport{HTTPClient: httpclient, Endpoint: mcpServer.URL}

		// Create a new client
		session, err := client.Connect(a.context, transport, nil)
		if err != nil {
			xlog.Error("Failed to connect to MCP server", "server", mcpServer, "error", err.Error())
			continue
		}
		a.mcpSessions = append(a.mcpSessions, session)

		xlog.Debug("Adding tools for MCP server", "server", mcpServer)
		actions, err := a.addTools(session)
		if err != nil {
			xlog.Error("Failed to add tools for MCP server", "server", mcpServer, "error", err.Error())
		}
		generatedActions = append(generatedActions, actions...)
	}

	// MCP STDIO Servers

	a.closeMCPSTDIOServers() // Make sure we stop all previous servers if any is active

	if a.options.mcpPrepareScript != "" {
		xlog.Debug("Preparing MCP", "script", a.options.mcpPrepareScript)

		prepareCmd := exec.Command("/bin/bash", "-c", a.options.mcpPrepareScript)
		output, err := prepareCmd.CombinedOutput()
		if err != nil {
			xlog.Error("Failed with error: '%s' - %s", err.Error(), output)
		}
		xlog.Debug("Prepared MCP: \n%s", output)
	}

	for _, mcpStdioServer := range a.options.mcpStdioServers {
		command := exec.Command(mcpStdioServer.Cmd, mcpStdioServer.Args...)
		command.Env = os.Environ()
		command.Env = append(command.Env, mcpStdioServer.Env...)

		// Create a new client
		session, err := client.Connect(a.context, &mcp.CommandTransport{
			Command: command}, nil)
		if err != nil {
			xlog.Error("Failed to connect to MCP server", "server", mcpStdioServer, "error", err.Error())
			continue
		}
		a.mcpSessions = append(a.mcpSessions, session)

		xlog.Debug("Adding tools for MCP server (stdio)", "server", mcpStdioServer)
		actions, err := a.addTools(session)
		if err != nil {
			xlog.Error("Failed to add tools for MCP server", "server", mcpStdioServer, "error", err.Error())
		}
		generatedActions = append(generatedActions, actions...)
	}

	a.mcpActions = generatedActions

	return err
}

func (a *Agent) closeMCPSTDIOServers() {
	for _, s := range a.mcpSessions {
		s.Close()
	}
}
