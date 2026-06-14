---
name: review
description: >-
  Critiques the implemented diff from multiple angles and checks it against the
  project's architectural principles. Use on "review", "review the changes",
  "code review", after develop. Do NOT use to decide which comments to act on
  (triage) or to run tests (test).
metadata:
  version: "0.1"
  phase: review
  requires: [develop]
  gate: none
  model_role: review
  consumes: ["diff", "03-plan.md"]
  produces: ["05-review.md"]
---

# Review

## When this runs
After `develop`. Run by a **different model than develop** (R6) for an
independent perspective.

## Inputs
- The diff for the ticket's commits, `works/<ticket>/03-plan.md`, and the space's
  declared principles (manifest `principles_ref`, e.g. a vault page).

## Protocol
1. Review the diff across these perspectives: correctness, simplicity, security,
   performance, readability, and test adequacy.
2. Run an **architectural-fitness check** against the declared project principles
   (constraint C10): flag anything that drifts from them.
3. Write `works/<ticket>/05-review.md` as a list of comments, each with a
   **severity** (blocker / major / minor / nit), file:line, and a concrete
   suggestion. Blockers must be unambiguous.
4. Record: `dock work set review done --artifact 05-review.md --model <model>`.

## Gate
- none — `triage` decides what to act on next.

## Done when
- `05-review.md` exists with severity-ranked comments (or an explicit "no
  findings").
