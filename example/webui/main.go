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
	os.MkdirAll(cwd+"/pool", 0755)

	pool, err := NewAgentPool(testModel, apiURL, cwd+"/pool")
	if err != nil {
		panic(err)

	}
	app := &App{
		htmx: htmx.New(),
		pool: pool,
	}

	if err := pool.StartAll(); err != nil {
		panic(err)
	}

	// go func() {
	// 	for {
	// 		clientsStr := ""
	// 		clients := sseManager.Clients()
	// 		for _, c := range clients {
	// 			clientsStr += c + ", "
	// 		}

	// 		time.Sleep(1 * time.Second) // Send a message every seconds
	// 		sseManager.Send(NewMessage(fmt.Sprintf("connected clients: %v", clientsStr)).WithEvent("clients"))
	// 	}
	// }()

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
			"Title": "Hello, World!",
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

	webapp.Get("/talk/:name", func(c *fiber.Ctx) error {
		return c.Render("chat.html", fiber.Map{
			//	"Character": agent.Character,
			"Name": c.Params("name"),
		})
	})

	log.Fatal(webapp.Listen(":3000"))

	// mux := http.NewServeMux()

	// mux.Handle("GET /", http.HandlerFunc(app.Home(agent)))

	// // External notifications (e.g. webhook)
	// mux.Handle("POST /notify", http.HandlerFunc(app.Notify))

	// // User chat
	// mux.Handle("POST /chat", http.HandlerFunc(app.Chat(sseManager)))

	// // Server Sent Events
	// //mux.Handle("GET /sse", http.HandlerFunc(app.SSE))

	// fmt.Print("Server started at http://localhost:3210")
	// err = http.ListenAndServe(":3210", mux)
	// log.Fatal(err)
}

// func (a *App) SSE(w http.ResponseWriter, r *http.Request) {
// 	cl := sse.NewClient(randStringRunes(10))
// 	sseManager.Handle(w, r, cl)
// }

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

func (a *App) Create(pool *AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		config := AgentConfig{}
		if err := c.BodyParser(&config); err != nil {
			return err
		}

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
					inputMessageDisabled(false), // show again the input
				).WithEvent("message_status"))

			//result := `<i>done</i>`
			//	_, _ = w.Write([]byte(result))
		}()

		manager.Send(
			NewMessage(
				loader() + inputMessageDisabled(true),
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
