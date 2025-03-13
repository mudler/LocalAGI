package actions

import (
	"context"
	"fmt"

	"github.com/mudler/LocalAgent/core/action"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const (
	MetadataImages = "images_url"
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

func (a *GenImageAction) Run(ctx context.Context, params action.ActionParams) (action.ActionResult, error) {
	result := struct {
		Prompt string `json:"prompt"`
		Size   string `json:"size"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		return action.ActionResult{}, err
	}

	if result.Prompt == "" {
		return action.ActionResult{}, fmt.Errorf("prompt is required")
	}

	req := openai.ImageRequest{
		Prompt: result.Prompt,
		Model:  a.imageModel,
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
		return action.ActionResult{Result: "Failed to generate image " + err.Error()}, err
	}

	if len(resp.Data) == 0 {
		return action.ActionResult{Result: "Failed to generate image"}, nil
	}

	return action.ActionResult{
		Result: fmt.Sprintf("The image was generated and available at: %s", resp.Data[0].URL),
		Metadata: map[string]interface{}{
			MetadataImages: []string{resp.Data[0].URL},
		}}, nil
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
