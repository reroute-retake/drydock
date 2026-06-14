---
name: handoff
description: >-
  Writes a clean handoff document so a later session can resume the ticket. Use
  on "handoff", "hand this off", "pause and summarize for later", "I need to
  stop". Do NOT use to archive a finished ticket (archive) or to plan work (plan).
metadata:
  version: "0.1"
  phase: handoff
  requires: []
  gate: none
  model_role: cheap
  consumes: ["state.yaml"]
  produces: ["HANDOFF.md"]
---

# Handoff

## When this runs
Any time a ticket needs to pause and be resumable by a future session.

## Inputs
- `works/<ticket>/state.yaml` (`dock work status` for the phase table) and the
  phase artifacts produced so far.

## Protocol
1. Read the current state and skim the phase artifacts.
2. Write `works/<ticket>/HANDOFF.md` covering:
   - **Goal** — one line, from `ticket.md`.
   - **Where we are** — current phase + what each completed phase concluded.
   - **In flight** — what's partially done (uncommitted work, open branches).
   - **Next steps** — the immediate actions to resume.
   - **Open questions / risks** — anything unresolved.
   - **How to resume** — exact commands (`dock start`, `dock shell`,
     `dock work status <ticket>`, which skill to run next).
3. Keep it self-contained — assume the next session has no memory of this one.

## Gate
- none.

## Done when
- `HANDOFF.md` exists and a fresh reader could resume the ticket from it alone.
