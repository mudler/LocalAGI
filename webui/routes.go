package webui

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/dave-gray101/v2keyauth"
	fiber "github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/keyauth"
	"github.com/google/uuid"
	"github.com/mudler/LocalAGI/core/sse"
	"github.com/mudler/LocalAGI/db"
	models "github.com/mudler/LocalAGI/dbmodels"
	"github.com/mudler/LocalAGI/services/connectors"

	"github.com/mudler/LocalAGI/core/state"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/mudler/LocalAGI/services"
)

//go:embed old/views/*
var viewsfs embed.FS

//go:embed old/public/*
var embeddedFiles embed.FS

//go:embed react-ui/dist/*
var reactUI embed.FS

func (app *App) registerRoutes(pool *state.AgentPool, webapp *fiber.App) {

	webapp.Use("/old/public", filesystem.New(filesystem.Config{
		Root:       http.FS(embeddedFiles),
		PathPrefix: "/old/public",
		Browse:     true,
	}))

	if len(app.config.ApiKeys) > 0 {
		kaConfig, err := GetKeyAuthConfig(app.config.ApiKeys)
		if err != nil || kaConfig == nil {
			panic(err)
		}
		webapp.Use(v2keyauth.New(*kaConfig))
	}

	webapp.Get("/old", func(c *fiber.Ctx) error {
		return c.Render("old/views/index", fiber.Map{
			"Agents":     pool.List(),
			"AgentCount": len(pool.List()),
			"Actions":    len(services.AvailableActions),
			"Connectors": len(services.AvailableConnectors),
		})
	})

	webapp.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/app")
	})

	webapp.Use("/app", filesystem.New(filesystem.Config{
		Root:       http.FS(reactUI),
		PathPrefix: "react-ui/dist",
	}))

	// Fallback route for SPA
	webapp.Get("/app/*", func(c *fiber.Ctx) error {
		indexHTML, err := reactUI.ReadFile("react-ui/dist/index.html")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Error reading index.html")
		}
		c.Set("Content-Type", "text/html")
		return c.Send(indexHTML)
	})

	webapp.Get("/old/agents", func(c *fiber.Ctx) error {
		statuses := map[string]bool{}
		for _, a := range pool.List() {
			agent := pool.GetAgent(a)
			if agent == nil {
				xlog.Error("Agent not found", "name", a)
				continue
			}
			statuses[a] = !agent.Paused()
		}
		return c.Render("old/views/agents", fiber.Map{
			"Agents": pool.List(),
			"Status": statuses,
		})
	})

	webapp.Get("/old/create", func(c *fiber.Ctx) error {
		return c.Render("old/views/create", fiber.Map{
			"Actions":      services.AvailableActions,
			"Connectors":   services.AvailableConnectors,
			"PromptBlocks": services.AvailableBlockPrompts,
		})
	})

	// Define a route for the GET method on the root path '/'
	webapp.Get("/sse/:id", app.RequireUser(), app.RequireActiveAgent(), func(c *fiber.Ctx) error {
		userID, ok := c.Locals("id").(string)

		if !ok || userID == "" {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		pool, ok := app.UserPools[userID]
		if !ok {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Agent pool not found",
			})
		}

		agentID := c.Params("id")

		if pool.GetAgent(agentID) == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Agent not found or unauthorized",
			})
		}

		manager := pool.GetManager(agentID)
		if manager == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "SSE stream not available for this agent",
			})
		}

		manager.Handle(c, sse.NewClient(randStringRunes(10)))
		return nil
	})

	webapp.Get("/old/status/:name", func(c *fiber.Ctx) error {
		history := pool.GetStatusHistory(c.Params("name"))
		if history == nil {
			history = &state.Status{ActionResults: []types.ActionState{}}
		}
		// reverse history

		return c.Render("old/views/status", fiber.Map{
			"Name":    c.Params("name"),
			"History": Reverse(history.Results()),
		})
	})

	webapp.Get("/api/notify/:name", app.RequireUser(), app.Notify(pool))
	webapp.Post("/old/chat/:name", app.RequireUser(), app.OldChat(pool))

	webapp.Post("/api/agent/create", app.RequireUser(), app.Create())
	webapp.Delete("/api/agent/:id", app.RequireUser(), app.RequireActiveAgent(), app.Delete())
	webapp.Put("/api/agent/:id/pause", app.RequireUser(), app.RequireActiveAgent(), app.Pause())
	webapp.Put("/api/agent/:id/start", app.RequireUser(), app.RequireActiveAgent(), app.Start())

	webapp.Post("/api/chat/:id", app.RequireUser(), app.RequireActiveAgent(), app.Chat())

	conversationTracker := connectors.NewConversationTracker[string](time.Minute)

	webapp.Post("/v1/responses", app.Responses(pool, conversationTracker))

	webapp.Get("/old/talk/:name", func(c *fiber.Ctx) error {
		return c.Render("old/views/chat", fiber.Map{
			//	"Character": agent.Character,
			"Name": c.Params("name"),
		})
	})

	webapp.Get("/old/settings/:name", func(c *fiber.Ctx) error {
		status := false
		for _, a := range pool.List() {
			if a == c.Params("name") {
				status = !pool.GetAgent(a).Paused()
			}
		}

		return c.Render("old/views/settings", fiber.Map{
			"Name":         c.Params("name"),
			"Status":       status,
			"Actions":      services.AvailableActions,
			"Connectors":   services.AvailableConnectors,
			"PromptBlocks": services.AvailableBlockPrompts,
		})
	})

	webapp.Get("/old/actions-playground", func(c *fiber.Ctx) error {
		return c.Render("old/views/actions", fiber.Map{})
	})

	webapp.Get("/old/group-create", func(c *fiber.Ctx) error {
		return c.Render("old/views/group-create", fiber.Map{
			"Actions":      services.AvailableActions,
			"Connectors":   services.AvailableConnectors,
			"PromptBlocks": services.AvailableBlockPrompts,
		})
	})

	// New API endpoints for getting and updating agent configuration
	webapp.Get("/api/agent/:id/config", app.RequireUser(), app.RequireActiveAgent(), app.GetAgentConfig())

	webapp.Put("/api/agent/:id/config", app.RequireUser(), app.RequireActiveAgent(), app.UpdateAgentConfig())

	// Metadata endpoint for agent configuration fields
	webapp.Get("/api/agent/config/metadata", app.RequireUser(), app.GetAgentConfigMeta())

	// Add endpoint for getting agent config metadata
	webapp.Get("/api/meta/agent/config", app.RequireUser(), app.GetAgentConfigMeta())

	webapp.Post("/api/action/:name/run", app.RequireUser(), app.ExecuteAction(pool))
	webapp.Get("/api/actions", app.ListActions())

	webapp.Post("/api/agent/group/generateProfiles", app.RequireUser(), app.GenerateGroupProfiles())
	webapp.Post("/api/agent/group/create", app.RequireUser(), app.CreateGroup())

	// Dashboard API endpoint for React UI
	webapp.Get("/api/agents", app.RequireUser(), func(c *fiber.Ctx) error {
		userID, ok := c.Locals("id").(string)
		if !ok || userID == "" {
			return errorJSONMessage(c, "User ID missing")
		}

		// 1. Parse UUID
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			return errorJSONMessage(c, "Invalid user ID")
		}

		// 2. Fetch non-archived agents directly from MySQL
		var dbAgents []models.Agent
		if err := db.DB.Where("UserId = ? AND archive = false", userUUID).Find(&dbAgents).Error; err != nil {
			return errorJSONMessage(c, "Failed to fetch agents: "+err.Error())
		}

		// 3. Build agent list response and check in-memory status
		agentList := make([]fiber.Map, 0, len(dbAgents))
		statuses := make(map[string]bool)

		// Use or init in-memory pool
		pool, ok := app.UserPools[userID]
		if !ok {
			pool, err = state.NewAgentPool(
				userID,
				"",
				os.Getenv("LOCALAGI_MULTIMODAL_MODEL"),
				os.Getenv("LOCALAGI_IMAGE_MODEL"),
				os.Getenv("LOCALAGI_LLM_API_URL"),
				os.Getenv("LOCALAGI_LLM_API_KEY"),
				os.Getenv("LOCALAGI_LOCALRAG_URL"),
				services.Actions,
				services.Connectors,
				services.DynamicPrompts,
				os.Getenv("LOCALAGI_TIMEOUT"),
				os.Getenv("LOCALAGI_ENABLE_CONVERSATIONS_LOGGING") == "true",
			)
			if err != nil {
				return errorJSONMessage(c, "Failed to load agent pool: "+err.Error())
			}
			app.UserPools[userID] = pool
		}

		for _, agent := range dbAgents {
			idStr := agent.ID.String()

			// Ensure config is loaded into pool memory
			if pool.GetAgent(idStr) == nil {
				var cfg state.AgentConfig
				if err := json.Unmarshal(agent.Config, &cfg); err == nil {
					cfg.Name = agent.Name // optionally enforce name
					_ = pool.CreateAgent(idStr, &cfg, true)
				}
			}

			// Do not start agent, just check if already running
			instance := pool.GetAgent(idStr)

			running := instance != nil && !instance.Paused()

			agentList = append(agentList, fiber.Map{
				"id":   agent.ID,
				"name": agent.Name,
			})
			statuses[idStr] = running
		}

		// 4. Return final response
		return c.JSON(fiber.Map{
			"agents":     agentList,
			"agentCount": len(agentList),
			"actions":    len(services.AvailableActions),
			"connectors": len(services.AvailableConnectors),
			"statuses":   statuses,
		})
	})

	// API endpoint for getting a specific agent's details
	webapp.Get("/api/agent/:id", app.RequireUser(), app.RequireActiveAgent(), app.GetAgentDetails())

	// API endpoint for agent status history
	webapp.Get("/api/agent/:id/status", app.RequireUser(), app.RequireActiveAgent(), func(c *fiber.Ctx) error {
		userID, ok := c.Locals("id").(string)
		if !ok || userID == "" {
			return errorJSONMessage(c, "User ID missing")
		}

		agentId := c.Params("id")
		if agentId == "" {
			return errorJSONMessage(c, "Agent id is required")
		}

		// Load or init in-memory agent pool
		pool, ok := app.UserPools[userID]
		if !ok {
			newPool, err := state.NewAgentPool(
				userID,
				"",
				os.Getenv("LOCALAGI_MULTIMODAL_MODEL"),
				os.Getenv("LOCALAGI_IMAGE_MODEL"),
				os.Getenv("LOCALAGI_LLM_API_URL"),
				os.Getenv("LOCALAGI_LLM_API_KEY"),
				os.Getenv("LOCALAGI_LOCALRAG_URL"),
				services.Actions,
				services.Connectors,
				services.DynamicPrompts,
				os.Getenv("LOCALAGI_TIMEOUT"),
				os.Getenv("LOCALAGI_ENABLE_CONVERSATIONS_LOGGING") == "true",
			)
			if err != nil {
				return errorJSONMessage(c, "Failed to load agent pool: "+err.Error())
			}
			app.UserPools[userID] = newPool
			pool = newPool
		}

		// Get agent status history
		history := pool.GetStatusHistory(agentId)
		if history == nil {
			history = &state.Status{ActionResults: []types.ActionState{}}
		}

		entries := []string{}
		for _, h := range Reverse(history.Results()) {
			entries = append(entries, fmt.Sprintf(
				"Result: %v Action: %v Params: %v Reasoning: %v",
				h.Result,
				h.Action.Definition().Name,
				h.Params,
				h.Reasoning,
			))
		}

		return c.JSON(fiber.Map{
			"id":      agentId,
			"active":  pool.IsAgentActive(agentId),
			"history": entries,
		})
	})

	webapp.Post("/settings/import", app.RequireUser(), app.ImportAgent(pool))
	webapp.Get("/settings/export/:name", app.RequireUser(), app.ExportAgent(pool))

	// webapp.Post("/api/openrouter/:id/chat", app.RequireUser(), app.ProxyOpenRouterChat())
	webapp.Get("/api/agent/:id/chat", app.RequireUser(), app.RequireActiveAgent(), app.GetChatHistory())
	webapp.Delete("/api/agent/:id/chat", app.RequireUser(), app.RequireActiveAgent(), app.ClearChat())

	// New API route to get usage for the user
	webapp.Get("/api/usage", app.RequireUser(), app.GetUsage())

}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func Reverse[T any](original []T) (reversed []T) {
	reversed = make([]T, len(original))
	copy(reversed, original)

	for i := len(reversed)/2 - 1; i >= 0; i-- {
		tmp := len(reversed) - 1 - i
		reversed[i], reversed[tmp] = reversed[tmp], reversed[i]
	}

	return
}

func GetKeyAuthConfig(apiKeys []string) (*v2keyauth.Config, error) {
	customLookup, err := v2keyauth.MultipleKeySourceLookup([]string{"header:Authorization", "header:x-api-key", "header:xi-api-key", "cookie:token"}, keyauth.ConfigDefault.AuthScheme)
	if err != nil {
		return nil, err
	}

	return &v2keyauth.Config{
		CustomKeyLookup: customLookup,
		Next:            func(c *fiber.Ctx) bool { return false },
		Validator:       getApiKeyValidationFunction(apiKeys),
		ErrorHandler:    getApiKeyErrorHandler(false, apiKeys),
		AuthScheme:      "Bearer",
	}, nil
}

func getApiKeyErrorHandler(opaqueErrors bool, apiKeys []string) fiber.ErrorHandler {
	return func(ctx *fiber.Ctx, err error) error {
		if errors.Is(err, v2keyauth.ErrMissingOrMalformedAPIKey) {
			if len(apiKeys) == 0 {
				return ctx.Next() // if no keys are set up, any error we get here is not an error.
			}
			ctx.Set("WWW-Authenticate", "Bearer")
			if opaqueErrors {
				return ctx.SendStatus(401)
			}
			return ctx.Status(401).Render("old/views/login", fiber.Map{})
		}
		if opaqueErrors {
			return ctx.SendStatus(500)
		}
		return err
	}
}

func getApiKeyValidationFunction(apiKeys []string) func(*fiber.Ctx, string) (bool, error) {

	return func(ctx *fiber.Ctx, apiKey string) (bool, error) {
		if len(apiKeys) == 0 {
			return true, nil // If no keys are setup, accept everything
		}
		for _, validKey := range apiKeys {
			if subtle.ConstantTimeCompare([]byte(apiKey), []byte(validKey)) == 1 {
				return true, nil
			}
		}
		return false, v2keyauth.ErrMissingOrMalformedAPIKey
	}

}
