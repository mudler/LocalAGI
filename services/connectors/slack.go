package connectors

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/mudler/LocalAgent/pkg/xlog"
	"github.com/mudler/LocalAgent/services/actions"
	"github.com/sashabaranov/go-openai"

	"github.com/mudler/LocalAgent/core/agent"

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
}

func NewSlack(config map[string]string) *Slack {
	return &Slack{
		appToken:    config["appToken"],
		botToken:    config["botToken"],
		channelID:   config["channelID"],
		alwaysReply: config["alwaysReply"] == "true",
	}
}

func (t *Slack) AgentResultCallback() func(state agent.ActionState) {
	return func(state agent.ActionState) {
		// Send the result to the bot
	}
}

func (t *Slack) AgentReasoningCallback() func(state agent.ActionCurrentState) bool {
	return func(state agent.ActionCurrentState) bool {
		// Send the reasoning to the bot
		return true
	}
}

func cleanUpUsernameFromMessage(message string, b *slack.AuthTestResponse) string {
	cleaned := strings.ReplaceAll(message, "<@"+b.UserID+">", "")
	cleaned = strings.ReplaceAll(cleaned, "<@"+b.BotID+">", "")
	cleaned = strings.TrimSpace(cleaned)
	return cleaned
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

func generateAttachmentsFromJobResponse(j *agent.JobResult) (attachments []slack.Attachment) {
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

	message := cleanUpUsernameFromMessage(ev.Text, b)

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

		agentOptions := []agent.JobOption{agent.WithText(message)}

		// If the last message has an image, we send it as a multi content message
		if len(imageBytes.Bytes()) > 0 {

			// // Encode the image to base64
			imgBase64, err := encodeImageFromURL(*imageBytes)
			if err != nil {
				xlog.Error(fmt.Sprintf("Error encoding image to base64: %v", err))
			} else {
				agentOptions = append(agentOptions, agent.WithImage(fmt.Sprintf("data:%s;base64,%s", mimeType, imgBase64)))
			}
		}

		res := a.Ask(
			agentOptions...,
		)

		res.Response = githubmarkdownconvertergo.Slack(res.Response)

		_, _, err = api.PostMessage(ev.Channel,
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
	message := cleanUpUsernameFromMessage(ev.Text, b)

	// strip our id from the message
	xlog.Info("Message", message)

	go func() {
		ts := ev.ThreadTimeStamp

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
										Text: cleanUpUsernameFromMessage(msg.Text, b),
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
								Content: cleanUpUsernameFromMessage(msg.Text, b),
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
								Text: cleanUpUsernameFromMessage(message, b),
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
					Content: cleanUpUsernameFromMessage(message, b),
				})
			}
		}

		res := a.Ask(
			//	agent.WithText(message),
			agent.WithConversationHistory(threadMessages),
		)

		res.Response = githubmarkdownconvertergo.Slack(res.Response)
		var err error
		if ts != "" {
			_, _, err = api.PostMessage(ev.Channel,
				slack.MsgOptionText(res.Response, true),
				slack.MsgOptionPostMessageParameters(
					postMessageParams,
				),
				slack.MsgOptionAttachments(generateAttachmentsFromJobResponse(res)...),
				slack.MsgOptionTS(ts))
		} else {
			_, _, err = api.PostMessage(ev.Channel,
				slack.MsgOptionText(res.Response, true),
				slack.MsgOptionAttachments(generateAttachmentsFromJobResponse(res)...),
				slack.MsgOptionPostMessageParameters(
					postMessageParams,
				),
				slack.MsgOptionTS(ev.TimeStamp))
		}
		if err != nil {
			xlog.Error(fmt.Sprintf("Error posting message: %v", err))
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
