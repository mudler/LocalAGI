package connectors

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/mudler/LocalAGI/pkg/xstrings"
	"github.com/mudler/LocalAGI/services/actions"
	"github.com/sashabaranov/go-openai"
)

const telegramThinkingMessage = "🤔 thinking..."

type Telegram struct {
	Token string
	bot   *bot.Bot
	agent *agent.Agent

	currentconversation map[int64][]openai.ChatCompletionMessage
	lastMessageTime     map[int64]time.Time
	lastMessageDuration time.Duration

	admins []string

	conversationTracker *ConversationTracker[int64]

	// To track placeholder messages
	placeholders     map[string]int // map[jobUUID]messageID
	placeholderMutex sync.RWMutex

	// Track active jobs for cancellation
	activeJobs      map[int64][]*types.Job // map[chatID]bool to track if a chat has active processing
	activeJobsMutex sync.RWMutex
}

// Send any text message to the bot after the bot has been started

func (t *Telegram) AgentResultCallback() func(state types.ActionState) {
	return func(state types.ActionState) {
		// Mark the job as completed when we get the final result
		if state.ActionCurrentState.Job != nil && state.ActionCurrentState.Job.Metadata != nil {
			if chatID, ok := state.ActionCurrentState.Job.Metadata["chatID"].(int64); ok && chatID != 0 {
				t.activeJobsMutex.Lock()
				delete(t.activeJobs, chatID)
				t.activeJobsMutex.Unlock()
			}
		}
	}
}

func (t *Telegram) AgentReasoningCallback() func(state types.ActionCurrentState) bool {
	return func(state types.ActionCurrentState) bool {
		// Check if we have a placeholder message for this job
		t.placeholderMutex.RLock()
		msgID, exists := t.placeholders[state.Job.UUID]
		chatID := int64(0)
		if state.Job.Metadata != nil {
			if ch, ok := state.Job.Metadata["chatID"].(int64); ok {
				chatID = ch
			}
		}
		t.placeholderMutex.RUnlock()

		if !exists || msgID == 0 || chatID == 0 || t.bot == nil {
			return true // Skip if we don't have a message to update
		}

		thought := telegramThinkingMessage + "\n\n"
		if state.Reasoning != "" {
			thought += "Current thought process:\n" + state.Reasoning
		}

		// Update the placeholder message with the current reasoning
		_, err := t.bot.EditMessageText(t.agent.Context(), &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: msgID,
			Text:      thought,
		})
		if err != nil {
			xlog.Error("Error updating reasoning message", "error", err)
		}
		return true
	}
}

// cancelActiveJobForChat cancels any active job for the given chat
func (t *Telegram) cancelActiveJobForChat(chatID int64) {
	t.activeJobsMutex.RLock()
	ctxs, exists := t.activeJobs[chatID]
	t.activeJobsMutex.RUnlock()

	if exists {
		xlog.Info("Cancelling active job for chat", "chatID", chatID)

		// Mark the job as inactive
		t.activeJobsMutex.Lock()
		for _, c := range ctxs {
			c.Cancel()
		}
		delete(t.activeJobs, chatID)
		t.activeJobsMutex.Unlock()
	}
}

func (t *Telegram) handleUpdate(ctx context.Context, b *bot.Bot, a *agent.Agent, update *models.Update) {
	username := update.Message.From.Username

	if len(t.admins) > 0 && !slices.Contains(t.admins, username) {
		xlog.Info("Unauthorized user", "username", username)
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "you are not authorized to use this bot!",
		})
		if err != nil {
			xlog.Error("Error sending unauthorized message", "error", err)
		}
		return
	}

	// Cancel any active job for this chat before starting a new one
	t.cancelActiveJobForChat(update.Message.Chat.ID)

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

	// Send initial placeholder message
	msg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   telegramThinkingMessage,
	})
	if err != nil {
		xlog.Error("Error sending initial message", "error", err)
		return
	}

	// Store the UUID->placeholder message mapping
	jobUUID := fmt.Sprintf("%d", msg.ID)

	t.placeholderMutex.Lock()
	t.placeholders[jobUUID] = msg.ID
	t.placeholderMutex.Unlock()

	// Add chat ID to metadata for tracking
	metadata := map[string]interface{}{
		"chatID": update.Message.Chat.ID,
	}

	// Create a new job with the conversation history and metadata
	job := types.NewJob(
		types.WithConversationHistory(currentConv),
		types.WithUUID(jobUUID),
		types.WithMetadata(metadata),
	)

	// Mark this chat as having an active job
	t.activeJobsMutex.Lock()
	t.activeJobs[update.Message.Chat.ID] = append(t.activeJobs[update.Message.Chat.ID], job)
	t.activeJobsMutex.Unlock()

	defer func() {
		// Mark job as complete
		t.activeJobsMutex.Lock()
		job.Cancel()
		for i, j := range t.activeJobs[update.Message.Chat.ID] {
			if j.UUID == job.UUID {
				t.activeJobs[update.Message.Chat.ID] = append(t.activeJobs[update.Message.Chat.ID][:i], t.activeJobs[update.Message.Chat.ID][i+1:]...)
				break
			}
		}
		t.activeJobsMutex.Unlock()

		// Clean up the placeholder map
		t.placeholderMutex.Lock()
		delete(t.placeholders, jobUUID)
		t.placeholderMutex.Unlock()
	}()

	res := a.Ask(
		types.WithConversationHistory(currentConv),
		types.WithUUID(jobUUID),
		types.WithMetadata(metadata),
	)

	if res.Response == "" {
		xlog.Error("Empty response from agent")
		_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.Message.Chat.ID,
			MessageID: msg.ID,
			Text:      "there was an internal error. try again!",
		})
		if err != nil {
			xlog.Error("Error updating error message", "error", err)
		}
		return
	}

	t.conversationTracker.AddMessage(
		update.Message.From.ID,
		openai.ChatCompletionMessage{
			Content: res.Response,
			Role:    "assistant",
		},
	)

	// Update the message with the final response
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    update.Message.Chat.ID,
		MessageID: msg.ID,
		Text:      res.Response,
	})
	if err != nil {
		xlog.Error("Error updating final message", "error", err)
	}

	// Handle any images or URLs in the response
	for _, res := range res.State {
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
					Photo: &models.InputFileUpload{
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
		xlog.Error("Error creating bot", "error", err)
		return
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
		placeholders:        make(map[string]int),
		activeJobs:          make(map[int64][]*types.Job),
	}, nil
}

// TelegramConfigMeta returns the metadata for Telegram connector configuration fields
func TelegramConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "token",
			Label:    "Telegram Token",
			Type:     config.FieldTypeText,
			Required: true,
		},
		{
			Name:     "admins",
			Label:    "Admins",
			Type:     config.FieldTypeText,
			HelpText: "Comma-separated list of Telegram usernames that are allowed to interact with the bot",
		},
		{
			Name:         "lastMessageDuration",
			Label:        "Last Message Duration",
			Type:         config.FieldTypeText,
			DefaultValue: "5m",
		},
	}
}
