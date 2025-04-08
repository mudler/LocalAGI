package actions

import (
	"context"
	"fmt"

	"github.com/mudler/LocalAgent/core/types"
	"github.com/mudler/LocalAgent/pkg/config"
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

func (a *GenImageAction) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Prompt string `json:"prompt"`
		Size   string `json:"size"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		return types.ActionResult{}, err
	}

	if result.Prompt == "" {
		return types.ActionResult{}, fmt.Errorf("prompt is required")
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
		return types.ActionResult{Result: "Failed to generate image " + err.Error()}, err
	}

	if len(resp.Data) == 0 {
		return types.ActionResult{Result: "Failed to generate image"}, nil
	}

	return types.ActionResult{
		Result: fmt.Sprintf("The image was generated and available at: %s", resp.Data[0].URL),
		Metadata: map[string]interface{}{
			MetadataImages: []string{resp.Data[0].URL},
		}}, nil
}

func (a *GenImageAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
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

func (a *GenImageAction) Plannable() bool {
	return true
}

// GenImageConfigMeta returns the metadata for GenImage action configuration fields
func GenImageConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "apiKey",
			Label:    "API Key",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "OpenAI API key for image generation",
		},
		{
			Name:     "apiURL",
			Label:    "API URL",
			Type:     config.FieldTypeText,
			Required: true,
			DefaultValue: "https://api.openai.com/v1",
			HelpText: "OpenAI API URL",
		},
		{
			Name:     "model",
			Label:    "Model",
			Type:     config.FieldTypeText,
			Required: true,
			DefaultValue: "dall-e-3",
			HelpText: "Image generation model to use (e.g., dall-e-3)",
		},
	}
}
