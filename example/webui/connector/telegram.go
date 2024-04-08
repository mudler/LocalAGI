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
	Token    string
	Conttext context.Context
}

// Send any text message to the bot after the bot has been started

func (t *Telegram) AgentResultCallback() func(state agent.ActionState) {
	return func(state agent.ActionState) {
		// Send the result to the bot
	}
}
func (t *Telegram) AgentReasoningCallback() func(state agent.ActionCurrentState) bool {
	return func(state agent.ActionCurrentState) bool {
		// Send the reasoning to the bot
		return true
	}

}
func (t *Telegram) Start(a *agent.Agent) {
	ctx, cancel := signal.NotifyContext(a.Context(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			res := a.Ask(
				agent.WithText(
					update.Message.Text,
				),
			)
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   res.Response,
			})
		}),
	}

	b, err := bot.New(t.Token, opts...)
	if err != nil {
		panic(err)
	}

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
