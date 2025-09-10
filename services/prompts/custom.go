package prompts

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

type DynamicCustomPrompt struct {
	config    map[string]string
	goPkgPath string
	i         *interp.Interpreter
}

func NewDynamicCustomPrompt(config map[string]string, goPkgPath string) (*DynamicCustomPrompt, error) {
	a := &DynamicCustomPrompt{
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

func (a *DynamicCustomPrompt) callInit() error {
	if a.i == nil {
		return nil
	}

	v, err := a.i.Eval(fmt.Sprintf("%s.Init", a.config["name"]))
	if err != nil {
		xlog.Warn("No init function found for custom prompt", "error", err, "action", a.config["name"])
		return nil
	}

	run := v.Interface().(func() error)

	return run()
}

func NewDynamicPromptConfigMeta() config.FieldGroup {
	return config.FieldGroup{
		Name:  "custom",
		Label: "Custom Prompt",
		Fields: []config.Field{
			{
				Name:        "name",
				Label:       "Name",
				Type:        config.FieldTypeText,
				Required:    true,
				HelpText:    "A unique name for your custom prompt",
				Placeholder: "Enter a unique name",
			},
			{
				Name:        "code",
				Label:       "Go Code",
				Type:        config.FieldTypeTextarea,
				Required:    true,
				HelpText:    "Enter code that implements the Render and Role functions here",
				Placeholder: "Write your Go code here",
			},
			{
				Name:     "unsafe",
				Label:    "Unsafe Code",
				Type:     config.FieldTypeCheckbox,
				Required: false,
				HelpText: "Enable if the code needs to use unsafe Go features",
			},
		},
	}
}

func (a *DynamicCustomPrompt) initializeInterpreter() error {
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

func (a *DynamicCustomPrompt) CanRender() bool {
	_, err := a.i.Eval(fmt.Sprintf("%s.Render", a.config["name"]))
	if err != nil {
		return false
	}

	return true
}

func (a *DynamicCustomPrompt) Render(c *agent.Agent) (types.PromptResult, error) {
	v, err := a.i.Eval(fmt.Sprintf("%s.Render", a.config["name"]))
	if err != nil {
		return types.PromptResult{}, err
	}

	run := v.Interface().(func() (string, string, error))
	content, image, err := run()
	if err != nil {
		return types.PromptResult{}, err
	}

	return types.PromptResult{
		Content:     content,
		ImageBase64: image,
	}, nil
}

func (a *DynamicCustomPrompt) Role() string {
	v, err := a.i.Eval(fmt.Sprintf("%s.Role", a.config["name"]))
	if err != nil {
		return "system"
	}

	run := v.Interface().(func() string)

	return run()
}
