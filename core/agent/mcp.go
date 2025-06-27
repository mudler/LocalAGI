package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/sashabaranov/go-openai/jsonschema"
)

var _ types.Action = &mcpAction{}

type MCPServer struct {
	URL string `json:"url"`
}

type mcpAction struct {
	mcpClient       *client.Client
	inputSchema     ToolInputSchema
	toolName        string
	toolDescription string
}

func (a *mcpAction) Plannable() bool {
	return true
}

func (m *mcpAction) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	// Convert params to map[string]interface{} for CallTool
	args := make(map[string]interface{})
	for k, v := range params {
		args[k] = v
	}

	// Use proper CallTool request format with Params structure
	req := mcp.CallToolRequest{
		Request: mcp.Request{
			Method: "tools/call",
		},
		Params: mcp.CallToolParams{
			Name:      m.toolName,
			Arguments: args,
		},
	}

	resp, err := m.mcpClient.CallTool(ctx, req)
	if err != nil {
		xlog.Error("Failed to call tool", "error", err.Error())
		return types.ActionResult{}, err
	}

	xlog.Debug("MCP response", "response", resp)

	textResult := ""
	for _, content := range resp.Content {
		// Handle different content types based on the mcp-go API
		if textContent, ok := mcp.AsTextContent(content); ok {
			textResult += textContent.Text + "\n"
		} else if imageContent, ok := mcp.AsImageContent(content); ok {
			xlog.Debug("Image content received", "mimeType", imageContent.MIMEType)
			textResult += "[Image content received]\n"
		} else if audioContent, ok := mcp.AsAudioContent(content); ok {
			xlog.Debug("Audio content received", "mimeType", audioContent.MIMEType)
			textResult += "[Audio content received]\n"
		} else if embeddedResource, ok := mcp.AsEmbeddedResource(content); ok {
			xlog.Debug("Embedded resource content received", "resource", embeddedResource.Resource)
			textResult += "[Embedded resource content received]\n"
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
		Properties:  props,
	}
}

type ToolInputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

func (a *Agent) initMCPActions() error {
	a.mcpActions = nil
	var err error

	generatedActions := types.Actions{}

	for _, mcpServer := range a.options.mcpServers {
		// Create a new client using the appropriate transport based on URL
		var mcpClient *client.Client
		var e error
		var isSSE bool

		if strings.Contains(mcpServer.URL, "/sse") {
			// Use SSE client for URLs containing "/sse"
			mcpClient, e = client.NewSSEMCPClient(mcpServer.URL)
			isSSE = true
		} else {
			// Use streamable HTTP client for other URLs
			mcpClient, e = client.NewStreamableHttpClient(mcpServer.URL)
			isSSE = false
		}

		if e != nil {
			xlog.Error("Failed to create MCP client", "error", e.Error(), "server", mcpServer)
			if err == nil {
				err = e
			} else {
				err = errors.Join(err, e)
			}
			continue
		}

		// Start the client if it's an SSE client (SSE transports require explicit start)
		if isSSE {
			if e := mcpClient.Start(a.context); e != nil {
				xlog.Error("Failed to start SSE MCP client", "error", e.Error(), "server", mcpServer)
				if err == nil {
					err = e
				} else {
					err = errors.Join(err, e)
				}
				continue
			}
			xlog.Debug("SSE client started successfully", "server", mcpServer.URL)
		}

		xlog.Debug("Initializing client", "server", mcpServer.URL)
		// Initialize the client with proper InitializeRequest
		initReq := mcp.InitializeRequest{
			Request: mcp.Request{
				Method: "initialize",
			},
			Params: mcp.InitializeParams{
				ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
				Capabilities:    mcp.ClientCapabilities{},
				ClientInfo: mcp.Implementation{
					Name:    "LocalAGI",
					Version: "1.0.0",
				},
			},
		}
		response, e := mcpClient.Initialize(a.context, initReq)
		if e != nil {
			xlog.Error("Failed to initialize client", "error", e.Error(), "server", mcpServer)
			if err == nil {
				err = e
			} else {
				err = errors.Join(err, e)
			}
			continue
		}

		xlog.Debug("Client initialized", "instructions", response.Instructions)

		var cursor *mcp.Cursor
		for {
			listReq := mcp.ListToolsRequest{
				PaginatedRequest: mcp.PaginatedRequest{
					Request: mcp.Request{
						Method: "tools/list",
					},
					Params: mcp.PaginatedParams{},
				},
			}
			if cursor != nil {
				listReq.Params.Cursor = *cursor
			}

			tools, err := mcpClient.ListTools(a.context, listReq)
			if err != nil {
				xlog.Error("Failed to list tools", "error", err.Error())
				return err
			}

			for _, t := range tools.Tools {
				desc := ""
				if t.Description != "" {
					desc = t.Description
				}

				xlog.Debug("Tool", "mcpServer", mcpServer, "name", t.Name, "description", desc)

				dat, err := json.Marshal(t.InputSchema)
				if err != nil {
					xlog.Error("Failed to marshal input schema", "error", err.Error())
				}

				xlog.Debug("Input schema", "mcpServer", mcpServer, "tool", t.Name, "schema", string(dat))

				// XXX: This is a wild guess, to verify (data types might be incompatible)
				var inputSchema ToolInputSchema
				err = json.Unmarshal(dat, &inputSchema)
				if err != nil {
					xlog.Error("Failed to unmarshal input schema", "error", err.Error())
				}

				// Create a new action with Client + tool
				generatedActions = append(generatedActions, &mcpAction{
					mcpClient:       mcpClient,
					toolName:        t.Name,
					inputSchema:     inputSchema,
					toolDescription: desc,
				})
			}

			if tools.NextCursor == "" {
				break // No more pages
			}
			cursor = &tools.NextCursor
		}

	}

	a.mcpActions = generatedActions

	return err
}
