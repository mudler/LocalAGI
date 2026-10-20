package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/mudler/LocalAGI/core/state"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/tmc/langchaingo/httputil"
)

type searchRoundTripper func(*http.Request) (*http.Response, error)

func (f searchRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Load saved action configuration through the same factory and tool adapter the
// agent uses. These are scripted tool calls, without model inference.
func configuredSearch(t *testing.T, ctx context.Context, config string) (string, any, error) {
	t.Helper()
	acts := Actions(nil)(&state.AgentConfig{Actions: []state.ActionsConfig{{Name: ActionSearch, Config: config}}})(ctx, nil)
	if len(acts) != 1 {
		t.Fatalf("loaded %d actions", len(acts))
	}
	tools := types.Actions(acts).ToCogitoTools(ctx, nil)
	if tools[0].Tool().Function.Name != "search_internet" {
		t.Fatal("search tool contract changed")
	}
	return tools[0].Execute(map[string]any{"query": "Go release notes"})
}

func routeSearch(t *testing.T, rt http.RoundTripper) {
	t.Helper()
	previous := http.DefaultTransport
	http.DefaultTransport = rt
	t.Cleanup(func() { http.DefaultTransport = previous })
}

func TestSearchProviderParallel(t *testing.T) {
	t.Setenv("PARALLEL_API_KEY", "ambient-must-not-be-sent")
	var mu sync.Mutex
	methods := map[string]int{}
	server := mcp.NewServer(&mcp.Implementation{Name: "search-fixture", Version: "1"}, nil)
	server.AddTool(&mcp.Tool{Name: "web_search", InputSchema: json.RawMessage(`{"type":"object"}`)}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args map[string]any
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, err
		}
		if args["objective"] != "Go release notes" || !reflect.DeepEqual(args["search_queries"], []any{"Go release notes"}) {
			t.Errorf("unexpected arguments: %v", args)
		}
		if _, ok := args["session_id"]; ok {
			t.Error("action must not invent a conversation ID")
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Go release notes: https://go.dev/doc/devel/release"}, &mcp.TextContent{Text: "Source: https://go.dev/doc/go1.26"}}}, nil
	})
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.UserAgent(), "LocalAGI") {
			t.Errorf("missing project User-Agent: %q", r.UserAgent())
		}
		if r.Header.Get("Authorization") != "" {
			t.Error("anonymous request included credentials")
		}
		if r.Method == http.MethodPost {
			b, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(strings.NewReader(string(b)))
			var msg struct{ Method string }
			_ = json.Unmarshal(b, &msg)
			mu.Lock()
			methods[msg.Method]++
			mu.Unlock()
		}
		handler.ServeHTTP(w, r)
	}))
	defer fixture.Close()
	target, _ := url.Parse(fixture.URL)
	realTransport := http.DefaultTransport
	routeSearch(t, searchRoundTripper(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://search.parallel.ai/mcp" {
			t.Errorf("unexpected destination %s", r.URL)
		}
		deadline, ok := r.Context().Deadline()
		if !ok || time.Until(deadline) > 60*time.Second {
			t.Error("request lacks bounded deadline")
		}
		r = r.Clone(r.Context())
		r.URL.Scheme = target.Scheme
		r.URL.Host = target.Host
		return realTransport.RoundTrip(r)
	}))
	for i := 0; i < 2; i++ {
		text, raw, err := configuredSearch(t, context.Background(), `{"provider":"parallel","results":"1"}`)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(text, "Go release notes") {
			t.Fatalf("tool consumer lost text: %q", text)
		}
		result, ok := raw.(types.ActionResult)
		if !ok {
			t.Fatalf("lost action result: %T", raw)
		}
		want := []string{"https://go.dev/doc/devel/release", "https://go.dev/doc/go1.26"}
		if !reflect.DeepEqual(result.Metadata["urls"], want) {
			t.Fatalf("source URLs: %v", result.Metadata)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if methods["initialize"] != 2 || methods["tools/call"] != 2 {
		t.Fatalf("actual MCP methods: %v", methods)
	}
}

func TestSearchProviderDefaults(t *testing.T) {
	previous := httputil.DefaultTransport
	defer func() { httputil.DefaultTransport = previous }()
	calls := 0
	httputil.DefaultTransport = searchRoundTripper(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Host != "html.duckduckgo.com" || r.URL.Query().Get("q") != "Go release notes" {
			t.Errorf("default request: %s", r.URL)
		}
		body := `<div class="web-result"><a class="result__a" rel="nofollow" href="https://go.dev/doc/devel/release">Go</a><span class="result__snippet">Release notes</span></div>`
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	routeSearch(t, searchRoundTripper(func(r *http.Request) (*http.Response, error) {
		t.Errorf("unexpected Parallel request: %s", r.URL)
		return nil, errors.New("unexpected request")
	}))
	for _, config := range []string{`{}`, `{"results":"2"}`, `{"provider":"duckduckgo"}`} {
		text, _, err := configuredSearch(t, context.Background(), config)
		if err != nil || !strings.Contains(text, "Release notes") {
			t.Fatalf("legacy/default selection: %q, %v", text, err)
		}
	}
	if calls != 3 {
		t.Fatalf("DuckDuckGo called %d times", calls)
	}
	acts := Actions(nil)(&state.AgentConfig{})(context.Background(), nil)
	if len(acts) != 0 {
		t.Fatal("search was enabled on an unconfigured agent")
	}
	_, _, err := configuredSearch(t, context.Background(), `{"provider":"unsupported"}`)
	if err == nil || !strings.Contains(err.Error(), "unknown search provider") {
		t.Fatalf("invalid provider: %v", err)
	}
}

func TestSearchProviderParallelErrors(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
		status           int
	}{
		{"quota", `{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"quota exceeded"}}`, "quota", 200},
		{"http", "service unavailable", "Service Unavailable", 503},
		{"malformed", "not json", "", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			routeSearch(t, searchRoundTripper(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tc.status, Status: http.StatusText(tc.status), Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			}))
			text, _, err := configuredSearch(t, context.Background(), `{"provider":"parallel"}`)
			if err == nil || text != "" || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected failure, got %q %v", text, err)
			}
		})
	}
	for _, canceled := range []bool{true, false} {
		name := "deadline"
		if canceled {
			name = "cancellation"
		}
		t.Run(name, func(t *testing.T) {
			routeSearch(t, searchRoundTripper(func(r *http.Request) (*http.Response, error) { <-r.Context().Done(); return nil, r.Context().Err() }))
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
			defer cancel()
			want := context.DeadlineExceeded
			if canceled {
				cancel()
				want = context.Canceled
			}
			_, _, err := configuredSearch(t, ctx, `{"provider":"parallel"}`)
			if !errors.Is(err, want) {
				t.Fatalf("expected %v, got %v", want, err)
			}
		})
	}
}

func TestSearchProviderParallelToolError(t *testing.T) {
	for _, text := range []string{"quota exceeded", ""} {
		t.Run(text, func(t *testing.T) {
			server := mcp.NewServer(&mcp.Implementation{Name: "error-fixture", Version: "1"}, nil)
			server.AddTool(&mcp.Tool{Name: "web_search", InputSchema: json.RawMessage(`{"type":"object"}`)}, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return &mcp.CallToolResult{IsError: text != "", Content: []mcp.Content{&mcp.TextContent{Text: text}}}, nil
			})
			fixture := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true}))
			defer fixture.Close()
			target, _ := url.Parse(fixture.URL)
			realTransport := http.DefaultTransport
			routeSearch(t, searchRoundTripper(func(r *http.Request) (*http.Response, error) {
				r = r.Clone(r.Context())
				r.URL.Scheme = target.Scheme
				r.URL.Host = target.Host
				return realTransport.RoundTrip(r)
			}))
			output, _, err := configuredSearch(t, context.Background(), `{"provider":"parallel"}`)
			if err == nil || output != "" {
				t.Fatalf("expected tool failure, got %q %v", output, err)
			}
		})
	}
}

func TestSearchProviderParallelLive(t *testing.T) {
	if os.Getenv("LOCALAGI_TEST_PARALLEL_LIVE") != "1" {
		t.Skip("set LOCALAGI_TEST_PARALLEL_LIVE=1 for anonymous live MCP search")
	}
	t.Setenv("PARALLEL_API_KEY", "")
	realTransport := http.DefaultTransport
	routeSearch(t, searchRoundTripper(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://search.parallel.ai/mcp" || !strings.Contains(r.UserAgent(), "LocalAGI") || r.Header.Get("Authorization") != "" {
			t.Errorf("unexpected live request destination, identity or authentication")
		}
		t.Logf("Anonymous MCP request: %s %s; User-Agent: %s", r.Method, r.URL, r.UserAgent())
		return realTransport.RoundTrip(r)
	}))
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	text, raw, err := configuredSearch(t, ctx, `{"provider":"parallel"}`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(text) == "" {
		t.Fatal("empty search output")
	}
	result := raw.(types.ActionResult)
	if len(result.Metadata["urls"].([]string)) == 0 {
		t.Fatalf("no source URLs: %s", text)
	}
	t.Logf("Anonymous search output: %s\nURL metadata: %v", text, result.Metadata["urls"])
}

func TestSearchProviderParallelCallCancellation(t *testing.T) {
	for _, canceled := range []bool{true, false} {
		name := "deadline"
		if canceled {
			name = "cancellation"
		}
		t.Run(name, func(t *testing.T) {
			server := mcp.NewServer(&mcp.Implementation{Name: "cancel-fixture", Version: "1"}, nil)
			fixture := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true}))
			defer fixture.Close()
			target, _ := url.Parse(fixture.URL)
			realTransport := http.DefaultTransport
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			called := false
			routeSearch(t, searchRoundTripper(func(r *http.Request) (*http.Response, error) {
				if r.Body != nil {
					body, _ := io.ReadAll(r.Body)
					r.Body = io.NopCloser(strings.NewReader(string(body)))
					var msg struct{ Method string }
					_ = json.Unmarshal(body, &msg)
					if msg.Method == "tools/call" {
						called = true
						if canceled {
							cancel()
						}
						<-r.Context().Done()
						return nil, r.Context().Err()
					}
				}
				r = r.Clone(r.Context())
				r.URL.Scheme = target.Scheme
				r.URL.Host = target.Host
				return realTransport.RoundTrip(r)
			}))
			text, _, err := configuredSearch(t, ctx, `{"provider":"parallel"}`)
			want := context.DeadlineExceeded
			if canceled {
				want = context.Canceled
			}
			if !called || text != "" || !errors.Is(err, want) {
				t.Fatalf("expected call-stage %v, called=%v, output=%q, err=%v", want, called, text, err)
			}
		})
	}
}

func TestSearchProviderParallelSourceURLs(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "source-fixture", Version: "1"}, nil)
	server.AddTool(&mcp.Tool{Name: "web_search", InputSchema: json.RawMessage(`{"type":"object"}`)}, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Match the live endpoint's JSON text response, including escaped excerpts.
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: `{"results":[{"url":"https://go.dev/doc/go1.26","title":"Go release notes","excerpts":["See https://go.dev/issue/123.\nNext line"]},{"url":"javascript:alert(1)"}]}`}}}, nil
	})
	fixture := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true}))
	defer fixture.Close()
	target, _ := url.Parse(fixture.URL)
	realTransport := http.DefaultTransport
	routeSearch(t, searchRoundTripper(func(r *http.Request) (*http.Response, error) {
		r = r.Clone(r.Context())
		r.URL.Scheme = target.Scheme
		r.URL.Host = target.Host
		return realTransport.RoundTrip(r)
	}))
	text, raw, err := configuredSearch(t, context.Background(), `{"provider":"parallel"}`)
	if err != nil || !strings.Contains(text, "Go release notes") {
		t.Fatalf("result consumption: %q %v", text, err)
	}
	if got := raw.(types.ActionResult).Metadata["urls"]; !reflect.DeepEqual(got, []string{"https://go.dev/doc/go1.26"}) {
		t.Fatalf("source URLs included escaped excerpt or invalid URL: %v", got)
	}
}
