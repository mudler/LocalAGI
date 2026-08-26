package cmd

import (
	"testing"
	"time"
)

func TestRetentionDefaults(t *testing.T) {
	t.Setenv("LOCALAGI_CONVERSATIONS_MAX_AGE", "")
	t.Setenv("LOCALAGI_CONVERSATIONS_MAX_PER_AGENT", "")
	t.Setenv("LOCALAGI_CONVERSATIONS_PRUNE_INTERVAL", "")
	t.Setenv("LOCALAGI_SCHEDULER_MAX_RUNS_PER_TASK", "")
	t.Setenv("LOCALAGI_SCHEDULER_MAX_RUN_AGE", "")

	env := LoadEnv()

	if env.ConversationsMaxAge != 720*time.Hour {
		t.Errorf("ConversationsMaxAge = %v, want 720h", env.ConversationsMaxAge)
	}
	if env.ConversationsMaxPerAgent != 200 {
		t.Errorf("ConversationsMaxPerAgent = %d, want 200", env.ConversationsMaxPerAgent)
	}
	if env.ConversationsPruneInterval != time.Hour {
		t.Errorf("ConversationsPruneInterval = %v, want 1h", env.ConversationsPruneInterval)
	}
	if env.SchedulerMaxRunsPerTask != 20 {
		t.Errorf("SchedulerMaxRunsPerTask = %d, want 20", env.SchedulerMaxRunsPerTask)
	}
	if env.SchedulerMaxRunAge != 720*time.Hour {
		t.Errorf("SchedulerMaxRunAge = %v, want 720h", env.SchedulerMaxRunAge)
	}
	if !env.SchedulerDedupeTasks {
		t.Error("SchedulerDedupeTasks = false, want true")
	}
	if env.SchedulerMaxTasksPerAgent != 100 {
		t.Errorf("SchedulerMaxTasksPerAgent = %d, want 100", env.SchedulerMaxTasksPerAgent)
	}
}

func TestCreationPolicyOverrides(t *testing.T) {
	t.Setenv("LOCALAGI_SCHEDULER_DEDUPE_TASKS", "false")
	t.Setenv("LOCALAGI_SCHEDULER_MAX_TASKS_PER_AGENT", "5")

	env := LoadEnv()

	if env.SchedulerDedupeTasks {
		t.Error("SchedulerDedupeTasks = true, want false")
	}
	if env.SchedulerMaxTasksPerAgent != 5 {
		t.Errorf("SchedulerMaxTasksPerAgent = %d, want 5", env.SchedulerMaxTasksPerAgent)
	}
}

// Dedupe is a safety default; a typo must not switch it off.
func TestCreationPolicyUnparseableBoolKeepsDedupeOn(t *testing.T) {
	t.Setenv("LOCALAGI_SCHEDULER_DEDUPE_TASKS", "yes-please")

	env := LoadEnv()

	if !env.SchedulerDedupeTasks {
		t.Error("SchedulerDedupeTasks = false, want the true default")
	}
}

func TestRetentionOverrides(t *testing.T) {
	t.Setenv("LOCALAGI_CONVERSATIONS_MAX_AGE", "48h")
	t.Setenv("LOCALAGI_CONVERSATIONS_MAX_PER_AGENT", "10")
	t.Setenv("LOCALAGI_CONVERSATIONS_PRUNE_INTERVAL", "5m")
	t.Setenv("LOCALAGI_SCHEDULER_MAX_RUNS_PER_TASK", "3")
	t.Setenv("LOCALAGI_SCHEDULER_MAX_RUN_AGE", "24h")

	env := LoadEnv()

	if env.ConversationsMaxAge != 48*time.Hour {
		t.Errorf("ConversationsMaxAge = %v, want 48h", env.ConversationsMaxAge)
	}
	if env.ConversationsMaxPerAgent != 10 {
		t.Errorf("ConversationsMaxPerAgent = %d, want 10", env.ConversationsMaxPerAgent)
	}
	if env.ConversationsPruneInterval != 5*time.Minute {
		t.Errorf("ConversationsPruneInterval = %v, want 5m", env.ConversationsPruneInterval)
	}
	if env.SchedulerMaxRunsPerTask != 3 {
		t.Errorf("SchedulerMaxRunsPerTask = %d, want 3", env.SchedulerMaxRunsPerTask)
	}
	if env.SchedulerMaxRunAge != 24*time.Hour {
		t.Errorf("SchedulerMaxRunAge = %v, want 24h", env.SchedulerMaxRunAge)
	}
}

// "0" is how an operator turns a single limit off, and it has to survive the
// fallback-to-default path that an empty value takes.
func TestLimitsCarriesCreationPolicy(t *testing.T) {
	t.Setenv("LOCALAGI_SCHEDULER_DEDUPE_TASKS", "true")
	t.Setenv("LOCALAGI_SCHEDULER_MAX_TASKS_PER_AGENT", "42")

	limits := LoadEnv().Limits()

	if !limits.SchedulerCreation.Dedupe {
		t.Error("Limits().SchedulerCreation.Dedupe = false, want true")
	}
	if limits.SchedulerCreation.MaxTasksPerAgent != 42 {
		t.Errorf("Limits().SchedulerCreation.MaxTasksPerAgent = %d, want 42", limits.SchedulerCreation.MaxTasksPerAgent)
	}
}

func TestRetentionZeroDisablesIndividualLimits(t *testing.T) {
	t.Setenv("LOCALAGI_CONVERSATIONS_MAX_AGE", "0")
	t.Setenv("LOCALAGI_CONVERSATIONS_MAX_PER_AGENT", "0")
	t.Setenv("LOCALAGI_SCHEDULER_MAX_RUNS_PER_TASK", "0")
	t.Setenv("LOCALAGI_SCHEDULER_MAX_RUN_AGE", "0")

	env := LoadEnv()

	if env.ConversationsMaxAge != 0 {
		t.Errorf("ConversationsMaxAge = %v, want 0", env.ConversationsMaxAge)
	}
	if env.ConversationsMaxPerAgent != 0 {
		t.Errorf("ConversationsMaxPerAgent = %d, want 0", env.ConversationsMaxPerAgent)
	}
	if env.SchedulerMaxRunsPerTask != 0 {
		t.Errorf("SchedulerMaxRunsPerTask = %d, want 0", env.SchedulerMaxRunsPerTask)
	}
	if env.SchedulerMaxRunAge != 0 {
		t.Errorf("SchedulerMaxRunAge = %v, want 0", env.SchedulerMaxRunAge)
	}
}

// A typo should not silently disable retention on a busy deployment.
func TestRetentionUnparseableValueFallsBackToDefault(t *testing.T) {
	t.Setenv("LOCALAGI_CONVERSATIONS_MAX_AGE", "not-a-duration")
	t.Setenv("LOCALAGI_CONVERSATIONS_MAX_PER_AGENT", "many")

	env := LoadEnv()

	if env.ConversationsMaxAge != 720*time.Hour {
		t.Errorf("ConversationsMaxAge = %v, want the 720h default", env.ConversationsMaxAge)
	}
	if env.ConversationsMaxPerAgent != 200 {
		t.Errorf("ConversationsMaxPerAgent = %d, want the 200 default", env.ConversationsMaxPerAgent)
	}
}

// Days are the natural unit for a retention window and time.ParseDuration
// rejects them, so "30d" has to work.
func TestRetentionAcceptsDayDurations(t *testing.T) {
	t.Setenv("LOCALAGI_CONVERSATIONS_MAX_AGE", "30d")
	t.Setenv("LOCALAGI_SCHEDULER_MAX_RUN_AGE", "7d")

	env := LoadEnv()

	if env.ConversationsMaxAge != 30*24*time.Hour {
		t.Errorf("ConversationsMaxAge = %v, want 720h", env.ConversationsMaxAge)
	}
	if env.SchedulerMaxRunAge != 7*24*time.Hour {
		t.Errorf("SchedulerMaxRunAge = %v, want 168h", env.SchedulerMaxRunAge)
	}
}
