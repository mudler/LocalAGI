package main

import (
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/donseba/go-htmx"
	"github.com/donseba/go-htmx/sse"
	fiber "github.com/gofiber/fiber/v3"
	external "github.com/mudler/local-agent-framework/external"
	"github.com/valyala/fasthttp/fasthttpadaptor"

	. "github.com/mudler/local-agent-framework/agent"
)

type (
	App struct {
		htmx *htmx.HTMX
	}
)

var (
	sseManager Manager
)
var testModel = os.Getenv("TEST_MODEL")
var apiModel = os.Getenv("API_MODEL")

func init() {
	if testModel == "" {
		testModel = "hermes-2-pro-mistral"
	}
	if apiModel == "" {
		apiModel = "http://192.168.68.113:8080"
	}
}

func htmlIfy(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", "<br>")
	return s
}

var agentInstance *Agent

func main() {
	app := &App{
		htmx: htmx.New(),
	}

	agent, err := New(
		WithLLMAPIURL(apiModel),
		WithModel(testModel),
		EnableHUD,
		DebugMode,
		EnableStandaloneJob,
		WithAgentReasoningCallback(func(state ActionCurrentState) bool {
			sseManager.Send(
				sse.NewMessage(
					fmt.Sprintf(`Thinking: %s`, htmlIfy(state.Reasoning)),
				).WithEvent("status"),
			)
			return true
		}),
		WithActions(external.NewSearch(3)),
		WithAgentResultCallback(func(state ActionState) {
			text := fmt.Sprintf(`Reasoning: %s
			Action taken: %+v
			Parameters: %+v
			Result: %s`,
				state.Reasoning,
				state.ActionCurrentState.Action.Definition().Name,
				state.ActionCurrentState.Params,
				state.Result)
			sseManager.Send(
				sse.NewMessage(
					htmlIfy(
						text,
					),
				).WithEvent("status"),
			)
		}),
		WithRandomIdentity(),
		WithPeriodicRuns("10m"),
		//WithPermanentGoal("get the weather of all the cities in italy and store the results"),
	)
	if err != nil {
		panic(err)
	}
	go agent.Run()
	defer agent.Stop()

	agentInstance = agent
	sseManager = NewManager(5)

	go func() {
		for {
			clientsStr := ""
			clients := sseManager.Clients()
			for _, c := range clients {
				clientsStr += c + ", "
			}

			time.Sleep(1 * time.Second) // Send a message every seconds
			sseManager.Send(NewMessage(fmt.Sprintf("connected clients: %v", clientsStr)).WithEvent("clients"))
		}
	}()

	go func() {
		for {
			time.Sleep(1 * time.Second) // Send a message every seconds
			sseManager.Send(NewMessage(
				htmlIfy(agent.State().String()),
			).WithEvent("hud"))
		}
	}()

	// Initialize a new Fiber app
	webapp := fiber.New()

	// Define a route for the GET method on the root path '/'
	webapp.Get("/sse", func(c fiber.Ctx) error {
		sseManager.Handle(c, NewClient(randStringRunes(10)))
		return nil
	})
	webapp.Get("/notify", wrapHandler(http.HandlerFunc(app.Notify)))
	webapp.Post("/chat", wrapHandler(http.HandlerFunc(app.Chat(sseManager))))
	webapp.Get("/talk", wrapHandler(http.HandlerFunc(app.Home(agent))))
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

func (a *App) Home(agent *Agent) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("chat.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tmpl.Execute(w,
			struct {
				Character Character
			}{
				Character: agent.Character,
			})
	}
}

// func (a *App) SSE(w http.ResponseWriter, r *http.Request) {
// 	cl := sse.NewClient(randStringRunes(10))
// 	sseManager.Handle(w, r, cl)
// }

func (a *App) Notify(w http.ResponseWriter, r *http.Request) {
	query := strings.ToLower(r.PostFormValue("message"))
	if query == "" {
		_, _ = w.Write([]byte("Please enter a message."))
		return
	}

	agentInstance.Ask(
		WithText(query),
	)
	_, _ = w.Write([]byte("Message sent"))
}

func wrapHandler(f func(http.ResponseWriter, *http.Request)) func(ctx fiber.Ctx) error {
	return func(ctx fiber.Ctx) error {
		fasthttpadaptor.NewFastHTTPHandler(http.HandlerFunc(f))(ctx.Context())
		return nil
	}
}

func (a *App) Chat(m Manager) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query := strings.ToLower(r.PostFormValue("message"))
		if query == "" {
			_, _ = w.Write([]byte("Please enter a message."))
			return
		}
		m.Send(
			NewMessage(
				chatDiv(query, "gray"),
			).WithEvent("messages"))

		go func() {
			res := agentInstance.Ask(
				WithText(query),
			)
			fmt.Println("response is", res.Response)
			m.Send(
				sse.NewMessage(
					chatDiv(res.Response, "blue"),
				).WithEvent("messages"))
			m.Send(
				sse.NewMessage(
					inputMessageDisabled(false), // show again the input
				).WithEvent("message_status"))

			//result := `<i>done</i>`
			//	_, _ = w.Write([]byte(result))
		}()

		m.Send(
			sse.NewMessage(
				loader() + inputMessageDisabled(true),
			).WithEvent("message_status"))
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
