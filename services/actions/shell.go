package actions

import (
	"context"
	"fmt"
	"log"

	"github.com/mudler/LocalAgent/core/action"
	"github.com/sashabaranov/go-openai/jsonschema"
	"golang.org/x/crypto/ssh"
)

func NewShell(config map[string]string) *ShellAction {
	return &ShellAction{
		privateKey: config["privateKey"],
	}
}

type ShellAction struct {
	privateKey string
}

func (a *ShellAction) Run(ctx context.Context, params action.ActionParams) (action.ActionResult, error) {
	result := struct {
		Command string `json:"command"`
		Host    string `json:"host"`
		User    string `json:"user"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return action.ActionResult{}, err
	}

	output, err := sshCommand(a.privateKey, result.Command, result.User, result.Host)
	if err != nil {
		return action.ActionResult{}, err
	}

	return action.ActionResult{Result: output}, nil
}

func (a *ShellAction) Definition() action.ActionDefinition {
	return action.ActionDefinition{
		Name:        "shell",
		Description: "Run a shell command on a remote server.",
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

func sshCommand(privateKey, command, user, host string) (string, error) {
	// Create signer from private key string
	key, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		log.Fatalf("failed to parse private key: %v", err)
	}

	// SSH client configuration
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
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
	output, err := session.CombinedOutput(command)
	if err != nil {
		return "", fmt.Errorf("failed to run: %v", err)
	}

	return string(output), nil
}
