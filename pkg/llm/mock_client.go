package llm

import (
	"context"
	"github.com/sashabaranov/go-openai"
)

type MockClient struct {
	CreateChatCompletionFunc func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
	CreateImageFunc func(ctx context.Context, req openai.ImageRequest) (openai.ImageResponse, error)
}

func (m *MockClient) CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	if m.CreateChatCompletionFunc != nil {
		return m.CreateChatCompletionFunc(ctx, req)
	}
	return openai.ChatCompletionResponse{}, nil
}

func (m *MockClient) CreateImage(ctx context.Context, req openai.ImageRequest) (openai.ImageResponse, error) {
	if m.CreateImageFunc != nil {
		return m.CreateImageFunc(ctx, req)
	}
	return openai.ImageResponse{}, nil
}
