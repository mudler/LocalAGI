package types

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mudler/LocalAGI/core/conversations"
)

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

const (
	DefaultLastMessageDuration = 5 * time.Minute
)

type ReminderActionResponse struct {
	Message     string    `json:"message"`
	CronExpr    string    `json:"cron_expr"`    // Cron expression for scheduling
	LastRun     time.Time `json:"last_run"`     // Last time this reminder was triggered
	NextRun     time.Time `json:"next_run"`     // Next scheduled run time
	IsRecurring bool      `json:"is_recurring"` // Whether this is a recurring reminder
}

type AgentSharedState struct {
	ConversationTracker *conversations.ConversationTracker[string] `json:"conversation_tracker"`
	Reminders           []ReminderActionResponse                   `json:"reminders"`
	UserID              uuid.UUID                                  `json:"user_id"`
	AgentID             uuid.UUID                                  `json:"agent_id"`
}

func NewAgentSharedState(lastMessageDuration time.Duration) *AgentSharedState {
	if lastMessageDuration == 0 {
		lastMessageDuration = DefaultLastMessageDuration
	}
	return &AgentSharedState{
		ConversationTracker: conversations.NewConversationTracker[string](lastMessageDuration),
		Reminders:           make([]ReminderActionResponse, 0),
	}
}

func NewAgentSharedStateWithIDs(lastMessageDuration time.Duration, userID, agentID uuid.UUID) *AgentSharedState {
	if lastMessageDuration == 0 {
		lastMessageDuration = DefaultLastMessageDuration
	}
	return &AgentSharedState{
		ConversationTracker: conversations.NewConversationTracker[string](lastMessageDuration),
		Reminders:           make([]ReminderActionResponse, 0),
		UserID:              userID,
		AgentID:             agentID,
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
