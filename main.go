package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/mudler/LocalAGI/db"
	"github.com/mudler/LocalAGI/webui"
)

var (
	apiURL string
	apiKey string
)

func init() {
	_ = godotenv.Load()

	apiURL = os.Getenv("LOCALAGI_LLM_API_URL")
	apiKey = os.Getenv("LOCALAGI_LLM_API_KEY")

	if apiURL == "" {
		panic("LOCALAGI_LLM_API_URL not set")
	}
}

func main() {
	db.ConnectDB()

	app := webui.NewApp(
		webui.WithLLMAPIUrl(apiURL),
		webui.WithLLMAPIKey(apiKey),
	)

	log.Fatal(app.Listen(":3000"))
}
