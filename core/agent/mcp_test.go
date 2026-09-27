package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/mudler/LocalAGI/core/types"
)

// newTestMCPServer builds an MCP server exposing a single "echo" tool.
func newTestMCPServer() *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{Name: "test-mcp", Version: "v1"}, nil)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "echo",
		Description: "Echo the input back.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args struct {
		Text string `json:"text"`
	}) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: args.Text}}}, nil, nil
	})
	return srv
}

// restartableMCPServer serves a stateful MCP endpoint whose handler can be
// swapped for a fresh one, which is what a real server restart looks like to a
// connected client: the session ID it holds is no longer known and every call
// comes back as "session not found".
type restartableMCPServer struct {
	handler atomic.Pointer[http.Handler]
	down    atomic.Bool
	http    *httptest.Server
}

func newRestartableMCPServer(t *testing.T) *restartableMCPServer {
	t.Helper()
	s := &restartableMCPServer{}
	s.restart()
	s.http = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.down.Load() {
			http.Error(w, "server is down", http.StatusServiceUnavailable)
			return
		}
		(*s.handler.Load()).ServeHTTP(w, r)
	}))
	t.Cleanup(s.http.Close)
	return s
}

func (s *restartableMCPServer) restart() {
	srv := newTestMCPServer()
	var h http.Handler = mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return srv }, nil,
	)
	s.handler.Store(&h)
}

func newTestAgent(t *testing.T, servers ...MCPServer) *Agent {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	a := &Agent{
		options: &options{mcpServers: servers},
		context: types.NewActionContext(ctx, cancel),
	}
	// Close the sessions before the test server: a live streamable session holds
	// a hanging GET that would block httptest.Server.Close.
	t.Cleanup(a.closeMCPServers)
	return a
}

// A server that restarts invalidates the session the agent holds. Before a
// turn, the agent must notice and re-dial it instead of carrying the dead
// session for the rest of its life.
func TestRefreshMCPSessionsReconnectsAfterServerRestart(t *testing.T) {
	server := newRestartableMCPServer(t)
	a := newTestAgent(t, MCPServer{URL: server.http.URL})

	if err := a.initMCPActions(); err != nil {
		t.Fatalf("initMCPActions: %v", err)
	}
	if got := len(a.mcpActionDefinitions); got != 1 {
		t.Fatalf("tools after init = %d, want 1", got)
	}
	first := a.liveMCPSessions()
	if len(first) != 1 {
		t.Fatalf("live sessions after init = %d, want 1", len(first))
	}

	server.restart()

	a.refreshMCPSessions()

	if got := len(a.mcpActionDefinitions); got != 1 {
		t.Fatalf("tools after restart = %d, want 1 (session was not re-dialed)", got)
	}
	second := a.liveMCPSessions()
	if len(second) != 1 {
		t.Fatalf("live sessions after restart = %d, want 1", len(second))
	}
	if second[0] == first[0] {
		t.Fatal("refreshMCPSessions kept the dead session instead of re-dialing")
	}
	if _, err := second[0].ListTools(context.Background(), nil); err != nil {
		t.Fatalf("re-dialed session is not usable: %v", err)
	}
}

// An MCP server that is unreachable must not stop the agent: it is recorded
// with no session, contributes no tools, and is picked up on a later turn once
// it answers again.
func TestUnreachableMCPServerIsSkippedThenRecovered(t *testing.T) {
	server := newRestartableMCPServer(t)
	server.down.Store(true)

	a := newTestAgent(t, MCPServer{URL: server.http.URL})

	if err := a.initMCPActions(); err != nil {
		t.Fatalf("initMCPActions returned an error for an unreachable server: %v", err)
	}
	if got := len(a.mcpActionDefinitions); got != 0 {
		t.Fatalf("tools from an unreachable server = %d, want 0", got)
	}
	if got := len(a.liveMCPSessions()); got != 0 {
		t.Fatalf("live sessions for an unreachable server = %d, want 0", got)
	}

	// The server comes back.
	server.down.Store(false)
	a.refreshMCPSessions()

	if got := len(a.mcpActionDefinitions); got != 1 {
		t.Fatalf("tools after the server came back = %d, want 1", got)
	}
	if got := len(a.liveMCPSessions()); got != 1 {
		t.Fatalf("live sessions after the server came back = %d, want 1", got)
	}
}
