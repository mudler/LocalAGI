package agent

import (
	"os"
	"os/exec"
	"time"

	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/mudler/LocalAGI/pkg/xlog"
)

type MCPServer struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

type MCPSTDIOServer struct {
	Args []string `json:"args"`
	Env  []string `json:"env"`
	Cmd  string   `json:"cmd"`
}

type ToolInputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
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

	a.closeMCPSTDIOServers() // Make sure we stop all previous servers if any is active

	var err error

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
	}

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
	}

	return err
}

func (a *Agent) closeMCPSTDIOServers() {
	for _, s := range a.mcpSessions {
		s.Close()
	}
}
