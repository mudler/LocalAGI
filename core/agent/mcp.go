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

// mcpPingTimeout bounds the health check run on every MCP session before a
// turn. It is deliberately short: a session that cannot answer a ping promptly
// is of no use to the turn anyway, and the turn proceeds without it.
const mcpPingTimeout = 5 * time.Second

// mcpSession pairs a live MCP client session with what is needed to bring it
// back after the remote server drops it, plus the tool wrappers it contributed
// to the agent's action list.
type mcpSession struct {
	session *mcp.ClientSession
	// describe identifies the server in log lines.
	describe string
	// dial re-establishes the session. It is nil for sessions the agent does
	// not own (extraMCPSessions, such as the in-process skills server): those
	// are dropped when they die, never re-dialed here and never closed here.
	dial func() (*mcp.ClientSession, error)
	// actions are the tool wrappers listed off this session.
	actions types.Actions
}

// dialMCP connects to an MCP server, preferring the streamable HTTP transport
// and falling back to SSE, which is what older servers speak.
func dialMCP(ctx context.Context, client *mcp.Client, server MCPServer) (*mcp.ClientSession, error) {
	httpclient := &http.Client{
		Timeout:   360 * time.Second,
		Transport: newBearerTokenRoundTripper(server.Token, http.DefaultTransport),
	}

	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{HTTPClient: httpclient, Endpoint: server.URL}, nil)
	if err == nil {
		return session, nil
	}
	xlog.Debug("Failed to connect to MCP server via StreamableClientTransport, trying SSE", "server", server.URL, "error", err.Error())

	return client.Connect(ctx, &mcp.SSEClientTransport{HTTPClient: httpclient, Endpoint: server.URL}, nil)
}

// dialMCPStdio starts an MCP server as a child process and connects to it over
// its stdio pipes.
func dialMCPStdio(ctx context.Context, client *mcp.Client, server MCPSTDIOServer) (*mcp.ClientSession, error) {
	command := exec.Command(server.Cmd, server.Args...)
	command.Env = append(os.Environ(), server.Env...)
	return client.Connect(ctx, &mcp.CommandTransport{Command: command}, nil)
}

func (a *Agent) initMCPActions() error {
	a.closeMCPServers() // Make sure we stop all previous servers if any is active

	a.mcpMutex.Lock()
	defer a.mcpMutex.Unlock()

	client := mcp.NewClient(&mcp.Implementation{Name: "LocalAI", Version: "v1.0.0"}, nil)

	// MCP HTTP Servers
	for _, mcpServer := range a.options.mcpServers {
		a.addMCPSession(mcpServer.URL, func() (*mcp.ClientSession, error) {
			return dialMCP(a.context.Context, client, mcpServer)
		})
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
		describe := mcpStdioServer.Name
		if describe == "" {
			describe = mcpStdioServer.Cmd
		}
		a.addMCPSession(describe, func() (*mcp.ClientSession, error) {
			return dialMCPStdio(a.context.Context, client, mcpStdioServer)
		})
	}

	// Pre-connected MCP sessions (e.g. in-process skills server). The agent does
	// not own these, so they get no dial func: they are never re-dialed or closed
	// here.
	for _, session := range a.options.extraMCPSessions {
		entry := &mcpSession{session: session, describe: "in-process MCP session"}
		a.mcpSessions = append(a.mcpSessions, entry)
		actions, err := a.addTools(session)
		if err != nil {
			xlog.Warn("Failed to add tools for extra MCP session", "error", err.Error())
			continue
		}
		entry.actions = actions
	}

	a.rebuildMCPActionDefinitions()

	return nil
}

// addMCPSession dials a server and records it, along with the closure that can
// dial it again. A server that is down at startup is still recorded, so a later
// turn can pick it up once it comes back.
func (a *Agent) addMCPSession(describe string, dial func() (*mcp.ClientSession, error)) {
	entry := &mcpSession{describe: describe, dial: dial}
	a.mcpSessions = append(a.mcpSessions, entry)

	session, err := dial()
	if err != nil {
		xlog.Warn("Failed to connect to MCP server, continuing without its tools", "server", describe, "error", err.Error())
		return
	}

	xlog.Debug("Adding tools for MCP server", "server", describe)
	actions, err := a.addTools(session)
	if err != nil {
		xlog.Warn("Failed to list tools for MCP server, continuing without them", "server", describe, "error", err.Error())
		session.Close()
		return
	}

	entry.session = session
	entry.actions = actions
}

// liveMCPSessions returns the client sessions to hand to cogito for this turn.
func (a *Agent) liveMCPSessions() []*mcp.ClientSession {
	a.mcpMutex.Lock()
	defer a.mcpMutex.Unlock()

	sessions := make([]*mcp.ClientSession, 0, len(a.mcpSessions))
	for _, s := range a.mcpSessions {
		if s.session != nil {
			sessions = append(sessions, s.session)
		}
	}
	return sessions
}

// refreshMCPSessions pings every MCP session and re-dials the ones that no
// longer answer. Sessions are opened once when the agent is built, but a remote
// server that restarts invalidates its session ID ("session not found"), and
// nothing would otherwise notice: the agent would hold a dead session for the
// rest of its life and silently lose those tools.
//
// A server that stays unreachable is a warning, never a turn failure. The agent
// runs with whatever tools it does have and picks the rest back up on a later
// turn, once the server answers again.
func (a *Agent) refreshMCPSessions() {
	a.mcpMutex.Lock()
	defer a.mcpMutex.Unlock()

	changed := false
	for _, s := range a.mcpSessions {
		if s.session != nil && a.pingMCPSession(s.session) == nil {
			continue
		}

		if s.session != nil {
			xlog.Warn("MCP session is no longer responding, reconnecting", "server", s.describe)
			s.session.Close()
			s.session = nil
			s.actions = nil
			changed = true
		}

		if s.dial == nil {
			// Not ours to re-open.
			xlog.Warn("MCP session cannot be re-dialed, skipping its tools", "server", s.describe)
			continue
		}

		session, err := s.dial()
		if err != nil {
			xlog.Warn("Failed to reconnect to MCP server, continuing without its tools", "server", s.describe, "error", err.Error())
			continue
		}

		actions, err := a.addTools(session)
		if err != nil {
			xlog.Warn("Reconnected to MCP server but failed to list its tools, continuing without them", "server", s.describe, "error", err.Error())
			session.Close()
			continue
		}

		xlog.Info("Reconnected to MCP server", "server", s.describe, "tools", len(actions))
		s.session = session
		s.actions = actions
		changed = true
	}

	if changed {
		a.rebuildMCPActionDefinitions()
	}
}

// pingMCPSession reports whether the session still answers, under a short
// deadline so a hung server cannot stall the turn.
func (a *Agent) pingMCPSession(session *mcp.ClientSession) error {
	ctx, cancel := context.WithTimeout(a.context.Context, mcpPingTimeout)
	defer cancel()
	return session.Ping(ctx, nil)
}

// rebuildMCPActionDefinitions flattens the per-session tool wrappers into the
// action list the agent exposes. Callers must hold mcpMutex.
func (a *Agent) rebuildMCPActionDefinitions() {
	var all types.Actions
	for _, s := range a.mcpSessions {
		all = append(all, s.actions...)
	}
	a.mcpActionDefinitions = all
}

// closeMCPServers closes every session the agent owns and forgets all of them.
// Sessions it does not own (extraMCPSessions) are left open; initMCPActions
// re-adds them from the options.
func (a *Agent) closeMCPServers() {
	a.mcpMutex.Lock()
	defer a.mcpMutex.Unlock()

	for _, s := range a.mcpSessions {
		if s.dial != nil && s.session != nil {
			s.session.Close()
		}
	}
	a.mcpSessions = nil
	a.mcpActionDefinitions = nil
}
