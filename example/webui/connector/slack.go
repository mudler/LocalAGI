package connector

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mudler/local-agent-framework/xlog"

	"github.com/mudler/local-agent-framework/agent"

	"github.com/slack-go/slack/socketmode"

	"github.com/slack-go/slack"
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

func (t *Slack) Start(a *agent.Agent) {
	api := slack.New(
		t.botToken,
		//	slack.OptionDebug(true),
		slack.OptionLog(log.New(os.Stdout, "api: ", log.Lshortfile|log.LstdFlags)),
		slack.OptionAppLevelToken(t.appToken),
	)

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
					fmt.Printf("Ignored %+v\n", evt)

					continue
				}

				fmt.Printf("Event received: %+v\n", eventsAPIEvent)

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
						if t.channelID == "" && !t.alwaysReply || // If we have set alwaysReply and no channelID
							t.channelID != ev.Channel { // If we have a channelID and it's not the same as the event channel
							// Skip messages from other channels
							xlog.Info("Skipping reply to channel", ev.Channel, t.channelID)
							continue
						}

						if b.UserID == ev.User {
							// Skip messages from ourselves
							continue
						}

						message := ev.Text
						go func() {
							res := a.Ask(
								agent.WithText(message),
							)

							_, _, err = api.PostMessage(ev.Channel,
								slack.MsgOptionText(res.Response, false),
								slack.MsgOptionPostMessageParameters(slack.PostMessageParameters{LinkNames: 1}))
							if err != nil {
								fmt.Printf("Error posting message: %v", err)
							}
						}()
					case *slackevents.AppMentionEvent:

						if b.UserID == ev.User {
							// Skip messages from ourselves
							continue
						}
						message := ev.Text

						// strip our id from the message
						message = strings.ReplaceAll(message, "<@"+b.UserID+"> ", "")
						xlog.Info("Message", message)

						go func() {
							res := a.Ask(
								agent.WithText(message),
							)

							_, _, err = api.PostMessage(ev.Channel,
								slack.MsgOptionText(res.Response, false),
								slack.MsgOptionPostMessageParameters(slack.PostMessageParameters{LinkNames: 1}))
							if err != nil {
								fmt.Printf("Error posting message: %v", err)
							}
						}()
					case *slackevents.MemberJoinedChannelEvent:
						fmt.Printf("user %q joined to channel %q", ev.User, ev.Channel)
					}
				default:
					client.Debugf("unsupported Events API event received")
				}
			default:
				fmt.Fprintf(os.Stderr, "Unexpected event type received: %s\n", evt.Type)
			}
		}
	}()

	client.RunContext(a.Context())
}
