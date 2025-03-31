package webui

import (
	"crypto/subtle"
	"embed"
	"errors"
	"math/rand"
	"net/http"
	"path/filepath"

	"github.com/dave-gray101/v2keyauth"
	fiber "github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/keyauth"
	"github.com/mudler/LocalAgent/core/sse"
	"github.com/mudler/LocalAgent/core/state"
	"github.com/mudler/LocalAgent/core/types"
	"github.com/mudler/LocalAgent/pkg/xlog"
	"github.com/mudler/LocalAgent/services"
)

//go:embed views/*
var viewsfs embed.FS

//go:embed public/*
var embeddedFiles embed.FS

//go:embed react-ui/dist/*
var reactUI embed.FS

func (app *App) registerRoutes(pool *state.AgentPool, webapp *fiber.App) {

	// Static avatars in a.pooldir/avatars
	webapp.Use("/avatars", filesystem.New(filesystem.Config{
		Root: http.Dir(filepath.Join(app.config.StateDir, "avatars")),
		//	PathPrefix: "avatars",
		Browse: true,
	}))

	/* 	webapp.Use("/public", filesystem.New(filesystem.Config{
	   		Root:       http.FS(embeddedFiles),
	   		PathPrefix: "public",
	   		Browse:     true,
	   	}))
	*/
	if len(app.config.ApiKeys) > 0 {
		kaConfig, err := GetKeyAuthConfig(app.config.ApiKeys)
		if err != nil || kaConfig == nil {
			panic(err)
		}
		webapp.Use(v2keyauth.New(*kaConfig))
	}

	/* 	webapp.Get("/", func(c *fiber.Ctx) error {
	   		return c.Render("views/index", fiber.Map{
	   			"Agents":     pool.List(),
	   			"AgentCount": len(pool.List()),
	   			"Actions":    len(services.AvailableActions),
	   			"Connectors": len(services.AvailableConnectors),
	   		})
	   	})
	*/
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

	/* 	webapp.Get("/agents", func(c *fiber.Ctx) error {
	   		statuses := map[string]bool{}
	   		for _, a := range pool.List() {
	   			agent := pool.GetAgent(a)
	   			if agent == nil {
	   				xlog.Error("Agent not found", "name", a)
	   				continue
	   			}
	   			statuses[a] = !agent.Paused()
	   		}
	   		return c.Render("views/agents", fiber.Map{
	   			"Agents": pool.List(),
	   			"Status": statuses,
	   		})
	   	})
	*/
	/* 		webapp.Get("/api/create", func(c *fiber.Ctx) error {
		return c.Render("views/create", fiber.Map{
			"Actions":      services.AvailableActions,
			"Connectors":   services.AvailableConnectors,
			"PromptBlocks": services.AvailableBlockPrompts,
		})
	})
	*/
	// Define a route for the GET method on the root path '/'
	webapp.Get("/sse/:name", func(c *fiber.Ctx) error {
		m := pool.GetManager(c.Params("name"))
		if m == nil {
			return c.SendStatus(404)
		}

		m.Handle(c, sse.NewClient(randStringRunes(10)))
		return nil
	})

	/*webapp.Get("/status/:name", func(c *fiber.Ctx) error {
		history := pool.GetStatusHistory(c.Params("name"))
		if history == nil {
			history = &state.Status{ActionResults: []types.ActionState{}}
		}
		// reverse history

		return c.Render("views/status", fiber.Map{
			"Name":    c.Params("name"),
			"History": Reverse(history.Results()),
		})
	})

	webapp.Get("/notify/:name", app.Notify(pool))
	webapp.Post("/chat/:name", app.Chat(pool))
	*/
	webapp.Post("/api/agent/create", app.Create(pool))
	webapp.Delete("/api/agent/:name", app.Delete(pool))
	webapp.Put("/api/agent/:name/pause", app.Pause(pool))
	webapp.Put("/api/agent/:name/start", app.Start(pool))

	// Add JSON-based chat API endpoint
	webapp.Post("/api/chat/:name", app.ChatAPI(pool))

	webapp.Post("/v1/responses", app.Responses(pool))

	/* 	webapp.Get("/talk/:name", func(c *fiber.Ctx) error {
	   		return c.Render("views/chat", fiber.Map{
	   			//	"Character": agent.Character,
	   			"Name": c.Params("name"),
	   		})
	   	})

	   	webapp.Get("/settings/:name", func(c *fiber.Ctx) error {
	   		status := false
	   		for _, a := range pool.List() {
	   			if a == c.Params("name") {
	   				status = !pool.GetAgent(a).Paused()
	   			}
	   		}

	   		return c.Render("views/settings", fiber.Map{
	   			"Name":         c.Params("name"),
	   			"Status":       status,
	   			"Actions":      services.AvailableActions,
	   			"Connectors":   services.AvailableConnectors,
	   			"PromptBlocks": services.AvailableBlockPrompts,
	   		})
	   	})

	   	webapp.Get("/actions-playground", func(c *fiber.Ctx) error {
	   		return c.Render("views/actions", fiber.Map{})
	   	})

	   	webapp.Get("/group-create", func(c *fiber.Ctx) error {
	   		return c.Render("views/group-create", fiber.Map{
	   			"Actions":      services.AvailableActions,
	   			"Connectors":   services.AvailableConnectors,
	   			"PromptBlocks": services.AvailableBlockPrompts,
	   		})
	   	})
	*/
	// New API endpoints for getting and updating agent configuration
	webapp.Get("/api/agent/:name/config", app.GetAgentConfig(pool))
	webapp.Put("/api/agent/:name/config", app.UpdateAgentConfig(pool))

	// Add endpoint for getting agent config metadata
	webapp.Get("/api/meta/agent/config", app.GetAgentConfigMeta())

	webapp.Post("/api/action/:name/run", app.ExecuteAction(pool))
	webapp.Get("/api/actions", app.ListActions())

	webapp.Post("/api/agent/group/generateProfiles", app.GenerateGroupProfiles(pool))
	webapp.Post("/api/agent/group/create", app.CreateGroup(pool))

	// Dashboard API endpoint for React UI
	webapp.Get("/api/agents", func(c *fiber.Ctx) error {
		statuses := map[string]bool{}
		agents := pool.List()
		for _, a := range agents {
			agent := pool.GetAgent(a)
			if agent == nil {
				xlog.Error("Agent not found", "name", a)
				continue
			}
			statuses[a] = !agent.Paused()
		}

		return c.JSON(fiber.Map{
			"agents":     agents,
			"agentCount": len(agents),
			"actions":    len(services.AvailableActions),
			"connectors": len(services.AvailableConnectors),
			"statuses":   statuses,
		})
	})

	// API endpoint for getting a specific agent's details
	webapp.Get("/api/agent/:name", func(c *fiber.Ctx) error {
		name := c.Params("name")
		agent := pool.GetAgent(name)
		if agent == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Agent not found",
			})
		}

		// Add the active status to the configuration
		return c.JSON(fiber.Map{
			"active": !agent.Paused(),
		})
	})

	// API endpoint for agent status history
	webapp.Get("/api/agent/:name/status", func(c *fiber.Ctx) error {
		history := pool.GetStatusHistory(c.Params("name"))
		if history == nil {
			history = &state.Status{ActionResults: []types.ActionState{}}
		}

		return c.JSON(fiber.Map{
			"Name":    c.Params("name"),
			"History": Reverse(history.Results()),
		})
	})

	webapp.Post("/settings/import", app.ImportAgent(pool))
	webapp.Get("/settings/export/:name", app.ExportAgent(pool))

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
			return ctx.Status(401).Render("views/login", fiber.Map{})
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
