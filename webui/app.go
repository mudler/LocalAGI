package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	coreTypes "github.com/mudler/LocalAgent/core/types"
	"github.com/mudler/LocalAgent/pkg/llm"
	"github.com/mudler/LocalAgent/pkg/xlog"
	"github.com/mudler/LocalAgent/services"
	"github.com/mudler/LocalAgent/webui/types"
	"github.com/sashabaranov/go-openai/jsonschema"

	"github.com/mudler/LocalAgent/core/sse"
	"github.com/mudler/LocalAgent/core/state"

	"github.com/donseba/go-htmx"
	fiber "github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

type (
	App struct {
		htmx   *htmx.HTMX
		config *Config
		*fiber.App
	}
)

func NewApp(opts ...Option) *App {
	config := NewConfig(opts...)
	engine := html.NewFileSystem(http.FS(viewsfs), ".html")

	// Initialize a new Fiber app
	// Pass the engine to the Views
	webapp := fiber.New(fiber.Config{
		Views: engine,
	})

	a := &App{
		htmx:   htmx.New(),
		config: config,
		App:    webapp,
	}

	a.registerRoutes(config.Pool, webapp)

	return a
}

func (a *App) Notify(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		payload := struct {
			Message string `form:"message"`
		}{}

		if err := c.BodyParser(&payload); err != nil {
			return err
		}

		query := payload.Message
		if query == "" {
			_, _ = c.Write([]byte("Please enter a message."))
			return nil
		}

		a := pool.GetAgent(c.Params("name"))
		a.Ask(
			coreTypes.WithText(query),
		)
		_, _ = c.Write([]byte("Message sent"))

		return nil
	}
}

func (a *App) Delete(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		if err := pool.Remove(c.Params("name")); err != nil {
			xlog.Info("Error removing agent", err)
			return errorJSONMessage(c, err.Error())
		}
		return statusJSONMessage(c, "ok")
	}
}

func errorJSONMessage(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusInternalServerError).JSON(struct {
		Error string `json:"error"`
	}{Error: message})
}

func statusJSONMessage(c *fiber.Ctx, message string) error {
	return c.JSON(struct {
		Status string `json:"status"`
	}{Status: message})
}

func (a *App) Pause(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		agent := pool.GetAgent(c.Params("name"))
		if agent != nil {
			xlog.Info("Pausing agent", "name", c.Params("name"))
			agent.Pause()
		}
		return statusJSONMessage(c, "ok")
	}
}

func (a *App) Start(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		agent := pool.GetAgent(c.Params("name"))
		if agent != nil {
			xlog.Info("Starting agent", "name", c.Params("name"))
			agent.Resume()
		}
		return statusJSONMessage(c, "ok")
	}
}

func (a *App) Create(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		config := state.AgentConfig{}
		if err := c.BodyParser(&config); err != nil {
			return errorJSONMessage(c, err.Error())
		}

		fmt.Printf("Agent configuration: %+v\n", config)

		if config.Name == "" {
			return errorJSONMessage(c, "Name is required")
		}
		if err := pool.CreateAgent(config.Name, &config); err != nil {
			return errorJSONMessage(c, err.Error())
		}
		return statusJSONMessage(c, "ok")
	}
}

// NEW FUNCTION: Get agent configuration
func (a *App) GetAgentConfig(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		config := pool.GetConfig(c.Params("name"))
		if config == nil {
			return errorJSONMessage(c, "Agent not found")
		}
		return c.JSON(config)
	}
}

// UpdateAgentConfig handles updating an agent's configuration
func (a *App) UpdateAgentConfig(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		agentName := c.Params("name")

		// First check if agent exists
		oldConfig := pool.GetConfig(agentName)
		if oldConfig == nil {
			return errorJSONMessage(c, "Agent not found")
		}

		// Parse the new configuration using the same approach as Create
		newConfig := state.AgentConfig{}
		if err := c.BodyParser(&newConfig); err != nil {
			xlog.Error("Error parsing agent config", "error", err)
			return errorJSONMessage(c, err.Error())
		}

		// Ensure the name doesn't change
		newConfig.Name = agentName

		// Remove the agent first
		if err := pool.Remove(agentName); err != nil {
			return errorJSONMessage(c, "Error removing agent: "+err.Error())
		}

		// Create agent with new config
		if err := pool.CreateAgent(agentName, &newConfig); err != nil {
			// Try to restore the old configuration if update fails
			if restoreErr := pool.CreateAgent(agentName, oldConfig); restoreErr != nil {
				return errorJSONMessage(c, fmt.Sprintf("Failed to update agent and restore failed: %v, %v", err, restoreErr))
			}
			return errorJSONMessage(c, "Error updating agent: "+err.Error())
		}

		xlog.Info("Updated agent", "name", agentName, "config", fmt.Sprintf("%+v", newConfig))

		return statusJSONMessage(c, "ok")
	}
}

func (a *App) ExportAgent(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		agent := pool.GetConfig(c.Params("name"))
		if agent == nil {
			return errorJSONMessage(c, "Agent not found")
		}

		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.json", agent.Name))
		return c.JSON(agent)
	}
}

func (a *App) ImportAgent(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		file, err := c.FormFile("file")
		if err != nil {
			// Handle error
			return err
		}

		os.MkdirAll("./uploads", os.ModePerm)

		destination := fmt.Sprintf("./uploads/%s", file.Filename)
		if err := c.SaveFile(file, destination); err != nil {
			// Handle error
			return err
		}

		data, err := os.ReadFile(destination)
		if err != nil {
			return err
		}

		config := state.AgentConfig{}
		if err := json.Unmarshal(data, &config); err != nil {
			return err
		}

		xlog.Info("Importing agent", config.Name)

		if config.Name == "" {
			return errorJSONMessage(c, "Name is required")
		}

		if err := pool.CreateAgent(config.Name, &config); err != nil {
			return errorJSONMessage(c, err.Error())
		}
		return statusJSONMessage(c, "ok")
	}
}

func (a *App) Chat(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		payload := struct {
			Message string `json:"message"`
		}{}

		if err := c.BodyParser(&payload); err != nil {
			return err
		}
		agentName := c.Params("name")
		manager := pool.GetManager(agentName)

		query := strings.Clone(payload.Message)
		if query == "" {
			_, _ = c.Write([]byte("Please enter a message."))
			return nil
		}
		manager.Send(
			sse.NewMessage(
				chatDiv(query, "gray"),
			).WithEvent("messages"))

		go func() {
			a := pool.GetAgent(agentName)
			if a == nil {
				xlog.Info("Agent not found in pool", c.Params("name"))
				return
			}
			res := a.Ask(
				coreTypes.WithText(query),
			)
			if res.Error != nil {
				xlog.Error("Error asking agent", "agent", agentName, "error", res.Error)
			} else {
				xlog.Info("we got a response from the agent", "agent", agentName, "response", res.Response)
			}
			manager.Send(
				sse.NewMessage(
					chatDiv(res.Response, "blue"),
				).WithEvent("messages"))
			manager.Send(
				sse.NewMessage(
					disabledElement("inputMessage", false), // show again the input
				).WithEvent("message_status"))

			//result := `<i>done</i>`
			//	_, _ = w.Write([]byte(result))
		}()

		manager.Send(
			sse.NewMessage(
				loader() + disabledElement("inputMessage", true),
			).WithEvent("message_status"))

		return nil
	}
}

func (a *App) ExecuteAction(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		payload := struct {
			Config map[string]string      `json:"config"`
			Params coreTypes.ActionParams `json:"params"`
		}{}

		if err := c.BodyParser(&payload); err != nil {
			xlog.Error("Error parsing action payload", "error", err)
			return errorJSONMessage(c, err.Error())
		}

		actionName := c.Params("name")

		xlog.Debug("Executing action", "action", actionName, "config", payload.Config, "params", payload.Params)
		a, err := services.Action(actionName, payload.Config, pool)
		if err != nil {
			xlog.Error("Error creating action", "error", err)
			return errorJSONMessage(c, err.Error())
		}

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Second)
		defer cancel()

		res, err := a.Run(ctx, payload.Params)
		if err != nil {
			xlog.Error("Error running action", "error", err)
			return errorJSONMessage(c, err.Error())
		}

		xlog.Info("Action executed", "action", actionName, "result", res)
		return c.JSON(res)
	}
}

func (a *App) ListActions() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		return c.JSON(services.AvailableActions)
	}
}

func (a *App) Responses(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var request types.RequestBody
		if err := c.BodyParser(&request); err != nil {
			return err
		}

		request.SetInputByType()

		agentName := request.Model

		messages := request.ToChatCompletionMessages()

		a := pool.GetAgent(agentName)
		if a == nil {
			xlog.Info("Agent not found in pool", c.Params("name"))
			return c.Status(http.StatusInternalServerError).JSON(types.ResponseBody{Error: "Agent not found"})
		}

		res := a.Ask(
			coreTypes.WithConversationHistory(messages),
		)
		if res.Error != nil {
			xlog.Error("Error asking agent", "agent", agentName, "error", res.Error)

			return c.Status(http.StatusInternalServerError).JSON(types.ResponseBody{Error: res.Error.Error()})
		} else {
			xlog.Info("we got a response from the agent", "agent", agentName, "response", res.Response)
		}

		response := types.ResponseBody{
			Object: "response",
			//   "created_at": 1741476542,
			CreatedAt: time.Now().Unix(),
			Status:    "completed",
			Output: []types.ResponseMessage{
				{
					Type:   "message",
					Status: "completed",
					Role:   "assistant",
					Content: []types.MessageContentItem{
						types.MessageContentItem{
							Type: "output_text",
							Text: res.Response,
						},
					},
				},
			},
		}

		return c.JSON(response)
	}
}

type AgentRole struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	SystemPrompt string `json:"system_prompt"`
}

func (a *App) GenerateGroupProfiles(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var request struct {
			Descript string `json:"description"`
		}

		if err := c.BodyParser(&request); err != nil {
			return errorJSONMessage(c, err.Error())
		}

		var results struct {
			Agents []AgentRole `json:"agents"`
		}

		xlog.Debug("Generating group", "description", request.Descript)
		client := llm.NewClient(a.config.LLMAPIKey, a.config.LLMAPIURL, "10m")
		err := llm.GenerateTypedJSON(c.Context(), client, request.Descript, a.config.LLMModel, jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"agents": {
					Type: jsonschema.Array,
					Items: &jsonschema.Definition{
						Type:     jsonschema.Object,
						Required: []string{"name", "description", "system_prompt"},
						Properties: map[string]jsonschema.Definition{
							"name": {
								Type:        jsonschema.String,
								Description: "The name of the agent",
							},
							"description": {
								Type:        jsonschema.String,
								Description: "The description of the agent",
							},
							"system_prompt": {
								Type:        jsonschema.String,
								Description: "The system prompt for the agent",
							},
						},
					},
				},
			},
		}, &results)
		if err != nil {
			return errorJSONMessage(c, err.Error())
		}

		return c.JSON(results.Agents)
	}
}

func (a *App) CreateGroup(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {

		var config struct {
			Agents      []AgentRole       `json:"agents"`
			AgentConfig state.AgentConfig `json:"agent_config"`
		}
		if err := c.BodyParser(&config); err != nil {
			return errorJSONMessage(c, err.Error())
		}

		agentConfig := &config.AgentConfig
		for _, agent := range config.Agents {
			xlog.Info("Creating agent", "name", agent.Name, "description", agent.Description)
			agentConfig.Name = agent.Name
			agentConfig.Description = agent.Description
			agentConfig.SystemPrompt = agent.SystemPrompt
			if err := pool.CreateAgent(agent.Name, agentConfig); err != nil {
				return errorJSONMessage(c, err.Error())
			}
		}

		return statusJSONMessage(c, "ok")
	}
}
