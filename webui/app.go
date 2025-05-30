package webui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	coreTypes "github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/db"
	models "github.com/mudler/LocalAGI/dbmodels"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/pkg/llm"
	"github.com/mudler/LocalAGI/pkg/utils"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/mudler/LocalAGI/services"
	"github.com/mudler/LocalAGI/services/connectors"
	"github.com/mudler/LocalAGI/webui/types"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"

	"github.com/mudler/LocalAGI/core/sse"
	"github.com/mudler/LocalAGI/core/state"

	"bytes"
	"io"

	"github.com/donseba/go-htmx"
	fiber "github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"gorm.io/gorm"
)

var (
	verificationKey      string
	privyAppId 			 string
	privyApiKey			 string
)



func init() {
	_ = godotenv.Load()
	var rawKey = os.Getenv("PRIVY_PUBLIC_KEY_PEM")
	verificationKey = strings.ReplaceAll(rawKey, `\n`, "\n")
	privyAppId = os.Getenv("PRIVY_APP_ID")
	privyApiKey = os.Getenv("PRIVY_APP_SECRET")
}

type (
	App struct {
		UserPools map[string]*state.AgentPool
		htmx      *htmx.HTMX
		config    *Config
		*fiber.App
	}
)

var poolRoot = "pools"

func NewApp(opts ...Option) *App {
	config := NewConfig(opts...)
	engine := html.NewFileSystem(http.FS(viewsfs), ".html")

	// Initialize a new Fiber app
	// Pass the engine to the Views
	webapp := fiber.New(fiber.Config{
		Views: engine,
	})

	a := &App{
		UserPools: make(map[string]*state.AgentPool),
		htmx:      htmx.New(),
		config:    config,
		App:       webapp,
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

func (a *App) Delete() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// 1. Get user ID
		userIDStr, ok := c.Locals("id").(string)
		if !ok || userIDStr == "" {
			return errorJSONMessage(c, "User ID missing")
		}
		userUUID, err := uuid.Parse(userIDStr)
		if err != nil {
			return errorJSONMessage(c, "Invalid user ID")
		}

		// 2. Get agent id
		agentId := c.Params("id")
		if agentId == "" {
			return errorJSONMessage(c, "Agent id is required")
		}

		// 3. Delete from DB
		if err := db.DB.
			Where("ID = ? AND UserID = ?", agentId, userUUID).
			Delete(&models.Agent{}).Error; err != nil {
			return errorJSONMessage(c, "Failed to delete agent from DB: "+err.Error())
		}

		// 4. Remove from in-memory pool if exists
		if pool, ok := a.UserPools[userIDStr]; ok {
			if err := pool.Remove(agentId); err != nil {
				xlog.Warn("Agent deleted from DB but failed to remove from memory", "error", err)
			}
		}

		xlog.Info("Agent deleted", "user", userIDStr, "agent", agentId)
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

func (a *App) Pause() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// 1. Get user ID
		userIDStr, ok := c.Locals("id").(string)
		if !ok || userIDStr == "" {
			return errorJSONMessage(c, "User ID missing")
		}

		// 2. Get agent id
		agentId := c.Params("id")
		if agentId == "" {
			return errorJSONMessage(c, "Agent id is required")
		}

		// 3. Get or init pool
		pool, ok := a.UserPools[userIDStr]
		if !ok {
			// Rehydrate pool from DB (no file fallback)
			newPool, err := state.NewAgentPool(
				userIDStr,
				os.Getenv("LOCALAGI_MODEL"),
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
			a.UserPools[userIDStr] = newPool
			pool = newPool
		}

		// 4. Pause agent if exists in memory
		if agent := pool.GetAgent(agentId); agent != nil {
			xlog.Info("Pausing agent", "Id", agentId)
			agent.Pause()
		} else {
			return errorJSONMessage(c, "Agent is not active in memory")
		}

		return statusJSONMessage(c, "ok")
	}
}


func (a *App) Start() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// 1. Get user ID
		userIDStr, ok := c.Locals("id").(string)
		if !ok || userIDStr == "" {
			return errorJSONMessage(c, "User ID missing")
		}

		// 2. Get agent id
		agentId := c.Params("id")
		if agentId == "" {
			return errorJSONMessage(c, "Agent id is required")
		}

		// 3. Load or create in-memory pool
		pool, ok := a.UserPools[userIDStr]
		if !ok {
			newPool, err := state.NewAgentPool(
				userIDStr,
				os.Getenv("LOCALAGI_MODEL"),
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
			a.UserPools[userIDStr] = newPool
			pool = newPool
		}

		// 4. Try to get the agent from memory
		agent := pool.GetAgent(agentId)
		
		if agent == nil {
			return errorJSONMessage(c, "Agent is not active in memory")
		}

		// 5. Resume agent
		xlog.Info("Starting agent", "id", agentId)
		agent.Resume()

		return statusJSONMessage(c, "ok")
	}
}


func (a *App) Create() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// 1. Get user ID
		userIDStr, ok := c.Locals("id").(string)
		if !ok || userIDStr == "" {
			return errorJSONMessage(c, "User ID missing")
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return errorJSONMessage(c, "Invalid user ID")
		}

		// 2. Parse request body
		var config state.AgentConfig
		if err := c.BodyParser(&config); err != nil {
			return errorJSONMessage(c, err.Error())
		}
		if config.Name == "" {
			return errorJSONMessage(c, "Name is required")
		}

		// 3. Serialize config to JSON
		configJSON, err := json.Marshal(config)
		if err != nil {
			return errorJSONMessage(c, "Failed to serialize config")
		}

		// 4. Store config in MySQL
		agent := models.Agent{
			ID:     uuid.New(),
			UserID: userID,
			Name:   config.Name,
			Config: configJSON,
		}

		if err := db.DB.Create(&agent).Error; err != nil {
			return errorJSONMessage(c, "Failed to store agent: " + err.Error())
		}

		// 5. Create in-memory agent pool (if needed)
		var pool *state.AgentPool
		if p, ok := a.UserPools[userIDStr]; ok {
			pool = p
		} else {
			pool, err = state.NewAgentPool(
				userIDStr,
				os.Getenv("LOCALAGI_MODEL"),
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
				return errorJSONMessage(c, "Failed to create agent pool: "+err.Error())
			}
			a.UserPools[userIDStr] = pool
		}

		// 6. Create agent in the pool
		if err := pool.CreateAgent(config.Name, &config); err != nil {
			return errorJSONMessage(c, "Failed to initialize agent: "+err.Error())
		}

		return statusJSONMessage(c, "ok")
	}
}


// NEW FUNCTION: Get agent configuration
func (a *App) GetAgentConfig() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// 1. Get user ID
		userID, ok := c.Locals("id").(string)
		if !ok || userID == "" {
			return errorJSONMessage(c, "User ID missing")
		}

		userUUID, err := uuid.Parse(userID)
		if err != nil {
			return errorJSONMessage(c, "Invalid user ID")
		}

		// 2. Get agent id from route param
		agentId := c.Params("id")
		if agentId == "" {
			return errorJSONMessage(c, "Agent id is required")
		}

		// 3. Fetch agent config from DB
		var agent models.Agent
		if err := db.DB.
			Where("ID = ? AND UserId = ?", agentId, userUUID).
			First(&agent).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error":   "Agent not found",
					"success": false,
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to query agent config",
				"success": false,
			})
		}

		// 4. Unmarshal config JSON
		var config state.AgentConfig
		if err := json.Unmarshal(agent.Config, &config); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to parse agent config",
				"success": false,
			})
		}

		// 5. Return the config
		return c.JSON(config)
	}
}


// UpdateAgentConfig handles updating an agent's configuration
func (a *App) UpdateAgentConfig() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// 1. Extract and validate user ID
		userIDStr, ok := c.Locals("id").(string)
		if !ok || userIDStr == "" {
			return errorJSONMessage(c, "User ID missing")
		}
		userUUID, err := uuid.Parse(userIDStr)
		if err != nil {
			return errorJSONMessage(c, "Invalid user ID")
		}

		// 2. Extract agent id
		agentId := c.Params("id")
		if agentId == "" {
			return errorJSONMessage(c, "Agent id is required")
		}

		// 3. Fetch agent from DB
		var agent models.Agent
		if err := db.DB.
			Where("ID = ? AND UserId = ?", agentId, userUUID).
			First(&agent).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errorJSONMessage(c, "Agent not found")
			}
			return errorJSONMessage(c, "Failed to fetch agent: "+err.Error())
		}

		// 4. Parse new config
		var newConfig state.AgentConfig
		if err := c.BodyParser(&newConfig); err != nil {
			xlog.Error("Error parsing agent config", "error", err)
			return errorJSONMessage(c, "Invalid agent config: "+err.Error())
		}

		newConfig.Name = agentId

		// 5. Update DB
		newConfigJSON, err := json.Marshal(newConfig)
		if err != nil {
			return errorJSONMessage(c, "Failed to serialize config")
		}
		agent.Config = newConfigJSON
		if err := db.DB.Save(&agent).Error; err != nil {
			return errorJSONMessage(c, "Failed to update config in DB: "+err.Error())
		}

		// 6. Reload in-memory agent if active
		pool, ok := a.UserPools[userIDStr]
		if ok {
			if err := pool.Remove(agentId); err != nil {
				xlog.Warn("Failed to remove old agent from pool", "error", err)
			}
			if err := pool.CreateAgent(agentId, &newConfig); err != nil {
				xlog.Error("Failed to recreate agent in memory", "error", err)
				return errorJSONMessage(c, "Agent config updated in DB but failed to reload in memory")
			}
		}

		xlog.Info("Updated agent", "id", agentId)
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

		// Safely save the file to prevent path traversal
		destination := filepath.Join("./uploads", file.Filename)
		if err := c.SaveFile(file, destination); err != nil {
			// Handle error
			return err
		}

		// Safely read the file
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

func (a *App) OldChat(pool *state.AgentPool) func(c *fiber.Ctx) error {
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

// Chat provides a JSON-based API for chat functionality
// This is designed to work better with the React UI
func (a *App) Chat(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// Parse the request body
		payload := struct {
			Message string `json:"message"`
		}{}

		if err := c.BodyParser(&payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(map[string]interface{}{
				"error": "Invalid request format",
			})
		}

		// Get agent name from URL parameter
		agentName := c.Params("name")

		// Validate message
		message := strings.TrimSpace(payload.Message)
		if message == "" {
			return c.Status(fiber.StatusBadRequest).JSON(map[string]interface{}{
				"error": "Message cannot be empty",
			})
		}

		// Get the agent from the pool
		agent := pool.GetAgent(agentName)
		if agent == nil {
			return c.Status(fiber.StatusNotFound).JSON(map[string]interface{}{
				"error": "Agent not found",
			})
		}

		// Get the SSE manager for this agent
		manager := pool.GetManager(agentName)

		// Create a unique message ID
		messageID := fmt.Sprintf("%d", time.Now().UnixNano())

		// Send user message event via SSE
		userMessageData, err := json.Marshal(map[string]interface{}{
			"id":        messageID + "-user",
			"sender":    "user",
			"content":   message,
			"timestamp": time.Now().Format(time.RFC3339),
		})
		if err != nil {
			xlog.Error("Error marshaling user message", "error", err)
		} else {
			manager.Send(
				sse.NewMessage(string(userMessageData)).WithEvent("json_message"))
		}

		// Send processing status
		statusData, err := json.Marshal(map[string]interface{}{
			"status":    "processing",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		if err != nil {
			xlog.Error("Error marshaling status message", "error", err)
		} else {
			manager.Send(
				sse.NewMessage(string(statusData)).WithEvent("json_status"))
		}

		// Process the message asynchronously
		go func() {
			// Ask the agent for a response
			response := agent.Ask(coreTypes.WithText(message))

			if response.Error != nil {
				// Send error message
				xlog.Error("Error asking agent", "agent", agentName, "error", response.Error)
				errorData, err := json.Marshal(map[string]interface{}{
					"error":     response.Error.Error(),
					"timestamp": time.Now().Format(time.RFC3339),
				})
				if err != nil {
					xlog.Error("Error marshaling error message", "error", err)
				} else {
					manager.Send(
						sse.NewMessage(string(errorData)).WithEvent("json_error"))
				}
			} else {
				// Send agent response
				xlog.Info("Response from agent", "agent", agentName, "response", response.Response)
				responseData, err := json.Marshal(map[string]interface{}{
					"id":        messageID + "-agent",
					"sender":    "agent",
					"content":   response.Response,
					"timestamp": time.Now().Format(time.RFC3339),
				})
				if err != nil {
					xlog.Error("Error marshaling agent response", "error", err)
				} else {
					manager.Send(
						sse.NewMessage(string(responseData)).WithEvent("json_message"))
				}
			}

			// Send completed status
			completedData, err := json.Marshal(map[string]interface{}{
				"status":    "completed",
				"timestamp": time.Now().Format(time.RFC3339),
			})
			if err != nil {
				xlog.Error("Error marshaling completed status", "error", err)
			} else {
				manager.Send(
					sse.NewMessage(string(completedData)).WithEvent("json_status"))
			}
		}()

		// Return immediate success response
		return c.Status(fiber.StatusAccepted).JSON(map[string]interface{}{
			"status":     "message_received",
			"message_id": messageID,
		})
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
		a, err := services.Action(actionName, "", payload.Config, pool)
		if err != nil {
			xlog.Error("Error creating action", "error", err)
			return errorJSONMessage(c, err.Error())
		}

		ctx, cancel := context.WithTimeout(c.Context(), 200*time.Second)
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

func (a *App) Responses(pool *state.AgentPool, tracker *connectors.ConversationTracker[string]) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var request types.RequestBody
		if err := c.BodyParser(&request); err != nil {
			return err
		}

		request.SetInputByType()

		var previousResponseID string
		conv := []openai.ChatCompletionMessage{}
		if request.PreviousResponseID != nil {
			previousResponseID = *request.PreviousResponseID
			conv = tracker.GetConversation(previousResponseID)
		}

		agentName := request.Model
		messages := append(conv, request.ToChatCompletionMessages()...)

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

		conv = append(conv, openai.ChatCompletionMessage{
			Role:    "assistant",
			Content: res.Response,
		})

		id := uuid.New().String()

		tracker.SetConversation(id, conv)

		response := types.ResponseBody{
			ID:     id,
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

// ListModels endpoint returns available models (local + filtered OpenRouter)
func (a *App) ListModels(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		allModels := getAvailableModels()
		return c.JSON(fiber.Map{"models": allModels})
	}
}

// GetAgentConfigMeta returns the metadata for agent configuration fields, including available models
func (a *App) GetAgentConfigMeta() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// 1. Get user ID
		userID, ok := c.Locals("id").(string)
		if !ok || userID == "" {
			return errorJSONMessage(c, "User ID missing from session")
		}

		// 2. Prepare config metadata for form rendering
		configMeta := state.NewAgentConfigMeta(
			services.ActionsConfigMeta(),
			services.ConnectorsConfigMeta(),
			services.DynamicPromptsConfigMeta(),
		)

		// 3. Add available models (could be filtered per-user later)
		models := getAvailableModels()
		for i, field := range configMeta.Fields {
			if field.Name == "model" {
				options := []config.FieldOption{}
				for _, m := range models {
					label := m["name"].(string)
					if src, ok := m["source"].(string); ok {
						label = "[" + src + "] " + label
					}
					options = append(options, config.FieldOption{
						Value: m["id"].(string),
						Label: label,
					})
				}
				configMeta.Fields[i].Options = options
			}
		}

		// 4. (Optional) Add user-specific flags/limits here if needed
		_ = userID // Placeholder if needed later

		return c.JSON(configMeta)
	}
}

// getLocalModels returns the local model(s) configured via environment variables
func getLocalModels() []map[string]interface{} {
	modelName := os.Getenv("MODEL_NAME")
	if modelName == "" {
		modelName = os.Getenv("LOCALAGI_MODEL")
	}
	if modelName == "" {
		return nil
	}
	return []map[string]interface{}{
		{"id": "local/" + modelName, "name": modelName, "description": "Local model: " + modelName},
	}
}

// getOpenRouterModels fetches and filters OpenRouter models for latest OpenAI, Anthropic, and Alibaba
func getOpenRouterModels() []map[string]interface{} {
	openrouterApiKey := os.Getenv("OPENROUTER_API_KEY")
	if openrouterApiKey == "" {
		return nil
	}
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+openrouterApiKey)
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer func() {
		io.Copy(io.Discard, resp.Body) // Ensure full body is read
		resp.Body.Close()
	}()

	var result struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}
	// Only allow latest OpenAI, Anthropic, and Alibaba models
	allowed := map[string]bool{
		"openai/gpt-4o":                        true,
		"openai/gpt-4-turbo":                   true,
		"openai/gpt-4.1":                       true,
		"openai/o4-mini":                       true,
		"openai/o4-mini-high":                  true,
		"openai/o3":                            true,
		"anthropic/claude-3.5-sonnet":          true,
		"anthropic/claude-3.7-sonnet":          true,
		"qwen/qwen-2.5-7b-instruct":            true,
		"qwen/qwq-32b":                         true,
		"qwen/qwen-2.5-72b-instruct":           true,
		"google/gemini-2.5-pro-exp-03-25:free": true,
		"deepseek/deepseek-chat-v3-0324:free":  true,
		"qwen/qwq-32b:free":                    true,
	}
	models := []map[string]interface{}{}
	for _, m := range result.Data {
		id, _ := m["id"].(string)
		if allowed[id] {
			m["source"] = "openrouter"
			m["id"] = "openrouter/" + id // Prefix to avoid collision
			models = append(models, m)
		}
	}
	return models
}

// getAvailableModels returns both local and filtered OpenRouter models
func getAvailableModels() []map[string]interface{} {
	localModels := getLocalModels()
	openrouterModels := getOpenRouterModels()
	return append(localModels, openrouterModels...)
}

// Proxy OpenRouter chat/completion endpoint
func (a *App) ProxyOpenRouterChat() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		apiKey := os.Getenv("OPENROUTER_API_KEY")
		if apiKey == "" {
			return c.Status(500).JSON(fiber.Map{"error": "OpenRouter API key not configured"})
		}
		// Forward the JSON body as-is
		body := c.Body()
		openrouterURL := "https://openrouter.ai/api/v1/chat/completions"
		if c.Query("type") == "completion" {
			openrouterURL = "https://openrouter.ai/api/v1/completions"
		}
		req, err := http.NewRequest("POST", openrouterURL, bytes.NewReader(body))
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer func() {
			io.Copy(io.Discard, resp.Body) // Ensure full body is read
			resp.Body.Close()
		}()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			xlog.Error("Error reading response body", "error", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to read response body"})
		}
		// xlog.Info("OpenRouter response status: %d", resp.StatusCode)
		// xlog.Info("OpenRouter response headers: %v", resp.Header)
		// xlog.Info("OpenRouter response body: %s", string(respBody))

		c.Status(resp.StatusCode)
		c.Set("Content-Type", resp.Header.Get("Content-Type"))
		c.Send(respBody)
		return nil
	}
}

// PrivyClaims holds the JWT fields
type PrivyClaims struct {
	AppId      string `json:"aud,omitempty"`
	Expiration uint64 `json:"exp,omitempty"`
	Issuer     string `json:"iss,omitempty"`
	UserId     string `json:"sub,omitempty"`
}

func (c *PrivyClaims) Valid() error {
	if c.AppId != privyAppId {
		return errors.New("aud claim must match your Privy App ID")
	}
	if c.Issuer != "privy.io" {
		return errors.New("iss claim must be 'privy.io'")
	}
	if c.Expiration < uint64(time.Now().Unix()) {
		return errors.New("token is expired")
	}
	return nil
}

func (a *App) RequireUser() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// 1. Get token from cookies
		tokenStr := c.Cookies("privy-token")
		if tokenStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Missing Privy token",
			})
		}

		// 2. Parse public key
		pubKey, err := jwt.ParseECPublicKeyFromPEM([]byte(verificationKey))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "Could not parse public key",
			})
		}

		// 3. Parse JWT
		claims := &PrivyClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != jwt.SigningMethodES256.Alg() {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return pubKey, nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Invalid or expired token",
			})
		}

		// 4. Validate claims
		if err := claims.Valid(); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "JWT validation failed: " + err.Error(),
			})
		}

		// 5. Find or create user
		var user models.User
		result := db.DB.First(&user, "privyID = ?", claims.UserId)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				// Fetch user info from Privy
				privyUser, err := utils.GetPrivyUserByDID(claims.UserId, privyAppId, privyApiKey)
				if err != nil {
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"success": false,
						"error":   "Failed to fetch user from Privy",
					})
				}

				user = models.User{
					Email:   privyUser.GetEmail(),
					PrivyID: claims.UserId,
				}

				if err := db.DB.Create(&user).Error; err != nil {
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"success": false,
						"error":   "Failed to create user",
					})
				}

				c.Locals("email", user.Email)
			} else {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"success": false,
					"error":   "DB error while fetching user",
				})
			}
		}

		// 6. Set user context
		c.Locals("id", user.ID.String())
		c.Locals("privyId", user.PrivyID)

		return c.Next()
	}
}



func (a *App) GetAgentDetails() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// 1. Get user ID
		userIDStr, ok := c.Locals("id").(string)
		if !ok || userIDStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User ID missing",
			})
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid user ID",
			})
		}

		// 2. Get agent id from URL param
		agentId := c.Params("id")
		if agentId == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Agent id is required",
			})
		}

		// 3. Look up agent config in MySQL
		var agent models.Agent
		if err := db.DB.
			Where("Id = ? AND UserId = ?", agentId, userID).
			First(&agent).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "Agent not found",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch agent",
			})
		}

		// 4. Check if agent is running in memory
		active := false
		if pool, ok := a.UserPools[userIDStr]; ok {
			if instance := pool.GetAgent(agentId); instance != nil {
				active = !instance.Paused()
			}
		}

		// 5. Return status
		return c.JSON(fiber.Map{
			"active":   active,
		})
	}
}

