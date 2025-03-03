package webui

import (
	"embed"
	"math/rand"
	"net/http"

	fiber "github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/mudler/LocalAgent/core/agent"
	"github.com/mudler/LocalAgent/core/sse"
	"github.com/mudler/LocalAgent/core/state"
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

	webapp.Get("/", func(c *fiber.Ctx) error {
		return c.Render("views/index", fiber.Map{
			"Agents": pool.List(),
		})
	})

	webapp.Get("/agents", func(c *fiber.Ctx) error {
		statuses := map[string]bool{}
		for _, a := range pool.List() {
			statuses[a] = !pool.GetAgent(a).Paused()
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
		return c.Render("views/settings", fiber.Map{
			//	"Character": agent.Character,
			"Name": c.Params("name"),
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
