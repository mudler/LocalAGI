package skills

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/types"

	skilldomain "github.com/mudler/skillserver/pkg/domain"
)

const defaultSkillsIntro = "You can use the following skills to help with the task.\nTo request the skill, you need to use the `request_skill` tool. The skill name is the name of the skill you want to use.\n"

// defaultSkillsTemplate is the default template that mimics the current XML behavior
const defaultSkillsTemplate = defaultSkillsIntro + `<available_skills>
{{range .Skills}}
  <skill>
    <name>{{escapeXML .Name}}</name>
    <description>{{escapeXML .Description}}</description>
  </skill>
{{end}}
</available_skills>`

// Skill is a local representation of a skill for template rendering
type Skill struct {
	Name        string
	Description string
	ID          string
}

// skillsPrompt implements agent.DynamicPrompt and injects the available skills XML block
type skillsPrompt struct {
	listSkills     func() ([]skilldomain.Skill, error)
	customTemplate string
}

// NewSkillsPrompt returns a DynamicPrompt that renders the list of available skills.
// If customTemplate is non-empty, it is used as a template with {{.Skills}} slice.
// Otherwise, the default template is used (mimics current XML behavior).
func NewSkillsPrompt(listSkills func() ([]skilldomain.Skill, error), customTemplate string) agent.DynamicPrompt {
	return &skillsPrompt{listSkills: listSkills, customTemplate: customTemplate}
}

func (p *skillsPrompt) Render(a *agent.Agent) (types.PromptResult, error) {
	skills, err := p.listSkills()
	if err != nil {
		return types.PromptResult{}, err
	}

	// Convert skilldomain.Skill to local Skill type for template rendering
	localSkills := make([]Skill, len(skills))
	for i, s := range skills {
		desc := ""
		if s.Metadata != nil && s.Metadata.Description != "" {
			desc = s.Metadata.Description
		}
		localSkills[i] = Skill{
			Name:        s.ID,
			Description: desc,
			ID:          s.ID,
		}
	}

	// Use custom template or default
	templ := p.customTemplate
	if templ == "" {
		templ = defaultSkillsTemplate
	}

	// Parse and execute the template
	tmpl, err := template.New("skillsPrompt").Funcs(template.FuncMap{
		"escapeXML": escapeXML,
	}).Funcs(sprig.FuncMap()).Parse(templ)
	if err != nil {
		return types.PromptResult{}, fmt.Errorf("failed to parse skills template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, struct {
		Skills []Skill
	}{
		Skills: localSkills,
	})
	if err != nil {
		return types.PromptResult{}, fmt.Errorf("failed to execute skills template: %w", err)
	}

	return types.PromptResult{Content: buf.String()}, nil
}

func (p *skillsPrompt) Role() string {
	return "system"
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
