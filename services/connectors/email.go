package connectors

import (
	"bytes"
	"fmt"
	"mime"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	imap "github.com/emersion/go-imap/v2"
	sasl "github.com/emersion/go-sasl"
	smtp "github.com/emersion/go-smtp"

	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/charset"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"

	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/sashabaranov/go-openai"
)

type Email struct {
	username     string
	name         string
	password     string
	email        string
	smtpServer   string
	smtpInsecure bool
	imapServer   string
	imapInsecure bool
}

func NewEmail(config map[string]string) *Email {

	return &Email{
		username:     config["username"],
		name:         config["name"],
		password:     config["password"],
		email:        config["email"],
		smtpServer:   config["smtpServer"],
		smtpInsecure: config["smtpInsecure"] == "true",
		imapServer:   config["imapServer"],
		imapInsecure: config["imapInsecure"] == "true",
	}
}

func EmailConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "smtpServer",
			Label:    "SMTP Host:port",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "SMTP server host:port (e.g., smtp.gmail.com:587)",
		},
		{
			Name:  "smtpInsecure",
			Label: "Insecure SMTP",
			Type:  config.FieldTypeCheckbox,
		},
		{
			Name:     "imapServer",
			Label:    "IMAP Host:port",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "IMAP server host:port (e.g., imap.gmail.com:993)",
		},
		{
			Name:  "imapInsecure",
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
			Label:    "Password",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "SMTP/IMAP password or app password",
		},
		{
			Name:     "email",
			Label:    "From Email",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "Agent email address",
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

func (e *Email) sendMail(to, subject, content, replyToID, references string, emails []string, html bool) {

	auth := sasl.NewPlainClient("", e.username, e.password)

	contentType := "text/plain"
	if html {
		contentType = "text/html"
	}

	var replyHeaders string
	if replyToID != "" {
		referenceLine := strings.ReplaceAll(references+" "+replyToID, "\n", "")
		replyHeaders = fmt.Sprintf("In-Reply-To: %s\r\nReferences: %s\r\n", replyToID, referenceLine)
	}

	// Build full message content
	var builder strings.Builder
	fmt.Fprintf(&builder, "To: %s\r\n", to)
	fmt.Fprintf(&builder, "From: %s <%s>\r\n", e.name, e.email)
	builder.WriteString(replyHeaders)
	fmt.Fprintf(&builder, "MIME-Version: 1.0\r\nContent-Type: %s;\r\n", contentType)
	fmt.Fprintf(&builder, "Subject: %s\r\n\r\n", subject)
	fmt.Fprintf(&builder, "%s\r\n", content)
	msg := strings.NewReader(builder.String())

	if !e.smtpInsecure {

		err := smtp.SendMail(e.smtpServer, auth, e.email, emails, msg)
		if err != nil {
			xlog.Error(fmt.Sprintf("Email send err: %v", err))
		}

	} else {

		c, err := smtp.Dial(e.smtpServer)
		if err != nil {
			xlog.Error(fmt.Sprintf("Email connection err: %v", err))
		}
		defer c.Close()

		err = c.Hello("client")
		if err != nil {
			xlog.Error(fmt.Sprintf("Email hello err: %v", err))
		}

		err = c.Auth(auth)
		if err != nil {
			xlog.Error(fmt.Sprintf("Email auth err: %v", err))
		}

		err = c.SendMail(e.email, emails, msg)
		if err != nil {
			xlog.Error(fmt.Sprintf("Email send err: %v", err))
		}

	}
}

func imapWorker(done chan bool, e *Email, a *agent.Agent, c *imapclient.Client, startIndex uint32) {

	currentIndex := startIndex

	for {
		select {
		case <-done:

			xlog.Info("Stopping imapWorker")
			err := c.Logout().Wait()
			if err != nil {
				xlog.Error(fmt.Sprintf("Email IMAP logout fail: %v", err))
			}
			return

		default:

			selectedMbox, err := c.Select("INBOX", nil).Wait()
			if err != nil {
				xlog.Error(fmt.Sprintf("Email IMAP mailbox err: %v", err))
			}

			// Loop over any new messages recieved in selected mailbox
			for currentIndex < selectedMbox.NumMessages {

				currentIndex++

				// Download email info
				seqSet := imap.SeqSetNum(currentIndex)
				bodySection := &imap.FetchItemBodySection{}
				fetchOptions := &imap.FetchOptions{
					Flags:       true,
					Envelope:    true,
					BodySection: []*imap.FetchItemBodySection{bodySection},
				}
				messageBuffers, err := c.Fetch(seqSet, fetchOptions).Collect()
				if err != nil {
					xlog.Error(fmt.Sprintf("Email IMAP fetch err: %v", err))
				}

				// Start conversation goroutine
				go func(e *Email, a *agent.Agent, c *imapclient.Client, fmb *imapclient.FetchMessageBuffer) {

					// Download Email contents
					r := bytes.NewReader(fmb.FindBodySection(bodySection))
					msg, err := message.Read(r)
					if err != nil {
						xlog.Error(fmt.Sprintf("Email reader err: %v", err))
					}
					buf := new(bytes.Buffer)
					buf.ReadFrom(msg.Body)

					xlog.Debug("New email!")
					xlog.Debug(fmt.Sprintf("From: %s", msg.Header.Get("From")))
					xlog.Debug(fmt.Sprintf("To: %s", msg.Header.Get("To")))
					xlog.Debug(fmt.Sprintf("Subject: %s", msg.Header.Get("Subject")))

					// In the event that an email account has multiple email addresses, only respond to the one configured
					if !strings.Contains(msg.Header.Get("To"), e.email) {
						xlog.Info(fmt.Sprintf("Email was sent to %s, but appeared in my inbox (%s). Ignoring!", msg.Header.Get("To"), e.email))
						return
					}

					content := buf.String()
					contentIsHTML := false

					// Convert email to markdown only if it's in HTML
					prefixes := []string{"<html", "<body", "<div", "<head"}
					for _, prefix := range prefixes {
						if strings.HasPrefix(strings.ToLower(content), prefix) {
							content, err = htmltomarkdown.ConvertString(buf.String())
							contentIsHTML = true
							if err != nil {
								xlog.Error(fmt.Sprintf("Email html => md err: %v", err))
								contentIsHTML = false
								content = buf.String()
							}
						}
					}

					xlog.Debug(fmt.Sprintf("Markdown:\n\n%s", content))

					// Construct prompt
					prompt := fmt.Sprintf("%s %s:\n\nFrom: %s\nTime: %s\nSubject: %s\n=====\n%s",
						"This email thread was sent to you. You are",
						e.email,
						msg.Header.Get("From"),
						fmb.Envelope.Date.Format(time.RFC3339),
						fmb.Envelope.Subject,
						content,
					)
					conv := []openai.ChatCompletionMessage{}
					conv = append(conv, openai.ChatCompletionMessage{Role: "user", Content: prompt})

					// Send prompt to agent and wait for result
					xlog.Debug(fmt.Sprintf("Starting conversation:\n\n%v", conv))
					jobResult := a.Ask(types.WithConversationHistory(conv))
					if jobResult.Error != nil {
						xlog.Error(fmt.Sprintf("Error asking agent: %v", jobResult.Error))
					}

					// Send agent response to user, replying to original email.
					xlog.Debug("Agent finished responding. Sending reply email to user")

					// Get a list of emails to respond to ("Reply All" logic)
					// This could be done through regex, but it's probably safer to rebuild explicitly
					fromEmail := fmt.Sprintf("%s@%s", fmb.Envelope.From[0].Mailbox, fmb.Envelope.From[0].Host)
					emails := []string{}
					emails = append(emails, fromEmail)

					for _, addr := range fmb.Envelope.To {
						if addr.Mailbox != "" && addr.Host != "" {
							email := fmt.Sprintf("%s@%s", addr.Mailbox, addr.Host)
							if email != e.email {
								emails = append(emails, email)
							}
						}
					}

					// Keep the original header, in case sender had contact names as part of the header
					newToHeader := msg.Header.Get("From") + ", " + filterEmailRecipients(msg.Header.Get("To"), e.email)

					// Create the body of the email
					replyContent := jobResult.Response
					if jobResult.Response == "" {
						replyContent =
							"System: I'm sorry, but it looks like the agent did not respond. " +
								"This could be in error, or maybe it had nothing to say."
					}

					// Quote the original message. This lets the agent see conversation history and is an email standard.
					quoteHeader := fmt.Sprintf("\r\n\r\nOn %s, %s wrote:\n",
						fmb.Envelope.Date.Format("Monday, Jan 2, 2006 at 15:04"),
						fmt.Sprintf("%s <%s>", fmb.Envelope.From[0].Name, fromEmail),
					)
					quotedLines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
					for i, line := range quotedLines {
						quotedLines[i] = "> " + line
					}
					replyContent = replyContent + quoteHeader + strings.Join(quotedLines, "\r\n")

					// If the original email was sent in HTML, reply with HTML
					if contentIsHTML {
						p := parser.NewWithExtensions(parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock)
						doc := p.Parse([]byte(replyContent))

						opts := html.RendererOptions{Flags: html.CommonFlags | html.HrefTargetBlank | html.CompletePage}
						renderer := html.NewRenderer(opts)

						replyContent = string(markdown.Render(doc, renderer))
					}

					// Send the email
					e.sendMail(newToHeader,
						fmt.Sprintf("Re: %s", msg.Header.Get("Subject")),
						replyContent,
						msg.Header.Get("Message-ID"),
						msg.Header.Get("References"),
						emails,
						contentIsHTML,
					)
				}(e, a, c, messageBuffers[0])
			}
			time.Sleep(5 * time.Second) // Refresh inbox every n seconds
		}
	}
}

func (e *Email) Start(a *agent.Agent) {
	go func() {

		xlog.Info("Email connector is now running.  Press CTRL-C to exit.")
		// IMAP dial
		imapOpts := &imapclient.Options{WordDecoder: &mime.WordDecoder{CharsetReader: charset.Reader}}
		var c *imapclient.Client
		var err error
		if e.imapInsecure {
			c, err = imapclient.DialInsecure(e.imapServer, imapOpts)
		} else {
			c, err = imapclient.DialTLS(e.imapServer, imapOpts)
		}

		if err != nil {
			xlog.Error(fmt.Sprintf("Email IMAP dial err: %v", err))
			return
		}
		defer c.Close()

		// IMAP login
		err = c.Login(e.username, e.password).Wait()
		if err != nil {
			xlog.Error(fmt.Sprintf("Email IMAP login err: %v", err))
			return
		}

		// IMAP mailbox
		mailboxes, err := c.List("", "%", nil).Collect()
		if err != nil {
			xlog.Error(fmt.Sprintf("Email IMAP mailbox err: %v", err))
			return
		}

		xlog.Debug(fmt.Sprintf("Email IMAP mailbox count: %v", len(mailboxes)))
		for _, mbox := range mailboxes {
			xlog.Debug(fmt.Sprintf(" - %v", mbox.Mailbox))
		}

		// Select INBOX
		selectedMbox, err := c.Select("INBOX", nil).Wait()
		if err != nil {
			xlog.Error(fmt.Sprintf("Cannot select INBOX mailbox! %v", err))
			return
		}
		xlog.Debug(fmt.Sprintf("INBOX contains %v messages", selectedMbox.NumMessages))

		// Start checking INBOX for new mail
		imapWorkerHandle := make(chan bool)
		go imapWorker(imapWorkerHandle, e, a, c, selectedMbox.NumMessages)

		<-a.Context().Done()
		imapWorkerHandle <- true
		xlog.Info("Email connector is now stopped.")

	}()
}
