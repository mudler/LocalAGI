package webui

import (
	"crypto/subtle"
	"embed"
	"errors"
	"math/rand"
	"net/http"

	"github.com/dave-gray101/v2keyauth"
	fiber "github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/keyauth"
	"github.com/mudler/LocalAgent/core/agent"
	"github.com/mudler/LocalAgent/core/sse"
	"github.com/mudler/LocalAgent/core/state"
	"github.com/mudler/LocalAgent/pkg/xlog"
	"github.com/mudler/LocalAgent/services"
)

//go:embed views/*
var viewsfs embed.FS

//go:embed public/*
var embeddedFiles embed.FS

func (app *App) registerRoutes(pool *state.AgentPool, webapp *fiber.App) {

	webapp.Use("/public", filesystem.New(filesystem.Config{
		Root:       http.FS(embeddedFiles),
		PathPrefix: "public",
		Browse:     true,
	}))

	if len(app.config.ApiKeys) > 0 {
		kaConfig, err := GetKeyAuthConfig(app.config.ApiKeys)
		if err != nil || kaConfig == nil {
			panic(err)
		}
		webapp.Use(v2keyauth.New(*kaConfig))
	}

	webapp.Get("/", func(c *fiber.Ctx) error {
		return c.Render("views/index", fiber.Map{
			"Agents":     pool.List(),
			"AgentCount": len(pool.List()),
			"Actions":    len(services.AvailableActions),
			"Connectors": len(services.AvailableConnectors),
		})
	})

	webapp.Get("/agents", func(c *fiber.Ctx) error {
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

	webapp.Get("/create", func(c *fiber.Ctx) error {
		return c.Render("views/create", fiber.Map{
			"Actions":      services.AvailableActions,
			"Connectors":   services.AvailableConnectors,
			"PromptBlocks": services.AvailableBlockPrompts,
		})
	})

	// Define a route for the GET method on the root path '/'
	webapp.Get("/sse/:name", func(c *fiber.Ctx) error {
		m := pool.GetManager(c.Params("name"))
		if m == nil {
			return c.SendStatus(404)
		}

		m.Handle(c, sse.NewClient(randStringRunes(10)))
		return nil
	})

	webapp.Get("/status/:name", func(c *fiber.Ctx) error {
		history := pool.GetStatusHistory(c.Params("name"))
		if history == nil {
			history = &state.Status{ActionResults: []agent.ActionState{}}
		}
		// reverse history

		return c.Render("views/status", fiber.Map{
			"Name":    c.Params("name"),
			"History": Reverse(history.Results()),
		})
	})

	webapp.Get("/notify/:name", app.Notify(pool))
	webapp.Post("/chat/:name", app.Chat(pool))
	webapp.Post("/create", app.Create(pool))
	webapp.Delete("/delete/:name", app.Delete(pool))
	webapp.Put("/pause/:name", app.Pause(pool))
	webapp.Put("/start/:name", app.Start(pool))

	webapp.Get("/talk/:name", func(c *fiber.Ctx) error {
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

	// New API endpoints for getting and updating agent configuration
	webapp.Get("/api/agent/:name/config", app.GetAgentConfig(pool))
	webapp.Put("/api/agent/:name/config", app.UpdateAgentConfig(pool))

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
