package connectors

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/mudler/LocalAgent/pkg/xlog"
	"github.com/mudler/LocalAgent/services/actions"
	"github.com/sashabaranov/go-openai"

	"github.com/mudler/LocalAgent/core/agent"
	"github.com/mudler/LocalAgent/core/types"

	"github.com/slack-go/slack/socketmode"

	"github.com/slack-go/slack"

	"github.com/eritikass/githubmarkdownconvertergo"
	"github.com/slack-go/slack/slackevents"
)

type Slack struct {
	appToken    string
	botToken    string
	channelID   string
	alwaysReply bool

	// To track placeholder messages
	placeholders     map[string]string // map[jobUUID]messageTS
	placeholderMutex sync.RWMutex
	apiClient        *slack.Client
}

const thinkingMessage = "thinking..."

func NewSlack(config map[string]string) *Slack {
	return &Slack{
		appToken:     config["appToken"],
		botToken:     config["botToken"],
		channelID:    config["channelID"],
		alwaysReply:  config["alwaysReply"] == "true",
		placeholders: make(map[string]string),
	}
}

func (t *Slack) AgentResultCallback() func(state types.ActionState) {
	return func(state types.ActionState) {
		// The final result callback is intentionally empty as we're handling
		// the final update in the handleMention function directly
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

func cleanUpUsernameFromMessage(message string, b *slack.AuthTestResponse) string {
	cleaned := strings.ReplaceAll(message, "<@"+b.UserID+">", "")
	cleaned = strings.ReplaceAll(cleaned, "<@"+b.BotID+">", "")
	cleaned = strings.TrimSpace(cleaned)
	return cleaned
}

func extractUserIDsFromMessage(message string) []string {
	var userIDs []string
	for _, part := range strings.Split(message, " ") {
		if strings.HasPrefix(part, "<@") && strings.HasSuffix(part, ">") {
			userIDs = append(userIDs, strings.TrimPrefix(strings.TrimSuffix(part, ">"), "<@"))
		}
	}
	return userIDs
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

func uniqueStringSlice(s []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range s {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func generateAttachmentsFromJobResponse(j *types.JobResult) (attachments []slack.Attachment) {
	for _, state := range j.State {
		// coming from the search action
		if urls, exists := state.Metadata[actions.MetadataUrls]; exists {
			for _, url := range uniqueStringSlice(urls.([]string)) {
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
			for _, url := range uniqueStringSlice(imagesUrls.([]string)) {
				attachment := slack.Attachment{
					Title:     "Image",
					TitleLink: url,
					ImageURL:  url,
				}
				attachments = append(attachments, attachment)
			}
		}
	}
	return
}

func (t *Slack) handleChannelMessage(
	a *agent.Agent,
	api *slack.Client, ev *slackevents.MessageEvent, b *slack.AuthTestResponse, postMessageParams slack.PostMessageParameters) {
	if t.channelID == "" && !t.alwaysReply || // If we have set alwaysReply and no channelID
		t.channelID != ev.Channel { // If we have a channelID and it's not the same as the event channel
		// Skip messages from other channels
		xlog.Info("Skipping reply to channel", ev.Channel, t.channelID)
		return
	}

	if b.UserID == ev.User {
		// Skip messages from ourselves
		return
	}

	message := replaceUserIDsWithNamesInMessage(api, cleanUpUsernameFromMessage(ev.Text, b))

	go func() {

		imageBytes := new(bytes.Buffer)
		mimeType := "image/jpeg"

		// Fetch the message using the API
		messages, _, _, err := api.GetConversationReplies(&slack.GetConversationRepliesParameters{
			ChannelID: ev.Channel,
			Timestamp: ev.TimeStamp,
		})

		if err != nil {
			xlog.Error(fmt.Sprintf("Error fetching messages: %v", err))
		} else {
			for _, msg := range messages {
				if len(msg.Files) == 0 {
					continue
				}
				for _, attachment := range msg.Files {
					if attachment.URLPrivate != "" {
						xlog.Debug(fmt.Sprintf("Getting Attachment: %+v", attachment))
						// download image with slack api
						mimeType = attachment.Mimetype
						if err := api.GetFile(attachment.URLPrivate, imageBytes); err != nil {
							xlog.Error(fmt.Sprintf("Error downloading image: %v", err))
						}
					}
				}
			}
		}

		agentOptions := []types.JobOption{
			types.WithUUID(ev.ThreadTimeStamp),
		}

		// If the last message has an image, we send it as a multi content message
		if len(imageBytes.Bytes()) > 0 {

			// // Encode the image to base64
			imgBase64, err := encodeImageFromURL(*imageBytes)
			if err != nil {
				xlog.Error(fmt.Sprintf("Error encoding image to base64: %v", err))
			} else {
				agentOptions = append(agentOptions, types.WithTextImage(message, fmt.Sprintf("data:%s;base64,%s", mimeType, imgBase64)))
			}
		} else {
			agentOptions = append(agentOptions, types.WithText(message))
		}

		res := a.Ask(
			agentOptions...,
		)

		//res.Response = githubmarkdownconvertergo.Slack(res.Response)

		_, _, err = api.PostMessage(ev.Channel,
			slack.MsgOptionLinkNames(true),
			slack.MsgOptionEnableLinkUnfurl(),
			slack.MsgOptionText(res.Response, true),
			slack.MsgOptionPostMessageParameters(postMessageParams),
			slack.MsgOptionAttachments(generateAttachmentsFromJobResponse(res)...),
		//	slack.MsgOptionTS(ts),
		)
		if err != nil {
			xlog.Error(fmt.Sprintf("Error posting message: %v", err))
		}
	}()
}

// Function to download the image from a URL and encode it to base64
func encodeImageFromURL(imageBytes bytes.Buffer) (string, error) {

	// WRITE THIS SOMEWHERE
	ioutil.WriteFile("image.jpg", imageBytes.Bytes(), 0644)

	// Encode the image data to base64
	base64Image := base64.StdEncoding.EncodeToString(imageBytes.Bytes())
	return base64Image, nil
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
		jobUUID := ts
		if jobUUID == "" {
			jobUUID = ev.TimeStamp
		}

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
				for i, msg := range messages {
					// Skip our placeholder message
					if msg.Timestamp == msgTs {
						continue
					}

					role := "assistant"
					if msg.User != b.UserID {
						role = "user"
					}

					imageBytes := new(bytes.Buffer)
					mimeType := "image/jpeg"

					xlog.Debug(fmt.Sprintf("Message: %+v", msg))
					if len(msg.Files) > 0 {
						for _, attachment := range msg.Files {

							if attachment.URLPrivate != "" {
								xlog.Debug(fmt.Sprintf("Getting Attachment: %+v", attachment))
								mimeType = attachment.Mimetype
								// download image with slack api
								if err := api.GetFile(attachment.URLPrivate, imageBytes); err != nil {
									xlog.Error(fmt.Sprintf("Error downloading image: %v", err))
								}
							}
						}
					}
					// If the last message has an image, we send it as a multi content message
					if len(imageBytes.Bytes()) > 0 && i == len(messages)-1 {

						// // Encode the image to base64
						imgBase64, err := encodeImageFromURL(*imageBytes)
						if err != nil {
							xlog.Error(fmt.Sprintf("Error encoding image to base64: %v", err))
						}

						threadMessages = append(
							threadMessages,
							openai.ChatCompletionMessage{
								Role: role,
								MultiContent: []openai.ChatMessagePart{
									{
										Text: replaceUserIDsWithNamesInMessage(api, cleanUpUsernameFromMessage(msg.Text, b)),
										Type: openai.ChatMessagePartTypeText,
									},
									{
										Type: openai.ChatMessagePartTypeImageURL,
										ImageURL: &openai.ChatMessageImageURL{
											URL: fmt.Sprintf("data:%s;base64,%s", mimeType, imgBase64),
											//	URL: imgUrl,
										},
									},
								},
							},
						)
					} else {
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

			imageBytes := new(bytes.Buffer)
			mimeType := "image/jpeg"

			// Fetch the message using the API
			messages, _, _, err := api.GetConversationReplies(&slack.GetConversationRepliesParameters{
				ChannelID: ev.Channel,
				Timestamp: ev.TimeStamp,
			})

			if err != nil {
				xlog.Error(fmt.Sprintf("Error fetching messages: %v", err))
			} else {
				for _, msg := range messages {
					if len(msg.Files) == 0 {
						continue
					}
					for _, attachment := range msg.Files {
						if attachment.URLPrivate != "" {
							xlog.Debug(fmt.Sprintf("Getting Attachment: %+v", attachment))
							// download image with slack api
							mimeType = attachment.Mimetype
							if err := api.GetFile(attachment.URLPrivate, imageBytes); err != nil {
								xlog.Error(fmt.Sprintf("Error downloading image: %v", err))
							}
						}
					}
				}
			}

			// If the last message has an image, we send it as a multi content message
			if len(imageBytes.Bytes()) > 0 {

				// // Encode the image to base64
				imgBase64, err := encodeImageFromURL(*imageBytes)
				if err != nil {
					xlog.Error(fmt.Sprintf("Error encoding image to base64: %v", err))
				}

				threadMessages = append(
					threadMessages,
					openai.ChatCompletionMessage{
						Role: "user",
						MultiContent: []openai.ChatMessagePart{
							{
								Text: replaceUserIDsWithNamesInMessage(api, cleanUpUsernameFromMessage(message, b)),
								Type: openai.ChatMessagePartTypeText,
							},
							{
								Type: openai.ChatMessagePartTypeImageURL,
								ImageURL: &openai.ChatMessageImageURL{
									//	URL: imgURL,
									URL: fmt.Sprintf("data:%s;base64,%s", mimeType, imgBase64),
								},
							},
						},
					},
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

		// get user id
		user, err := api.GetUserInfo(ev.User)
		if err != nil {
			xlog.Error(fmt.Sprintf("Error getting user info: %v", err))
		}

		// Format the final response
		//finalResponse := githubmarkdownconvertergo.Slack(res.Response)
		finalResponse := fmt.Sprintf("@%s %s", user.Name, res.Response)

		// Update the placeholder message with the final result
		t.placeholderMutex.RLock()
		msgTs, exists := t.placeholders[jobUUID]
		t.placeholderMutex.RUnlock()

		if exists && msgTs != "" {
			_, _, _, err = api.UpdateMessage(
				ev.Channel,
				msgTs,
				slack.MsgOptionLinkNames(true),
				slack.MsgOptionEnableLinkUnfurl(),
				slack.MsgOptionText(finalResponse, true),
				slack.MsgOptionAttachments(generateAttachmentsFromJobResponse(res)...),
			)
			if err != nil {
				xlog.Error(fmt.Sprintf("Error updating final message: %v", err))
			}

			// Clean up the placeholder map
			t.placeholderMutex.Lock()
			delete(t.placeholders, jobUUID)
			t.placeholderMutex.Unlock()
		}
	}()
}

func (t *Slack) Start(a *agent.Agent) {
	api := slack.New(
		t.botToken,
		//	slack.OptionDebug(true),
		slack.OptionLog(log.New(os.Stdout, "api: ", log.Lshortfile|log.LstdFlags)),
		slack.OptionAppLevelToken(t.appToken),
	)

	t.apiClient = api

	postMessageParams := slack.PostMessageParameters{
		LinkNames: 1,
		Markdown:  true,
	}

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
