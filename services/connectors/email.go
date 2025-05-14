package connectors

import (
	"bytes"
	"fmt"
	"mime"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	imap "github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/charset"
	sasl "github.com/emersion/go-sasl"
	smtp "github.com/emersion/go-smtp"
	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/sashabaranov/go-openai"
)

type Email struct {
	username string
	name     string
	password string
	email    string
	smtpUri  string
	smtpIns  bool
	imapUri string
	imapIns  bool
}

func NewEmail(config map[string]string) *Email {

	return &Email{
		username: config["username"],
		name:     config["name"],
		password: config["password"],
		email:    config["email"],
		smtpUri:  config["smtpUri"],
		smtpIns:  config["smtpIns"] == "true",
		imapUri:  config["imapUri"],
		imapIns:  config["imapIns"] == "true",
	}
}

func EmailConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "smtpUri",
			Label:    "SMTP Host:port",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "SMTP server host:port (e.g., smtp.gmail.com:587)",
		},
		{
			Name:  "smtpIns",
			Label: "Insecure SMTP",
			Type:  config.FieldTypeCheckbox,
		},
		{
			Name:     "imapUri",
			Label:    "IMAP Host:port",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "IMAP server host:port (e.g., imap.gmail.com:993)",
		},
		{
			Name:  "imapIns",
			Label: "Insecure IMAP",
			Type:  config.FieldTypeCheckbox,
		},
		{
			Name:     "username",
			Label:    "Username",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "Username/email address",
		},
		{
			Name:     "name",
			Label:    "Friendly Name",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "Friendly name of sender",
		},
		{
			Name:     "password",
			Label:    "SMTP Password",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "SMTP password or app password",
		},
		{
			Name:     "email",
			Label:    "From Email",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "Sender email address",
		},
	}
}

func (e *Email) AgentResultCallback() func(state types.ActionState) {
	return func(state types.ActionState) {
		// Send the result to the bot
	}
}

func (e *Email) AgentReasoningCallback() func(state types.ActionCurrentState) bool {
	return func(state types.ActionCurrentState) bool {
		// Send the reasoning to the bot
		return true
	}
}

func (e *Email) sendMail(toHeader, subject, content, replyTo, references string, sendTo []string) {
	auth := sasl.NewPlainClient("", e.username, e.password)

	replyHeaders := ""
	if replyTo != "" { replyHeaders = fmt.Sprintf(
		"In-Reply-To: %s\r\nReferences: %s\r\n",
		replyTo,
		strings.ReplaceAll(references + " " + replyTo, "\n", ""))
	}

	msg := strings.NewReader(
		fmt.Sprintf("To: %s\r\n", toHeader) +
		fmt.Sprintf("From: %s <%s>\r\n", e.name, e.email) +
		replyHeaders + 
		fmt.Sprintf("Subject: %s\r\n\r\n", subject) +
		fmt.Sprintf("%s\r\n", content),
	)
	if !e.smtpIns {

		err := smtp.SendMail(e.smtpUri, auth, e.email, sendTo, msg)
		if err != nil { xlog.Info(fmt.Sprintf("Email send err: %v", err)) }

	} else {

		c, err := smtp.Dial(e.smtpUri)
		if err != nil { xlog.Info(fmt.Sprintf("Email connection err: %v", err)) }
		defer c.Close()

		err = c.Hello("hello")
		if err != nil { xlog.Info(fmt.Sprintf("Email hello err: %v", err)) }

		err = c.Auth(auth)
		if err != nil { xlog.Info(fmt.Sprintf("Email auth err: %v", err)) }

		err = c.SendMail(e.email, sendTo, msg)
		if err != nil { xlog.Info(fmt.Sprintf("Email send err: %v", err)) }

	}
}

func filterEmailRecipients(input string, emailToRemove string) string {

	addresses := strings.Split(strings.TrimPrefix(input, "To: "), ",")

	var filtered []string
	for _, address := range addresses {
		address = strings.TrimSpace(address)
		if !strings.Contains(address, emailToRemove) {
			filtered = append(filtered, address)
		}
	}

	if len(filtered) > 0 {
		return strings.Join(filtered, ", ")
	}
	return ""
}


func imapWorker(done chan bool, e *Email, a *agent.Agent, c *imapclient.Client, startIndex uint32) {

	currentIndex := startIndex

    for {
        select {
        case <-done:
            xlog.Info("Stopping imapWorker")
			err := c.Logout().Wait()
			if err != nil { xlog.Info(fmt.Sprintf("Email IMAP logout fail: %v", err)) }
            return
        default:
			selectedMbox, err := c.Select("INBOX", nil).Wait()
			if err != nil { xlog.Info(fmt.Sprintf("Email IMAP mailbox err: %v", err)) }

			for currentIndex < selectedMbox.NumMessages {
				currentIndex++

				seqSet := imap.SeqSetNum(currentIndex)
				bodySection := &imap.FetchItemBodySection{}
				fetchOptions := &imap.FetchOptions{
					Flags:       true,
					Envelope:    true,
					BodySection: []*imap.FetchItemBodySection{bodySection},
				}
				messages, err := c.Fetch(seqSet, fetchOptions).Collect()
				if err != nil { xlog.Info(fmt.Sprintf("Email IMAP fetch err: %v", err)) }
	
				msg := messages[0]

				go func(e *Email, a *agent.Agent, c *imapclient.Client, msg *imapclient.FetchMessageBuffer){
					r := bytes.NewReader(msg.FindBodySection(bodySection))
					message, err := message.Read(r)
					if err != nil { xlog.Info(fmt.Sprintf("Email reader err: %v", err)) }
		
					xlog.Info(fmt.Sprintf("From: %s", message.Header.Get("From")))
					xlog.Info(fmt.Sprintf("To: %s", message.Header.Get("To")))
					xlog.Info(fmt.Sprintf("Subject: %s", message.Header.Get("Subject")))
					xlog.Info(fmt.Sprintf("Message-ID: %s", message.Header.Get("Message-ID")))
					xlog.Info(fmt.Sprintf("Envelope From: %s", msg.Envelope.From))
					xlog.Info(fmt.Sprintf("Envelope To: %s", msg.Envelope.To))
					xlog.Info(fmt.Sprintf("Envelope Subject: %s", msg.Envelope.Subject))
		
					// Print the body content
					buf := new(bytes.Buffer)
					buf.ReadFrom(message.Body)
				
					// TODO: for some reason, it outputs &gt; when it comes to these >quotes
					markdown, err := htmltomarkdown.ConvertString(buf.String())
					if err != nil { xlog.Info(fmt.Sprintf("Email html => md err: %v", err)) }
					xlog.Info(fmt.Sprintf("Markdown:\n\n%s", markdown))
					
					prompt := fmt.Sprintf("From: %s\nTime: %s\nSubject: %s\n=====\n%s",
						message.Header.Get("From"),
						msg.Envelope.Date.Format(time.RFC3339),
						msg.Envelope.Subject,
						markdown,
					)
					conv := []openai.ChatCompletionMessage{}
					conv = append(conv, openai.ChatCompletionMessage{
						Role:    "user",
						Content: prompt,
					})

					xlog.Info(fmt.Sprintf("Starting conversation:\n\n%v", conv))

					jobResult := a.Ask( types.WithConversationHistory(conv), )
					if jobResult.Error != nil { xlog.Info(fmt.Sprintf("Error asking agent: %v", jobResult.Error)) }

					xlog.Info("Sending reply email")
					emails := []string{}
					emails = append(emails, fmt.Sprintf("%s@%s", 
						msg.Envelope.From[0].Mailbox, msg.Envelope.From[0].Host))

					for _, addr := range msg.Envelope.To {
						if addr.Mailbox != "" && addr.Host != "" {
							email := fmt.Sprintf("%s@%s", addr.Mailbox, addr.Host)
							if email != e.email { emails = append(emails, email) }
						}
					}

					newTos := 
						message.Header.Get("From") +
						", " + filterEmailRecipients(message.Header.Get("To"), e.email)

					replyContent := jobResult.Response
					if jobResult.Response == "" { replyContent = 
						"System: I'm sorry, but it looks like the agent did not respond. This could be in error, or maybe it had nothing to say." }
					quoteHeader := fmt.Sprintf("\n\nOn %s, %s wrote:\n", 
						msg.Envelope.Date.Format("Monday, Jan 2, 2006 at 15:04"),
						fmt.Sprintf("%s <%s@%s>", msg.Envelope.From[0].Name, msg.Envelope.From[0].Mailbox, msg.Envelope.From[0].Host),
					)
					quotedContent := strings.ReplaceAll(markdown, "\r\n", "\n")
					quotedLines := strings.Split(quotedContent, "\n")
					for i, line := range quotedLines {
						quotedLines[i] = "> " + line
					}
					replyContent = replyContent + quoteHeader + strings.Join(quotedLines, "\n")
					e.sendMail(newTos, 
						fmt.Sprintf("Re: %s", message.Header.Get("Subject")),
						replyContent,
						message.Header.Get("Message-ID"),
						message.Header.Get("References"),
						emails,
					)
				}(e, a, c, msg)
			}
            time.Sleep(5 * time.Second)
        }
    }
}

func (e *Email) Start(a *agent.Agent) {
	go func() {
		xlog.Info("Email connector is now running.  Press CTRL-C to exit.")

		options := &imapclient.Options{
			WordDecoder: &mime.WordDecoder{CharsetReader: charset.Reader},
		}
		c, err := imapclient.DialInsecure(e.imapUri, options)
		if err != nil { xlog.Info(fmt.Sprintf("Email IMAP dial err: %v", err)) }
		defer c.Close()

		err = c.Login(e.username, e.password).Wait()
		if err != nil { xlog.Info(fmt.Sprintf("Email IMAP login err: %v", err)) }

		mailboxes, err := c.List("", "%", nil).Collect()
		if err != nil { xlog.Info(fmt.Sprintf("Email IMAP mailbox err: %v", err)) }

		xlog.Info(fmt.Sprintf("Email IMAP mailbox count: %v", len(mailboxes)))
		for _, mbox := range mailboxes {
			xlog.Info(fmt.Sprintf(" - %v", mbox.Mailbox))
		}

		selectedMbox, err := c.Select("INBOX", nil).Wait()
		if err != nil { xlog.Info(fmt.Sprintf("Email IMAP mailbox err: %v", err)) }

		xlog.Info(fmt.Sprintf("Email IMAP mailbox contains %v messages", selectedMbox.NumMessages))

		imapWorkerHandle := make(chan bool)
		go imapWorker(imapWorkerHandle, e, a, c, selectedMbox.NumMessages)

		<-a.Context().Done()
		imapWorkerHandle <- true
		xlog.Info("Email connector is now stopped.")
	}()
}