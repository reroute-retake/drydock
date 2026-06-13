---
name: analyze
description: >-
  Brainstorms and researches a new ticket into a written analysis before any
  planning or coding. Use when starting a ticket, or on "analyze", "scope this",
  "research the problem", "what are the options". Do NOT use to verify/challenge
  an existing analysis (that's grill) or to break work into tasks (that's plan).
metadata:
  version: "0.1"
  phase: analyze
  requires: []
  gate: human
  model_role: reason
  consumes: ["ticket.md"]
  produces: ["01-analysis.md"]
---

# Analyze

## When this runs
First phase of a ticket, before `grill` and `plan`.

## Inputs
- `works/<ticket>/ticket.md` (the ask).
- Grounding: the codebase (Serena), the vault (`/workspace/vault`), MCP servers
  (Context7 for library docs, GitHub for repo/issue context), and web research (Exa).

## Protocol
1. Read `ticket.md`. Pull relevant prior knowledge from the vault and the codebase.
2. Explore the problem: restate the goal, list constraints, surface 2-3 solution
   options with trade-offs, and call out unknowns/risks.
3. Write `works/<ticket>/01-analysis.md`: problem statement, options + recommendation,
   affected areas, open questions. Mark assumptions explicitly (they feed `grill`).
4. Discuss with the user; iterate the document until they agree.
5. Record progress: `dock work set analyze in_progress --model <model>` when you
   start; `dock work set analyze agreed --artifact 01-analysis.md` once agreed.

## Gate
- `human`: do not advance to `grill`/`plan` until the user explicitly agrees the
  analysis is correct. Summarize what changed each round; never infer agreement.

## Done when
- `01-analysis.md` exists and the user has marked it agreed.
