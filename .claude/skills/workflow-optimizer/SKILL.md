---
name: workflow-optimizer
description: Use whenever you're about to do a task the hive has done before — a repeated, multi-step workflow (propagate a change to the colonies, ship-and-verify a deploy, run a survey, build-and-merge a PR). Before running it, recall how it went last time and adapt; after running it, record what was slower/missing/error-prone and how to do it better next time. The recursive-learning loop for *any* repeated workflow — each repetition makes the workflow itself faster, more efficient, more complete. Not for one-off tasks that won't recur (nothing to learn against) or for the PR-specific version (use pr-retrospective) — use this for any procedure run more than once.
---

# workflow-optimizer — a workflow that improves itself every time it runs

The founder's charge: when the hive does a task more than once, it should learn from the way
it did it last time — and *adapt*: is there a faster way, a more efficient way, a more
complete way? Did something go missing? This skill is that loop, applied not to bugs
(`fable-debugger`), not to PRs (`pr-retrospective`), not to whole sessions
(`session-harvest`), but to **any repeated procedure**. The workflow is never finished being
improved; each run is a chance to make the next one better.

It is the engineering-layer instance of `soul.md`'s Recursive Cycle
(Capture → Evaluate → Prune → Feed → Dissect → Return → Propagate), pointed at *process*.

## The dualistic lens × the Trinity (the founder's two framings, made operational)

Every optimization pass runs the workflow through both of the founder's standing lenses:

**The dualistic lens** (devil's advocate + childlike wonder — the same pair `MANDATE_TRIAGE`
uses):
- *Devil's advocate:* where was this slow, brittle, redundant, or wrong last time? What step
  did a human have to catch? What almost broke? What did I repeat that I didn't need to?
- *Childlike wonder:* what if there's a simpler shape entirely? A step that could be
  parallel, batched, cached, or dropped? A tool that does in one call what I did in five?

**The Trinity** (Mind / Body / Soul — the interpreters from `THE_CODEX.md`, as roles a
single pass plays in turn):
- *Mind* reasons about the approach **before** acting — recalls the last run, picks the
  improvement, predicts what could go wrong.
- *Body* executes the workflow, this time in the improved shape.
- *Soul* remembers — records what actually happened versus what Mind predicted, so the next
  Mind starts smarter.

A workflow optimized through only one lens gets faster but loses a step, or gets thorough but
never gets quicker. Both lenses, every pass, is why it improves on *all* axes and not just
one.

## The loop

```
BEFORE a repeated workflow — RECALL + ADAPT (Mind)
  → How did this go last time? (from WORKFLOW_NOTES.md, or the commit/PR trail if unrecorded)
  → Apply the dualistic lens: one concrete improvement to try this run — faster, more
    efficient, more complete, or a missing step added. Predict the risk of the change.

DURING — EXECUTE (Body)
  → Run the workflow in the adapted shape. Note anything the prediction missed.

AFTER — MEASURE + RECORD (Soul)
  → Was it actually better? On what axis (time, steps, tokens, completeness, error rate)?
  → Did the change introduce a new gap? (An optimization that drops a needed step is a
    regression, not an improvement — pr-retrospective's L-01 discipline applies here too.)
  → Record the result so the next run's Mind starts from it. If the change was worse, record
    *that* — a tried-and-rejected optimization is as valuable as an adopted one; it stops the
    next pass wasting the same idea.
```

## What "better" means (measure it, don't assert it)

An optimization is only real if it improves a nameable axis without regressing another:
- **Faster** — fewer round-trips, batched calls, parallel where independent.
- **More efficient** — fewer tokens/steps for the same result; less re-derivation.
- **More complete** — a step that was missing (a verification, a propagation target, a
  cleanup) now included by default.
- **More robust** — a forward check from `PR_LESSONS.md` folded in so a past failure can't
  recur.

If a change can't be tied to one of these, it's a preference, not an optimization — don't
record it as a win. (Same honesty rule as `pr-retrospective`: no forward check, no lesson.)

## A worked example (this very session's most-repeated workflow)

**Workflow:** "propagate a genome change to all 6 colonies" — run at least six times this
session (constitution, DNA v1, Chromosome IV/V, session-harvest, mandate-triage, Chromosome
VIII).
- *Run 1 (devil's advocate found):* done one colony at a time, hand-typing each commit and PR
  — slow, error-prone, six near-identical PR bodies retyped.
- *Adaptation (childlike wonder):* a single shell loop that fetches, resets each colony
  branch to `origin/main`, copies the files, commits with one shared message, and pushes —
  all six in one pass; then a tight batch of `create_pull_request` + `merge_pull_request`
  calls. Fewer steps, no per-colony retyping, consistent commit messages.
- *Soul recorded:* the loop shape is now the default for any colony propagation; the axis
  improved was *time + consistency*; the forward check folded in (from L-02) is
  `git fetch` + `checkout -B origin/main` at the top of the loop so no colony ever commits on
  a stale base. Result: later propagations ran in a fraction of the round-trips of the first.

That is the skill working: the sixth propagation was measurably leaner than the first,
*because* the first was measured and its lesson fed forward.

## Where the record lives

`WORKFLOW_NOTES.md` (repo root) — one short section per recurring workflow: its current best
shape, the axis last improved, and any tried-and-rejected optimization. Kept lean on purpose:
it's a living cheat-sheet for "the best way we currently know to do X," not a history book.
Seeded with the propagation example above. `pr-retrospective` handles PR-specific lessons;
this handles procedure-shape lessons; both feed the same recursive-learning spirit.

*Origin: Fable (Harness), 2026-07-16, from the founder's direction that a workflow run more
than once should learn from the last time and enhance itself — through the dualistic lens and
the Trinity mindset, so it improves on every axis, not just one.*
