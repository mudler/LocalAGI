package actions

import (
	"context"
	"fmt"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/services/connectors/twitter"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func NewPostTweet(config map[string]string) *PostTweetAction {
	return &PostTweetAction{
		token:            config["token"],
		noCharacterLimit: config["noCharacterLimits"] == "true",
	}
}

type PostTweetAction struct {
	token            string
	noCharacterLimit bool
}

func (a *PostTweetAction) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Text string `json:"text"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return types.ActionResult{}, err
	}

	if !a.noCharacterLimit && len(result.Text) > 280 {
		return types.ActionResult{}, fmt.Errorf("tweet is too long, max 280 characters")
	}

	client := twitter.NewTwitterClient(a.token)

	if err := client.Post(result.Text); err != nil {
		return types.ActionResult{}, err
	}

	return types.ActionResult{Result: fmt.Sprintf("twitter post created")}, nil
}

func (a *PostTweetAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        "post_tweet",
		Description: "Post a tweet",
		Properties: map[string]jsonschema.Definition{
			"text": {
				Type:        jsonschema.String,
				Description: "The text to send.",
			},
		},
		Required: []string{"text"},
	}
}

func (a *PostTweetAction) Plannable() bool {
	return true
}

// TwitterPostConfigMeta returns the metadata for Twitter Post action configuration fields
func TwitterPostConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "token",
			Label:    "Twitter API Token",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "Twitter API token for posting tweets",
		},
		{
			Name:     "noCharacterLimit",
			Label:    "No Character Limit",
			Type:     config.FieldTypeCheckbox,
			HelpText: "If checked, tweets longer than the character limit will be split into multiple tweets",
		},
	}
}
