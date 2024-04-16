package llm

import (
	"net/http"
	"time"

	"github.com/sashabaranov/go-openai"
)

func NewClient(APIKey, URL, timeout string) *openai.Client {
	// Set up OpenAI client
	if APIKey == "" {
		//log.Fatal("OPENAI_API_KEY environment variable not set")
		APIKey = "sk-xxx"
	}
	config := openai.DefaultConfig(APIKey)
	config.BaseURL = URL

	dur, err := time.ParseDuration(timeout)
	if err != nil {
		dur = 150 * time.Second
	}

	config.HTTPClient = &http.Client{
		Timeout: dur,
	}
	return openai.NewClientWithConfig(config)
}
