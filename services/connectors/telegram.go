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
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/services/connectors/common"
	"github.com/mudler/LocalAGI/pkg/xstrings"
	"github.com/mudler/LocalAGI/services/actions"
	"github.com/mudler/xlog"
	"github.com/sashabaranov/go-openai"
)

const telegramThinkingMessage = "🤔 thinking..."
const telegramMaxMessageLength = 3000

type Telegram struct {
	Token string
	bot   *bot.Bot
	agent *agent.Agent

	admins []string

	// To track placeholder messages
	placeholders     map[string]int // map[jobUUID]messageID
	placeholderMutex sync.RWMutex
	jobStatus        map[string]*common.StatusAccumulator // map[jobUUID]accumulator

	// Track active jobs for cancellation
	activeJobs      map[int64][]*types.Job // map[chatID]bool to track if a chat has active processing
	activeJobsMutex sync.RWMutex

	channelID   string
	groupMode   bool
	mentionOnly bool
}

// isBotMentioned checks if the bot is mentioned in the message
func (t *Telegram) isBotMentioned(message string, botUsername string) bool {
	return strings.Contains(message, "@"+botUsername)
}

func (t *Telegram) chatFromMessage(update *models.Update) (openai.ChatCompletionMessage, error) {
	// Handle audio messages
	if update.Message.Voice != nil || update.Message.Audio != nil {
		return t.handleAudioMessage(update)
	}

	// Handle photo messages
	if len(update.Message.Photo) > 0 {
		return t.handlePhotoMessage(update)
	}

	// Handle text messages
	return openai.ChatCompletionMessage{
		Content: update.Message.Text,
		Role:    "user",
	}, nil
}

func (t *Telegram) handlePhotoMessage(update *models.Update) (openai.ChatCompletionMessage, error) {
	xlog.Debug("Image", "found image")
	// Get the largest photo
	photo := update.Message.Photo[len(update.Message.Photo)-1]

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	// Download the photo
	file, err := t.bot.GetFile(ctx, &bot.GetFileParams{
		FileID: photo.FileID,
	})
	if err != nil {
		xlog.Error("Error getting file", "error", err)
		return openai.ChatCompletionMessage{}, err
	}

	// Construct the full URL for downloading the file
	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", t.Token, file.FilePath)

	// Download the file content
	resp, err := http.Get(fileURL)
	if err != nil {
		xlog.Error("Error downloading file", "error", err)
		return openai.ChatCompletionMessage{}, err
	}
	defer resp.Body.Close()

	imageBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		xlog.Error("Error reading image", "error", err)
		return openai.ChatCompletionMessage{}, err
	}

	// Encode to base64
	imgBase64 := base64.StdEncoding.EncodeToString(imageBytes)
	xlog.Debug("Image", "sending encoded image")
	// Add to conversation as multi-content message
	return openai.ChatCompletionMessage{
		Role: "user",
		MultiContent: []openai.ChatMessagePart{
			{
				Text: update.Message.Caption,
				Type: openai.ChatMessagePartTypeText,
			},
			{
				Type: openai.ChatMessagePartTypeImageURL,
				ImageURL: &openai.ChatMessageImageURL{
					URL: fmt.Sprintf("data:image/jpeg;base64,%s", imgBase64),
				},
			},
		},
	}, nil
}

func (t *Telegram) handleAudioMessage(update *models.Update) (openai.ChatCompletionMessage, error) {
	var fileID string
	var audioType string

	if update.Message.Voice != nil {
		fileID = update.Message.Voice.FileID
		audioType = "voice"
	} else if update.Message.Audio != nil {
		fileID = update.Message.Audio.FileID
		audioType = "audio"
	}

	xlog.Debug("Audio message received", "type", audioType, "fileID", fileID)

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Download the audio file
	file, err := t.bot.GetFile(ctx, &bot.GetFileParams{
		FileID: fileID,
	})
	if err != nil {
		xlog.Error("Error getting audio file", "error", err)
		return openai.ChatCompletionMessage{}, err
	}

	// Construct the full URL for downloading the file
	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", t.Token, file.FilePath)

	// Download the file content
	resp, err := http.Get(fileURL)
	if err != nil {
		xlog.Error("Error downloading audio file", "error", err)
		return openai.ChatCompletionMessage{}, err
	}
	defer resp.Body.Close()

	audioBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		xlog.Error("Error reading audio file", "error", err)
		return openai.ChatCompletionMessage{}, err
	}

	// Create a temporary file for transcription
	tempFile, err := os.CreateTemp("", "telegram_audio_*.ogg")
	if err != nil {
		xlog.Error("Error creating temp file", "error", err)
		return openai.ChatCompletionMessage{}, err
	}
	defer os.Remove(tempFile.Name())

	// Write audio data to temp file
	if _, err := tempFile.Write(audioBytes); err != nil {
		tempFile.Close()
		xlog.Error("Error writing audio to temp file", "error", err)
		return openai.ChatCompletionMessage{}, err
	}
	tempFile.Close()

	// Transcribe the audio using the agent's Transcribe method
	transcription, err := t.agent.Transcribe(ctx, tempFile.Name())
	if err != nil {
		xlog.Error("Error transcribing audio", "error", err)
		return openai.ChatCompletionMessage{
			Content: fmt.Sprintf("I received an audio message but couldn't transcribe it: %v", err),
			Role:    "user",
		}, nil
	}

	xlog.Debug("Audio transcribed successfully", "transcription", transcription)
	return openai.ChatCompletionMessage{
		Content: transcription,
		Role:    "user",
	}, nil
}

// handleGroupMessage handles messages in group chats
func (t *Telegram) handleGroupMessage(ctx context.Context, b *bot.Bot, a *agent.Agent, update *models.Update) {
	xlog.Debug("Handling group message", "update", update)
	if !t.groupMode {
		xlog.Debug("Group mode is disabled, skipping group message", "chatID", update.Message.Chat.ID)
		return
	}

	// Get bot info to check username
	botInfo, err := b.GetMe(ctx)
	if err != nil {
		xlog.Error("Error getting bot info", "error", err)
		return
	}

	// Skip messages from ourselves
	if update.Message.From.Username == botInfo.Username {
		return
	}

	// If mention-only mode is enabled, check if bot is mentioned
	if t.mentionOnly && !t.isBotMentioned(update.Message.Text, botInfo.Username) {
		xlog.Debug("Bot not mentioned in message, skipping", "chatID", update.Message.Chat.ID)
		return
	}

	// Cancel any active job for this chat before starting a new one
	t.cancelActiveJobForChat(update.Message.Chat.ID)

	// Clean up the message by removing bot mentions
	message := strings.ReplaceAll(update.Message.Text, "@"+botInfo.Username, "")
	update.Message.Text = strings.TrimSpace(message)

	// Send initial placeholder message
	msg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      bot.EscapeMarkdown(telegramThinkingMessage),
		ParseMode: models.ParseModeMarkdown,
		ReplyParameters: &models.ReplyParameters{
			MessageID: update.Message.ID,
		},
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

	// Track if the original message was audio for TTS response
	if update.Message.Voice != nil || update.Message.Audio != nil {
		metadata["originalMessageType"] = "audio"
	}

	chatMessage, err := t.chatFromMessage(update)
	if err != nil {
		xlog.Error("Error extracting chat message", "error", err)
	}

	a.SharedState().ConversationTracker.AddMessage(
		fmt.Sprintf("telegram:%d", update.Message.Chat.ID),
		chatMessage,
	)

	currentConv := a.SharedState().ConversationTracker.GetConversation(fmt.Sprintf("telegram:%d", update.Message.Chat.ID))

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

		// Clean up the placeholder map and job status
		t.placeholderMutex.Lock()
		delete(t.placeholders, jobUUID)
		delete(t.jobStatus, jobUUID)
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

	a.SharedState().ConversationTracker.AddMessage(
		fmt.Sprintf("telegram:%d", update.Message.Chat.ID),
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

	// Check if original message was audio and generate TTS response
	if metadata["originalMessageType"] == "audio" && res.Response != "" {

		xlog.Debug("Original message was audio, generating TTS response")
		audioData, err := t.agent.TTS(ctx, res.Response)
		if err != nil {
			xlog.Error("Error generating TTS", "error", err)
		} else {
			// Send audio response
			err = sendAudioToTelegram(ctx, t.bot, update.Message.Chat.ID, audioData, res.Response)
			if err != nil {
				xlog.Error("Error sending audio response", "error", err)
			} else {
				xlog.Debug("Audio response sent successfully")
				// Remove the thinking placeholder message before returning
				_, err := t.bot.DeleteMessage(ctx, &bot.DeleteMessageParams{
					ChatID:    update.Message.Chat.ID,
					MessageID: msg.ID,
				})
				if err != nil {
					xlog.Error("Error deleting thinking placeholder", "error", err)
				}
				// Don't send text response if audio was sent successfully
				return
			}
		}
	}

	// Update the message with the final response
	formattedResponse := formatResponseWithURLs(res.Response, urls)

	// Split the message if it's too long
	messages := xstrings.SplitParagraph(formattedResponse, telegramMaxMessageLength)

	if len(messages) == 0 {
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

	// Update the first message
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    update.Message.Chat.ID,
		MessageID: msg.ID,
		Text:      messages[0],
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		xlog.Error("Error updating message", "error", err)
		return
	}

	// Send additional chunks as new messages
	for i := 1; i < len(messages); i++ {
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      messages[i],
			ParseMode: models.ParseModeMarkdown,
			ReplyParameters: &models.ReplyParameters{
				MessageID: update.Message.ID,
			},
		})
		if err != nil {
			xlog.Error("Error sending additional message", "error", err)
		}
	}
}

// Send any text message to the bot after the bot has been started

func (t *Telegram) AgentResultCallback() func(state types.ActionState) {
	return func(state types.ActionState) {
		job := state.ActionCurrentState.Job
		if job == nil || job.Metadata == nil {
			return
		}
		chatID, ok := job.Metadata["chatID"].(int64)
		if !ok || chatID == 0 {
			return
		}

		// Update placeholder with tool result if still in progress
		t.placeholderMutex.Lock()
		msgID, exists := t.placeholders[job.UUID]
		if exists && msgID != 0 && t.bot != nil {
			acc, ok := t.jobStatus[job.UUID]
			if !ok {
				acc = common.NewStatusAccumulator()
				t.jobStatus[job.UUID] = acc
			}
			acc.AppendToolResult(common.ActionDisplayName(state.Action), state.Result)
			thought := acc.BuildMessage(telegramThinkingMessage, telegramMaxMessageLength)
			t.placeholderMutex.Unlock()
			_, err := t.bot.EditMessageText(t.agent.Context(), &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: msgID,
				Text:      thought,
			})
			if err != nil {
				xlog.Error("Error updating tool result message", "error", err)
			}
			t.placeholderMutex.Lock()
		}
		t.placeholderMutex.Unlock()

		t.activeJobsMutex.Lock()
		delete(t.activeJobs, chatID)
		t.activeJobsMutex.Unlock()
	}
}

func (t *Telegram) AgentReasoningCallback() func(state types.ActionCurrentState) bool {
	return func(state types.ActionCurrentState) bool {
		t.placeholderMutex.Lock()
		msgID, exists := t.placeholders[state.Job.UUID]
		chatID := int64(0)
		if state.Job.Metadata != nil {
			if ch, ok := state.Job.Metadata["chatID"].(int64); ok {
				chatID = ch
			}
		}
		if !exists || msgID == 0 || chatID == 0 || t.bot == nil {
			t.placeholderMutex.Unlock()
			return true
		}

		if state.Reasoning == "" && state.Action == nil {
			t.placeholderMutex.Unlock()
			return true
		}

		acc, ok := t.jobStatus[state.Job.UUID]
		if !ok {
			acc = common.NewStatusAccumulator()
			t.jobStatus[state.Job.UUID] = acc
		}
		if state.Reasoning != "" {
			acc.AppendReasoning(state.Reasoning)
		}
		if state.Action != nil {
			acc.AppendToolCall(common.ActionDisplayName(state.Action), state.Params.String())
		}
		thought := acc.BuildMessage(telegramThinkingMessage, telegramMaxMessageLength)
		t.placeholderMutex.Unlock()

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

// sendAudioToTelegram sends audio data to Telegram
func sendAudioToTelegram(ctx context.Context, b *bot.Bot, chatID int64, audioData []byte, caption string) error {
	// Send audio with caption
	_, err := b.SendVoice(ctx, &bot.SendVoiceParams{
		ChatID: chatID,
		Voice: &models.InputFileUpload{
			Filename: "response.mp3",
			Data:     bytes.NewReader(audioData),
		},
		Caption: caption,
	})
	if err != nil {
		return fmt.Errorf("error sending audio: %w", err)
	}

	return nil
}

// sendSongToTelegram reads a song file from path and sends it to Telegram as audio.
func sendSongToTelegram(ctx context.Context, b *bot.Bot, chatID int64, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading song file: %w", err)
	}
	filename := filepath.Base(path)
	if filename == "" || filename == "." {
		filename = "audio"
	}
	_, err = b.SendAudio(ctx, &bot.SendAudioParams{
		ChatID: chatID,
		Audio: &models.InputFileUpload{
			Filename: filename,
			Data:     bytes.NewReader(data),
		},
		Caption: "Generated song",
	})
	if err != nil {
		return fmt.Errorf("error sending audio: %w", err)
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

		// Handle songs from generate_song action (local file paths)
		if songPaths, exists := state.Metadata[actions.MetadataSongs]; exists {
			for _, path := range xstrings.UniqueSlice(songPaths.([]string)) {
				xlog.Debug("Sending song", "path", path)
				if err := sendSongToTelegram(ctx, t.bot, chatID, path); err != nil {
					xlog.Error("Error sending song", "error", err)
				}
			}
		}

		// Handle PDFs from generate_pdf action (local file paths)
		if pdfPaths, exists := state.Metadata[actions.MetadataPDFs]; exists {
			for _, path := range xstrings.UniqueSlice(pdfPaths.([]string)) {
				data, err := os.ReadFile(path)
				if err != nil {
					xlog.Error("Error reading PDF file", "path", path, "error", err)
					continue
				}

				filename := filepath.Base(path)
				if filename == "" || filename == "." {
					filename = "document.pdf"
				}

				xlog.Debug("Sending PDF document", "filename", filename, "size", len(data))
				_, err = t.bot.SendDocument(ctx, &bot.SendDocumentParams{
					ChatID: chatID,
					Document: &models.InputFileUpload{
						Filename: filename,
						Data:     bytes.NewReader(data),
					},
					Caption: "Generated PDF",
				})
				if err != nil {
					xlog.Error("Error sending PDF", "error", err)
				}
			}
		}

		// Handle browser agent screenshots
		// if history, exists := state.Metadata[actions.MetadataBrowserAgentHistory]; exists {
		// 	if historyStruct, ok := history.(*localoperator.StateHistory); ok {
		// 		state := historyStruct.States[len(historyStruct.States)-1]
		// 		if state.Screenshot != "" {
		// 			// Decode base64 screenshot
		// 			screenshotData, err := base64.StdEncoding.DecodeString(state.Screenshot)
		// 			if err != nil {
		// 				xlog.Error("Error decoding screenshot", "error", err)
		// 				continue
		// 			}

		// 			// Send screenshot with caption
		// 			_, err = t.bot.SendPhoto(ctx, &bot.SendPhotoParams{
		// 				ChatID: chatID,
		// 				Photo: &models.InputFileUpload{
		// 					Filename: "screenshot.png",
		// 					Data:     bytes.NewReader(screenshotData),
		// 				},
		// 				Caption: "Browser Agent Screenshot",
		// 			})
		// 			if err != nil {
		// 				xlog.Error("Error sending screenshot", "error", err)
		// 			}
		// 		}
		// 	}
		// }
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
	if update.Message == nil || update.Message.From == nil {
		xlog.Debug("Message or user is nil", "update", update)
		return
	}

	username := update.Message.From.Username

	xlog.Debug("Received message from user", "username", username, "chatID", update.Message.Chat.ID, "message", update.Message.Text)
	internalError := func(err error, msg *models.Message) {
		xlog.Error("Error updating final message", "error", err)
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.Message.Chat.ID,
			MessageID: msg.ID,
			Text:      "there was an internal error. try again!",
		})
	}

	xlog.Debug("Handling message", "update", update)
	// Handle group messages
	if update.Message.Chat.Type == "group" || update.Message.Chat.Type == "supergroup" {
		t.handleGroupMessage(ctx, b, a, update)
		return
	}

	// Handle private messages
	if len(t.admins) > 0 && !slices.Contains(t.admins, username) {
		xlog.Info("Unauthorized user", "username", username, "admins", t.admins)
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

	currentConv := a.SharedState().ConversationTracker.GetConversation(fmt.Sprintf("telegram:%d", update.Message.From.ID))

	message, err := t.chatFromMessage(update)
	if err != nil {
		xlog.Error("Error extracting chat message", "error", err)
		return
	}

	currentConv = append(currentConv, message)

	a.SharedState().ConversationTracker.AddMessage(
		fmt.Sprintf("telegram:%d", update.Message.From.ID),
		message,
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

	// Track if the original message was audio for TTS response
	if update.Message.Voice != nil || update.Message.Audio != nil {
		metadata["originalMessageType"] = "audio"
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

		// Clean up the placeholder map and job status
		t.placeholderMutex.Lock()
		delete(t.placeholders, jobUUID)
		delete(t.jobStatus, jobUUID)
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

	a.SharedState().ConversationTracker.AddMessage(
		fmt.Sprintf("telegram:%d", update.Message.From.ID),
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

	// Check if original message was audio and generate TTS response
	if metadata["originalMessageType"] == "audio" && res.Response != "" {
		xlog.Debug("Original message was audio, generating TTS response")
		audioData, err := t.agent.TTS(ctx, res.Response)
		if err != nil {
			xlog.Error("Error generating TTS", "error", err)
		} else {
			// Send audio response
			err = sendAudioToTelegram(ctx, t.bot, update.Message.Chat.ID, audioData, res.Response)
			if err != nil {
				xlog.Error("Error sending audio response", "error", err)
			} else {
				xlog.Debug("Audio response sent successfully")
				// Remove the thinking placeholder message before returning
				_, err := t.bot.DeleteMessage(ctx, &bot.DeleteMessageParams{
					ChatID:    update.Message.Chat.ID,
					MessageID: msg.ID,
				})
				if err != nil {
					xlog.Error("Error deleting thinking placeholder", "error", err)
				}
				// Don't send text response if audio was sent successfully
				return
			}
		}
	}

	// Update the message with the final response
	formattedResponse := formatResponseWithURLs(res.Response, urls)

	// Split the message if it's too long
	messages := xstrings.SplitParagraph(formattedResponse, telegramMaxMessageLength)

	if len(messages) == 0 {
		_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.Message.Chat.ID,
			MessageID: msg.ID,
			Text:      "there was an internal error. try again!",
		})
		if err != nil {
			xlog.Error("Error updating error message", "error", err)
			internalError(fmt.Errorf("error updating error message: %w", err), msg)
		}
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
		xlog.Error("Error updating message", "error", err)
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
			xlog.Error("Error sending additional message", "error", err)
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
	// 	forc m := range a.ConversationChannel() {
	// 		t.handleNewMessage(ctx, b, m)
	// 	}
	// }()

	if t.channelID != "" {
		// handle new conversations
		a.AddSubscriber(func(ccm *types.ConversationMessage) {
			xlog.Debug("Subscriber(telegram)", "message", ccm.Message.Content)

			// First, handle any multimedia content from metadata
			if ccm.Metadata != nil {
				// Handle images from gen image actions
				if imagesUrls, exists := ccm.Metadata[actions.MetadataImages]; exists {
					for _, url := range xstrings.UniqueSlice(imagesUrls.([]string)) {
						xlog.Debug("Sending photo from new conversation", "url", url)
						chatID, _ := strconv.ParseInt(t.channelID, 10, 64)
						if err := sendImageToTelegram(ctx, t.bot, chatID, url); err != nil {
							xlog.Error("Error handling image", "error", err)
						}
					}
				}

				// Handle songs from generate_song action (local file paths)
				if songPaths, exists := ccm.Metadata[actions.MetadataSongs]; exists {
					for _, path := range xstrings.UniqueSlice(songPaths.([]string)) {
						xlog.Debug("Sending song from new conversation", "path", path)
						chatID, _ := strconv.ParseInt(t.channelID, 10, 64)
						if err := sendSongToTelegram(ctx, t.bot, chatID, path); err != nil {
							xlog.Error("Error sending song", "error", err)
						}
					}
				}

				// Handle PDFs from generate_pdf action (local file paths)
				if pdfPaths, exists := ccm.Metadata[actions.MetadataPDFs]; exists {
					for _, path := range xstrings.UniqueSlice(pdfPaths.([]string)) {
						data, err := os.ReadFile(path)
						if err != nil {
							xlog.Error("Error reading PDF file", "path", path, "error", err)
							continue
						}

						filename := filepath.Base(path)
						if filename == "" || filename == "." {
							filename = "document.pdf"
						}

						xlog.Debug("Sending PDF document from new conversation", "filename", filename, "size", len(data))
						chatID, _ := strconv.ParseInt(t.channelID, 10, 64)
						_, err = t.bot.SendDocument(ctx, &bot.SendDocumentParams{
							ChatID: chatID,
							Document: &models.InputFileUpload{
								Filename: filename,
								Data:     bytes.NewReader(data),
							},
							Caption: "Generated PDF",
						})
						if err != nil {
							xlog.Error("Error sending PDF", "error", err)
						}
					}
				}
			}

			// Then send the text message
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: t.channelID,
				Text:   ccm.Message.Content,
			})
			if err != nil {
				xlog.Error("Error sending message", "error", err)
				return
			}

			t.agent.SharedState().ConversationTracker.AddMessage(
				fmt.Sprintf("telegram:%s", t.channelID),
				openai.ChatCompletionMessage{
					Content: ccm.Message.Content,
					Role:    "assistant",
				},
			)
		})
	}

	b.Start(ctx)
}

func NewTelegramConnector(config map[string]string) (*Telegram, error) {
	token, ok := config["token"]
	if !ok {
		return nil, errors.New("token is required")
	}

	admins := []string{}

	if _, ok := config["admins"]; ok && strings.Contains(config["admins"], ",") {
		admins = append(admins, strings.Split(config["admins"], ",")...)
	}

	return &Telegram{
		Token:        token,
		admins:       admins,
		placeholders: make(map[string]int),
		jobStatus:    make(map[string]*common.StatusAccumulator),
		activeJobs:   make(map[int64][]*types.Job),
		channelID:    config["channel_id"],
		groupMode:    config["group_mode"] == "true",
		mentionOnly:  config["mention_only"] == "true",
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
			Name:     "channel_id",
			Label:    "Channel ID",
			Type:     config.FieldTypeText,
			HelpText: "Telegram channel ID to send messages to if the agent needs to initiate a conversation",
		},
		{
			Name:     "group_mode",
			Label:    "Group Mode",
			Type:     config.FieldTypeCheckbox,
			HelpText: "Enable bot to respond in group chats",
		},
		{
			Name:     "mention_only",
			Label:    "Mention Only",
			Type:     config.FieldTypeCheckbox,
			HelpText: "Bot will only respond when mentioned in group chats",
		},
	}
}
