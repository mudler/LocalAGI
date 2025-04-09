package connectors

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/mudler/LocalAGI/services/connectors/twitter"
	"github.com/sashabaranov/go-openai"
)

type Twitter struct {
	token            string
	botUsername      string
	client           *twitter.TwitterClient
	noCharacterLimit bool
}

func (t *Twitter) AgentResultCallback() func(state types.ActionState) {
	return func(state types.ActionState) {

	}
}

func (t *Twitter) AgentReasoningCallback() func(state types.ActionCurrentState) bool {
	return func(state types.ActionCurrentState) bool {

		return true
	}
}

func NewTwitterConnector(config map[string]string) (*Twitter, error) {
	return &Twitter{
		token:            config["token"],
		botUsername:      config["botUsername"],
		client:           twitter.NewTwitterClient(config["token"]),
		noCharacterLimit: config["noCharacterLimit"] == "true",
	}, nil
}

func (t *Twitter) Start(a *agent.Agent) {
	ctx, cancel := signal.NotifyContext(a.Context(), os.Interrupt)
	defer cancel()

	// Step 1: Setup stream rules
	xlog.Info("Setting up stream rules...")
	err := t.client.AddStreamRule(t.botUsername)
	if err != nil {
		xlog.Error("Failed to add stream rule:", err)
	}

	// Step 2: Listen for mentions and respond
	fmt.Println("Listening for mentions...")

	go t.loop(ctx, a)

}

func (t *Twitter) loop(ctx context.Context, a *agent.Agent) {

	for {
		select {
		case <-ctx.Done():
			xlog.Info("Shutting down Twitter connector...")
			return

		default:
			if err := t.run(a); err != nil {
				xlog.Error("Error running Twitter connector", "err", err)
				return
			}
		}
	}

}

func (t *Twitter) run(a *agent.Agent) error {
	tweet, err := t.client.ListenForMentions()
	if err != nil {
		xlog.Error("Error getting mention", "error", err)
		return nil
	}

	xlog.Info("Got mention", "tweet", tweet)
	// Check if bot has already replied
	hasReplied, err := t.client.HasReplied(tweet.ID, t.botUsername)
	if err != nil {
		xlog.Error("Error checking if bot has replied", "error", err)
		return nil
	}

	if hasReplied {
		xlog.Info("Bot has already replied to this tweet")
		return nil
	}

	res := a.Ask(
		types.WithConversationHistory(
			[]openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: "You are replying to a twitter mention, keep answer short",
				},
				{
					Role:    "user",
					Content: tweet.Text,
				},
			},
		),
	)

	if res.Error != nil {
		xlog.Error("Error getting response from agent", "error", res.Error)
		return nil
	}

	if len(res.Response) > 280 && !t.noCharacterLimit {
		xlog.Error("Tweet is too long, max 280 characters")
		return nil
	}

	// Reply to tweet
	err = t.client.ReplyToTweet(tweet.ID, res.Response)
	if err != nil {
		xlog.Error("Error replying to tweet", "error", err)
		return nil
	}

	xlog.Debug("Replied successfully!")

	return nil
}

// TwitterConfigMeta returns the metadata for Twitter connector configuration fields
func TwitterConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "token",
			Label:    "Twitter API Token",
			Type:     config.FieldTypeText,
			Required: true,
		},
		{
			Name:     "botUsername",
			Label:    "Bot Username",
			Type:     config.FieldTypeText,
			Required: true,
		},
		{
			Name:  "noCharacterLimit",
			Label: "No Character Limit",
			Type:  config.FieldTypeCheckbox,
		},
	}
}
