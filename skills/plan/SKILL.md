---
name: plan
description: >-
  Breaks an agreed (grilled) analysis into a small, ordered set of implementation
  steps. Use on "plan", "break this down", "what's the sequence", after the
  analysis is settled. Do NOT use to write the analysis (analyze) or to implement
  code (develop).
metadata:
  version: "0.1"
  phase: plan
  requires: [grill]
  gate: human
  model_role: reason
  consumes: ["02-analysis.grilled.md"]
  produces: ["03-plan.md"]
---

# Plan

## When this runs
After `grill` has produced a de-risked analysis (or, on the lightweight path,
directly after `analyze`). Before `develop`.

## Inputs
- `works/<ticket>/02-analysis.grilled.md` (fallback: `01-analysis.md` if grill was skipped).

## Protocol
1. Decompose the solution into the smallest sensible tasks. Each task should be
   independently testable and committable.
2. Sequence them: declare dependencies, and mark which tasks can run in parallel.
3. For each task note the acceptance check and any docs it must add/update
   (docs travel with code — design P7).
4. Write `works/<ticket>/03-plan.md` as an ordered task list (the human-readable
   plan). If the space's `tooling.tasks: beads`, also create the task graph:
   `bd create` per task and `bd dep add` for dependencies, so `develop` can pull
   ready tasks deterministically.
5. Record: `dock work set plan agreed --artifact 03-plan.md --model <model>`.

## Gate
- `human`: get the user's sign-off on the task breakdown and sequencing before
  `develop` starts.

## Done when
- `03-plan.md` exists (and, if enabled, the beads graph), agreed by the user.
