package scheduler

import (
	"fmt"
	"strings"
	"unicode"
)

// Action names the scheduler quotes back to an agent that has run out of room.
// They mirror the reminder action names the agent already has.
const (
	ListTasksActionName  = "list_tasks"
	RemoveTaskActionName = "remove_task"
)

// CreationPolicy bounds what an agent may schedule for itself.
//
// An agent with a task-creating tool can re-create the same reminder on every
// run. One deployment reached 6,399 tasks, 74% of them byte-identical copies.
type CreationPolicy struct {
	// Dedupe returns the existing task instead of creating a second copy of it.
	Dedupe bool
	// MaxTasksPerAgent refuses new tasks past this count. Zero means no limit.
	MaxTasksPerAgent int
}

// NormalizePrompt reduces a prompt to the form used for duplicate comparison:
// lowercased, with runs of whitespace collapsed and punctuation dropped.
//
// This catches capitalisation and punctuation drift. It does not catch a
// reworded prompt, which is a deliberate limit — merging prompts that differ in
// their actual words risks dropping a task somebody meant to schedule, and the
// per-agent ceiling is what bounds that case.
func NormalizePrompt(prompt string) string {
	var b strings.Builder
	b.Grow(len(prompt))

	for _, r := range strings.ToLower(prompt) {
		switch {
		case unicode.IsSpace(r):
			b.WriteRune(' ')
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			// dropped
		default:
			b.WriteRune(r)
		}
	}

	return strings.Join(strings.Fields(b.String()), " ")
}

// sameTask reports whether two tasks are the same scheduled intent.
func sameTask(a, b *Task) bool {
	return a.AgentName == b.AgentName &&
		a.ScheduleType == b.ScheduleType &&
		a.ScheduleValue == b.ScheduleValue &&
		NormalizePrompt(a.Prompt) == NormalizePrompt(b.Prompt)
}

// findDuplicate returns the agent's existing task matching this one, if any.
func (s *Scheduler) findDuplicate(task *Task) (*Task, error) {
	existing, err := s.store.GetByAgent(task.AgentName)
	if err != nil {
		return nil, err
	}

	for _, candidate := range existing {
		if sameTask(candidate, task) {
			return candidate, nil
		}
	}

	return nil, nil
}

// checkCapacity reports whether the agent has room for another task.
func (s *Scheduler) checkCapacity(agentName string) error {
	if s.policy.MaxTasksPerAgent <= 0 {
		return nil
	}

	existing, err := s.store.GetByAgent(agentName)
	if err != nil {
		return err
	}

	if len(existing) < s.policy.MaxTasksPerAgent {
		return nil
	}

	return fmt.Errorf(
		"agent has %d scheduled tasks (limit %d); review them with %s and remove the ones you no longer need with %s",
		len(existing), s.policy.MaxTasksPerAgent, ListTasksActionName, RemoveTaskActionName,
	)
}
