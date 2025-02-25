package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/mudler/local-agent-framework/pkg/xlog"

	"github.com/mudler/local-agent-framework/core/agent"
	"github.com/mudler/local-agent-framework/core/sse"
	"github.com/mudler/local-agent-framework/core/state"

	"github.com/donseba/go-htmx"
	"github.com/dslipak/pdf"
	fiber "github.com/gofiber/fiber/v2"
)

type (
	App struct {
		htmx *htmx.HTMX
		pool *state.AgentPool
	}
)

func (a *App) KnowledgeBaseReset(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		db := pool.GetAgentMemory(c.Params("name"))
		db.Reset()
		return c.Redirect("/knowledgebase/" + c.Params("name"))
	}
}

func (a *App) KnowledgeBaseExport(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		db := pool.GetAgentMemory(c.Params("name"))
		knowledgeBase := db.Data()

		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.knowledgebase.json", c.Params("name")))
		return c.JSON(knowledgeBase)
	}
}

func (a *App) KnowledgeBaseImport(pool *state.AgentPool) func(c *fiber.Ctx) error {
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

		knowledge := []string{}
		if err := json.Unmarshal(data, &knowledge); err != nil {
			return err
		}

		if len(knowledge) > 0 {
			xlog.Info("Importing agent KB")
			db := pool.GetAgentMemory(c.Params("name"))
			db.Reset()

			for _, k := range knowledge {
				db.Store(k)
			}

		} else {
			return fmt.Errorf("Empty knowledge base")
		}

		return c.Redirect("/agents")
	}
}

func (a *App) KnowledgeBaseFile(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		agent := pool.GetAgent(c.Params("name"))
		db := agent.Memory()

		// https://golang.withcodeexample.com/blog/file-upload-handling-golang-fiber-guide/
		file, err := c.FormFile("file")
		if err != nil {
			// Handle error
			return err
		}

		payload := struct {
			ChunkSize int `form:"chunk_size"`
		}{}

		if err := c.BodyParser(&payload); err != nil {
			return err
		}

		os.MkdirAll("./uploads", os.ModePerm)

		destination := fmt.Sprintf("./uploads/%s", file.Filename)
		if err := c.SaveFile(file, destination); err != nil {
			// Handle error
			return err
		}

		xlog.Info("File uploaded to: " + destination)
		fmt.Printf("Payload: %+v\n", payload)

		content, err := readPdf(destination) // Read local pdf file
		if err != nil {
			panic(err)
		}

		xlog.Info("Content is", content)
		chunkSize := defaultChunkSize
		if payload.ChunkSize > 0 {
			chunkSize = payload.ChunkSize
		}

		go state.StringsToKB(db, chunkSize, content)

		_, err = c.WriteString(chatDiv("File uploaded", "gray"))

		return err
	}
}

func (a *App) KnowledgeBase(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		agent := pool.GetAgent(c.Params("name"))
		db := agent.Memory()

		payload := struct {
			URL       string `form:"url"`
			ChunkSize int    `form:"chunk_size"`
		}{}

		if err := c.BodyParser(&payload); err != nil {
			return err
		}

		website := payload.URL
		if website == "" {
			return fmt.Errorf("please enter a URL")
		}
		chunkSize := defaultChunkSize
		if payload.ChunkSize > 0 {
			chunkSize = payload.ChunkSize
		}

		go state.WebsiteToKB(website, chunkSize, db)

		return c.Redirect("/knowledgebase/" + c.Params("name"))
	}
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
			agent.WithText(query),
		)
		_, _ = c.Write([]byte("Message sent"))

		return nil
	}
}

func (a *App) Delete(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		if err := pool.Remove(c.Params("name")); err != nil {
			xlog.Info("Error removing agent", err)
			return c.Status(http.StatusInternalServerError).SendString(err.Error())
		}
		return c.Redirect("/agents")
	}
}

func (a *App) Pause(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		xlog.Info("Pausing agent", c.Params("name"))
		agent := pool.GetAgent(c.Params("name"))
		if agent != nil {
			agent.Pause()
		}
		return c.Redirect("/agents")
	}
}

func (a *App) Start(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		agent := pool.GetAgent(c.Params("name"))
		if agent != nil {
			agent.Resume()
		}
		return c.Redirect("/agents")
	}
}

func (a *App) Create(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		config := state.AgentConfig{}
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
		return c.Redirect("/agents")
	}
}

func (a *App) ExportAgent(pool *state.AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		agent := pool.GetConfig(c.Params("name"))
		if agent == nil {
			return c.Status(http.StatusNotFound).SendString("Agent not found")
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
			c.Status(http.StatusBadRequest).SendString("Name is required")
			return nil
		}

		if err := pool.CreateAgent(config.Name, &config); err != nil {
			c.Status(http.StatusInternalServerError).SendString(err.Error())
			return nil
		}
		return c.Redirect("/agents")
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
				agent.WithText(query),
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

func readPdf(path string) (string, error) {
	r, err := pdf.Open(path)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	b, err := r.GetPlainText()
	if err != nil {
		return "", err
	}
	buf.ReadFrom(b)
	return buf.String(), nil
}
