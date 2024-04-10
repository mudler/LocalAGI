package main

import (
	"math/rand"

	fiber "github.com/gofiber/fiber/v2"
)

func RegisterRoutes(webapp *fiber.App, pool *AgentPool, db *InMemoryDatabase, app *App) {

	// Serve static files
	webapp.Static("/", "./public")

	webapp.Get("/", func(c *fiber.Ctx) error {
		return c.Render("views/index", fiber.Map{
			"Agents": pool.List(),
		})
	})

	webapp.Get("/agents", func(c *fiber.Ctx) error {
		return c.Render("views/agents", fiber.Map{
			"Agents": pool.List(),
		})
	})

	webapp.Get("/create", func(c *fiber.Ctx) error {
		return c.Render("views/create", fiber.Map{
			"Title":      "Hello, World!",
			"Actions":    AvailableActions,
			"Connectors": AvailableConnectors,
		})
	})

	webapp.Get("/knowledgebase", func(c *fiber.Ctx) error {
		return c.Render("views/knowledgebase", fiber.Map{
			"Title":                   "Hello, World!",
			"KnowledgebaseItemsCount": len(db.Database),
		})
	})

	// Define a route for the GET method on the root path '/'
	webapp.Get("/sse/:name", func(c *fiber.Ctx) error {

		m := pool.GetManager(c.Params("name"))
		if m == nil {
			return c.SendStatus(404)
		}

		m.Handle(c, NewClient(randStringRunes(10)))
		return nil
	})

	webapp.Get("/notify/:name", app.Notify(pool))
	webapp.Post("/chat/:name", app.Chat(pool))
	webapp.Post("/create", app.Create(pool))
	webapp.Get("/delete/:name", app.Delete(pool))
	webapp.Post("/knowledgebase", app.KnowledgeBase(db))

	webapp.Get("/talk/:name", func(c *fiber.Ctx) error {
		return c.Render("chat.html", fiber.Map{
			//	"Character": agent.Character,
			"Name": c.Params("name"),
		})
	})

}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
