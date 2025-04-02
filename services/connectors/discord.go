package connectors

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/mudler/LocalAgent/core/agent"
	"github.com/mudler/LocalAgent/core/types"
	"github.com/mudler/LocalAgent/pkg/config"
	"github.com/mudler/LocalAgent/pkg/xlog"
	"github.com/sashabaranov/go-openai"
)

type Discord struct {
	token               string
	defaultChannel      string
	conversationTracker *ConversationTracker[string]
}

// NewDiscord creates a new Discord connector
// with the given configuration
// - token: Discord token
// - defaultChannel: Discord channel to always answer even if not mentioned
func NewDiscord(config map[string]string) *Discord {

	duration, err := time.ParseDuration(config["lastMessageDuration"])
	if err != nil {
		duration = 5 * time.Minute
	}

	return &Discord{
		conversationTracker: NewConversationTracker[string](duration),
		token:               config["token"],
		defaultChannel:      config["defaultChannel"],
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
	messages, err = s.ChannelMessages(m.ChannelID, 100, "", m.MessageReference.MessageID, "")
	if err != nil {
		xlog.Info("error getting messages,", err)
		return
	}

	conv := []openai.ChatCompletionMessage{}

	for _, message := range messages {
		if message.Author.ID == s.State.User.ID {
			conv = append(conv, openai.ChatCompletionMessage{
				Role:    "assistant",
				Content: message.Content,
			})
		} else {
			conv = append(conv, openai.ChatCompletionMessage{
				Role:    "user",
				Content: message.Content,
			})
		}
	}

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

	conv := d.conversationTracker.GetConversation(m.ChannelID)

	d.conversationTracker.AddMessage(m.ChannelID, openai.ChatCompletionMessage{
		Role:    "user",
		Content: m.Content,
	})

	jobResult := a.Ask(
		types.WithConversationHistory(conv),
	)

	if jobResult.Error != nil {
		xlog.Info("error asking agent,", jobResult.Error)
		return
	}

	d.conversationTracker.AddMessage(m.ChannelID, openai.ChatCompletionMessage{
		Role:    "assistant",
		Content: jobResult.Response,
	})

	_, err := s.ChannelMessageSend(m.ChannelID, jobResult.Response)
	if err != nil {
		xlog.Info("error sending message,", err)
	}
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

		// Interact if we are mentioned
		mentioned := false
		for _, mention := range m.Mentions {
			if mention.ID == s.State.User.ID {
				mentioned = true
				return
			}
		}

		if !mentioned && d.defaultChannel == "" {
			xlog.Debug("Not mentioned")
			return
		}

		// check if the message is in a thread and get all messages in the thread
		if m.MessageReference != nil &&
			((d.defaultChannel != "" && m.ChannelID == d.defaultChannel) || (mentioned && d.defaultChannel == "")) {
			d.handleThreadMessage(a, s, m)
			return
		}

		// Or we are in the default channel (if one is set!)
		if (d.defaultChannel != "" && m.ChannelID == d.defaultChannel) || (mentioned && d.defaultChannel == "") {
			d.handleChannelMessage(a, s, m)
			return
		}
	}
}
