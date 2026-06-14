---
name: triage
description: >-
  Decides which review comments to implement, defer, or reject — with reasons —
  and loops accepted fixes back to develop. Use on "triage", "which comments
  should we fix", after review. Do NOT use to produce the review (review) or to
  ship (ship).
metadata:
  version: "0.1"
  phase: triage
  requires: [review]
  gate: human
  model_role: cheap
  consumes: ["05-review.md"]
  produces: ["06-triage.md"]
---

# Triage

## When this runs
After `review`. Turns review comments into decisions.

## Inputs
- `works/<ticket>/05-review.md`.

## Protocol
1. For each comment, decide: **accept** (fix now), **defer** (file follow-up), or
   **reject** — and give a one-line rationale. Every reject must be justified.
2. The user may override any decision; record overrides explicitly.
3. Write `works/<ticket>/06-triage.md` (comment → decision → rationale, with a
   clear list of what `develop` must fix).
4. For accepted fixes, loop back: re-enter `develop` (a back-edge), implement,
   then re-run `review`. Repeat until no blockers remain.
5. Record: `dock work set triage done --artifact 06-triage.md`.

## Gate
- `human override`: surface the decisions; the user can flip any of them before
  proceeding. `ship` must not start while accepted blockers are unresolved.

## Done when
- Every comment has a decision, accepted blockers are resolved (via develop→
  review), and `06-triage.md` records the rationale.
