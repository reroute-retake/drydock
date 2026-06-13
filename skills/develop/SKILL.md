---
name: develop
description: >-
  Implements an agreed plan task-by-task, test-first, committing each task with
  its docs. Use on "develop", "implement", "build the plan", "write the code",
  once a plan exists. Do NOT use to plan the work (plan) or to open a PR (ship).
metadata:
  version: "0.1"
  phase: develop
  requires: [plan]
  calls: [tdd]
  gate: none
  model_role: code
  consumes: ["03-plan.md"]
  produces: ["04-dev-log.md", "docs/"]
---

# Develop

## When this runs
After `plan` is agreed. Produces code; no human gate (review/triage come after).

## Inputs
- `works/<ticket>/03-plan.md` (or the beads ready-queue: `bd ready --json`).

## Protocol
For each task (one at a time; spawn a fresh subagent per task to avoid drift):
1. Claim it (`bd update <id> --claim` if using beads).
2. **TDD where the stack permits**: write a failing test, implement, make it pass.
3. Update the docs that the change touches, in the same commit (constraint C5).
4. Append to `works/<ticket>/04-dev-log.md`: the task, issues hit, any workarounds,
   and future-improvement notes (this is what `hindsight` later mines).
5. Commit the task locally (one commit per task, C4). Close it (`bd close <id>`).
   If you discover new work, record it (`bd create ... discovered-from`).

When all tasks are done: `dock work set develop done --model <model> --artifact 04-dev-log.md`.

## Gate
- none. Correctness is checked next by `review` (run by a different model, R6).

## Done when
- All plan tasks are implemented with tests and docs, each committed, and
  `04-dev-log.md` is complete.
