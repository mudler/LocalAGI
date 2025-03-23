package actions

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/mudler/LocalAgent/core/types"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func NewSendMail(config map[string]string) *SendMailAction {
	return &SendMailAction{
		username: config["username"],
		password: config["password"],
		email:    config["email"],
		smtpHost: config["smtpHost"],
		smtpPort: config["smtpPort"],
	}
}

type SendMailAction struct {
	username string
	password string
	email    string
	smtpHost string
	smtpPort string
}

func (a *SendMailAction) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Message string `json:"message"`
		To      string `json:"to"`
		Subject string `json:"subject"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return types.ActionResult{}, err
	}

	// Authentication.
	auth := smtp.PlainAuth("", a.email, a.password, a.smtpHost)

	// Sending email.
	err = smtp.SendMail(
		fmt.Sprintf("%s:%s", a.smtpHost, a.smtpPort),
		auth, a.email, []string{
			result.To,
		}, []byte(result.Message))
	if err != nil {
		return types.ActionResult{}, err
	}
	return types.ActionResult{Result: fmt.Sprintf("Email sent to %s", result.To)}, nil
}

func (a *SendMailAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        "send_email",
		Description: "Send an email.",
		Properties: map[string]jsonschema.Definition{
			"to": {
				Type:        jsonschema.String,
				Description: "The email address to send the email to.",
			},
			"subject": {
				Type:        jsonschema.String,
				Description: "The subject of the email.",
			},
			"message": {
				Type:        jsonschema.String,
				Description: "The message to send.",
			},
		},
		Required: []string{"to", "subject", "message"},
	}
}

func (a *SendMailAction) Plannable() bool {
	return true
}
