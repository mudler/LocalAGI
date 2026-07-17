---
name: merge-readiness
description: Use when a change is ready for a PR and you want it to land green without babysitting each CI failure by hand — the formalized verify-PR-autofix-merge loop this hive already practices. Also use when watching an open PR for CI failures that need re-diagnosis and a re-push. Not a license to self-merge beyond whatever scope the founder has already authorized — the founder-only gate on irreversible/shared-truth actions is a hard boundary this skill never expands.
---

# merge-readiness — the loop, formalized (with the boundary stated plainly)

The founder's research proposed a "Merge-Back Protocol": automated PR review, CI repair,
base-branch sync, merge loops. Read completely literally, "automated... merge loops" could
mean self-merging without any ceiling — that would conflict with the hive's own standing
rule that irreversible or shared-truth actions wait for the founder. This skill keeps
everything genuinely useful about the idea and states the boundary explicitly so it can't
drift.

## The loop

```
1. VERIFY LOCALLY   → the real check for the surface touched: node --check, tsc --noEmit,
                       pytest, a build — whatever proves the change works before it ships.
2. OPEN THE PR       → with a Rationale/Verification line (F-004) — what changed, why, how
                       it was proven.
3. WATCH CI          → if it's green, proceed to step 5.
4. IF CI FAILS       → re-diagnose (fable-debugger: root cause, not the first guess) →
                       fix → re-push → back to step 3. Repeat until green or genuinely
                       stuck (then escalate with the exact failure, don't loop forever).
5. MERGE             → only within whatever authorization the founder has already granted
                       for this repo/branch/session. If that authorization doesn't cover
                       this merge, open the PR and stop — hand it to the founder.
6. SWEEP             → after merging, check the base branch is still healthy (no broken
                       CI introduced), same as any post-merge regression check.
```

## The boundary (this is the point of writing this skill down at all)

- **"Ship before breakfast" applies to reversible, already-in-scope work** — building a
  skill, fixing a bug, opening a PR. It does not mean "no permission is ever needed."
- **Self-merging is only ever done within authorization the founder has explicitly given**
  — for a specific repo, a specific pattern of work, a specific session. This skill does not
  grant new merge authority; it only formalizes *how* to use whatever authority already
  exists, cleanly and with a real CI-repair loop instead of a single fragile attempt.
- **Irreversible or destructive actions** (force-push, `git reset --hard`, deleting
  branches/history, anything affecting shared infrastructure beyond what's already
  authorized) are never in scope for automatic repetition in this loop — they stop and ask,
  every time, regardless of how many times this loop has run cleanly before.

## Why formalize something already being done

This session has run this exact loop informally, dozens of times, across THEHIVE and six
colonies — the value in writing it down is that a *future* session (or a founder deciding
whether to grant merge authorization at all) can see the actual shape of the loop and its
boundary in one place, rather than re-deriving it from scattered examples.

Origin: Fable (Harness), 2026-07-13, from the founder's Merge-Back Protocol research —
adopted, with the founder-authorization boundary made explicit rather than implied.
