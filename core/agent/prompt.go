package agent

import (
	"fmt"
	"strings"

	"github.com/mudler/LocalAgent/pkg/xlog"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

type PromptBlock interface {
	Render(a *Agent) (string, error)
	Role() string
}

type DynamicPrompt struct {
	config    map[string]string
	goPkgPath string
	i         *interp.Interpreter
}

func NewDynamicPrompt(config map[string]string, goPkgPath string) (*DynamicPrompt, error) {
	a := &DynamicPrompt{
		config:    config,
		goPkgPath: goPkgPath,
	}

	if err := a.initializeInterpreter(); err != nil {
		return nil, err
	}

	if err := a.callInit(); err != nil {
		xlog.Error("Error calling custom action init", "error", err)
	}

	return a, nil
}

func (a *DynamicPrompt) callInit() error {
	if a.i == nil {
		return nil
	}

	v, err := a.i.Eval(fmt.Sprintf("%s.Init", a.config["name"]))
	if err != nil {
		return err
	}

	run := v.Interface().(func() error)

	return run()
}

func (a *DynamicPrompt) initializeInterpreter() error {
	if _, exists := a.config["code"]; exists && a.i == nil {
		unsafe := strings.ToLower(a.config["unsafe"]) == "true"
		i := interp.New(interp.Options{
			GoPath:       a.goPkgPath,
			Unrestricted: unsafe,
		})
		if err := i.Use(stdlib.Symbols); err != nil {
			return err
		}

		if _, exists := a.config["name"]; !exists {
			a.config["name"] = "custom"
		}

		_, err := i.Eval(fmt.Sprintf("package %s\n%s", a.config["name"], a.config["code"]))
		if err != nil {
			return err
		}

		a.i = i
	}

	return nil
}

func (a *DynamicPrompt) Render(c *Agent) (string, error) {
	v, err := a.i.Eval(fmt.Sprintf("%s.Render", a.config["name"]))
	if err != nil {
		return "", err
	}

	run := v.Interface().(func() (string, error))

	return run()
}

func (a *DynamicPrompt) Role() string {
	v, err := a.i.Eval(fmt.Sprintf("%s.Role", a.config["name"]))
	if err != nil {
		return "system"
	}

	run := v.Interface().(func() string)

	return run()
}
