package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"

	"github.com/donseba/go-htmx"
	fiber "github.com/gofiber/fiber/v2"

	. "github.com/mudler/local-agent-framework/agent"
)

type (
	App struct {
		htmx *htmx.HTMX
		pool *AgentPool
	}
)

var testModel = os.Getenv("TEST_MODEL")
var apiURL = os.Getenv("API_URL")
var apiKey = os.Getenv("API_KEY")

func init() {
	if testModel == "" {
		testModel = "hermes-2-pro-mistral"
	}
	if apiURL == "" {
		apiURL = "http://192.168.68.113:8080"
	}
}

func htmlIfy(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", "<br>")
	return s
}

func main() {
	// current dir
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	stateDir := cwd + "/pool"
	os.MkdirAll(stateDir, 0755)

	pool, err := NewAgentPool(testModel, apiURL, stateDir)
	if err != nil {
		panic(err)
	}

	db, err := NewInMemoryDB(stateDir)
	if err != nil {
		panic(err)
	}

	// Reload store
	//	if err := db.SaveToStore(apiKey, apiURL); err != nil {
	//		fmt.Println("Error storing in the KB", err)
	//	}

	app := &App{
		htmx: htmx.New(),
		pool: pool,
	}

	if err := pool.StartAll(); err != nil {
		panic(err)
	}

	// Initialize a new Fiber app
	webapp := fiber.New()

	// Serve static files
	webapp.Static("/", "./public")

	webapp.Get("/", func(c *fiber.Ctx) error {
		return c.Render("index.html", fiber.Map{
			"Agents": pool.List(),
		})
	})

	webapp.Get("/create", func(c *fiber.Ctx) error {
		return c.Render("create.html", fiber.Map{
			"Title":      "Hello, World!",
			"Actions":    AvailableActions,
			"Connectors": AvailableConnectors,
		})
	})

	webapp.Get("/knowledgebase", func(c *fiber.Ctx) error {
		return c.Render("knowledgebase.html", fiber.Map{
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

	log.Fatal(webapp.Listen(":3000"))
}

func (a *App) KnowledgeBase(db *InMemoryDatabase) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		payload := struct {
			URL       string `json:"url"`
			ChunkSize int    `json:"chunk_size"`
		}{}

		if err := c.BodyParser(&payload); err != nil {
			return err
		}

		website := payload.URL
		if website == "" {
			return fmt.Errorf("please enter a URL")
		}

		go func() {
			content, err := Sitemap(website)
			if err != nil {
				fmt.Println("Error walking sitemap for website", err)
			}
			fmt.Println("Found pages: ", len(content))

			for _, c := range content {
				chunks := splitParagraphIntoChunks(c, payload.ChunkSize)
				fmt.Println("chunks: ", len(chunks))
				for _, chunk := range chunks {
					db.AddEntry(chunk)
				}

				db.SaveDB()
			}

			if err := db.SaveToStore(apiKey, apiURL); err != nil {
				fmt.Println("Error storing in the KB", err)
			}
		}()

		return c.Redirect("/knowledgebase")
	}
}

func (a *App) Notify(pool *AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		payload := struct {
			Message string `json:"message"`
		}{}

		if err := c.BodyParser(&payload); err != nil {
			return err
		}

		query := payload.Message
		if query == "" {
			_, _ = c.Write([]byte("Please enter a message."))
			return nil
		}

		agent := pool.GetAgent(c.Params("name"))
		agent.Ask(
			WithText(query),
		)
		_, _ = c.Write([]byte("Message sent"))

		return nil
	}
}

func (a *App) Delete(pool *AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		if err := pool.Remove(c.Params("name")); err != nil {
			fmt.Println("Error removing agent", err)
			return c.Status(http.StatusInternalServerError).SendString(err.Error())
		}
		return c.Redirect("/")
	}
}

func (a *App) Create(pool *AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		config := AgentConfig{}
		if err := c.BodyParser(&config); err != nil {
			return err
		}

		fmt.Printf("Agent configuration: %+v\n", config)

		if config.Name == "" {
			c.Status(http.StatusBadRequest).SendString("Name is required")
			return nil
		}
		if err := pool.CreateAgent(config.Name, &config); err != nil {
			c.Status(http.StatusInternalServerError).SendString(err.Error())
			return nil
		}
		return c.Redirect("/")
	}
}

func (a *App) Chat(pool *AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		payload := struct {
			Message string `json:"message"`
		}{}

		if err := c.BodyParser(&payload); err != nil {
			return err
		}
		agentName := c.Params("name")
		manager := pool.GetManager(agentName)

		query := payload.Message
		if query == "" {
			_, _ = c.Write([]byte("Please enter a message."))
			return nil
		}
		manager.Send(
			NewMessage(
				chatDiv(query, "gray"),
			).WithEvent("messages"))

		go func() {
			agent := pool.GetAgent(agentName)
			if agent == nil {
				fmt.Println("Agent not found in pool", c.Params("name"))
				return
			}
			res := agent.Ask(
				WithText(query),
			)
			fmt.Println("response is", res.Response)
			manager.Send(
				NewMessage(
					chatDiv(res.Response, "blue"),
				).WithEvent("messages"))
			manager.Send(
				NewMessage(
					disabledElement("inputMessage", false), // show again the input
				).WithEvent("message_status"))

			//result := `<i>done</i>`
			//	_, _ = w.Write([]byte(result))
		}()

		manager.Send(
			NewMessage(
				loader() + disabledElement("inputMessage", true),
			).WithEvent("message_status"))

		return nil
	}
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
