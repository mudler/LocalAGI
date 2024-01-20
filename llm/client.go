package llm

import "github.com/sashabaranov/go-openai"

func NewClient(APIKey, URL string) *openai.Client {
	// Set up OpenAI client
	if APIKey == "" {
		//log.Fatal("OPENAI_API_KEY environment variable not set")
		APIKey = "sk-xxx"
	}
	config := openai.DefaultConfig(APIKey)
	config.BaseURL = URL
	return openai.NewClientWithConfig(config)
}
