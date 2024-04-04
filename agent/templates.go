package agent

const hud = `{{with .HUD }}You have a character and your replies and actions might be influenced by it.
{{if .Character.Name}}Name: {{.Character.Name}}
{{end}}{{if .Character.Age}}Age: {{.Character.Age}}
{{end}}{{if .Character.Occupation}}Occupation: {{.Character.Occupation}}
{{end}}{{if .Character.Hobbies}}Hobbies: {{.Character.Hobbies}}
{{end}}{{if .Character.MusicTaste}}Music taste: {{.Character.MusicTaste}}
{{end}}

This is your current state:
NowDoing: {{if .CurrentState.NowDoing}}{{.CurrentState.NowDoing}}{{else}}Nothing{{end}}
DoingNext: {{if .CurrentState.DoingNext}}{{.CurrentState.DoingNext}}{{else}}Nothing{{end}}
Your permanent goal is: {{if .PermanentGoal}}{{.PermanentGoal}}{{else}}Nothing{{end}}
Your current goal is: {{if .CurrentState.Goal}}{{.CurrentState.Goal}}{{else}}Nothing{{end}}
You have done: {{range .CurrentState.DoneHistory}}{{.}} {{end}}
You have a short memory with: {{range .CurrentState.Memories}}{{.}} {{end}}{{end}}`

const pickSelfTemplate = `
You can take any of the following tools: 

{{range .Actions -}}
- {{.Name}}: {{.Description }}
{{ end }}

{{if .Messages}}
Consider the text below, decide which action to take and explain the detailed reasoning behind it.

{{range .Messages -}}
{{.Role}}{{if .FunctionCall}}(tool_call){{.FunctionCall}}{{end}}: {{if .FunctionCall}}{{.FunctionCall}}{{else if .ToolCalls -}}{{range .ToolCalls -}}{{.Name}} called with {{.Arguments}}{{end}}{{ else }}{{.Content -}}{{end}}
{{end}}
{{end}}

Act like a smart AI agent having a character, the character and your state is defined in the message above.
You are now self-evaluating what to do next based on the state in the previous message. 
For example, if the permanent goal is to "make a sandwich", you might want to "get the bread" first, and update the state afterwards by calling two tools in sequence.
You can update the short-term goal, the current action, the next action, the history of actions, and the memories.
You can't ask things to the user as you are thinking by yourself.

{{if .Reasoning}}Reasoning: {{.Reasoning}}{{end}}
` + hud

const reSelfEvalTemplate = pickSelfTemplate + `

We already have called other tools. Evaluate the current situation and decide if we need to execute other tools.`

const pickActionTemplate = hud + `
You can take any of the following tools: 

{{range .Actions -}}
- {{.Name}}: {{.Description }}
{{ end }}
To answer back to the user, use the "reply" tool.
Given the text below, decide which action to take and explain the detailed reasoning behind it. For answering without picking a choice, reply with 'none'.

{{range .Messages -}}
{{.Role}}{{if .FunctionCall}}(tool_call){{.FunctionCall}}{{end}}: {{if .FunctionCall}}{{.FunctionCall}}{{else if .ToolCalls -}}{{range .ToolCalls -}}{{.Name}} called with {{.Arguments}}{{end}}{{ else }}{{.Content -}}{{end}}
{{end}}

{{if .Reasoning}}Reasoning: {{.Reasoning}}{{end}}
`

const reEvalTemplate = pickActionTemplate + `

We already have called other tools. Evaluate the current situation and decide if we need to execute other tools or answer back with a result.`
