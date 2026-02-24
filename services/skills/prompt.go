package skills

import (
	"bytes"
	"html/template"
	"strings"

	"github.com/mudler/LocalAGI/core/agent"
)

// Skill represents a skill that can be used by the agent
type Skill struct {
	Name        string
	Description string
	ID          string
}

// defaultSkillsTemplate is the default template that mimics the current XML behavior
const defaultSkillsTemplate = `{{.Intro}}<available_skills>
{{range .Skills}}<skill>
  <name>{{.Name}}</name>
  <description>{{.Description}}</description>
</skill>
{{end}}</available_skills>`

// SkillsTemplateData represents the data available for skills template rendering
type SkillsTemplateData struct {
	Intro  string
	Skills []Skill
}

// SkillsPrompt implements agent.DynamicPrompt and injects the available skills using a template
type SkillsPrompt struct {
	listSkills     func() []Skill
	customIntro    string
	customTemplate string
}

// NewSkillsPrompt returns a DynamicPrompt that renders the list of available skills using a template.
// If customTemplate is non-empty, it is used as the template; otherwise the default template is used.
// If customIntro is non-empty, it is used as the intro; otherwise the default intro is used.
func NewSkillsPrompt(listSkills func() []Skill, customIntro string, customTemplate string) agent.DynamicPrompt {
	return &SkillsPrompt{
		listSkills:     listSkills,
		customIntro:    customIntro,
		customTemplate: customTemplate,
	}
}

func (p *SkillsPrompt) Render(a *agent.Agent) (string, error) {
	skills := p.listSkills()

	// Prepare intro
	intro := "You can use the following skills to help with the task.\nTo request the skill, you need to use the `request_skill` tool. The skill name is the name of the skill you want to use.\n"
	if p.customIntro != "" {
		intro = strings.TrimSpace(p.customIntro) + "\n"
	}

	// Prepare template
	templ := p.customTemplate
	if templ == "" {
		templ = defaultSkillsTemplate
	}

	// Render the template
	tmpl, err := template.New("skillsPrompt").Parse(templ)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, SkillsTemplateData{
		Intro:  intro,
		Skills: skills,
	})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (p *SkillsPrompt) Role() string {
	return "system"
}

// SkillsFromSlice creates a simple listSkills function from a slice of skills
func SkillsFromSlice(skills []Skill) func() []Skill {
	return func() []Skill {
		return skills
	}
}
