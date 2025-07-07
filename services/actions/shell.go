package actions

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/sashabaranov/go-openai/jsonschema"
	"golang.org/x/crypto/ssh"
)

func NewShell(config map[string]string, sshBoxURL string) *ShellAction {
	return &ShellAction{
		privateKey:        config["privateKey"],
		user:              config["user"],
		host:              config["host"],
		password:          config["password"],
		customName:        config["customName"],
		customDescription: config["customDescription"],
		sshBoxURL:         sshBoxURL,
	}
}

type ShellAction struct {
	privateKey           string
	user, host, password string
	customName           string
	customDescription    string
	sshBoxURL            string
}

func (a *ShellAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Command string `json:"command"`
		Host    string `json:"host"`
		User    string `json:"user"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return types.ActionResult{}, err
	}

	if a.host != "" && a.user != "" {
		result.Host = a.host
		result.User = a.user
	}

	password := a.password
	if a.sshBoxURL != "" && result.Host == "" && result.User == "" && password == "" {
		// sshbox url can be root:root@localhost:2222
		parts := strings.Split(a.sshBoxURL, "@")
		if len(parts) == 2 {
			if strings.Contains(parts[0], ":") {
				userPass := strings.Split(parts[0], ":")
				result.User = userPass[0]
				password = userPass[1]
			} else {
				result.User = parts[0]
			}
			result.Host = parts[1]
		}
	}

	output, err := sshCommand(a.privateKey, result.Command, result.User, result.Host, password)
	if err != nil {
		return types.ActionResult{}, err
	}

	return types.ActionResult{Result: output}, nil
}

func (a *ShellAction) Definition() types.ActionDefinition {
	name := "run_command"
	description := "Run a command on a linux environment."
	if a.customName != "" {
		name = a.customName
	}
	if a.customDescription != "" {
		description = a.customDescription
	}
	if (a.host != "" && a.user != "") || a.sshBoxURL != "" {
		return types.ActionDefinition{
			Name:        types.ActionDefinitionName(name),
			Description: description,
			Properties: map[string]jsonschema.Definition{
				"command": {
					Type:        jsonschema.String,
					Description: "The command to run on the remote server.",
				},
			},
			Required: []string{"command"},
		}
	}
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(name),
		Description: description,
		Properties: map[string]jsonschema.Definition{
			"command": {
				Type:        jsonschema.String,
				Description: "The command to run on the remote server.",
			},
			"host": {
				Type:        jsonschema.String,
				Description: "The host of the remote server. e.g. ip:port",
			},
			"user": {
				Type:        jsonschema.String,
				Description: "The user to connect to the remote server.",
			},
		},
		Required: []string{"command", "host", "user"},
	}
}

// ShellConfigMeta returns the metadata for Shell action configuration fields
func ShellConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "privateKey",
			Label:    "Private Key",
			Type:     config.FieldTypeTextarea,
			Required: false,
			HelpText: "SSH private key for connecting to remote servers",
		},
		{
			Name:     "user",
			Label:    "Default User",
			Type:     config.FieldTypeText,
			HelpText: "Default SSH user for connecting to remote servers",
		},
		{
			Name:     "password",
			Label:    "Default Password",
			Type:     config.FieldTypeText,
			HelpText: "Default SSH password for connecting to remote servers",
		},
		{
			Name:     "host",
			Label:    "Default Host",
			Type:     config.FieldTypeText,
			HelpText: "Default host for SSH connections (e.g., hostname:port)",
		},
		{
			Name:     "customName",
			Label:    "Custom Action Name",
			Type:     config.FieldTypeText,
			HelpText: "Custom name for this action",
		},
		{
			Name:     "customDescription",
			Label:    "Custom Description",
			Type:     config.FieldTypeTextarea,
			HelpText: "Custom description for this action",
		},
	}
}

func sshCommand(privateKey, command, user, host, password string) (string, error) {

	authMethods := []ssh.AuthMethod{}
	if password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}
	if privateKey != "" {
		// Create signer from private key string
		key, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err != nil {
			log.Fatalf("failed to parse private key: %v", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(key))
	}

	// SSH client configuration
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to SSH server
	client, err := ssh.Dial("tcp", host, config)
	if err != nil {
		return "", fmt.Errorf("failed to dial: %v", err)
	}
	defer client.Close()

	// Open a new session
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	// Run a command
	cmdOut, err := session.CombinedOutput(command)
	result := string(cmdOut)
	if strings.TrimSpace(result) == "" {
		result += "\nCommand has exited with no output"
	}
	if err != nil {
		result += "\nError: " + err.Error()
	}
	return result, nil
}

func (a *ShellAction) Plannable() bool {
	return true
}
