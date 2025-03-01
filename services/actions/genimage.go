package actions

import (
	"context"
	"fmt"

	"github.com/mudler/LocalAgent/core/action"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func NewGenImage(config map[string]string) *GenImageAction {
	defaultConfig := openai.DefaultConfig(config["apiKey"])
	defaultConfig.BaseURL = config["apiURL"]

	return &GenImageAction{
		client:     openai.NewClientWithConfig(defaultConfig),
		imageModel: config["model"],
	}
}

type GenImageAction struct {
	client     *openai.Client
	imageModel string
}

func (a *GenImageAction) Run(ctx context.Context, params action.ActionParams) (string, error) {
	result := struct {
		Prompt string `json:"prompt"`
		Size   string `json:"size"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return "", err
	}

	req := openai.ImageRequest{
		Prompt: result.Prompt,
	}

	switch result.Size {
	case "256x256":
		req.Size = openai.CreateImageSize256x256
	case "512x512":
		req.Size = openai.CreateImageSize512x512
	case "1024x1024":
		req.Size = openai.CreateImageSize1024x1024
	default:
		req.Size = openai.CreateImageSize256x256
	}

	resp, err := a.client.CreateImage(ctx, req)
	if err != nil {
		return "Failed to generate image " + err.Error(), err
	}

	if len(resp.Data) == 0 {
		return "Failed to generate image", nil
	}

	return resp.Data[0].URL, nil
}

func (a *GenImageAction) Definition() action.ActionDefinition {
	return action.ActionDefinition{
		Name:        "generate_image",
		Description: "Generate image with.",
		Properties: map[string]jsonschema.Definition{
			"prompt": {
				Type:        jsonschema.String,
				Description: "The image prompt to generate the image.",
			},
			"size": {
				Type:        jsonschema.String,
				Description: "The image prompt to generate the image.",
				Enum:        []string{"256x256", "512x512", "1024x1024"},
			},
		},
		Required: []string{"prompt"},
	}
}
