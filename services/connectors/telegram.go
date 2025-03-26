package connectors

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/mudler/LocalAgent/core/agent"
	"github.com/mudler/LocalAgent/core/types"
	"github.com/mudler/LocalAgent/pkg/xlog"
	"github.com/mudler/LocalAgent/pkg/xstrings"
	"github.com/mudler/LocalAgent/services/actions"
	"github.com/sashabaranov/go-openai"
)

type Telegram struct {
	Token string
	bot   *bot.Bot
	agent *agent.Agent

	currentconversation map[int64][]openai.ChatCompletionMessage
	lastMessageTime     map[int64]time.Time
	lastMessageDuration time.Duration

	admins []string

	conversationTracker *ConversationTracker[int64]
}

// Send any text message to the bot after the bot has been started

func (t *Telegram) AgentResultCallback() func(state types.ActionState) {
	return func(state types.ActionState) {
		t.bot.SetMyDescription(t.agent.Context(), &bot.SetMyDescriptionParams{
			Description: state.Reasoning,
		})
	}
}

func (t *Telegram) AgentReasoningCallback() func(state types.ActionCurrentState) bool {
	return func(state types.ActionCurrentState) bool {
		t.bot.SetMyDescription(t.agent.Context(), &bot.SetMyDescriptionParams{
			Description: state.Reasoning,
		})
		return true
	}
}

func (t *Telegram) handleUpdate(ctx context.Context, b *bot.Bot, a *agent.Agent, update *models.Update) {
	username := update.Message.From.Username

	if len(t.admins) > 0 && !slices.Contains(t.admins, username) {
		xlog.Info("Unauthorized user", "username", username)
		return
	}

	currentConv := t.conversationTracker.GetConversation(update.Message.From.ID)
	currentConv = append(currentConv, openai.ChatCompletionMessage{
		Content: update.Message.Text,
		Role:    "user",
	})

	t.conversationTracker.AddMessage(
		update.Message.From.ID,
		openai.ChatCompletionMessage{
			Content: update.Message.Text,
			Role:    "user",
		},
	)

	xlog.Info("New message", "username", username, "conversation", currentConv)
	res := a.Ask(
		types.WithConversationHistory(currentConv),
	)

	xlog.Debug("Response", "response", res.Response)

	if res.Response == "" {
		xlog.Error("Empty response from agent")
		return
	}

	t.conversationTracker.AddMessage(
		update.Message.From.ID,
		openai.ChatCompletionMessage{
			Content: res.Response,
			Role:    "assistant",
		},
	)

	xlog.Debug("Sending message back to telegram", "response", res.Response)

	for _, res := range res.State {
		// coming from the search action
		// if urls, exists := res.Metadata[actions.MetadataUrls]; exists {
		// 	for _, url := range uniqueStringSlice(urls.([]string)) {

		// 	}
		// }

		// coming from the gen image actions
		if imagesUrls, exists := res.Metadata[actions.MetadataImages]; exists {
			for _, url := range xstrings.UniqueSlice(imagesUrls.([]string)) {
				xlog.Debug("Sending photo", "url", url)

				resp, err := http.Get(url)
				if err != nil {
					xlog.Error("Error downloading image", "error", err.Error())
					continue
				}
				defer resp.Body.Close()
				_, err = b.SendPhoto(ctx, &bot.SendPhotoParams{
					ChatID: update.Message.Chat.ID,
					Photo: models.InputFileUpload{
						Filename: "image.jpg",
						Data:     resp.Body,
					},
				})
				if err != nil {
					xlog.Error("Error sending photo", "error", err.Error())
				}
			}
		}

	}
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		//	ParseMode: models.ParseModeMarkdown,
		ChatID: update.Message.Chat.ID,
		Text:   res.Response,
	})
	if err != nil {
		xlog.Error("Error sending message", "error", err)
	}
}

// func (t *Telegram) handleNewMessage(ctx context.Context, b *bot.Bot, m openai.ChatCompletionMessage) {
// 	if t.lastChatID == 0 {
// 		return
// 	}
// 	b.SendMessage(ctx, &bot.SendMessageParams{
// 		ChatID: t.lastChatID,
// 		Text:   m.Content,
// 	})
// }

func (t *Telegram) Start(a *agent.Agent) {
	ctx, cancel := signal.NotifyContext(a.Context(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			go t.handleUpdate(ctx, b, a, update)
		}),
	}

	b, err := bot.New(t.Token, opts...)
	if err != nil {
		panic(err)
	}

	t.bot = b
	t.agent = a

	// go func() {
	// 	for m := range a.ConversationChannel() {
	// 		t.handleNewMessage(ctx, b, m)
	// 	}
	// }()

	b.Start(ctx)
}

func NewTelegramConnector(config map[string]string) (*Telegram, error) {
	token, ok := config["token"]
	if !ok {
		return nil, errors.New("token is required")
	}

	duration, err := time.ParseDuration(config["lastMessageDuration"])
	if err != nil {
		duration = 5 * time.Minute
	}

	admins := []string{}

	if _, ok := config["admins"]; ok {
		admins = append(admins, strings.Split(config["admins"], ",")...)
	}

	return &Telegram{
		Token:               token,
		lastMessageDuration: duration,
		admins:              admins,
		currentconversation: map[int64][]openai.ChatCompletionMessage{},
		lastMessageTime:     map[int64]time.Time{},
		conversationTracker: NewConversationTracker[int64](duration),
	}, nil
}
