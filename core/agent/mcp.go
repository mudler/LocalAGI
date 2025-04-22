package agent

import (
	"context"
	"encoding/json"

	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/http"
	stdioTransport "github.com/metoro-io/mcp-golang/transport/stdio"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/stdio"
	"github.com/mudler/LocalAGI/pkg/xlog"

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
	mcpClient       *mcp.Client
	inputSchema     ToolInputSchema
	toolName        string
	toolDescription string
}

func (a *mcpAction) Plannable() bool {
	return true
}

func (m *mcpAction) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	resp, err := m.mcpClient.CallTool(ctx, m.toolName, params)
	if err != nil {
		xlog.Error("Failed to call tool", "error", err.Error())
		return types.ActionResult{}, err
	}

	xlog.Debug("MCP response", "response", resp)

	textResult := ""
	for _, c := range resp.Content {
		switch c.Type {
		case mcp.ContentTypeText:
			textResult += c.TextContent.Text + "\n"
		case mcp.ContentTypeImage:
			xlog.Error("Image content not supported yet")
		case mcp.ContentTypeEmbeddedResource:
			xlog.Error("Embedded resource content not supported yet")
		}
	}

	return types.ActionResult{
		Result: textResult,
	}, nil
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

func (a *Agent) addTools(client *mcp.Client) (types.Actions, error) {

	var generatedActions types.Actions
	xlog.Debug("Initializing client")
	// Initialize the client
	response, e := client.Initialize(a.context)
	if e != nil {
		xlog.Error("Failed to initialize client", "error", e.Error())
		return nil, e
	}

	xlog.Debug("Client initialized: %v", response.Instructions)

	var cursor *string
	for {
		tools, err := client.ListTools(a.context, cursor)
		if err != nil {
			xlog.Error("Failed to list tools", "error", err.Error())
			return nil, err
		}

		for _, t := range tools.Tools {
			desc := ""
			if t.Description != nil {
				desc = *t.Description
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

		if tools.NextCursor == nil {
			break // No more pages
		}
		cursor = tools.NextCursor
	}

	return generatedActions, nil

}

func (a *Agent) initMCPActions() error {

	a.mcpActions = nil
	var err error

	generatedActions := types.Actions{}

	// MCP HTTP Servers
	for _, mcpServer := range a.options.mcpServers {
		transport := http.NewHTTPClientTransport("/mcp")
		transport.WithBaseURL(mcpServer.URL)
		if mcpServer.Token != "" {
			transport.WithHeader("Authorization", "Bearer "+mcpServer.Token)
		}

		// Create a new client
		client := mcp.NewClient(transport)
		xlog.Debug("Adding tools for MCP server", "server", mcpServer)
		generatedActions, err = a.addTools(client)
		if err != nil {
			xlog.Error("Failed to add tools for MCP server", "server", mcpServer, "error", err.Error())
		}
	}

	// MCP STDIO Servers
	a.closeMCPSTDIOServers() // Make sure we stop all previous servers if any is active
	for _, mcpStdioServer := range a.options.mcpStdioServers {
		client := stdio.NewClient(a.options.mcpBoxURL)
		p, err := client.CreateProcess(a.context,
			mcpStdioServer.Cmd,
			mcpStdioServer.Args,
			mcpStdioServer.Env,
			a.Character.Name)
		if err != nil {
			xlog.Error("Failed to create process", "error", err.Error())
			continue
		}
		read, writer, err := client.GetProcessIO(p.ID)
		if err != nil {
			xlog.Error("Failed to get process IO", "error", err.Error())
			continue
		}

		transport := stdioTransport.NewStdioServerTransportWithIO(read, writer)

		// Create a new client
		mcpClient := mcp.NewClient(transport)

		xlog.Debug("Adding tools for MCP server (stdio)", "server", mcpStdioServer)
		generatedActions, err = a.addTools(mcpClient)
		if err != nil {
			xlog.Error("Failed to add tools for MCP server", "server", mcpStdioServer, "error", err.Error())
		}
	}

	a.mcpActions = generatedActions

	return err
}

func (a *Agent) closeMCPSTDIOServers() {
	client := stdio.NewClient(a.options.mcpBoxURL)
	client.StopGroup(a.Character.Name)
}
