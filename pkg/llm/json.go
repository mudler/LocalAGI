package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mudler/LocalAGI/db"
	models "github.com/mudler/LocalAGI/dbmodels"
	"github.com/mudler/LocalAGI/pkg/utils"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func GenerateTypedJSONWithGuidance(ctx context.Context, client *openai.Client, guidance, model string, userID, agentID uuid.UUID, i jsonschema.Definition, dst any) error {
	return GenerateTypedJSONWithConversation(ctx, client, []openai.ChatCompletionMessage{
		{
			Role:    "user",
			Content: guidance,
		},
	}, model, userID, agentID, i, dst)
}

func GenerateTypedJSONWithConversation(ctx context.Context, client *openai.Client, conv []openai.ChatCompletionMessage, model string, userID, agentID, i jsonschema.Definition, dst any) error {
	toolName := "json"
	decision := openai.ChatCompletionRequest{
		Model:    model,
		Messages: conv,
		Tools: []openai.Tool{
			{

				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:       toolName,
					Parameters: i,
				},
			},
		},
		ToolChoice: openai.ToolChoice{
			Type:     openai.ToolTypeFunction,
			Function: openai.ToolFunction{Name: toolName},
		},
	}

	resp, err := client.CreateChatCompletion(ctx, decision)

	if err == nil && len(resp.Choices) == 1 && resp.Choices[0].Message.Content != "" {
		// Track usage after successful API call
		usage := utils.GetOpenRouterUsage(resp.ID)
		llmUsage := &models.LLMUsage{
			ID:               uuid.New(),
			UserID:           userID,
			AgentID:          agentID,
			Model:            model,
			PromptTokens:     usage.PromptTokens,
			CompletionTokens: usage.CompletionTokens,
			TotalTokens:      usage.TotalTokens,
			Cost:             usage.Cost,
			RequestType:      "chat",
			GenID:            resp.ID,
			CreatedAt:        time.Now(),
		}
		if err := db.DB.Create(llmUsage).Error; err != nil {
			xlog.Error("Error tracking LLM usage", "error", err)
		}
	}

	if err != nil {
		return err
	}

	if len(resp.Choices) != 1 {
		return fmt.Errorf("no choices: %d", len(resp.Choices))
	}

	jsonSchema, _ := json.MarshalIndent(i, "", "  ")
	fmt.Println("JSON Schema:", string(jsonSchema))

	jsonResp, _ := json.MarshalIndent(resp.Choices, "", "  ")
	fmt.Println("Response choices:", string(jsonResp))

	msg := resp.Choices[0].Message

	if len(msg.ToolCalls) == 0 {
		return fmt.Errorf("no tool calls: %d", len(msg.ToolCalls))
	}

	fmt.Println("JSON generated", "Arguments", msg.ToolCalls)

	return json.Unmarshal([]byte(msg.ToolCalls[0].Function.Arguments), dst)
}
