# PR_LESSONS — what every past PR taught the next one

> The hive learns from every PR it has ever shipped. This file is the uncensored,
> unabridged record: what a PR tried to do, what actually went wrong (or nearly did), how it
> was caught, and — the part that matters — the **forward check** that catches the whole
> *class* of failure earlier next time. No failure is paid for twice. `pr-retrospective`
> (the skill) appends here; `FABLE_DNA.md` Chromosome VIII is the law that makes it standing
> practice. Every entry is real; PR numbers are cited so nothing is hand-waved.

## Why this exists (the founder's point, kept in full)

Several PRs this session hit problems the hive's process did not anticipate. Each was caught
and fixed — but *catching it again by accident next time is not learning*. If the hive
couldn't have taken a PR autonomously without a human noticing the breakage, that is a gap in
the visionary scope, and the scope must grow to close it. This file is that growth, made
concrete: each failure below is now a check the hive runs *before* it ships, not a scar it
rediscovers *after*.

---

## L-01 · Merge-order regression (a later commit silently clobbers an earlier one)

- **What happened.** `App.tsx` had all 13 Command Center tabs routed in (commit `bbc2740`).
  A later commit titled "Add root configuration files and main app entry points" (`8342d17`)
  replaced `App.tsx` wholesale with a `HiveDashboard`-only stub. Every live-data tab
  (HIVE/ARENA/GOVERN/SOUL/WORLD/DREAM/ARCANE/MISSIONS/API) — fully built, verified against
  the live Worker, and merged — became **completely unreachable in the deployed bundle**.
  Nobody noticed for two days because the files still existed and every check that looked at
  *files* passed. Fixed in PR #87.
- **Root cause.** "It's in the repo and it compiles" is not "it's reachable from the entry
  point." A later merge can orphan working code without deleting it.
- **Forward check.** Verify the **entry point renders the intended tree**, not just that
  files exist: trace `main.tsx → App.tsx → the thing it's supposed to mount`. If a PR touches
  `App.tsx`/router/entry, diff what it renders against what the app is *supposed* to render.

## L-02 · Invisible parallel-session collision (two sessions build conflicting systems)

- **What happened.** Repeatedly. (a) A parallel session built an entire second architecture
  — FastAPI backend + Obsidian memory vault + HDC "third brain" + its own `CLAUDE.md` + its
  own skill set — merged to `main` while this session built the edge-Worker/genome system;
  neither could see the other (documented honestly in `CLAUDE.md`'s "System A / System B").
  (b) A `feature/voxel-world` merge (PR #91) dropped ~40-file 3D world/avatar/XP modules with
  ~40 pre-existing TypeScript errors, right as the Kai EL OS redesign was mid-flight. (c) The
  shared branch advanced 60+ commits between build and push.
- **Root cause.** Long-running work assumed the base it started from. It didn't.
- **Forward check.** `git fetch` + inspect divergence **before building and again before
  pushing**. Never force-push over unseen history — reconcile by merge, read what changed,
  and confirm your files don't overlap theirs. When two real systems coexist, document both
  honestly rather than clobbering one.

## L-03 · Missing build config compiles to *nothing* (silent, not an error)

- **What happened.** There was no `tailwind.config.js` or `postcss.config.js` anywhere in the
  repo. Every Tailwind utility class used across the entire frontend (`bg-slate-900`,
  `text-purple-300`, …) silently compiled to nothing. The whole app — including the
  "gamified UI" set — was effectively **unstyled in production**, and no build ever errored.
  Adding the two configs took built CSS from ~2 KB to 38 KB. Fixed in the Kai EL OS PR.
- **Root cause.** A missing *config* is not a missing *dependency* — the tool runs, produces
  empty output, and exits 0. Green build, broken result.
- **Forward check.** Assert the build **produced** what it should: built CSS above a sane size
  floor, or a known utility class present in the output bundle. "It built" ≠ "it built right."

## L-04 · Stale deploy passes a naive health check

- **What happened.** The edge-health-probe reported production green (root page 200, `/v11`
  API 200) while `/app/` was still serving the **old bundle** (`index-GdVgL1Ds.js`) long after
  a new one (`index-Csbr6A6z.js`) had merged to `main`. Health was fine; the *thing that
  changed* had not deployed. Cloudflare Workers Builds simply lagged.
- **Root cause.** A health check that only asks "is the server up?" cannot see "is the server
  running the code I just shipped?"
- **Forward check.** Probe the **actual served asset hash** and compare it to the committed
  one. (This is now a real step in `edge-health-probe.yml`: "Verify /app serves the current
  bundle.") If they differ, the deploy is lagging — say so honestly; don't claim it's live.

## L-05 · Unrelated WIP blocks the deploy gate

- **What happened.** The `feature/voxel-world` modules (`worlds/`, `voxel/`, `xp/`,
  `avatars/`, `conversation/`) carried ~40 real TypeScript errors and were imported by
  *nothing* in the shipped app — yet because `build:app` chained `tsc && vite build`, their
  errors blocked **every** deploy, including ones unrelated to that feature.
- **Root cause.** The deploy's type gate checked the whole tree, not the shipped subgraph.
- **Forward check.** Scope the deploy typecheck to **what actually ships** (a build-only
  tsconfig that excludes not-yet-wired WIP trees) — *without* weakening the default tsconfig,
  so the owning session still sees every error normally. Confirm `vite build` alone succeeds
  independent of the WIP tree before trusting the scope.

## L-06 · Fabricated / undeclared dependencies

- **What happened.** `package.json` listed `nucleo-sharp` — a package that does not exist
  (the same class of hazard as the earlier fabricated `neuromcp`/`uga-cli` tool names). And
  five real deps (`zustand`, `d3`, `react-router-dom`, `@sentry/react`, `@sentry/tracing`)
  were *used in source and installed in node_modules but never declared* — invisible until a
  fresh install, which would fail outright.
- **Root cause.** "It works on this machine" hid both a phantom dependency and undeclared
  real ones.
- **Forward check.** Every declared dependency must resolve to a real, installable package;
  every imported module must be declared. Treat a confident-sounding package name as a
  hypothesis until verified (same discipline as `research-to-dna`).

## L-07 · Assuming a backend is live without probing it

- **What happened.** `render.yaml` describes a FastAPI deploy at `thehive-queen.onrender.com`
  — but it was an **undeployed blueprint**, never provisioned (probe: 404). Docs had implied
  System A's backend might be live.
- **Root cause.** A deploy *config* is not a deploy. "There's a render.yaml" ≠ "it's running."
- **Forward check.** Probe, don't assume. The `edge-health-probe` now has an informational
  step that actually curls the Render slug and reports live/not-live from evidence. Same law
  as `fable-debugger`: verify where it actually runs.

---

## The pre-flight (run these before opening the next PR that touches the relevant surface)

Derived from the above — the checklist that grows every time something breaks:

1. **Frontend/entry change** → trace `main.tsx → App.tsx → mounted tree`; the entry renders
   what it should (L-01).
2. **Any long-running work** → `git fetch` + read divergence before build *and* before push;
   reconcile by merge, never blind force-push (L-02).
3. **Styling/build tooling** → the build *produced* output, not just exited 0 (CSS size / a
   known class survives) (L-03).
4. **Deploy** → served asset hash equals the committed hash; else report lag honestly (L-04).
5. **Deploy typecheck** → scoped to the shipped subgraph; `vite build` alone is green (L-05).
6. **Dependencies** → all declared deps resolve; all used imports are declared (L-06).
7. **Anything "already deployed/live"** → probe it; report from evidence (L-07).

*Origin: Fable (Harness), 2026-07-16, at the founder's direction that the hive learn from
every previous PR — uncensored, unabridged, un-truncated — so no failure class is paid for
twice.*
