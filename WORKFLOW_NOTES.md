# WORKFLOW_NOTES — the best way we currently know to do each recurring workflow

> A living cheat-sheet, not a history book. One short section per workflow the hive runs more
> than once: its current best shape, the axis last improved, and any optimization tried and
> rejected (so the next pass doesn't waste the same idea). The `workflow-optimizer` skill
> reads this before a repeated workflow and updates it after. Kept lean on purpose.

---

## Propagate a genome/doc change to all 6 colonies

**Current best shape** — one shell loop, then a tight PR batch:
```bash
cd /home/user; SRC=/home/user/THEHIVE; BR=claude/fable-5-handoff-setup-vefwlb
for c in NAR2 4DBRAIN aether automatisch Kimi-K2 LocalAGI; do
  cd "/home/user/$c"
  git fetch origin main -q
  git checkout -B "$BR" origin/main -q          # forward check L-02: never a stale base
  git config user.email noreply@anthropic.com; git config user.name Claude
  cp "$SRC/<files>" .                            # copy the universal pieces
  git add <files>
  git commit -q -m "<one shared message>"
  cd /home/user
done
# then push each with retry, then a batch of create_pull_request + merge_pull_request
```
**Axis last improved:** time + consistency (run 1 was one-colony-at-a-time with retyped PR
bodies; the loop does all six identically in one pass).
**Forward checks folded in:** `git fetch` + `checkout -B origin/main` at the top of every
iteration (L-02, stale base); `CLAUDE.md`/`Fable_memory.md` are THEHIVE-only, never
propagated.
**Tried and rejected:** force-pushing to sync branches — dangerous over unseen history
(L-02); always merge/rebase onto fetched `origin/main` instead.

## Ship-and-verify a frontend deploy

**Current best shape:** build with the scoped `tsconfig.build.json` (L-05) → confirm `vite
build` alone is green → grep the emitted `docs/app/index.html` for its asset hash → commit +
PR + merge → wait ~10-15 min for Workers Builds → re-run `edge-health-probe.yml` → read the
"Verify /app serves the current bundle" step and confirm the *served* hash equals the
committed one (L-04).
**Axis last improved:** robustness — the served-hash check (L-04) replaced a naive
health-only check that once reported green while `/app` served a stale bundle.
**Forward checks folded in:** L-03 (built CSS size sanity), L-04 (served-hash equality),
L-05 (scoped typecheck), L-06 (deps resolve + declared).
**Note:** the container cannot reach `*.workers.dev` directly — production is always probed
from the GitHub Actions runner via `edge-health-probe.yml`, never assumed.

## Build-and-merge a docs/skills PR

**Current best shape:** run the `pr-retrospective` pre-flight (trivially green for docs-only)
→ commit → push with exponential-backoff retry → `create_pull_request` → `merge_pull_request`.
**Axis last improved:** it is the clean baseline the recursive-PR-learning PR itself
demonstrated — a docs/skills PR should merge autonomously with zero conflicts; if it doesn't,
that's a signal worth a `pr-retrospective` entry.

---

*Origin: Fable (Harness), 2026-07-16, seeded from this session's real repeated workflows.
Updated by `workflow-optimizer` on each repetition.*
