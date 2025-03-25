package connectors

import (
	"fmt"
	"strings"
	"time"

	"github.com/mudler/LocalAgent/core/agent"
	"github.com/mudler/LocalAgent/core/types"
	"github.com/mudler/LocalAgent/pkg/xlog"
	"github.com/mudler/LocalAgent/services/actions"
	"github.com/sashabaranov/go-openai"
	irc "github.com/thoj/go-ircevent"
)

type IRC struct {
	server              string
	port                string
	nickname            string
	channel             string
	conn                *irc.Connection
	alwaysReply         bool
	conversationTracker *ConversationTracker[string]
}

func NewIRC(config map[string]string) *IRC {

	duration, err := time.ParseDuration(config["lastMessageDuration"])
	if err != nil {
		duration = 5 * time.Minute
	}
	return &IRC{
		server:              config["server"],
		port:                config["port"],
		nickname:            config["nickname"],
		channel:             config["channel"],
		alwaysReply:         config["alwaysReply"] == "true",
		conversationTracker: NewConversationTracker[string](duration),
	}
}

func (i *IRC) AgentResultCallback() func(state types.ActionState) {
	return func(state types.ActionState) {
		// Send the result to the bot
	}
}

func (i *IRC) AgentReasoningCallback() func(state types.ActionCurrentState) bool {
	return func(state types.ActionCurrentState) bool {
		// Send the reasoning to the bot
		return true
	}
}

// cleanUpUsernameFromMessage removes the bot's nickname from the message
func cleanUpMessage(message string, nickname string) string {
	cleaned := strings.ReplaceAll(message, nickname+":", "")
	cleaned = strings.ReplaceAll(cleaned, nickname+",", "")
	cleaned = strings.TrimSpace(cleaned)
	return cleaned
}

// isMentioned checks if the bot is mentioned in the message
func isMentioned(message string, nickname string) bool {
	return strings.Contains(message, nickname+":") ||
		strings.Contains(message, nickname+",") ||
		strings.HasPrefix(message, nickname)
}

// Start connects to the IRC server and starts listening for messages
func (i *IRC) Start(a *agent.Agent) {
	i.conn = irc.IRC(i.nickname, i.nickname)
	if i.conn == nil {
		xlog.Error("Failed to create IRC client")
		return
	}
	i.conn.UseTLS = false
	i.conn.AddCallback("001", func(e *irc.Event) {
		xlog.Info("Connected to IRC server", "server", i.server)
		i.conn.Join(i.channel)
		xlog.Info("Joined channel", "channel", i.channel)
	})

	i.conn.AddCallback("JOIN", func(e *irc.Event) {
		if e.Nick == i.nickname {
			xlog.Info("Bot joined channel", "channel", e.Arguments[0])
			time.Sleep(1 * time.Second) // Small delay to ensure join is complete
			i.conn.Privmsg(e.Arguments[0], "Hello! I've just (re)started and am ready to assist.")
		}
	})

	i.conn.AddCallback("PRIVMSG", func(e *irc.Event) {
		message := e.Message()
		sender := e.Nick
		channel := e.Arguments[0]
		isDirect := false

		if channel == i.nickname {
			channel = sender
			isDirect = true
		}

		// Skip messages from ourselves
		if sender == i.nickname {
			return
		}

		if !(i.alwaysReply || isMentioned(message, i.nickname) || isDirect) {
			return
		}

		xlog.Info("Recv message", "message", message, "sender", sender, "channel", channel)
		cleanedMessage := cleanUpMessage(message, i.nickname)

		go func() {
			conv := i.conversationTracker.GetConversation(channel)

			conv = append(conv,
				openai.ChatCompletionMessage{
					Content: cleanedMessage,
					Role:    "user",
				},
			)

			res := a.Ask(
				types.WithConversationHistory(conv),
			)

			if res.Response == "" {
				xlog.Info("No response from agent")
				return
			}

			// Update the conversation history
			i.conversationTracker.AddMessage(channel, openai.ChatCompletionMessage{
				Content: res.Response,
				Role:    "assistant",
			})

			xlog.Info("Sending message", "message", res.Response, "channel", channel)

			// Split the response into multiple messages if it's too long
			// IRC typically has a message length limit
			maxLength := 400 // Safe limit for most IRC servers
			response := res.Response

			// Handle multiline responses
			lines := strings.Split(response, "\n")
			for _, line := range lines {
				if line == "" {
					continue
				}

				// Split long lines
				for len(line) > 0 {
					var chunk string
					if len(line) > maxLength {
						chunk = line[:maxLength]
						line = line[maxLength:]
					} else {
						chunk = line
						line = ""
					}

					// Send the message to the channel
					i.conn.Privmsg(channel, chunk)

					// Small delay to prevent flooding
					time.Sleep(500 * time.Millisecond)
				}
			}

			// Handle any attachments or special content from actions
			for _, state := range res.State {
				// Handle URLs from search action
				if urls, exists := state.Metadata[actions.MetadataUrls]; exists {
					for _, url := range urls.([]string) {
						i.conn.Privmsg(channel, fmt.Sprintf("URL: %s", url))
						time.Sleep(500 * time.Millisecond)
					}
				}

				// Handle image URLs
				if imagesUrls, exists := state.Metadata[actions.MetadataImages]; exists {
					for _, url := range imagesUrls.([]string) {
						i.conn.Privmsg(channel, fmt.Sprintf("Image: %s", url))
						time.Sleep(500 * time.Millisecond)
					}
				}
			}
		}()
	})

	// Connect to the server
	err := i.conn.Connect(i.server + ":" + i.port)
	if err != nil {
		xlog.Error("Failed to connect to IRC server", "error", err)
		return
	}

	// Start the IRC client in a goroutine
	go i.conn.Loop()
}
