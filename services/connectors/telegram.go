package connectors

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
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
	"github.com/mudler/LocalAGI/pkg/localoperator"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/mudler/LocalAGI/pkg/xstrings"
	"github.com/mudler/LocalAGI/services/actions"
	"github.com/sashabaranov/go-openai"
)

const telegramThinkingMessage = "🤔 thinking..."
const telegramMaxMessageLength = 3000

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

// sendImageToTelegram downloads and sends an image to Telegram
func sendImageToTelegram(ctx context.Context, b *bot.Bot, chatID int64, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error downloading image: %w", err)
	}
	defer resp.Body.Close()

	// Read the entire body into memory
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading image body: %w", err)
	}

	// Send image with caption
	_, err = b.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID: chatID,
		Photo: &models.InputFileUpload{
			Filename: "image.jpg",
			Data:     bytes.NewReader(bodyBytes),
		},
		Caption: "Generated image",
	})
	if err != nil {
		return fmt.Errorf("error sending photo: %w", err)
	}

	return nil
}

// handleMultimediaContent processes and sends multimedia content from the agent's response
func (t *Telegram) handleMultimediaContent(ctx context.Context, chatID int64, res *types.JobResult) ([]string, error) {
	var urls []string

	for _, state := range res.State {
		// Collect URLs from search action
		if urlList, exists := state.Metadata[actions.MetadataUrls]; exists {
			urls = append(urls, xstrings.UniqueSlice(urlList.([]string))...)
		}

		// Handle images from gen image actions
		if imagesUrls, exists := state.Metadata[actions.MetadataImages]; exists {
			for _, url := range xstrings.UniqueSlice(imagesUrls.([]string)) {
				xlog.Debug("Sending photo", "url", url)
				if err := sendImageToTelegram(ctx, t.bot, chatID, url); err != nil {
					xlog.Error("Error handling image", "error", err)
				}
			}
		}

		// Handle browser agent screenshots
		if history, exists := state.Metadata[actions.MetadataBrowserAgentHistory]; exists {
			if historyStruct, ok := history.(*localoperator.StateHistory); ok {
				state := historyStruct.States[len(historyStruct.States)-1]
				if state.Screenshot != "" {
					// Decode base64 screenshot
					screenshotData, err := base64.StdEncoding.DecodeString(state.Screenshot)
					if err != nil {
						xlog.Error("Error decoding screenshot", "error", err)
						continue
					}

					// Send screenshot with caption
					_, err = t.bot.SendPhoto(ctx, &bot.SendPhotoParams{
						ChatID: chatID,
						Photo: &models.InputFileUpload{
							Filename: "screenshot.png",
							Data:     bytes.NewReader(screenshotData),
						},
						Caption: "Browser Agent Screenshot",
					})
					if err != nil {
						xlog.Error("Error sending screenshot", "error", err)
					}
				}
			}
		}
	}

	return urls, nil
}

// formatResponseWithURLs formats the response text and creates message entities for URLs
func formatResponseWithURLs(response string, urls []string) string {
	finalResponse := response
	if len(urls) > 0 {
		finalResponse += "\n\nReferences:\n"
		for i, url := range urls {
			finalResponse += fmt.Sprintf("🔗 %d. %s\n", i+1, url)
		}
	}

	return bot.EscapeMarkdown(finalResponse)
}

func (t *Telegram) handleUpdate(ctx context.Context, b *bot.Bot, a *agent.Agent, update *models.Update) {
	username := update.Message.From.Username

	internalError := func(err error, msg *models.Message) {
		xlog.Error("Error updating final message", "error", err)
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.Message.Chat.ID,
			MessageID: msg.ID,
			Text:      "there was an internal error. try again!",
		})
	}
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
		ChatID:    update.Message.Chat.ID,
		Text:      bot.EscapeMarkdown(telegramThinkingMessage),
		ParseMode: models.ParseModeMarkdown,
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

	// Handle any multimedia content in the response and collect URLs
	urls, err := t.handleMultimediaContent(ctx, update.Message.Chat.ID, res)
	if err != nil {
		xlog.Error("Error handling multimedia content", "error", err)
	}

	// Update the message with the final response
	formattedResponse := formatResponseWithURLs(res.Response, urls)

	// Split the message if it's too long
	messages := xstrings.SplitParagraph(formattedResponse, telegramMaxMessageLength)

	if len(messages) == 0 {
		internalError(errors.New("empty response from agent"), msg)
		return
	}

	// Update the first message
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    update.Message.Chat.ID,
		MessageID: msg.ID,
		Text:      messages[0],
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		internalError(fmt.Errorf("internal error: %w", err), msg)
		return
	}

	// Send additional chunks as new messages
	for i := 1; i < len(messages); i++ {
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      messages[i],
			ParseMode: models.ParseModeMarkdown,
		})
		if err != nil {
			internalError(fmt.Errorf("internal error: %w", err), msg)
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
