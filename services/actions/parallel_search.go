package actions

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/mudler/LocalAGI/core/types"
	"mvdan.cc/xurls/v2"
)

const parallelSearchEndpoint = "https://search.parallel.ai/mcp"

// parallelSearchTransport identifies LocalAGI on every MCP request, including
// initialization and session cleanup. It never reads ambient API credentials.
type parallelSearchTransport struct {
	base http.RoundTripper
	ctx  context.Context
}

func (t parallelSearchTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// The SDK detaches its HTTP context from the caller, including while
	// initializing. Bind all requests and response reads to this action's
	// deadline and cancellation rather than the session's detached lifetime.
	req = req.Clone(t.ctx)
	if req.Method == http.MethodDelete {
		// The SDK detaches the session cleanup context from the caller. Keep
		// cleanup short and let job cancellation stop it too.
		ctx, cancel := context.WithTimeout(t.ctx, 2*time.Second)
		defer cancel()
		req = req.Clone(ctx)
	}
	req.Header.Set("User-Agent", "LocalAGI (https://github.com/mudler/LocalAGI)")
	return t.base.RoundTrip(req)
}

func parallelSearch(ctx context.Context, query string) (types.ActionResult, error) {
	if strings.TrimSpace(query) == "" {
		return types.ActionResult{}, fmt.Errorf("Parallel search requires a query")
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{Name: "LocalAGI", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: parallelSearchEndpoint,
		HTTPClient: &http.Client{
			Timeout:   60 * time.Second,
			Transport: parallelSearchTransport{base: http.DefaultTransport, ctx: ctx},
		},
	}, nil)
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("connect to Parallel Search MCP: %w", err)
	}
	defer session.Close()

	// Actions receive a job context, not a stable conversation identity. Leave
	// the optional session_id unset rather than sharing one across agent chats
	// or changing it on each tool call. The MCP connection is not a conversation.
	response, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "web_search",
		Arguments: map[string]any{
			"objective":      query,
			"search_queries": []string{query},
		},
	})
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("Parallel search: %w", err)
	}
	var text []string
	for _, content := range response.Content {
		if c, ok := content.(*mcp.TextContent); ok {
			text = append(text, c.Text)
		}
	}
	result := strings.Join(text, "\n")
	if response.IsError {
		return types.ActionResult{}, fmt.Errorf("Parallel search: %s", result)
	}
	if strings.TrimSpace(result) == "" {
		return types.ActionResult{}, fmt.Errorf("Parallel search returned no text")
	}
	urls := xurls.Strict().FindAllString(result, -1)
	return types.ActionResult{Result: result, Metadata: map[string]interface{}{MetadataUrls: urls}}, nil
}
