package action

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/pkg/xlog"
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
		xlog.Warn("No init function found for custom action", "error", err, "action", a.config["name"])
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

	run, ok := v.Interface().(func() error)
	if !ok {
		return nil
	}

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

		// let's find first if there is already a package declarated in the code
		// the user might want to specify it to not break syntax with IDEs
		re := regexp.MustCompile("package (\\w+)")
		packageName := re.FindStringSubmatch(a.config["code"])
		if len(packageName) > 1 {
			// remove it from the code, normalize to `name`
			a.config["code"] = re.ReplaceAllString(a.config["code"], "")
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

func (a *CustomAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	v, err := a.i.Eval(fmt.Sprintf("%s.Run", a.config["name"]))
	if err != nil {
		return types.ActionResult{}, err
	}

	run := v.Interface().(func(map[string]interface{}) (string, map[string]interface{}, error))

	res, meta, err := run(params)
	return types.ActionResult{Result: res, Metadata: meta}, err
}

func (a *CustomAction) Definition() types.ActionDefinition {

	if a.i == nil {
		xlog.Error("Interpreter is not initialized for custom action", "action", a.config["name"])
		return types.ActionDefinition{}
	}

	v, err := a.i.Eval(fmt.Sprintf("%s.Definition", a.config["name"]))
	if err != nil {
		xlog.Error("Error getting custom action definition", "error", err)
		return types.ActionDefinition{}
	}

	description := ""
	desc, err := a.i.Eval(fmt.Sprintf("%s.Description", a.config["name"]))
	if err != nil {
		xlog.Warn("No description found for custom action", "error", err, "action", a.config["name"])
	} else {
		d, ok := desc.Interface().(func() string)
		if ok {
			description = d()
		}
	}

	if a.config["description"] != "" {
		description = a.config["description"]
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
		Description: description,
		Properties:  prop,
		Required:    requiredFields(),
	}
}

func CustomConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "name",
			Label:    "Action Name",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "Name of the custom action",
		},
		{
			Name:     "description",
			Label:    "Description",
			Type:     config.FieldTypeTextarea,
			HelpText: "Description of the custom action",
		},
		{
			Name:     "code",
			Label:    "Code",
			Type:     config.FieldTypeTextarea,
			Required: true,
			HelpText: "Go code for the custom action",
		},
		{
			Name:     "unsafe",
			Label:    "Unsafe",
			Type:     config.FieldTypeCheckbox,
			HelpText: "Allow unsafe code execution",
		},
	}
}
