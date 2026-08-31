package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/xlog"

	"github.com/sashabaranov/go-openai/jsonschema"
)

var _ types.Action = &mcpWrapperAction{}

type MCPServer struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

type MCPSTDIOServer struct {
	Name string   `json:"name,omitempty"`
	Args []string `json:"args"`
	Env  []string `json:"env"`
	Cmd  string   `json:"cmd"`
}

type mcpWrapperAction struct {
	mcpClient       *mcp.ClientSession
	inputSchema     ToolInputSchema
	toolName        string
	toolDescription string
}

func (m *mcpWrapperAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	// We don't call the method here, it is used by cogito.
	// We will just use these to have a list of actions that MCP server provides for resolving internal states
	return types.ActionResult{Result: "MCP action called"}, fmt.Errorf("not implemented")
}

func (m *mcpWrapperAction) Definition() types.ActionDefinition {
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
		generatedActions = append(generatedActions, &mcpWrapperAction{
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
	a.closeMCPServers() // Make sure we stop all previous servers if any is active

	var err error

	client := a.mcpClient

	// MCP HTTP Servers
	for i := range a.options.mcpServers {
		mcpServer := &a.options.mcpServers[i]
		err := a.startStreamableClientTransport(mcpServer, client)
		if err != nil {
			xlog.Error("Failed to start MCP session", "server", mcpServer, "error", err.Error())
		}
	}

	// MCP STDIO Servers
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

		xlog.Debug("Adding tools for MCP server (stdio)", "server", mcpStdioServer)
		actions, err := a.addTools(session)
		if err != nil {
			session.Close()
			xlog.Error("Failed to add tools for MCP server", "server", mcpStdioServer, "error", err.Error())
			continue
		}
		a.mcpSessions = append(a.mcpSessions, session)
		a.mcpSessionActions[session] = actions
	}

	for _, session := range a.options.extraMCPSessions {
		actions, err := a.addTools(session)
		if err != nil {
			xlog.Error("Failed to add tools for extra MCP session", "error", err.Error())
			continue
		}
		// Pre-connected sessions are not added to mcpSessions because Agent doesn't manage their lifecycle - but we still want to add their actions
		a.mcpSessionActions[session] = actions
	}

	return err
}

func (a *Agent) closeMCPServers() {
	for _, session := range a.mcpSessions {
		session.Close()
	}

	clear(a.mcpSessionActions)
	clear(a.mcpSessions)
	clear(a.mcpServerSessions)
}

func (a *Agent) startStreamableClientTransport(mcpServer *MCPServer, client *mcp.Client) error {
	// Create HTTP client with custom roundtripper for bearer token injection
	httpclient := &http.Client{
		Timeout:   360 * time.Second,
		Transport: newBearerTokenRoundTripper(mcpServer.Token, http.DefaultTransport),
	}

	xlog.Debug("Connecting to MCP server", "server", mcpServer)
	streamableTransport := &mcp.StreamableClientTransport{HTTPClient: httpclient, Endpoint: mcpServer.URL}
	session, err := client.Connect(a.context, streamableTransport, nil)
	if err != nil {
		xlog.Error("Failed to connect to MCP server", "server", mcpServer.URL, "error", err.Error())
		return fmt.Errorf("Failed to connect to MCP server: %w", err)
	}

	xlog.Debug("Adding tools for MCP server", "server", mcpServer)
	actions, err := a.addTools(session)
	if err != nil {
		session.Close()
		xlog.Error("Failed to add tools for MCP server", "server", mcpServer, "error", err.Error())
		return fmt.Errorf("Failed to add tools for MCP server: %w", err)
	}

	a.mcpSessions = append(a.mcpSessions, session)
	a.mcpSessionActions[session] = actions
	a.mcpServerSessions[mcpServer] = session
	return nil
}

func (a *Agent) closeStreamableClientTransport(mcpServer *MCPServer) {
	if session, ok := a.mcpServerSessions[mcpServer]; ok {
		xlog.Debug("Closing MCP server session", "server", mcpServer.URL)
		session.Close()
		delete(a.mcpServerSessions, mcpServer)
		delete(a.mcpSessionActions, session)

		var newSessions []*mcp.ClientSession
		for _, s := range a.mcpSessions {
			if s != session {
				newSessions = append(newSessions, s)
			}
		}
		a.mcpSessions = newSessions
	} else {
		xlog.Warn("No session found for MCP server during close", "server", mcpServer.URL)
	}
}

func (a *Agent) mcpStreamableClientHealthCheck() {
	for i := range a.options.mcpServers {
		mcpServer := &a.options.mcpServers[i]
		if session, ok := a.mcpServerSessions[mcpServer]; ok {
			err := session.Ping(a.context, &mcp.PingParams{})
			if err != nil {
				xlog.Warn("Error pinging MCP server, will reconnect", "server", mcpServer.URL, "error", err)
				a.closeStreamableClientTransport(mcpServer)
				err := a.startStreamableClientTransport(mcpServer, a.mcpClient)
				if err != nil {
					xlog.Error("Failed to reconnect MCP server", "server", mcpServer, "error", err.Error())
				}
			}
		} else {
			xlog.Warn("No session found for MCP server during health check, will try to connect", "server", mcpServer.URL)
			err := a.startStreamableClientTransport(mcpServer, a.mcpClient)
			if err != nil {
				xlog.Error("Failed to connect MCP server", "server", mcpServer, "error", err.Error())
			}
		}
	}
}
