package connector

import (
	"context"
	"errors"
	"os"
	"os/signal"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/mudler/local-agent-framework/agent"
)

type Telegram struct {
	Token      string
	lastChatID int64
	bot        *bot.Bot
	agent      *agent.Agent
}

// Send any text message to the bot after the bot has been started

func (t *Telegram) AgentResultCallback() func(state agent.ActionState) {
	return func(state agent.ActionState) {
		t.bot.SetMyDescription(t.agent.Context(), &bot.SetMyDescriptionParams{
			Description: state.Reasoning,
		})
	}
}

func (t *Telegram) AgentReasoningCallback() func(state agent.ActionCurrentState) bool {
	return func(state agent.ActionCurrentState) bool {
		t.bot.SetMyDescription(t.agent.Context(), &bot.SetMyDescriptionParams{
			Description: state.Reasoning,
		})
		return true
	}
}

func (t *Telegram) Start(a *agent.Agent) {
	ctx, cancel := signal.NotifyContext(a.Context(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			go func() {
				res := a.Ask(
					agent.WithText(
						update.Message.Text,
					),
				)
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   res.Response,
				})
				t.lastChatID = update.Message.Chat.ID
			}()
		}),
	}

	b, err := bot.New(t.Token, opts...)
	if err != nil {
		panic(err)
	}

	t.bot = b
	t.agent = a

	go func() {
		for m := range a.ConversationChannel() {
			if t.lastChatID == 0 {
				continue
			}
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: t.lastChatID,
				Text:   m.Content,
			})
		}
	}()

	b.Start(ctx)
}

func NewTelegramConnector(config map[string]string) (*Telegram, error) {
	token, ok := config["token"]
	if !ok {
		return nil, errors.New("token is required")
	}

	return &Telegram{
		Token: token,
	}, nil
}
