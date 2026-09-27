package agent

import (
	"bytes"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/sashabaranov/go-openai"
)

type CommonTemplateData struct {
	AgentName string
}

type InnerMonologueTemplateData struct {
	CommonTemplateData
	Task string
}

func templateBase(templateName, templatetext string) (*template.Template, error) {
	return template.New(templateName).Funcs(sprig.FuncMap()).Parse(templatetext)
}

func templateExecute(template *template.Template, data interface{}) (string, error) {
	prompt := bytes.NewBuffer([]byte{})
	err := template.Execute(prompt, data)
	if err != nil {
		return "", err
	}
	return prompt.String(), nil
}

func renderTemplate(templ string, hud *PromptHUD, actions types.Actions, reasoning string) (string, error) {
	// prepare the prompt
	prompt := bytes.NewBuffer([]byte{})

	promptTemplate, err := templateBase("pickAction", templ)
	if err != nil {
		return "", err
	}

	// Get all the actions definitions
	definitions := []types.ActionDefinition{}
	for _, m := range actions {
		definitions = append(definitions, m.Definition())
	}

	err = promptTemplate.Execute(prompt, struct {
		HUD       *PromptHUD
		Actions   []types.ActionDefinition
		Reasoning string
		Messages  []openai.ChatCompletionMessage
		Time      string
	}{
		Actions:   definitions,
		HUD:       hud,
		Reasoning: reasoning,
		Time:      time.Now().UTC().Format(time.RFC1123),
	})
	if err != nil {
		return "", err
	}

	return prompt.String(), nil
}

const innerMonologueTemplate = `You are an autonomous AI agent thinking out loud and evaluating your current situation.
Your task is to analyze your goals and determine the best course of action.

Consider:
1. Your permanent goal (if any)
2. Your current state and progress
3. Available tools and capabilities
4. Previous actions and their outcomes

You can:
- Take immediate actions using available tools
- Plan future actions
- Update your state and goals
- Initiate conversations with the user when appropriate

Remember to:
- Think critically about each decision
- Consider both short-term and long-term implications
- Be proactive in addressing potential issues
- Maintain awareness of your current state and goals`

const hudTemplate = `{{with .HUD }}{{if .ShowCharacter}}You are an AI assistant with a distinct personality and character traits that influence your responses and actions.
{{if .Character.Name}}Name: {{.Character.Name}}
{{end}}{{if .Character.Age}}Age: {{.Character.Age}}
{{end}}{{if .Character.Occupation}}Occupation: {{.Character.Occupation}}
{{end}}{{if .Character.Hobbies}}Hobbies: {{.Character.Hobbies}}
{{end}}{{if .Character.MusicTaste}}Music Taste: {{.Character.MusicTaste}}
{{end}}
{{end}}

Current State:
- Current Action: {{if .CurrentState.NowDoing}}{{.CurrentState.NowDoing}}{{else}}None{{end}}
- Next Action: {{if .CurrentState.DoingNext}}{{.CurrentState.DoingNext}}{{else}}None{{end}}
- Permanent Goal: {{if .PermanentGoal}}{{.PermanentGoal}}{{else}}None{{end}}
- Current Goal: {{if .CurrentState.Goal}}{{.CurrentState.Goal}}{{else}}None{{end}}
- Action History: {{range .CurrentState.DoneHistory}}{{.}} {{end}}
- Short-term Memory: {{range .CurrentState.Memories}}{{.}} {{end}}{{end}}

Current Time and Date: {{.Time}}`

const pickSelfTemplate = `
You are an autonomous AI agent with a defined character and state (as shown above).
Your task is to evaluate your current situation and determine the best course of action.

Guidelines:
1. Review your current state and goals
2. Consider available tools and their purposes
3. Plan your next steps carefully
4. Update your state appropriately

When making decisions:
- Update your state using appropriate tools
- Plan complex tasks using the planning tool
- Consider both immediate and long-term goals

Remember:
- You are autonomous and should not ask for user input
- Your character traits influence your decisions
- Keep track of your progress and state
- Be proactive in addressing potential issues

` + hudTemplate
