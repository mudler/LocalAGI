package llm

import (
	"context"
	"net/http"
	"time"

	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/sashabaranov/go-openai"
)

type LLMClient interface {
	CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
	CreateImage(ctx context.Context, req openai.ImageRequest) (openai.ImageResponse, error)
}

type realClient struct {
	*openai.Client
}

func (r *realClient) CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	return r.Client.CreateChatCompletion(ctx, req)
}

func (r *realClient) CreateImage(ctx context.Context, req openai.ImageRequest) (openai.ImageResponse, error) {
	return r.Client.CreateImage(ctx, req)
}

// NewClient returns a real OpenAI client as LLMClient
func NewClient(APIKey, URL, timeout string) LLMClient {
	// Set up OpenAI client
	if APIKey == "" {
		//log.Fatal("OPENAI_API_KEY environment variable not set")
		APIKey = "sk-xxx"
	}
	config := openai.DefaultConfig(APIKey)
	config.BaseURL = URL

	dur, err := time.ParseDuration(timeout)
	if err != nil {
		xlog.Error("Failed to parse timeout", "error", err)
		dur = 150 * time.Second
	}

	config.HTTPClient = &http.Client{
		Timeout: dur,
	}
	return &realClient{openai.NewClientWithConfig(config)}
}
