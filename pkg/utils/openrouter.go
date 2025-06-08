package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/mudler/LocalAGI/pkg/xlog"
)

func init() {
	_ = godotenv.Load()
	openRouterAPIKey = os.Getenv("LOCALAGI_LLM_API_KEY")
}

var (
	openRouterAPIKey string
)

type OpenRouterUsage struct {
	Cost             float64 `json:"total_cost"`
	PromptTokens     int     `json:"tokens_prompt"`
	CompletionTokens int     `json:"tokens_completion"`
	TotalTokens      int     `json:"tokens_total"`
}

type OpenRouterResponse struct {
	Data struct {
		ID                     string  `json:"id"`
		TotalCost              float64 `json:"total_cost"`
		CreatedAt              string  `json:"created_at"`
		Model                  string  `json:"model"`
		Origin                 string  `json:"origin"`
		Usage                  float64 `json:"usage"`
		IsByok                 bool    `json:"is_byok"`
		UpstreamID             string  `json:"upstream_id"`
		CacheDiscount          float64 `json:"cache_discount"`
		AppID                  int     `json:"app_id"`
		Streamed               bool    `json:"streamed"`
		Cancelled              bool    `json:"cancelled"`
		ProviderName           string  `json:"provider_name"`
		Latency                int     `json:"latency"`
		ModerationLatency      int     `json:"moderation_latency"`
		GenerationTime         int     `json:"generation_time"`
		FinishReason           string  `json:"finish_reason"`
		NativeFinishReason     string  `json:"native_finish_reason"`
		TokensPrompt           int     `json:"tokens_prompt"`
		TokensCompletion       int     `json:"tokens_completion"`
		NativeTokensPrompt     int     `json:"native_tokens_prompt"`
		NativeTokensCompletion int     `json:"native_tokens_completion"`
		NativeTokensReasoning  int     `json:"native_tokens_reasoning"`
		NumMediaPrompt         int     `json:"num_media_prompt"`
		NumMediaCompletion     int     `json:"num_media_completion"`
		NumSearchResults       int     `json:"num_search_results"`
	} `json:"data"`
}

func GetOpenRouterUsage(generationID string) OpenRouterUsage {
	// If no generation ID is provided, return empty usage
	if generationID == "" {
		return OpenRouterUsage{}
	}

	// Make request to OpenRouter API
	url := fmt.Sprintf("https://openrouter.ai/api/v1/generation?id=%s", generationID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		xlog.Error("Error creating request to OpenRouter", "error", err)
		return OpenRouterUsage{}
	}

	req.Header.Add("Authorization", "Bearer "+openRouterAPIKey)

	println("OPENROUTER_API_KEY", openRouterAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		xlog.Error("Error making request to OpenRouter", "error", err)
		return OpenRouterUsage{}
	}
	defer resp.Body.Close()

	var result OpenRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		xlog.Error("Error parsing OpenRouter response", "error", err)
		return OpenRouterUsage{}
	}

	prettyJSON, err := json.MarshalIndent(result.Data, "", "  ")
	if err != nil {
		xlog.Error("Error pretty printing response", "error", err)
	} else {
		fmt.Println("OpenRouter Response Data:", string(prettyJSON))
	}

	return OpenRouterUsage{
		Cost:             result.Data.TotalCost,
		PromptTokens:     result.Data.TokensPrompt,
		CompletionTokens: result.Data.TokensCompletion,
		TotalTokens:      result.Data.TokensPrompt + result.Data.TokensCompletion,
	}
}
