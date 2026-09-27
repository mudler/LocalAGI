package agent

import (
	"context"

	"github.com/mudler/LocalAGI/core/action"
	"github.com/mudler/LocalAGI/core/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// namedAction is a no-op action that only carries a name, enough to exercise
// name-based filtering.
type namedAction struct{ name string }

func (n namedAction) Run(context.Context, *types.AgentSharedState, types.ActionParams) (types.ActionResult, error) {
	return types.ActionResult{}, nil
}

func (n namedAction) Plannable() bool { return false }

func (n namedAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{Name: types.ActionDefinitionName(n.name)}
}

func actionsNamed(names ...string) types.Actions {
	acts := make(types.Actions, 0, len(names))
	for _, n := range names {
		acts = append(acts, namedAction{name: n})
	}
	return acts
}

func namesOf(acts types.Actions) []string {
	out := make([]string, 0, len(acts))
	for _, a := range acts {
		out = append(out, a.Definition().Name.String())
	}
	return out
}

var _ = Describe("tool allow/deny-list", func() {
	all := func() types.Actions {
		return actionsNamed("search", "search_memory", "add_memory", "get_document_content",
			action.ConversationActionName, action.StopActionName, action.StateActionName)
	}

	Context("with no filter configured", func() {
		It("returns nil and leaves every tool in place", func() {
			f := newToolFilter(nil, []string{"", "  "})
			Expect(f).To(BeNil())
			Expect(namesOf(f.filterActions(all()))).To(Equal(namesOf(all())))
			Expect(f.mcpToolFilter()).To(BeNil())
		})
	})

	Context("with an allow-list", func() {
		It("keeps only the allowed tools plus the control actions", func() {
			f := newToolFilter([]string{" get_document_content ", "search"}, nil)
			Expect(namesOf(f.filterActions(all()))).To(Equal([]string{
				"search", "get_document_content",
				action.ConversationActionName, action.StopActionName, action.StateActionName,
			}))
		})
	})

	Context("with a deny-list", func() {
		It("removes the denied tools and keeps the rest", func() {
			f := newToolFilter(nil, []string{"search_memory", "add_memory"})
			Expect(namesOf(f.filterActions(all()))).To(Equal([]string{
				"search", "get_document_content",
				action.ConversationActionName, action.StopActionName, action.StateActionName,
			}))
		})

		It("cannot remove the control actions", func() {
			f := newToolFilter(nil, []string{action.ConversationActionName, action.StopActionName, action.StateActionName})
			Expect(namesOf(f.filterActions(all()))).To(ContainElements(
				action.ConversationActionName, action.StopActionName, action.StateActionName))
		})
	})

	Context("with both lists", func() {
		It("lets the deny-list win over the allow-list", func() {
			f := newToolFilter([]string{"search", "search_memory"}, []string{"search_memory"})
			Expect(namesOf(f.filterActions(all()))).To(Equal([]string{
				"search",
				action.ConversationActionName, action.StopActionName, action.StateActionName,
			}))
		})
	})

	Context("for MCP tool discovery", func() {
		It("applies the same rules to MCP tool names", func() {
			fn := newToolFilter([]string{"get_document_content"}, []string{"delete_document"}).mcpToolFilter()
			Expect(fn).NotTo(BeNil())
			Expect(fn(nil, "get_document_content")).To(BeTrue())
			Expect(fn(nil, "search_collections")).To(BeFalse())
			Expect(fn(nil, "delete_document")).To(BeFalse())
		})

		It("with only a deny-list, lets every other MCP tool through", func() {
			fn := newToolFilter(nil, []string{"delete_document"}).mcpToolFilter()
			Expect(fn(nil, "get_document_content")).To(BeTrue())
			Expect(fn(nil, "delete_document")).To(BeFalse())
		})
	})

	Context("on an agent", func() {
		It("filters availableActions and keeps per-request user tools", func() {
			a := &Agent{options: &options{
				userActions: actionsNamed("search", "search_memory", "get_document_content"),
				toolFilter:  newToolFilter([]string{"get_document_content"}, nil),
				enableHUD:   true,
			}}
			job := types.NewJob(types.WithUserTools([]types.ActionDefinition{{Name: "client_tool"}}))
			Expect(namesOf(a.getAvailableActionsForJob(job))).To(Equal([]string{
				"get_document_content", action.StateActionName, "client_tool",
			}))
		})
	})
})
