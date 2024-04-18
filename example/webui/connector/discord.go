package connector

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/mudler/local-agent-framework/agent"
	"github.com/mudler/local-agent-framework/xlog"
)

type Discord struct {
	token          string
	defaultChannel string
}

func NewDiscord(config map[string]string) *Discord {
	return &Discord{
		token:          config["token"],
		defaultChannel: config["defaultChannel"],
	}
}

func (d *Discord) AgentResultCallback() func(state agent.ActionState) {
	return func(state agent.ActionState) {
		// Send the result to the bot
	}
}

func (d *Discord) AgentReasoningCallback() func(state agent.ActionCurrentState) bool {
	return func(state agent.ActionCurrentState) bool {
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

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func (d *Discord) messageCreate(a *agent.Agent) func(s *discordgo.Session, m *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore all messages created by the bot itself
		// This isn't required in this specific example but it's a good practice.
		if m.Author.ID == s.State.User.ID {
			return
		}
		interact := func() {
			//m := m.ContentWithMentionsReplaced()
			content := m.Content

			content = strings.ReplaceAll(content, "<@"+s.State.User.ID+"> ", "")
			xlog.Info("Received message", "content", content)
			job := a.Ask(
				agent.WithText(
					content,
				),
			)
			if job.Error != nil {
				xlog.Info("error asking agent,", job.Error)
				return
			}

			xlog.Info("Response", "response", job.Response)
			_, err := s.ChannelMessageSend(m.ChannelID, job.Response)
			if err != nil {
				xlog.Info("error sending message,", err)
			}
		}

		// Interact if we are mentioned
		for _, mention := range m.Mentions {
			if mention.ID == s.State.User.ID {
				go interact()
				return
			}
		}

		// Or we are in the default channel (if one is set!)
		if d.defaultChannel != "" && m.ChannelID == d.defaultChannel {
			go interact()
			return
		}
	}
}
