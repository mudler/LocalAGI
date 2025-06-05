package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/mudler/LocalAGI/db"
	"github.com/mudler/LocalAGI/webui"
)

var (
	baseModel            string
	apiURL               string
	apiKey               string
	timeout              string
	conversationDuration string
)

func init() {
	_ = godotenv.Load()

	baseModel = os.Getenv("LOCALAGI_MODEL")
	apiURL = os.Getenv("LOCALAGI_LLM_API_URL")
	apiKey = os.Getenv("LOCALAGI_LLM_API_KEY")
	timeout = os.Getenv("LOCALAGI_TIMEOUT")
	conversationDuration = os.Getenv("LOCALAGI_CONVERSATION_DURATION")

	if baseModel == "" {
		panic("LOCALAGI_MODEL not set")
	}
	if apiURL == "" {
		panic("LOCALAGI_API_URL not set")
	}
	if timeout == "" {
		timeout = "5m"
	}
}

func main() {
	db.ConnectDB()

	app := webui.NewApp(
		webui.WithConversationStoreduration(conversationDuration),
		webui.WithLLMAPIUrl(apiURL),
		webui.WithLLMAPIKey(apiKey),
		webui.WithLLMModel(baseModel),
	)

	log.Fatal(app.Listen(":3000"))
}
