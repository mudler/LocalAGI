package skills

import (
	"fmt"
	"strings"

	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/types"

	skilldomain "github.com/mudler/skillserver/pkg/domain"
)

// skillsPrompt implements agent.DynamicPrompt and injects the available skills XML block
type skillsPrompt struct {
	listSkills func() ([]skilldomain.Skill, error)
}

// NewSkillsPrompt returns a DynamicPrompt that renders the list of available skills as XML
func NewSkillsPrompt(listSkills func() ([]skilldomain.Skill, error)) agent.DynamicPrompt {
	return &skillsPrompt{listSkills: listSkills}
}

func (p *skillsPrompt) Render(a *agent.Agent) (types.PromptResult, error) {
	skills, err := p.listSkills()
	if err != nil {
		return types.PromptResult{}, err
	}
	var sb strings.Builder
	sb.WriteString("You can use the following skills to help with the task.\n")
	sb.WriteString("To request the skill, you need to use the `request_skill` tool. The skill name is the name of the skill you want to use.\n")
	sb.WriteString("<available_skills>\n")
	for _, s := range skills {
		name := s.ID
		desc := ""
		if s.Metadata != nil && s.Metadata.Description != "" {
			desc = s.Metadata.Description
		}
		sb.WriteString("  <skill>\n")
		sb.WriteString(fmt.Sprintf("    <name>%s</name>\n", escapeXML(name)))
		sb.WriteString(fmt.Sprintf("    <description>%s</description>\n", escapeXML(desc)))
		sb.WriteString("  </skill>\n")
	}
	sb.WriteString("</available_skills>")
	return types.PromptResult{Content: sb.String()}, nil
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
