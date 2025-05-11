package actions

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/pkg/xstrings"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const (
	MetadataTelegramMessageSent = "telegram_message_sent"
	telegramMaxMessageLength    = 3000
)

type SendTelegramMessageRunner struct {
	token             string
	chatID            int64
	bot               *bot.Bot
	customName        string
	customDescription string
}

func NewSendTelegramMessageRunner(config map[string]string) *SendTelegramMessageRunner {
	token := config["token"]
	if token == "" {
		return nil
	}

	// Parse chat ID from config if present
	var chatID int64
	if configChatID := config["chat_id"]; configChatID != "" {
		var err error
		chatID, err = strconv.ParseInt(configChatID, 10, 64)
		if err != nil {
			return nil
		}
	}

	b, err := bot.New(token)
	if err != nil {
		return nil
	}

	return &SendTelegramMessageRunner{
		token:             token,
		chatID:            chatID,
		bot:               b,
		customName:        config["custom_name"],
		customDescription: config["custom_description"],
	}
}

type TelegramMessageParams struct {
	ChatID  int64  `json:"chat_id"`
	Message string `json:"message"`
}

func (s *SendTelegramMessageRunner) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	var messageParams TelegramMessageParams
	err := params.Unmarshal(&messageParams)
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to unmarshal params: %w", err)
	}

	if s.chatID != 0 {
		messageParams.ChatID = s.chatID
	}

	if messageParams.ChatID == 0 {
		return types.ActionResult{}, fmt.Errorf("chat_id is required either in config or parameters")
	}

	if messageParams.Message == "" {
		return types.ActionResult{}, fmt.Errorf("message is required")
	}

	// Split the message if it's too long
	messages := xstrings.SplitParagraph(messageParams.Message, telegramMaxMessageLength)

	if len(messages) == 0 {
		return types.ActionResult{}, fmt.Errorf("empty message after splitting")
	}

	// Send each message part
	for i, msg := range messages {
		_, err = s.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    messageParams.ChatID,
			Text:      msg,
			ParseMode: models.ParseModeMarkdown,
		})
		if err != nil {
			return types.ActionResult{}, fmt.Errorf("failed to send telegram message part %d: %w", i+1, err)
		}
	}

	sharedState.ConversationTracker.AddMessage(fmt.Sprintf("telegram:%d", messageParams.ChatID), openai.ChatCompletionMessage{
		Content: messageParams.Message,
		Role:    "assistant",
	})

	return types.ActionResult{
		Result: fmt.Sprintf("Message sent successfully to chat ID %d in %d parts", messageParams.ChatID, len(messages)),
		Metadata: map[string]interface{}{
			MetadataTelegramMessageSent: true,
		},
	}, nil
}

func (s *SendTelegramMessageRunner) Definition() types.ActionDefinition {

	customName := "send_telegram_message"
	if s.customName != "" {
		customName = s.customName
	}

	customDescription := "Send a message to a Telegram user or group"
	if s.customDescription != "" {
		customDescription = s.customDescription
	}

	if s.chatID != 0 {
		return types.ActionDefinition{
			Name:        types.ActionDefinitionName(customName),
			Description: customDescription,
			Properties: map[string]jsonschema.Definition{
				"message": {
					Type:        jsonschema.String,
					Description: "The message to send",
				},
			},
			Required: []string{"message"},
		}
	}

	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(customName),
		Description: customDescription,
		Properties: map[string]jsonschema.Definition{
			"chat_id": {
				Type:        jsonschema.Number,
				Description: "The Telegram chat ID to send the message to (optional if configured in config)",
			},
			"message": {
				Type:        jsonschema.String,
				Description: "The message to send",
			},
		},
		Required: []string{"message", "chat_id"},
	}
}

func (s *SendTelegramMessageRunner) Plannable() bool {
	return true
}

// SendTelegramMessageConfigMeta returns the metadata for Send Telegram Message action configuration fields
func SendTelegramMessageConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "token",
			Label:    "Telegram Token",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "Telegram bot token for sending messages",
		},
		{
			Name:     "chat_id",
			Label:    "Default Chat ID",
			Type:     config.FieldTypeText,
			Required: false,
			HelpText: "Default Telegram chat ID to send messages to (can be overridden in parameters)",
		},
		{
			Name:     "custom_name",
			Label:    "Custom Name",
			Type:     config.FieldTypeText,
			Required: false,
			HelpText: "Custom name for the action (optional, defaults to 'send_telegram_message')",
		},
		{
			Name:     "custom_description",
			Label:    "Custom Description",
			Type:     config.FieldTypeText,
			Required: false,
			HelpText: "Custom description for the action (optional, defaults to 'Send a message to a Telegram user or group')",
		},
	}
}
