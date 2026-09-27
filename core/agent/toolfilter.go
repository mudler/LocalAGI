package agent

import (
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/mudler/LocalAGI/core/action"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/cogito"
)

// controlActionNames are the built-in actions that drive the agent loop itself.
// An allow-list written with only domain tools in mind would otherwise remove
// them and leave the agent unable to reply to a scheduled run, stop, or update
// its HUD state. Whether they are offered at all is already decided by
// initiate_conversations, can_stop_itself and hud, so the tool filter leaves
// them alone in both directions.
var controlActionNames = map[string]struct{}{
	action.ConversationActionName: {},
	action.StopActionName:         {},
	action.StateActionName:        {},
}

// isControlAction reports whether name is one of the agent's control actions.
func isControlAction(name string) bool {
	_, ok := controlActionNames[name]
	return ok
}

// toolFilter is a per-agent allow/deny-list over tool names. A nil *toolFilter
// allows everything, so callers never need to check whether one is configured.
type toolFilter struct {
	allow map[string]struct{}
	deny  map[string]struct{}
}

// newToolFilter builds a filter from tool names. Names are trimmed and empty
// entries dropped, so a form that submits "" or " a, b " behaves as expected.
// It returns nil when neither list names a tool.
func newToolFilter(allow, deny []string) *toolFilter {
	f := &toolFilter{allow: toNameSet(allow), deny: toNameSet(deny)}
	if len(f.allow) == 0 && len(f.deny) == 0 {
		return nil
	}
	return f
}

func toNameSet(names []string) map[string]struct{} {
	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		if n = strings.TrimSpace(n); n != "" {
			set[n] = struct{}{}
		}
	}
	return set
}

// allows reports whether a tool with this name may be offered to the model.
func (f *toolFilter) allows(name string) bool {
	if f == nil || isControlAction(name) {
		return true
	}
	if _, denied := f.deny[name]; denied {
		return false
	}
	if len(f.allow) == 0 {
		return true
	}
	_, allowed := f.allow[name]
	return allowed
}

// filterActions returns the actions the filter allows, preserving order. With
// no filter configured the input is returned as is.
func (f *toolFilter) filterActions(acts types.Actions) types.Actions {
	if f == nil {
		return acts
	}
	out := make(types.Actions, 0, len(acts))
	for _, act := range acts {
		if f.allows(act.Definition().Name.String()) {
			out = append(out, act)
		}
	}
	return out
}

// mcpToolFilter adapts the filter to cogito's MCP tool discovery. cogito lists
// tools straight from the live MCP sessions, so filtering the agent's own copy
// of the MCP actions is not enough to hide a tool from the model. Returns nil
// (no filtering) when no filter is configured.
func (f *toolFilter) mcpToolFilter() cogito.MCPToolFilter {
	if f == nil {
		return nil
	}
	return func(_ *mcp.ClientSession, toolName string) bool {
		return f.allows(toolName)
	}
}
