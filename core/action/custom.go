package action

import (
	"context"
	"fmt"
	"strings"

	"github.com/mudler/LocalAgent/core/types"
	"github.com/mudler/LocalAgent/pkg/xlog"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

func NewCustom(config map[string]string, goPkgPath string) (*CustomAction, error) {
	a := &CustomAction{
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

type CustomAction struct {
	config    map[string]string
	goPkgPath string
	i         *interp.Interpreter
}

func (a *CustomAction) callInit() error {
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

func (a *CustomAction) initializeInterpreter() error {
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

func (a *CustomAction) Plannable() bool {
	return true
}

func (a *CustomAction) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	v, err := a.i.Eval(fmt.Sprintf("%s.Run", a.config["name"]))
	if err != nil {
		return types.ActionResult{}, err
	}

	run := v.Interface().(func(map[string]interface{}) (string, map[string]interface{}, error))

	res, meta, err := run(params)
	return types.ActionResult{Result: res, Metadata: meta}, err
}

func (a *CustomAction) Definition() types.ActionDefinition {

	v, err := a.i.Eval(fmt.Sprintf("%s.Definition", a.config["name"]))
	if err != nil {
		xlog.Error("Error getting custom action definition", "error", err)
		return types.ActionDefinition{}
	}

	properties := v.Interface().(func() map[string][]string)

	v, err = a.i.Eval(fmt.Sprintf("%s.RequiredFields", a.config["name"]))
	if err != nil {
		xlog.Error("Error getting custom action definition", "error", err)
		return types.ActionDefinition{}
	}

	requiredFields := v.Interface().(func() []string)

	prop := map[string]jsonschema.Definition{}

	for k, v := range properties() {
		if len(v) != 2 {
			xlog.Error("Invalid property definition", "property", k)
			continue
		}
		prop[k] = jsonschema.Definition{
			Type:        jsonschema.DataType(v[0]),
			Description: v[1],
		}
	}
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(a.config["name"]),
		Description: a.config["description"],
		Properties:  prop,
		Required:    requiredFields(),
	}
}
