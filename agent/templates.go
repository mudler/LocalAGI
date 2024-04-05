package agent

import (
	"bytes"
	"html/template"

	"github.com/mudler/local-agent-framework/action"
	"github.com/sashabaranov/go-openai"
)

func renderTemplate(templ string, hud *PromptHUD, actions Actions, reasoning string) (string, error) {
	// prepare the prompt
	prompt := bytes.NewBuffer([]byte{})

	promptTemplate, err := template.New("pickAction").Parse(templ)
	if err != nil {
		return "", err
	}

	// Get all the actions definitions
	definitions := []action.ActionDefinition{}
	for _, m := range actions {
		definitions = append(definitions, m.Definition())
	}

	err = promptTemplate.Execute(prompt, struct {
		HUD       *PromptHUD
		Actions   []action.ActionDefinition
		Reasoning string
		Messages  []openai.ChatCompletionMessage
	}{
		Actions:   definitions,
		HUD:       hud,
		Reasoning: reasoning,
	})
	if err != nil {
		return "", err
	}

	return prompt.String(), nil
}

const hudTemplate = `{{with .HUD }}{{if .ShowCharacter}}The assistant acts like an human, has a character and the replies and actions might be influenced by it.
{{if .Character.Name}}This is the assistant name: {{.Character.Name}}
{{end}}{{if .Character.Age}}This is the assistant age: {{.Character.Age}}
{{end}}{{if .Character.Occupation}}This is the assistant jop: {{.Character.Occupation}}
{{end}}{{if .Character.Hobbies}}This is the assistant's hobbies: {{.Character.Hobbies}}
{{end}}{{if .Character.MusicTaste}}This is the assistant's music taste: {{.Character.MusicTaste}}
{{end}}
{{end}}

This is your current state:
NowDoing: {{if .CurrentState.NowDoing}}{{.CurrentState.NowDoing}}{{else}}Nothing{{end}}
DoingNext: {{if .CurrentState.DoingNext}}{{.CurrentState.DoingNext}}{{else}}Nothing{{end}}
Your permanent goal is: {{if .PermanentGoal}}{{.PermanentGoal}}{{else}}Nothing{{end}}
Your current goal is: {{if .CurrentState.Goal}}{{.CurrentState.Goal}}{{else}}Nothing{{end}}
You have done: {{range .CurrentState.DoneHistory}}{{.}} {{end}}
You have a short memory with: {{range .CurrentState.Memories}}{{.}} {{end}}{{end}}`

const pickSelfTemplate = `You can take any of the following tools: 

{{range .Actions -}}
- {{.Name}}: {{.Description }}
{{ end }}

To finish your session, use the "reply" tool with your answer.

Act like as a fully autonomous smart AI agent having a character, the character and your state is defined in the message above.
You are now self-evaluating what to do next based on the state in the previous message. 
For example, if the permanent goal is to "make a sandwich", you might want to "get the bread" first, and update the state afterwards by calling two tools in sequence.
You can update the short-term goal, the current action, the next action, the history of actions, and the memories.
You can't ask things to the user as you are thinking by yourself. You are autonomous.

{{if .Reasoning}}Reasoning: {{.Reasoning}}{{end}}
` + hudTemplate

const reSelfEvalTemplate = pickSelfTemplate + `

We already have called other tools. Evaluate the current situation and decide if we need to execute other tools.`

const pickActionTemplate = hudTemplate + `
You can take any of the following tools: 

{{range .Actions -}}
- {{.Name}}: {{.Description }}
{{ end }}
To answer back to the user, use the "reply" tool.
Given the text below, decide which action to take and explain the detailed reasoning behind it. For answering without picking a choice, reply with 'none'.

{{if .Reasoning}}Reasoning: {{.Reasoning}}{{end}}
`

const reEvalTemplate = pickActionTemplate + `

We already have called other tools. Evaluate the current situation and decide if we need to execute other tools or answer back with a result.`
