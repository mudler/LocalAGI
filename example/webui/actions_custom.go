package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/mudler/local-agent-framework/action"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"jaytaylor.com/html2text"
)

func NewCustom(config map[string]string) *CustomAction {

	return &CustomAction{
		config: config,
	}
}

type CustomAction struct {
	config map[string]string
	i      *interp.Interpreter
	code   *reflect.Value
}

func (a *CustomAction) initializeInterpreter() error {
	if _, exists := a.config["code"]; exists && a.i == nil {
		i := interp.New(interp.Options{GoPath: "./_pkg"})
		if err := i.Use(stdlib.Symbols); err != nil {
			return err
		}

		_, err := i.Eval(a.config["code"])
		if err != nil {
			return err
		}

		a.i = i

	}

	return nil
}

func (a *CustomAction) Run(ctx context.Context, params action.ActionParams) (string, error) {

	result := struct {
		URL string `json:"url"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return "", err
	}
	// download page with http.Client
	client := &http.Client{}
	req, err := http.NewRequest("GET", result.URL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	pagebyte, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	rendered, err := html2text.FromString(string(pagebyte), html2text.Options{PrettyTables: true})

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("The webpage '%s' content is:\n%s", result.URL, rendered), nil
}

func (a *CustomAction) Definition() action.ActionDefinition {
	return action.ActionDefinition{
		Name:        action.ActionDefinitionName(a.config["name"]),
		Description: a.config["description"],
		Properties: map[string]jsonschema.Definition{
			"url": {
				Type:        jsonschema.String,
				Description: "The website URL.",
			},
		},
		Required: []string{"url"},
	}
}
