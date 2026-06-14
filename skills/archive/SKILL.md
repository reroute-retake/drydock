---
name: archive
description: >-
  Ingests a shipped ticket's artifacts into the vault and clears the works/
  folder for the next task. Use on "archive", "wrap up the ticket", "ingest into
  the vault", after the PR is shipped. Do NOT use to pause/resume a ticket
  (handoff) or to open the PR (ship).
metadata:
  version: "0.1"
  phase: archive
  requires: [ship]
  gate: none
  model_role: cheap
  consumes: ["works/"]
  produces: ["vault inbox bundle"]
---

# Archive

## When this runs
After `ship` (PR open/merged). The final phase; closes the loop into the vault.

## Protocol
1. Confirm the ticket is shipped and its artifacts are complete.
2. Stage the artifacts into the vault inbox: run `dock archive <ticket>`. This
   copies `works/<ticket>/` to `<vault>/inbox/<ticket>/` (skipping any `.env`)
   and records the archive phase done.
3. Ingest: drydock runs the vault's `bin/ingest` hook if present; otherwise
   invoke the **`vault:ingest`** skill on the inbox path. Ingestion (dedup, merge
   into canonical pages, human review) is owned by the vault project (design 8).
4. Once ingestion succeeds, clear the workspace for the next task:
   `dock archive <ticket> --clean` (removes `works/<ticket>/`). Do NOT clean
   before ingestion is confirmed.

## Gate
- none. (The vault's own ingest step may have a human review gate; that lives in
  the vault project.)

## Done when
- The ticket's artifacts are in the vault inbox (and ingested), and `works/` is
  cleared.
