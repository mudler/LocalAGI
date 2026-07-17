---
name: pocket-dimensions
description: Use when a task should be tried in an isolated, disposable workspace before it touches shared truth — parallel exploration, a risky refactor, an agent working unsupervised, or any "try it and see" change. Formalizes the create → work → review → merge/discard lifecycle (the pattern behind BranchFS, agent-cow, Sandcastle, dmux) using tools the hive already has natively — git worktrees and GitHub PRs — with zero new dependencies. This is the honest, buildable half of the founder's "Pocket Dimensions" research: real isolation without importing an unvetted third party.
---

# Pocket Dimensions — isolated work, honestly built

The founder's research surveyed real 2026 tools that give AI agents disposable, isolated
workspaces: `multikernel/branchfs` (FUSE copy-on-write filesystem), `trail-ml/agent-cow`
(database copy-on-write), `mattpocock/sandcastle` and `standardagents/dmux` (sandboxed /
worktree agent orchestration). All confirmed to exist by direct search (2026-07-13) — see
`SOURCED_SKILLS_INDEX.md` Wave 3 for the full verdict on each.

None of them are dependencies of this skill. **The hive already runs this exact lifecycle**
every time a change ships in this project — branch, isolated commit, PR, gate, merge, delete
— using git and GitHub, which cost nothing and need no new package to vet. This skill makes
that lifecycle explicit and repeatable so it is used *deliberately* for isolation, not just
incidentally as "how commits happen."

## The lifecycle

```
CREATE   → git checkout -B <branch> origin/<base>   (a fresh, disposable dimension)
WORK     → make the change; the working tree elsewhere is untouched
REVIEW   → node --check / tsc --noEmit / tests / a probe against the real thing —
           machine-verified, not asserted (fable-debugger discipline)
GATE     → constitutional check (F-001…F-006) — does this change hold up?
MERGE    → PR → merge — the dimension's work joins shared truth atomically
DESTROY  → delete the branch — the dimension leaves no trace once merged
```

This is BranchFS's "instant branch → work → atomic commit → zero-cost abort," agent-cow's
"isolated writes → review → selectively commit," and Sandcastle/dmux's "isolated agent →
merge cleanly" — the same shape, implemented with `git worktree` + GitHub instead of a FUSE
layer or a database CoW table.

## When to reach for a real worktree, not just a branch

A branch alone shares your working directory — fine for sequential work, not enough when
you need true parallelism (two agents editing at once) or want a crash/bad-edit in one
attempt to be unable to touch another. For that, use an actual second working tree:

```bash
git worktree add ../hive-dimension-<name> -b <branch> origin/main
# work inside ../hive-dimension-<name> — a fully separate directory, same .git object store
# when done: PR + merge from there, then:
git worktree remove ../hive-dimension-<name>
```

In this environment, prefer the `EnterWorktree`/`ExitWorktree` tools where available — they
wrap exactly this pattern (isolated working copy, auto-cleaned if nothing changed).

## The merge-back gate (what makes it safe, not just isolated)

Isolation alone doesn't make a change good — it makes a *bad* change cheap to discard. What
actually protects shared truth is the gate before merge:

1. **Machine-verified, not asserted** — `node --check`, `tsc --noEmit`, the test suite, or a
   production probe actually ran and passed. See `fable-debugger`.
2. **Constitutional** — F-004 (a Rationale/Verification line), F-001 (no sovereign data
   leaves without consent), F-005 (fixed law beats a convenient shortcut).
3. **Founder-gated for the irreversible** — merges to shared truth, secrets, and anything
   that can't be undone still wait for the founder's hands, same as always.

Only after all three does DESTROY happen — the dimension is deleted, having left behind
only what it proved.

## Why not import BranchFS / agent-cow / Sandcastle / dmux today

Each does something this git+PR lifecycle doesn't yet: BranchFS gives filesystem-level CoW
without git's history overhead; agent-cow gives database-level isolation (useful once the
hive has a production DB agents write to speculatively, which D1 currently isn't used for);
Sandcastle/dmux orchestrate *many* agents in parallel containers automatically. Those are
real future upgrades, catalogued for the founder to greenlight individually — not
silently added as dependencies in this pass. Adopting any of them means: clone it, read the
source, check the license, run it against a real hive task, *then* decide — the same
`skill-harvester` discipline as any other external import.

Origin: Fable (Harness), 2026-07-13, from the founder's Pocket Dimensions research —
verified against real repositories, then built using only what the hive already has.
