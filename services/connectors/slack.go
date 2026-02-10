package connectors

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/pkg/xstrings"
	"github.com/mudler/LocalAGI/services/actions"
	"github.com/mudler/xlog"
	"github.com/sashabaranov/go-openai"

	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/types"

	"github.com/slack-go/slack/socketmode"

	"github.com/slack-go/slack"

	"github.com/eritikass/githubmarkdownconvertergo"
	"github.com/slack-go/slack/slackevents"
)

type Slack struct {
	appToken    string
	botToken    string
	channelID   string
	channelMode bool

	// To track placeholder messages
	placeholders     map[string]string // map[jobUUID]messageTS
	placeholderMutex sync.RWMutex
	apiClient        *slack.Client

	// Track active jobs for cancellation
	activeJobs      map[string][]*types.Job // map[channelID]bool to track if a channel has active processing
	activeJobsMutex sync.RWMutex
}

const thinkingMessage = ":hourglass: thinking..."

func NewSlack(config map[string]string) *Slack {

	return &Slack{
		appToken:     config["appToken"],
		botToken:     config["botToken"],
		channelID:    config["channelID"],
		channelMode:  config["channelMode"] == "true",
		placeholders: make(map[string]string),
		activeJobs:   make(map[string][]*types.Job),
	}
}

func (t *Slack) AgentResultCallback() func(state types.ActionState) {
	return func(state types.ActionState) {
		// Mark the job as completed when we get the final result
		if state.ActionCurrentState.Job != nil && state.ActionCurrentState.Job.Metadata != nil {
			if channel, ok := state.ActionCurrentState.Job.Metadata["channel"].(string); ok && channel != "" {
				t.activeJobsMutex.Lock()
				delete(t.activeJobs, channel)
				t.activeJobsMutex.Unlock()
			}
		}
	}
}

func (t *Slack) AgentReasoningCallback() func(state types.ActionCurrentState) bool {
	return func(state types.ActionCurrentState) bool {
		// Check if we have a placeholder message for this job
		t.placeholderMutex.RLock()
		msgTs, exists := t.placeholders[state.Job.UUID]
		channel := ""
		if state.Job.Metadata != nil {
			if ch, ok := state.Job.Metadata["channel"].(string); ok {
				channel = ch
			}
		}
		t.placeholderMutex.RUnlock()

		if !exists || msgTs == "" || channel == "" || t.apiClient == nil {
			return true // Skip if we don't have a message to update
		}

		thought := thinkingMessage + "\n\n"
		if state.Reasoning != "" {
			thought += "Current thought process:\n" + state.Reasoning
		}

		// Update the placeholder message with the current reasoning
		_, _, _, err := t.apiClient.UpdateMessage(
			channel,
			msgTs,
			slack.MsgOptionText(githubmarkdownconvertergo.Slack(thought), false),
		)
		if err != nil {
			xlog.Error(fmt.Sprintf("Error updating reasoning message: %v", err))
		}
		return true
	}
}

// cancelActiveJobForChannel cancels any active job for the given channel
func (t *Slack) cancelActiveJobForChannel(channelID string) {
	t.activeJobsMutex.RLock()
	ctxs, exists := t.activeJobs[channelID]
	t.activeJobsMutex.RUnlock()

	if exists {
		xlog.Info(fmt.Sprintf("Cancelling active job for channel: %s", channelID))

		// Mark the job as inactive
		t.activeJobsMutex.Lock()
		for _, c := range ctxs {
			c.Cancel()
		}
		delete(t.activeJobs, channelID)
		t.activeJobsMutex.Unlock()
	}
}

func cleanUpUsernameFromMessage(message string, b *slack.AuthTestResponse) string {
	cleaned := strings.ReplaceAll(message, "<@"+b.UserID+">", "")
	cleaned = strings.ReplaceAll(cleaned, "<@"+b.BotID+">", "")
	cleaned = strings.TrimSpace(cleaned)
	return cleaned
}

func replaceUserIDsWithNamesInMessage(api *slack.Client, message string) string {
	for _, part := range strings.Split(message, " ") {
		if strings.HasPrefix(part, "<@") && strings.HasSuffix(part, ">") {
			xlog.Debug(fmt.Sprintf("Part: %s", part))
			userID := strings.TrimPrefix(strings.TrimSuffix(part, ">"), "<@")
			xlog.Debug(fmt.Sprintf("UserID: %s", userID))
			userInfo, err := api.GetUserInfo(userID)
			if err != nil {
				xlog.Error(fmt.Sprintf("Error getting user info: %v", err))
				continue
			}
			message = strings.ReplaceAll(message, part, "@"+userInfo.Name)
			xlog.Debug(fmt.Sprintf("Message: %s", message))
		}
	}
	return message
}

func generateAttachmentsFromJobResponse(j *types.JobResult, api *slack.Client, channelID, ts string) (attachments []slack.Attachment) {
	for _, state := range j.State {
		// coming from the browser agent
		// if history, exists := state.Metadata[actions.MetadataBrowserAgentHistory]; exists {
		// 	if historyStruct, ok := history.(*localoperator.StateHistory); ok {
		// 		state := historyStruct.States[len(historyStruct.States)-1]
		// 		// Decode base64 screenshot and upload to Slack
		// 		if state.Screenshot != "" {
		// 			screenshotData, err := base64.StdEncoding.DecodeString(state.Screenshot)
		// 			if err != nil {
		// 				xlog.Error(fmt.Sprintf("Error decoding screenshot: %v", err))
		// 				continue
		// 			}

		// 			data := string(screenshotData)
		// 			// Upload the file to Slack
		// 			_, err = api.UploadFileV2(slack.UploadFileV2Parameters{
		// 				Reader:          bytes.NewReader(screenshotData),
		// 				FileSize:        len(data),
		// 				ThreadTimestamp: ts,
		// 				Channel:         channelID,
		// 				Filename:        "screenshot.png",
		// 				InitialComment:  "Browser Agent Screenshot",
		// 			})
		// 			if err != nil {
		// 				xlog.Error(fmt.Sprintf("Error uploading screenshot: %v", err))
		// 				continue
		// 			}
		// 		}
		// 	}
		// }

		// coming from the search action
		if urls, exists := state.Metadata[actions.MetadataUrls]; exists {
			for _, url := range xstrings.UniqueSlice(urls.([]string)) {
				attachment := slack.Attachment{
					Title:     "URL",
					TitleLink: url,
					Text:      url,
				}
				attachments = append(attachments, attachment)
			}
		}

		// coming from the gen image actions
		if imagesUrls, exists := state.Metadata[actions.MetadataImages]; exists {
			for _, url := range xstrings.UniqueSlice(imagesUrls.([]string)) {
				attachment := slack.Attachment{
					Title:     "Image",
					TitleLink: url,
					ImageURL:  url,
				}
				attachments = append(attachments, attachment)
			}
		}

		// coming from the generate_song action (local file paths)
		if songPaths, exists := state.Metadata[actions.MetadataSongs]; exists {
			for _, path := range xstrings.UniqueSlice(songPaths.([]string)) {
				data, err := os.ReadFile(path)
				if err != nil {
					xlog.Error(fmt.Sprintf("Error reading song file %s: %v", path, err))
					continue
				}
				filename := filepath.Base(path)
				if filename == "" || filename == "." {
					filename = "audio"
				}
				_, err = api.UploadFileV2(slack.UploadFileV2Parameters{
					Reader:          bytes.NewReader(data),
					FileSize:        len(data),
					ThreadTimestamp: ts,
					Channel:         channelID,
					Filename:        filename,
					InitialComment:  "Generated song",
				})
				if err != nil {
					xlog.Error(fmt.Sprintf("Error uploading song to Slack: %v", err))
				}
			}
		}

		// coming from the generate_pdf action (local file paths)
		if pdfPaths, exists := state.Metadata[actions.MetadataPDFs]; exists {
			for _, path := range xstrings.UniqueSlice(pdfPaths.([]string)) {
				data, err := os.ReadFile(path)
				if err != nil {
					xlog.Error(fmt.Sprintf("Error reading PDF file %s: %v", path, err))
					continue
				}
				filename := filepath.Base(path)
				if filename == "" || filename == "." {
					filename = "document.pdf"
				}

				_, err = api.UploadFileV2(slack.UploadFileV2Parameters{
					Reader:          bytes.NewReader(data),
					FileSize:        len(data),
					ThreadTimestamp: ts,
					Channel:         channelID,
					Filename:        filename,
					InitialComment:  "Generated PDF document",
				})
				if err != nil {
					xlog.Error(fmt.Sprintf("Error uploading PDF to Slack: %v", err))
				}
			}
		}
	}
	return
}

// ImageData represents a single image with its metadata
type ImageData struct {
	Data     []byte
	MimeType string
}

// scanImagesInMessages scans for all images in a message and returns them as a slice
func scanImagesInMessages(api *slack.Client, ev *slackevents.MessageEvent) []ImageData {
	var images []ImageData

	// Fetch the message using the API
	messages, _, _, err := api.GetConversationReplies(&slack.GetConversationRepliesParameters{
		ChannelID: ev.Channel,
		Timestamp: ev.TimeStamp,
	})

	if err != nil {
		xlog.Error(fmt.Sprintf("Error fetching messages: %v", err))
		return images
	}

	xlog.Debug("Scanning images in messages", "messages", messages)
	for _, msg := range messages {
		if len(msg.Files) == 0 {
			xlog.Debug("No files in message", "message", msg.Text)
			continue
		}
		xlog.Debug("Files in message", "files", msg.Files)
		for _, attachment := range msg.Files {
			if attachment.URLPrivate != "" {
				xlog.Debug(fmt.Sprintf("Getting Attachment: %+v", attachment))
				// download image with slack api
				imageBytes := new(bytes.Buffer)
				if err := api.GetFile(attachment.URLPrivate, imageBytes); err != nil {
					xlog.Error(fmt.Sprintf("Error downloading image: %v", err))
					continue
				}

				images = append(images, ImageData{
					Data:     imageBytes.Bytes(),
					MimeType: attachment.Mimetype,
				})
			}
		}
	}

	return images
}

// scanImagesInAppMentionEvent scans for all images in an app mention event
func scanImagesInAppMentionEvent(api *slack.Client, ev *slackevents.AppMentionEvent) []ImageData {
	var images []ImageData

	// Fetch the message using the API
	messages, _, _, err := api.GetConversationReplies(&slack.GetConversationRepliesParameters{
		ChannelID: ev.Channel,
		Timestamp: ev.TimeStamp,
	})

	if err != nil {
		xlog.Error(fmt.Sprintf("Error fetching messages: %v", err))
		return images
	}

	xlog.Debug("Scanning images in app mention event", "messages", messages)
	for _, msg := range messages {
		if len(msg.Files) == 0 {
			xlog.Debug("No files in message", "message", msg.Text)
			continue
		}
		xlog.Debug("Files in message", "files", msg.Files)
		for _, attachment := range msg.Files {
			if attachment.URLPrivate != "" {
				xlog.Debug(fmt.Sprintf("Getting Attachment: %+v", attachment))
				// download image with slack api
				imageBytes := new(bytes.Buffer)
				if err := api.GetFile(attachment.URLPrivate, imageBytes); err != nil {
					xlog.Error(fmt.Sprintf("Error downloading image: %v", err))
					continue
				}

				images = append(images, ImageData{
					Data:     imageBytes.Bytes(),
					MimeType: attachment.Mimetype,
				})
			}
		}
	}

	return images
}

// scanImagesInThreadMessage scans for all images in a single thread message
func scanImagesInThreadMessage(api *slack.Client, msg slack.Message) []ImageData {
	var images []ImageData

	if len(msg.Files) == 0 {
		return images
	}

	xlog.Debug("found files in the message", "files", len(msg.Files))
	for _, attachment := range msg.Files {
		if attachment.URLPrivate != "" {
			xlog.Debug(fmt.Sprintf("Getting Attachment: %+v", attachment))
			// download image with slack api
			imageBytes := new(bytes.Buffer)
			if err := api.GetFile(attachment.URLPrivate, imageBytes); err != nil {
				xlog.Error(fmt.Sprintf("Error downloading image: %v", err))
				continue
			}

			images = append(images, ImageData{
				Data:     imageBytes.Bytes(),
				MimeType: attachment.Mimetype,
			})
		}
	}

	return images
}

// createMultiContentMessage creates a ChatCompletionMessage with text and multiple images
func createMultiContentMessage(role, text string, images []ImageData) openai.ChatCompletionMessage {
	multiContent := []openai.ChatMessagePart{
		{
			Text: text,
			Type: openai.ChatMessagePartTypeText,
		},
	}

	for _, img := range images {
		imgBase64, err := encodeImageFromBytes(img.Data)
		if err != nil {
			xlog.Error(fmt.Sprintf("Error encoding image to base64: %v", err))
			continue
		}

		multiContent = append(multiContent, openai.ChatMessagePart{
			Type: openai.ChatMessagePartTypeImageURL,
			ImageURL: &openai.ChatMessageImageURL{
				URL: fmt.Sprintf("data:%s;base64,%s", img.MimeType, imgBase64),
			},
		})
	}

	return openai.ChatCompletionMessage{
		Role:         role,
		MultiContent: multiContent,
	}
}

// encodeImageFromBytes encodes image bytes to base64
func encodeImageFromBytes(imageData []byte) (string, error) {
	return base64.StdEncoding.EncodeToString(imageData), nil
}

func (t *Slack) handleChannelMessage(
	a *agent.Agent,
	api *slack.Client, ev *slackevents.MessageEvent, b *slack.AuthTestResponse, postMessageParams slack.PostMessageParameters) {
	if t.channelID == "" ||
		t.channelID != "" && !t.channelMode ||
		t.channelID != ev.Channel { // If we have a channelID and it's not the same as the event channel
		// Skip messages from other channels
		xlog.Info("Skipping reply to channel", ev.Channel, t.channelID)
		return
	}

	if b.UserID == ev.User {
		// Skip messages from ourselves
		return
	}

	// Cancel any active job for this channel before starting a new one
	t.cancelActiveJobForChannel(ev.Channel)

	currentConv := a.SharedState().ConversationTracker.GetConversation(fmt.Sprintf("slack:%s", t.channelID))

	message := replaceUserIDsWithNamesInMessage(api, cleanUpUsernameFromMessage(ev.Text, b))

	go func() {

		images := scanImagesInMessages(api, ev)

		agentOptions := []types.JobOption{
			types.WithUUID(ev.ThreadTimeStamp),
		}

		// If the last message has an image, we send it as a multi content message
		if len(images) > 0 {
			currentConv = append(currentConv, createMultiContentMessage("user", message, images))
		} else {
			currentConv = append(currentConv, openai.ChatCompletionMessage{
				Role:    "user",
				Content: message,
			})
		}

		a.SharedState().ConversationTracker.AddMessage(
			fmt.Sprintf("slack:%s", t.channelID), currentConv[len(currentConv)-1],
		)

		agentOptions = append(agentOptions, types.WithConversationHistory(currentConv))

		// Add channel to metadata for tracking
		metadata := map[string]interface{}{
			"channel": ev.Channel,
		}
		agentOptions = append(agentOptions, types.WithMetadata(metadata))

		job := types.NewJob(agentOptions...)

		// Mark this channel as having an active job
		t.activeJobsMutex.Lock()
		t.activeJobs[ev.Channel] = append(t.activeJobs[ev.Channel], job)
		t.activeJobsMutex.Unlock()

		defer func() {
			// Mark job as complete
			t.activeJobsMutex.Lock()
			job.Cancel()
			for i, j := range t.activeJobs[ev.Channel] {
				if j.UUID == job.UUID {
					t.activeJobs[ev.Channel] = append(t.activeJobs[ev.Channel][:i], t.activeJobs[ev.Channel][i+1:]...)
					break
				}
			}

			t.activeJobsMutex.Unlock()
		}()

		res := a.Ask(
			agentOptions...,
		)

		if res.Response == "" {
			xlog.Debug(fmt.Sprintf("Empty response from agent"))
			return
		}

		if res.Error != nil {
			xlog.Error(fmt.Sprintf("Error from agent: %v", res.Error))
			return
		}

		a.SharedState().ConversationTracker.AddMessage(
			fmt.Sprintf("slack:%s", t.channelID), openai.ChatCompletionMessage{
				Role:    "assistant",
				Content: res.Response,
			},
		)

		xlog.Debug("After adding message to conversation tracker", "conversation", a.SharedState().ConversationTracker.GetConversation(fmt.Sprintf("slack:%s", t.channelID)))

		//res.Response = githubmarkdownconvertergo.Slack(res.Response)

		replyWithPostMessage(res.Response, api, ev, postMessageParams, res)

	}()
}

func replyWithPostMessage(finalResponse string, api *slack.Client, ev *slackevents.MessageEvent, postMessageParams slack.PostMessageParameters, res *types.JobResult) {
	if len(finalResponse) > 4000 {
		// split response in multiple messages, and update the first

		messages := xstrings.SplitParagraph(finalResponse, 3000)

		for _, message := range messages {
			_, _, err := api.PostMessage(ev.Channel,
				slack.MsgOptionLinkNames(true),
				slack.MsgOptionEnableLinkUnfurl(),
				slack.MsgOptionText(message, true),
				slack.MsgOptionPostMessageParameters(postMessageParams),
				slack.MsgOptionAttachments(generateAttachmentsFromJobResponse(res, api, ev.Channel, "")...),
			)
			if err != nil {
				xlog.Error(fmt.Sprintf("Error posting message: %v", err))
			}
		}
	} else {
		_, _, err := api.PostMessage(ev.Channel,
			slack.MsgOptionLinkNames(true),
			slack.MsgOptionEnableLinkUnfurl(),
			slack.MsgOptionText(res.Response, true),
			slack.MsgOptionPostMessageParameters(postMessageParams),
			slack.MsgOptionAttachments(generateAttachmentsFromJobResponse(res, api, ev.Channel, "")...),
		//	slack.MsgOptionTS(ts),
		)
		if err != nil {
			xlog.Error(fmt.Sprintf("Error updating final message: %v", err))
		}
	}
}

func replyToUpdateMessage(finalResponse string, api *slack.Client, ev *slackevents.AppMentionEvent, msgTs string, ts string, res *types.JobResult) {
	if len(finalResponse) > 3000 {
		// split response in multiple messages, and update the first

		messages := xstrings.SplitParagraph(finalResponse, 3000)

		_, _, _, err := api.UpdateMessage(
			ev.Channel,
			msgTs,
			slack.MsgOptionLinkNames(true),
			slack.MsgOptionEnableLinkUnfurl(),
			slack.MsgOptionText(messages[0], true),
			slack.MsgOptionAttachments(generateAttachmentsFromJobResponse(res, api, ev.Channel, msgTs)...),
		)
		if err != nil {
			xlog.Error(fmt.Sprintf("Error updating final message: %v", err))
		}

		for i, message := range messages {
			if i == 0 {
				continue
			}
			_, _, err = api.PostMessage(ev.Channel,
				slack.MsgOptionLinkNames(true),
				slack.MsgOptionEnableLinkUnfurl(),
				slack.MsgOptionText(message, true),
				slack.MsgOptionTS(ts),
			)
			if err != nil {
				xlog.Error(fmt.Sprintf("Error posting message: %v", err))
			}
		}
	} else {
		_, _, _, err := api.UpdateMessage(
			ev.Channel,
			msgTs,
			slack.MsgOptionLinkNames(true),
			slack.MsgOptionEnableLinkUnfurl(),
			slack.MsgOptionText(finalResponse, true),
			slack.MsgOptionAttachments(generateAttachmentsFromJobResponse(res, api, ev.Channel, msgTs)...),
		)
		if err != nil {
			xlog.Error(fmt.Sprintf("Error updating final message: %v", err))
		}
	}
}

func (t *Slack) handleMention(
	a *agent.Agent, api *slack.Client, ev *slackevents.AppMentionEvent,
	b *slack.AuthTestResponse, postMessageParams slack.PostMessageParameters) {

	if b.UserID == ev.User {
		// Skip messages from ourselves
		return
	}
	message := replaceUserIDsWithNamesInMessage(api, cleanUpUsernameFromMessage(ev.Text, b))

	// strip our id from the message
	xlog.Info("Message", message)

	go func() {
		ts := ev.ThreadTimeStamp
		var msgTs string // Timestamp of our placeholder message
		var err error

		// Store the API client for use in the callbacks
		t.apiClient = api

		// Send initial placeholder message
		if ts != "" {
			// If we're in a thread, post the placeholder there
			_, respTs, err := api.PostMessage(ev.Channel,
				slack.MsgOptionText(thinkingMessage, false),
				slack.MsgOptionLinkNames(true),
				slack.MsgOptionEnableLinkUnfurl(),
				slack.MsgOptionPostMessageParameters(postMessageParams),
				slack.MsgOptionTS(ts))
			if err != nil {
				xlog.Error(fmt.Sprintf("Error posting initial message: %v", err))
			} else {
				msgTs = respTs
			}
		} else {
			// Starting a new thread
			_, respTs, err := api.PostMessage(ev.Channel,
				slack.MsgOptionText(thinkingMessage, false),
				slack.MsgOptionLinkNames(true),
				slack.MsgOptionEnableLinkUnfurl(),
				slack.MsgOptionPostMessageParameters(postMessageParams),
				slack.MsgOptionTS(ev.TimeStamp))
			if err != nil {
				xlog.Error(fmt.Sprintf("Error posting initial message: %v", err))
			} else {
				msgTs = respTs
				// We're creating a new thread, so use this as our thread timestamp
				ts = ev.TimeStamp
			}
		}

		// Store the UUID->placeholder message mapping
		// We'll use the thread timestamp as our UUID
		jobUUID := msgTs

		t.placeholderMutex.Lock()
		t.placeholders[jobUUID] = msgTs
		t.placeholderMutex.Unlock()

		var threadMessages []openai.ChatCompletionMessage

		// A thread already exists
		// so we reconstruct the conversation
		if ts != "" {
			// Fetch the thread messages
			messages, _, _, err := api.GetConversationReplies(&slack.GetConversationRepliesParameters{
				ChannelID: ev.Channel,
				Timestamp: ts,
			})
			if err != nil {
				xlog.Error(fmt.Sprintf("Error fetching thread messages: %v", err))
			} else {
				for _, msg := range messages {
					// Skip our placeholder message
					if msg.Timestamp == msgTs {
						continue
					}

					role := "assistant"
					if msg.User != b.UserID {
						role = "user"
					}

					images := scanImagesInThreadMessage(api, msg)

					// If the last message has an image, we send it as a multi content message
					if len(images) > 0 {

						xlog.Debug("found image in an existing thread", "image", len(images))

						threadMessages = append(
							threadMessages,
							createMultiContentMessage(role, replaceUserIDsWithNamesInMessage(api, cleanUpUsernameFromMessage(msg.Text, b)), images),
						)
					} else {
						xlog.Debug("no image in the last message of the thread", "message", msg.Text)
						threadMessages = append(
							threadMessages,
							openai.ChatCompletionMessage{
								Role:    role,
								Content: replaceUserIDsWithNamesInMessage(api, cleanUpUsernameFromMessage(msg.Text, b)),
							},
						)
					}
				}
			}
		} else {

			images := scanImagesInAppMentionEvent(api, ev)

			// If the last message has an image, we send it as a multi content message
			if len(images) > 0 {

				xlog.Debug("found image in the last message of the thread", "image", len(images))
				threadMessages = append(
					threadMessages,
					createMultiContentMessage("user", replaceUserIDsWithNamesInMessage(api, cleanUpUsernameFromMessage(message, b)), images),
				)
			} else {
				threadMessages = append(threadMessages, openai.ChatCompletionMessage{
					Role:    "user",
					Content: replaceUserIDsWithNamesInMessage(api, cleanUpUsernameFromMessage(message, b)),
				})
			}
		}

		// Add channel to job metadata for use in callbacks
		metadata := map[string]interface{}{
			"channel": ev.Channel,
		}

		// Call the agent with the conversation history
		res := a.Ask(
			types.WithConversationHistory(threadMessages),
			types.WithUUID(jobUUID),
			types.WithMetadata(metadata),
		)

		if res.Response == "" {
			xlog.Debug(fmt.Sprintf("Empty response from agent"))
			replyToUpdateMessage("there was an internal error. try again!", api, ev, msgTs, ts, res)

			// _, _, err := api.DeleteMessage(ev.Channel, msgTs)
			// if err != nil {
			// 	xlog.Error(fmt.Sprintf("Error deleting message: %v", err))
			// }
			return
		}

		// get user id
		user, err := api.GetUserInfo(ev.User)
		if err != nil {
			xlog.Error(fmt.Sprintf("Error getting user info: %v", err))
		}

		// Format the final response
		//finalResponse := githubmarkdownconvertergo.Slack(res.Response)
		finalResponse := fmt.Sprintf("@%s %s", user.Name, res.Response)
		xlog.Debug("Send final response to slack", "response", finalResponse)

		replyToUpdateMessage(finalResponse, api, ev, msgTs, ts, res)

		// Clean up the placeholder map
		t.placeholderMutex.Lock()
		delete(t.placeholders, jobUUID)
		t.placeholderMutex.Unlock()
	}()
}

func (t *Slack) Start(a *agent.Agent) {

	postMessageParams := slack.PostMessageParameters{
		LinkNames: 1,
		Markdown:  true,
	}

	api := slack.New(
		t.botToken,
		//	slack.OptionDebug(true),
		slack.OptionLog(log.New(os.Stdout, "api: ", log.Lshortfile|log.LstdFlags)),
		slack.OptionAppLevelToken(t.appToken),
	)

	if t.channelID != "" {
		xlog.Debug(fmt.Sprintf("Listening for messages in channel %s", t.channelID))
		// handle new conversations
		a.AddSubscriber(func(ccm *types.ConversationMessage) {
			xlog.Debug("Subscriber(slack)", "message", ccm.Message.Content)
			_, _, err := api.PostMessage(t.channelID,
				slack.MsgOptionLinkNames(true),
				slack.MsgOptionEnableLinkUnfurl(),
				slack.MsgOptionText(ccm.Message.Content, true),
				slack.MsgOptionPostMessageParameters(postMessageParams),
			)
			if err != nil {
				xlog.Error(fmt.Sprintf("Error posting message: %v", err))
			}
			a.SharedState().ConversationTracker.AddMessage(
				fmt.Sprintf("slack:%s", t.channelID),
				openai.ChatCompletionMessage{
					Content: ccm.Message.Content,
					Role:    "assistant",
				},
			)
		})
	}

	t.apiClient = api

	client := socketmode.New(
		api,
		//socketmode.OptionDebug(true),
		//socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)
	go func() {
		for evt := range client.Events {
			switch evt.Type {
			case socketmode.EventTypeConnecting:
				xlog.Info("Connecting to Slack with Socket Mode...")
			case socketmode.EventTypeConnectionError:
				xlog.Info("Connection failed. Retrying later...")
			case socketmode.EventTypeConnected:
				xlog.Info("Connected to Slack with Socket Mode.")
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					xlog.Debug(fmt.Sprintf("Ignored %+v\n", evt))

					continue
				}

				client.Ack(*evt.Request)

				switch eventsAPIEvent.Type {
				case slackevents.CallbackEvent:
					innerEvent := eventsAPIEvent.InnerEvent

					b, err := api.AuthTest()
					if err != nil {
						fmt.Printf("Error getting auth test: %v", err)
					}

					switch ev := innerEvent.Data.(type) {
					case *slackevents.MessageEvent:
						t.handleChannelMessage(a, api, ev, b, postMessageParams)
					case *slackevents.AppMentionEvent:
						t.handleMention(a, api, ev, b, postMessageParams)
					case *slackevents.MemberJoinedChannelEvent:
						xlog.Error(fmt.Sprintf("user %q joined to channel %q", ev.User, ev.Channel))
					}
				default:
					client.Debugf("unsupported Events API event received")
				}
			default:
				xlog.Error(fmt.Sprintf("Unexpected event type received: %s", evt.Type))
			}
		}
	}()

	client.RunContext(a.Context())
}

// SlackConfigMeta returns the metadata for Slack connector configuration fields
func SlackConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "appToken",
			Label:    "App Token",
			Type:     config.FieldTypeText,
			Required: true,
		},
		{
			Name:     "botToken",
			Label:    "Bot Token",
			Type:     config.FieldTypeText,
			Required: true,
		},
		{
			Name:  "channelID",
			Label: "Channel ID",
			Type:  config.FieldTypeText,
		},
		{
			Name:  "alwaysReply",
			Label: "Always Reply",
			Type:  config.FieldTypeCheckbox,
		},
	}
}
