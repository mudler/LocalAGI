---
name: pr-retrospective
description: Use after any PR merges or fails, and again as a pre-flight before opening the next PR that touches a risky surface (frontend entry point, build tooling, deploy, dependencies). Turns every past PR into a forward check so the hive stops paying for the same failure class twice — the recursive-learning loop made concrete at the PR boundary. Reads and appends to PR_LESSONS.md; run its accumulated pre-flight before shipping. Not for trivial doc-only PRs (nothing to learn, pre-flight is a no-op) — use it when a PR touched real code or its outcome revealed a failure class worth catching next time.
---

# pr-retrospective — every PR teaches the next one

The founder's charge: the hive must learn from **every previous PR** — what to build, and
what goes wrong along the way — so it grows more able to take a PR autonomously each time.
This skill is that loop, made standing. It is the engineering-layer instance of `soul.md`'s
Recursive Cycle (Capture → Evaluate → Prune → Feed → Dissect → Return → Propagate): a PR
retrospective is one full turn of that cycle over a unit of shipped work.

`FABLE_DNA.md` Chromosome VIII is the law; this skill is how it runs; `PR_LESSONS.md` is the
growing record it reads from and writes to.

## Two moments, one loop

```
AFTER a PR (merged or failed) — CAPTURE
   → What did it intend to do?
   → What went wrong, or nearly did? (a real break, a near-miss, a thing a human caught)
   → What is the FORWARD CHECK that would catch this whole class earlier next time?
   → Append it to PR_LESSONS.md as a new L-## entry (or sharpen an existing one).

BEFORE the next PR (that touches a risky surface) — PRE-FLIGHT
   → Run the accumulated forward checks in PR_LESSONS.md that apply to this PR's surface.
   → Only ship once they pass. A failure class already in the file must never recur silently.
```

## The pre-flight (current, from PR_LESSONS.md — grows over time)

Run the ones that apply to what the PR touches:

1. **Frontend / entry change** → trace `main.tsx → App.tsx → the mounted tree`; the entry
   actually renders what it's supposed to, not just that files exist (L-01).
2. **Any long-running work** → `git fetch` + read divergence **before build and again before
   push**; reconcile by merge; never force-push over unseen history (L-02).
3. **Styling / build tooling** → the build *produced* output, not merely exited 0 — assert a
   CSS-size floor or that a known utility class survives into the bundle (L-03).
4. **Deploy** → the served asset hash equals the committed hash; if not, the deploy is
   lagging — report it honestly, don't claim it's live (L-04).
5. **Deploy typecheck** → scoped to the shipped subgraph; `vite build` alone is green
   independent of any not-yet-wired WIP tree; the default tsconfig is left untouched (L-05).
6. **Dependencies** → every declared dep resolves to a real installable package; every used
   import is declared (L-06).
7. **"It's already deployed / live"** → probe it and report from evidence (L-07).

## How to write a good lesson (so it actually prevents recurrence)

A lesson is only worth recording if it produces a **forward check** — a concrete,
runnable-or-checkable test, not a moral. "Be more careful with merges" is not a lesson;
"`git fetch` + diff divergence before push" is. Each entry: what happened (cite the real PR
number), the root cause in one sentence, how it was caught, and the forward check. Keep it
uncensored — a near-miss that a human caught is exactly the kind of thing that will become a
silent failure once no human is watching, so record it in full rather than tidying it away.

## Relationship to the other skills

- **`fable-debugger`** diagnoses a single failure (root cause, proven-path contrast). This
  skill generalizes that one diagnosis into a check that guards against the *class*.
- **`merge-readiness`** runs the verify → PR → CI-autofix → merge loop. This skill's
  pre-flight runs *before* that loop, so the loop starts from safer ground.
- **`session-harvest`** captures a whole session's work at its close. This skill captures a
  single PR's lesson at its boundary — finer-grained, and it feeds the same memory.
- **`nine-miss-truths`** widens the hypothesis space before naming a root cause; use it when
  a PR broke in a way whose cause isn't obvious, then record the survivor here.

*Origin: Fable (Harness), 2026-07-16, from the founder's direction that the hive learn from
every previous PR — so the pre-flight checklist grows every time something breaks, and no
failure class is paid for twice.*
