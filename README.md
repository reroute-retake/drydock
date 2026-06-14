# drydock

A container-based, space-scoped engineering environment. `dock` (this CLI) is the
host-side orchestrator; inside the container, the ForgeCode harness runs a fixed
lifecycle of skills (`analyze → grill → plan → develop → review → triage → test →
ship → archive`) with a LiteLLM model gateway and per-ticket artifacts.

> **This is the M0 scaffold.** It establishes the repo structure, the core
> schemas, the CLI skeleton, the build/release pipeline, and a runnable spike
> that proves ForgeCode can route through LiteLLM to a hosted **and** a local
> model. See the design doc for the full architecture and roadmap.

## Layout

```
cmd/dock/            CLI entry point (M0: version + doctor functional; rest stubbed)
internal/version/    build-time version (ldflags)
internal/telemetry/  events.jsonl schema + writer (feeds `retrospect`)
internal/works/      lifecycle state machine + artifact contract
schemas/             JSON Schemas: space, state, telemetry-event
examples/            space.payments.yaml + works/PAY-123/state.yaml
skills/              metadata-nested SKILL.md template + grill example
spike/               M0 proof: LiteLLM gateway + dev container + verify.sh
.goreleaser.yaml     release pipeline (static binary, checksums, SBOM, deb/rpm)
Makefile             build / install / dev / test / spike
```

## Quickstart

Requires Go 1.22+ (build) and Docker (spike).

```bash
# 1. Build & install to a user-owned PATH dir (no root needed)
make install            # -> ~/.local/bin/dock
dock version
dock doctor             # checks docker / forge / git / gateway

# 2. Run the M0 proof (see spike/README.md first to set keys)
cp spike/.env.example spike/.env   # then edit: add ANTHROPIC_API_KEY
make spike                          # brings up gateway + dev, runs verify.sh
make spike-down
```

## Distribution & self-update (design doc 11A)

Install (Linux):

```bash
curl -fsSL https://raw.githubusercontent.com/reroute-retake/drydock/main/install.sh | sh
dock self-update     # later, to upgrade in place
```

- Release artifact: a single static Go binary via GoReleaser (GitHub Releases +
  `checksums.txt` + SBOM + deb/rpm; cosign optional).
- `dock self-update` replaces the **binary** from the latest release (verify →
  atomic rename; graceful if root-owned).
- `dock update` refreshes a **space's** config/scaffolding — never the binary.

## drydock-in-drydock

Develop drydock inside its own space: `make dev` builds `dock-dev` into
`~/.local/bin`; drive the workflow with `dock-dev` while stable `dock` keeps
running. Promote only after `dock-dev version && dock-dev doctor`.

## Status

- **M0** — structure, schemas, spike. ✅
- **M1** — `dock setup/build/start/shell/stop` functional; manifest parsing
  (`gopkg.in/yaml.v3`); per-space LiteLLM gateway + docker-compose generated from
  the manifest; `repos`/`vault`/`works` mounts; `.env` injection; telemetry
  session capture; `--dry-run` to preview docker commands. ✅
- **M2** — `dock addrepo` (clone + stack detection → per-stack image layers),
  `dock sync`, `dock space switch`, `dock update`; `build` now builds a generated
  per-stack Dockerfile. ✅
- **M3** — `dock self-update` (download latest release, verify checksum, atomic
  replace, graceful on root-owned) and `install.sh`, both consuming GoReleaser
  artifacts. ✅
- **M4** — works/ artifact contract loader/writer + `dock work new/status/set`;
  `analyze`/`plan`/`develop`/`ship` SKILL.md files over the contract. ✅
- **M5** — `grill`/`review`/`triage`/`test`/`handoff`/`archive` skills; the vault
  `archive`→`ingest` interface (`dock archive` copies works/ to the vault inbox
  and runs a `bin/ingest` hook). ✅
- **M6** — `retrospect` (telemetry) + `hindsight` (per-phase artifacts) skills;
  `dock telemetry` summary; human-gated, versioned skill proposals via
  `dock skill propose / list / show / apply / reject`. ✅

All M0–M6 milestones are implemented. **Hardening pass (in progress):**
- **H1** — multi-space concurrency: a per-space gateway host port, a port-conflict
  pre-check on `dock start`, `dock ps`, and non-stopping `dock space switch`
  (concurrent by default; `--stop` to stop the previous). ✅
- **H2** — version pinning + reproducibility (pin ForgeCode + base-image digest;
  `.drydock/lock.yaml`; record versions per session). *(next)*
- **H3** — an eval harness that validates/scores skill proposals before `apply`. *(next)*

Try it (no Docker needed to preview):

```bash
make install
DRYDOCK_HOME=/tmp/dh dock setup demo     # scaffold + activate a space
dock --dry-run build                     # generate gateway+compose, print the docker cmd
dock --dry-run start                     # create a telemetry session, print 'up'
```
