package connectors

import (
	"fmt"
	"strings"

	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/pkg/xlog"
	sasl "github.com/emersion/go-sasl"
	smtp "github.com/emersion/go-smtp"
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

func (e *Email) sendMail(toAddr, subject, content string, secure bool) {
	auth := sasl.NewPlainClient("", e.username, e.password)

	to := []string{toAddr}
	if secure {
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
		
		<-a.Context().Done()
		xlog.Info("Email connector is now stopped.")
	}()
}