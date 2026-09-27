package conversations

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/mudler/xlog"
)

// conversationTimeLayout mirrors the timestamp format saveConversation uses
// when naming dumps: "<agent>-2006-01-02-15-04-05.json".
const conversationTimeLayout = "2006-01-02-15-04-05"

// conversationFile matches a saved conversation dump, capturing the agent key
// and the timestamp. The agent group is greedy so that a "decision-" prefixed
// dump buckets separately from the agent's own conversations: those decision
// rounds outnumber real conversations by an order of magnitude, and a shared
// budget would let them evict the history worth keeping.
var conversationFile = regexp.MustCompile(`^(.+)-(\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-\d{2})\.json$`)

// RetentionPolicy bounds how much conversation history survives a sweep.
// A zero value for either field disables that limit.
type RetentionPolicy struct {
	// MaxAge removes dumps older than this.
	MaxAge time.Duration
	// MaxPerAgent keeps only this many of the newest dumps per agent.
	MaxPerAgent int
}

// Enabled reports whether the policy would remove anything at all.
func (p RetentionPolicy) Enabled() bool {
	return p.MaxAge > 0 || p.MaxPerAgent > 0
}

type conversationEntry struct {
	name string
	at   time.Time
}

// Prune applies the retention policy to a conversation directory and returns
// how many files it removed.
//
// The timestamp comes from the filename rather than the inode, so a sweep is a
// single ReadDir with no stat per file — the directory routinely holds tens of
// thousands of entries. Files that do not match the naming scheme are never
// touched.
func Prune(dir string, p RetentionPolicy, now time.Time) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}

	if !p.Enabled() {
		return 0, nil
	}

	byAgent := map[string][]conversationEntry{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := conversationFile.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		at, err := time.ParseInLocation(conversationTimeLayout, m[2], time.Local)
		if err != nil {
			continue
		}
		byAgent[m[1]] = append(byAgent[m[1]], conversationEntry{name: e.Name(), at: at})
	}

	doomed := map[string]struct{}{}
	for _, dumps := range byAgent {
		// Newest first, with the name as a tiebreaker so a sweep is deterministic
		// when two dumps share a one-second timestamp.
		sort.Slice(dumps, func(i, j int) bool {
			if dumps[i].at.Equal(dumps[j].at) {
				return dumps[i].name > dumps[j].name
			}
			return dumps[i].at.After(dumps[j].at)
		})

		for i, d := range dumps {
			if p.MaxAge > 0 && now.Sub(d.at) > p.MaxAge {
				doomed[d.name] = struct{}{}
			}
			if p.MaxPerAgent > 0 && i >= p.MaxPerAgent {
				doomed[d.name] = struct{}{}
			}
		}
	}

	removed := 0
	for name := range doomed {
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			// A dump that vanished under us is not a failure worth aborting the
			// sweep for; the next tick picks up whatever is left.
			xlog.Warn("Failed to prune conversation", "file", name, "error", err)
			continue
		}
		removed++
	}

	return removed, nil
}
