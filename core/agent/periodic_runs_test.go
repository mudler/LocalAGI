package agent

import (
	"testing"
	"time"
)

// WithPeriodicRuns falls back to 10m when the duration does not parse. The fallback was
// assigned and then immediately overwritten by the zero value from the failed parse, so a
// misconfigured periodic_runs silently became 0 instead of 10m -- and an agent with a zero
// interval does not run periodically at all. Silent, because nothing logs and nothing fails.
func TestWithPeriodicRunsFallsBackOnUnparsableDuration(t *testing.T) {
	for _, bad := range []string{"", "10 minutes", "banana", "10x"} {
		t.Run(bad, func(t *testing.T) {
			var o options
			if err := WithPeriodicRuns(bad)(&o); err != nil {
				t.Fatalf("option returned %v", err)
			}
			if o.periodicRuns != 10*time.Minute {
				t.Fatalf("unparsable %q must fall back to 10m, got %v", bad, o.periodicRuns)
			}
		})
	}
}

func TestWithPeriodicRunsKeepsAValidDuration(t *testing.T) {
	var o options
	if err := WithPeriodicRuns("45s")(&o); err != nil {
		t.Fatalf("option returned %v", err)
	}
	if o.periodicRuns != 45*time.Second {
		t.Fatalf("valid duration must be kept, got %v", o.periodicRuns)
	}
}
