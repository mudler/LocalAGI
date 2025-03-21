package action

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai/jsonschema"
)

const StateActionName = "update_state"

func NewState() *StateAction {
	return &StateAction{}
}

type StateAction struct{}

// State is the structure
// that is used to keep track of the current state
// and the Agent's short memory that it can update
// Besides a long term memory that is accessible by the agent (With vector database),
// And a context memory (that is always powered by a vector database),
// this memory is the shorter one that the LLM keeps across conversation and across its
// reasoning process's and life time.
// TODO: A special action is then used to let the LLM itself update its memory
// periodically during self-processing, and the same action is ALSO exposed
// during the conversation to let the user put for example, a new goal to the agent.
type AgentInternalState struct {
	NowDoing    string   `json:"doing_now"`
	DoingNext   string   `json:"doing_next"`
	DoneHistory []string `json:"done_history"`
	Memories    []string `json:"memories"`
	Goal        string   `json:"goal"`
}

func (a *StateAction) Run(context.Context, ActionParams) (ActionResult, error) {
	return ActionResult{Result: "internal state has been updated"}, nil
}

func (a *StateAction) Plannable() bool {
	return false
}

func (a *StateAction) Definition() ActionDefinition {
	return ActionDefinition{
		Name:        StateActionName,
		Description: "update the agent state (short memory) with the current state of the conversation.",
		Properties: map[string]jsonschema.Definition{
			"goal": {
				Type:        jsonschema.String,
				Description: "The current goal of the agent.",
			},
			"doing_next": {
				Type:        jsonschema.String,
				Description: "The next action the agent will do.",
			},
			"done_history": {
				Type: jsonschema.Array,
				Items: &jsonschema.Definition{
					Type: jsonschema.String,
				},
				Description: "A list of actions that the agent has done.",
			},
			"now_doing": {
				Type:        jsonschema.String,
				Description: "The current action the agent is doing.",
			},
			"memories": {
				Type: jsonschema.Array,
				Items: &jsonschema.Definition{
					Type: jsonschema.String,
				},
				Description: "A list of memories to keep between conversations.",
			},
		},
	}
}

const fmtT = `=====================
NowDoing: %s
DoingNext: %s
Your current goal is: %s
You have done: %+v
You have a short memory with: %+v
=====================
`

func (c AgentInternalState) String() string {
	return fmt.Sprintf(
		fmtT,
		c.NowDoing,
		c.DoingNext,
		c.Goal,
		c.DoneHistory,
		c.Memories,
	)
}
