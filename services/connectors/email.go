package connectors

import (
	"fmt"
	"strings"
	"mime"
	"time"

	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/pkg/xlog"
	sasl "github.com/emersion/go-sasl"
	smtp "github.com/emersion/go-smtp"
	imap "github.com/emersion/go-imap/v2"
    "github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/charset"
	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
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

func (e *Email) sendMail(toAddr, subject, content string) {
	auth := sasl.NewPlainClient("", e.username, e.password)

	to := []string{toAddr}
	if !e.smtpIns {
		msg := strings.NewReader(
			fmt.Sprintf("To: %s\r\n", toAddr) +
			fmt.Sprintf("From: %s <%s>\r\n", e.name, e.email) +
			fmt.Sprintf("Subject: %s\r\n\r\n", subject) +
			fmt.Sprintf("%s\r\n", content),
		)
		err := smtp.SendMail(e.smtpUri, auth, e.email, to, msg)
		if err != nil { xlog.Info(fmt.Sprintf("Email send err: %v", err)) }
	} else {
		c, err := smtp.Dial(e.smtpUri)
		if err != nil { xlog.Info(fmt.Sprintf("Email connection err: %v", err)) }
		defer c.Close()

		err = c.Hello("hello")
		if err != nil { xlog.Info(fmt.Sprintf("Email hello err: %v", err)) }

		err = c.Auth(auth)
		if err != nil { xlog.Info(fmt.Sprintf("Email auth err: %v", err)) }

		msg := strings.NewReader(
			fmt.Sprintf("To: %s\r\n", toAddr) +
			fmt.Sprintf("From: %s <%s>\r\n", e.name, e.email) +
			fmt.Sprintf("Subject: %s\r\n\r\n", subject) +
			fmt.Sprintf("%s\r\n", content),
		)
		err = c.SendMail(e.email, to, msg)
		if err != nil { xlog.Info(fmt.Sprintf("Email send err: %v", err)) }
	}
}

func (e *Email) Start(a *agent.Agent) {
	go func() {
		xlog.Info("Email connector is now running.  Press CTRL-C to exit.")

		searchSinceTime := time.Now().Add(-100 * time.Hour)
		xlog.Info(fmt.Sprintf("%v", searchSinceTime))

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

		// data, err := c.UIDSearch(&imap.SearchCriteria{
		// 	NotFlag: []imap.Flag{imap.FlagSeen},
		// 	//Since: searchSinceTime,
		// }, nil).Wait()
		// if err != nil { xlog.Info(fmt.Sprintf("Email IMAP search err: %v", err)) }
		// xlog.Info(fmt.Sprintf("Email search UUID: %v", data.AllUIDs()))
		// xlog.Info(fmt.Sprintf("Email search AllSeqNums: %v", data.AllSeqNums()))
		// xlog.Info(fmt.Sprintf("Email search AllString: %v", data.All.String()))

		if true {
			// Do something with each seqNum
			seqSet := imap.SeqSetNum(1)
			bodySection := &imap.FetchItemBodySection{}
			fetchOptions := &imap.FetchOptions{
				Flags:       true,
				Envelope:    true,
				BodySection: []*imap.FetchItemBodySection{bodySection},
			}
			messages, err := c.Fetch(seqSet, fetchOptions).Collect()
			if err != nil { xlog.Info(fmt.Sprintf("Email IMAP fetch err: %v", err)) }

			msg := messages[0]
			header, body, _ := strings.Cut(string(msg.FindBodySection(bodySection)), "\r\n\r\n")
			markdown, err := htmltomarkdown.ConvertString(body)
	        if err != nil { xlog.Info(fmt.Sprintf("Email html => md err: %v", err)) }

			xlog.Info(fmt.Sprintf("Flags: %v", msg.Flags))
			xlog.Info(fmt.Sprintf("Subject: %v", msg.Envelope.Subject))
			xlog.Info(fmt.Sprintf("Header:\n%v", header))
			xlog.Info(fmt.Sprintf("Body:\n%s", markdown))

			xlog.Info(fmt.Sprintf("Processing email with sequence number: %v", 1))
		}
		
		
		// if selectedMbox.NumMessages > 0 {
		// 	seqSet := imap.SeqSetNum(1)
		// 	fetchOptions := &imap.FetchOptions{Envelope: true}
		// 	messages, err := c.Fetch(seqSet, fetchOptions).Collect()
		// 	if err != nil {
		// 		log.Fatalf("failed to fetch first message in INBOX: %v", err)
		// 	}
		// 	log.Printf("subject of first message in INBOX: %v", messages[0].Envelope.Subject)
		// }

		err = c.Logout().Wait()
		if err != nil { xlog.Info(fmt.Sprintf("Email IMAP logout fail: %v", err)) }
		
		<-a.Context().Done()
		xlog.Info("Email connector is now stopped.")
	}()
}