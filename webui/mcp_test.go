package webui

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"

	fiber "github.com/gofiber/fiber/v2"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/mudler/LocalAGI/core/state"
	"github.com/mudler/LocalAGI/services"
	"github.com/mudler/LocalAGI/services/skills"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// newTestPool builds an AgentPool rooted in a throwaway directory, wired with
// the same action/connector/prompt/filter providers the server uses.
func newTestPool(dir string) *state.AgentPool {
	skillsService, err := skills.NewService(dir)
	Expect(err).ToNot(HaveOccurred())

	pool, err := state.NewAgentPool(
		"test-model", "", "", "", "",
		"http://127.0.0.1:1/v1", "",
		dir,
		services.Actions(map[string]string{services.ConfigStateDir: dir}),
		services.Connectors,
		services.DynamicPrompts(map[string]string{services.ConfigStateDir: dir}),
		services.Filters,
		"5m",
		false,
		skillsService,
	)
	Expect(err).ToNot(HaveOccurred())
	return pool
}

// connectMCP runs the app's MCP server over an in-memory transport and returns
// a connected client session.
func connectMCP(ctx context.Context, pool *state.AgentPool) *mcp.ClientSession {
	app := &App{config: NewConfig(WithPool(pool))}

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	srv := app.newMCPServer(pool)
	go func() {
		defer GinkgoRecover()
		_ = srv.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	Expect(err).ToNot(HaveOccurred())
	return session
}

// callTool invokes a tool and returns the raw result.
func callTool(ctx context.Context, session *mcp.ClientSession, name string, args any) *mcp.CallToolResult {
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	Expect(err).ToNot(HaveOccurred())
	return res
}

// callToolOK invokes a tool, asserts it succeeded, and decodes its structured
// output into out.
func callToolOK(ctx context.Context, session *mcp.ClientSession, name string, args any, out any) {
	res := callTool(ctx, session, name, args)
	Expect(res.IsError).To(BeFalse(), "tool %s failed: %s", name, textOf(res))
	if out == nil {
		return
	}
	data, err := json.Marshal(res.StructuredContent)
	Expect(err).ToNot(HaveOccurred())
	Expect(json.Unmarshal(data, out)).To(Succeed())
}

// textOf concatenates the textual content of a tool result.
func textOf(res *mcp.CallToolResult) string {
	out := ""
	for _, c := range res.Content {
		if t, ok := c.(*mcp.TextContent); ok {
			out += t.Text
		}
	}
	return out
}

var _ = Describe("MCP server", func() {
	var (
		ctx     context.Context
		cancel  context.CancelFunc
		session *mcp.ClientSession
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		session = connectMCP(ctx, newTestPool(GinkgoT().TempDir()))
	})

	AfterEach(func() {
		session.Close()
		cancel()
	})

	It("exposes the agent management tools", func() {
		tools, err := session.ListTools(ctx, nil)
		Expect(err).ToNot(HaveOccurred())

		names := []string{}
		for _, t := range tools.Tools {
			names = append(names, t.Name)
		}

		Expect(names).To(ConsistOf(
			"list_agents",
			"get_agent_config",
			"create_agent",
			"update_agent_config",
			"delete_agent",
			"pause_agent",
			"start_agent",
			"get_agent_config_schema",
		))
	})
	It("describes the agent configuration schema", func() {
		meta := state.AgentConfigMeta{}
		callToolOK(ctx, session, "get_agent_config_schema", struct{}{}, &meta)

		fieldNames := []string{}
		for _, f := range meta.Fields {
			fieldNames = append(fieldNames, f.Name)
		}
		Expect(fieldNames).To(ContainElements("name", "model", "system_prompt"))
		Expect(meta.Actions).ToNot(BeEmpty())
		Expect(meta.Connectors).ToNot(BeEmpty())
	})
	It("creates an agent that can be read back", func() {
		callToolOK(ctx, session, "create_agent", map[string]any{
			"name":          "researcher",
			"description":   "digs things up",
			"model":         "custom-model",
			"system_prompt": "You are a researcher.",
		}, nil)

		cfg := state.AgentConfig{}
		callToolOK(ctx, session, "get_agent_config", agentNameArgs{Name: "researcher"}, &cfg)

		Expect(cfg.Name).To(Equal("researcher"))
		Expect(cfg.Description).To(Equal("digs things up"))
		Expect(cfg.Model).To(Equal("custom-model"))
		Expect(cfg.SystemPrompt).To(Equal("You are a researcher."))
	})
	It("lists the agents in the pool", func() {
		callToolOK(ctx, session, "create_agent", map[string]any{
			"name": "alpha", "description": "the first one", "model": "model-a",
		}, nil)
		callToolOK(ctx, session, "create_agent", map[string]any{"name": "beta"}, nil)

		listed := listAgentsResult{}
		callToolOK(ctx, session, "list_agents", struct{}{}, &listed)

		names := []string{}
		for _, a := range listed.Agents {
			names = append(names, a.Name)
		}
		Expect(names).To(ConsistOf("alpha", "beta"))

		for _, a := range listed.Agents {
			if a.Name == "alpha" {
				Expect(a.Description).To(Equal("the first one"))
				Expect(a.Model).To(Equal("model-a"))
				Expect(a.Paused).To(BeFalse())
			}
		}
	})

	It("replaces the configuration of an existing agent", func() {
		callToolOK(ctx, session, "create_agent", map[string]any{
			"name": "editme", "system_prompt": "before",
		}, nil)

		callToolOK(ctx, session, "update_agent_config", map[string]any{
			"name": "editme",
			"config": map[string]any{
				"name":          "editme",
				"system_prompt": "after",
				"enable_kb":     true,
			},
		}, nil)

		cfg := state.AgentConfig{}
		callToolOK(ctx, session, "get_agent_config", agentNameArgs{Name: "editme"}, &cfg)
		Expect(cfg.SystemPrompt).To(Equal("after"))
		Expect(cfg.EnableKnowledgeBase).To(BeTrue())
	})

	It("deletes an agent", func() {
		callToolOK(ctx, session, "create_agent", map[string]any{"name": "temporary"}, nil)
		callToolOK(ctx, session, "delete_agent", agentNameArgs{Name: "temporary"}, nil)

		res := callTool(ctx, session, "get_agent_config", agentNameArgs{Name: "temporary"})
		Expect(res.IsError).To(BeTrue())
	})

	It("pauses and resumes an agent", func() {
		callToolOK(ctx, session, "create_agent", map[string]any{"name": "sleepy"}, nil)

		callToolOK(ctx, session, "pause_agent", agentNameArgs{Name: "sleepy"}, nil)
		listed := listAgentsResult{}
		callToolOK(ctx, session, "list_agents", struct{}{}, &listed)
		Expect(listed.Agents[0].Paused).To(BeTrue())

		callToolOK(ctx, session, "start_agent", agentNameArgs{Name: "sleepy"}, nil)
		callToolOK(ctx, session, "list_agents", struct{}{}, &listed)
		Expect(listed.Agents[0].Paused).To(BeFalse())
	})

	It("reports a tool error for an unknown agent", func() {
		for _, tool := range []string{"get_agent_config", "delete_agent", "pause_agent", "start_agent"} {
			res := callTool(ctx, session, tool, agentNameArgs{Name: "ghost"})
			Expect(res.IsError).To(BeTrue(), "%s should have failed", tool)
			Expect(textOf(res)).To(ContainSubstring("ghost"))
		}
	})

	It("rejects a create call that omits the name", func() {
		_, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "create_agent",
			Arguments: map[string]any{"description": "nameless"},
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("name"))
	})

	It("refuses to create an agent with an empty name", func() {
		res := callTool(ctx, session, "create_agent", map[string]any{"name": ""})
		Expect(res.IsError).To(BeTrue())
		Expect(textOf(res)).To(ContainSubstring("name is required"))
	})

	It("refuses to create an agent that already exists", func() {
		callToolOK(ctx, session, "create_agent", map[string]any{"name": "twice"}, nil)

		res := callTool(ctx, session, "create_agent", map[string]any{"name": "twice"})
		Expect(res.IsError).To(BeTrue())
		Expect(textOf(res)).To(ContainSubstring("already exists"))
	})

	It("refuses to update an agent that does not exist", func() {
		res := callTool(ctx, session, "update_agent_config", map[string]any{
			"name":   "ghost",
			"config": map[string]any{"name": "ghost"},
		})
		Expect(res.IsError).To(BeTrue())
		Expect(textOf(res)).To(ContainSubstring("ghost"))
	})
})

var _ = Describe("MCP HTTP endpoint", func() {
	It("serves the MCP protocol at /mcp", func() {
		pool := newTestPool(GinkgoT().TempDir())
		app := &App{config: NewConfig(WithPool(pool))}

		webapp := fiber.New()
		app.registerMCPRoutes(pool, webapp)

		body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{` +
			`"protocolVersion":"2025-06-18","capabilities":{},` +
			`"clientInfo":{"name":"test","version":"v1"}}}`

		req := httptest.NewRequest("POST", "/mcp", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")

		resp, err := webapp.Test(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(200))

		payload, err := io.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(payload)).To(ContainSubstring("LocalAGI"))
	})
})
