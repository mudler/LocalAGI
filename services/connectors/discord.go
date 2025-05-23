package connectors

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/sashabaranov/go-openai"
)

type Discord struct {
	token          string
	defaultChannel string
}

// NewDiscord creates a new Discord connector
// with the given configuration
// - token: Discord token
// - defaultChannel: Discord channel to always answer even if not mentioned
func NewDiscord(config map[string]string) *Discord {

	token := config["token"]

	if !strings.HasPrefix(token, "Bot ") {
		token = "Bot " + token
	}

	return &Discord{
		token:          token,
		defaultChannel: config["defaultChannel"],
	}
}

func DiscordConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "token",
			Label:    "Discord Token",
			Type:     config.FieldTypeText,
			Required: true,
		},
		{
			Name:  "defaultChannel",
			Label: "Default Channel",
			Type:  config.FieldTypeText,
		},
		{
			Name:         "lastMessageDuration",
			Label:        "Last Message Duration",
			Type:         config.FieldTypeText,
			DefaultValue: "5m",
		},
	}
}

func (d *Discord) AgentResultCallback() func(state types.ActionState) {
	return func(state types.ActionState) {
		// Send the result to the bot
	}
}

func (d *Discord) AgentReasoningCallback() func(state types.ActionCurrentState) bool {
	return func(state types.ActionCurrentState) bool {
		// Send the reasoning to the bot
		return true
	}
}

func (d *Discord) Start(a *agent.Agent) {

	Token := d.token
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New(Token)
	if err != nil {
		xlog.Info("error creating Discord session,", err)
		return
	}

	dg.StateEnabled = true

	if d.defaultChannel != "" {
		// handle new conversations
		a.AddSubscriber(func(ccm openai.ChatCompletionMessage) {
			xlog.Debug("Subscriber(discord)", "message", ccm.Content)

			// Send the message to the default channel
			_, err := dg.ChannelMessageSend(d.defaultChannel, ccm.Content)
			if err != nil {
				xlog.Error(fmt.Sprintf("Error sending message: %v", err))
			}

			a.SharedState().ConversationTracker.AddMessage(
				fmt.Sprintf("discord:%s", d.defaultChannel),
				openai.ChatCompletionMessage{
					Content: ccm.Content,
					Role:    "assistant",
				},
			)
		})
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(d.messageCreate(a))

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentMessageContent

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		xlog.Info("error opening connection,", err)
		return
	}

	go func() {
		xlog.Info("Discord bot is now running.  Press CTRL-C to exit.")
		<-a.Context().Done()
		dg.Close()
		xlog.Info("Discord bot is now stopped.")
	}()
}

func (d *Discord) handleThreadMessage(a *agent.Agent, s *discordgo.Session, m *discordgo.MessageCreate) {
	var messages []*discordgo.Message
	var err error

	messages, err = s.ChannelMessages(m.ChannelID, 100, "", "", "")
	if err != nil {
		xlog.Info("error getting messages,", err)
		return
	}

	conv := []openai.ChatCompletionMessage{}

	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]
		if message.Author.ID == s.State.User.ID {
			conv = append(conv, openai.ChatCompletionMessage{
				Role:    "assistant",
				Content: removeBotID(s, message.Content),
			})
		} else {
			conv = append(conv, openai.ChatCompletionMessage{
				Role:    "user",
				Content: removeBotID(s, message.Content),
			})
		}
	}

	xlog.Debug("Conversation", "conversation", conv)

	jobResult := a.Ask(
		types.WithConversationHistory(conv),
	)

	if jobResult.Error != nil {
		xlog.Info("error asking agent,", jobResult.Error)
		return
	}

	_, err = s.ChannelMessageSend(m.ChannelID, jobResult.Response)
	if err != nil {
		xlog.Info("error sending message,", err)
	}
}

func (d *Discord) handleChannelMessage(a *agent.Agent, s *discordgo.Session, m *discordgo.MessageCreate) {

	a.SharedState().ConversationTracker.AddMessage(fmt.Sprintf("discord:%s", m.ChannelID), openai.ChatCompletionMessage{
		Role:    "user",
		Content: m.Content,
	})

	conv := a.SharedState().ConversationTracker.GetConversation(fmt.Sprintf("discord:%s", m.ChannelID))

	jobResult := a.Ask(
		types.WithConversationHistory(conv),
	)

	if jobResult.Error != nil {
		xlog.Info("error asking agent,", jobResult.Error)
		return
	}

	a.SharedState().ConversationTracker.AddMessage(fmt.Sprintf("discord:%s", m.ChannelID), openai.ChatCompletionMessage{
		Role:    "assistant",
		Content: jobResult.Response,
	})

	thread, err := s.MessageThreadStartComplex(m.ChannelID, m.ID, &discordgo.ThreadStart{
		Name:                "Thread for " + m.Author.Username,
		AutoArchiveDuration: 60,
	})
	if err != nil {
		xlog.Error("error creating thread", "err", err.Error())
		// Thread already exists
		_, err = s.ChannelMessageSend(m.ChannelID, jobResult.Response)
		if err != nil {
			xlog.Error("error sending message to thread", "err", err.Error())
		}
	} else {
		_, err = s.ChannelMessageSend(thread.ID, jobResult.Response)
		if err != nil {
			xlog.Error("error sending message,", err)
		}
	}

}

func removeBotID(s *discordgo.Session, m string) string {
	return strings.ReplaceAll(m, "<@"+s.State.User.ID+">", "")
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func (d *Discord) messageCreate(a *agent.Agent) func(s *discordgo.Session, m *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore all messages created by the bot itself
		// This isn't required in this specific example but it's a good practice.
		if m.Author.ID == s.State.User.ID {
			return
		}

		m.Content = removeBotID(s, m.Content)

		xlog.Debug("Message received", "content", m.Content, "connector", "discord")

		// Interact if we are mentioned
		mentioned := false
		for _, mention := range m.Mentions {
			if mention.ID == s.State.User.ID {
				mentioned = true
				break
			}
		}

		if !mentioned && d.defaultChannel == "" {
			xlog.Debug("Not mentioned")
			return
		}

		mm, _ := json.Marshal(m)
		xlog.Debug("Discord message", "message", string(mm))

		isThread := func() bool {
			// NOTE: this doesn't seem to work,
			// even if used in https://github.com/bwmarrin/discordgo/blob/5571950c905ff94d898501e5a0d76895fa140069/examples/threads/main.go#L33
			ch, err := s.State.Channel(m.ChannelID)
			return !(err != nil || !ch.IsThread())
		}

		// check if the message is in a thread and get all messages in the thread
		if isThread() {
			xlog.Debug("Thread message")
			if (d.defaultChannel != "" && m.ChannelID == d.defaultChannel) || (mentioned && d.defaultChannel == "") {
				xlog.Debug("Thread message")
				d.handleThreadMessage(a, s, m)
			}
			xlog.Info("ignoring thread message")
			return
		}

		// Or we are in the default channel (if one is set!)
		if (d.defaultChannel != "" && m.ChannelID == d.defaultChannel) || (mentioned && d.defaultChannel == "") {
			xlog.Debug("Channel message")
			d.handleChannelMessage(a, s, m)
			return
		}
	}
}
