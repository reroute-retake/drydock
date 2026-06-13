# M0 Spike — ForgeCode → LiteLLM → (hosted + local)

Proves the core routing assumption: ForgeCode (configured with an OpenAI-compatible
base URL) talks to a **LiteLLM gateway** that fronts both a **hosted** model and a
**local** model. This is the M0 exit criterion.

## What's here
- `docker-compose.yaml` — `gateway` (LiteLLM) + `dev` (ForgeCode) services.
- `litellm.config.yaml` — two aliases: `dock/reason` (hosted), `dock/local` (Ollama).
- `Dockerfile` — minimal dev image (curl, git, jq, ForgeCode).
- `verify.sh` — smoke test (gateway `/models`, a chat to each alias, forge presence).
- `.env.example` — copy to `.env` and add your key.

## Prerequisites
- Docker + Docker Compose v2 (Linux).
- A hosted key (e.g. `ANTHROPIC_API_KEY`).
- A local model: `ollama serve` then `ollama pull llama3.1` on the host.

## Run
```bash
cp .env.example .env          # then edit: set ANTHROPIC_API_KEY
docker compose up -d --build
docker compose exec dev bash /usr/local/bin/verify.sh
# ... expect: "result: N passed, 0 failed"
docker compose down -v
```
Or from the repo root: `make spike` (and `make spike-down`).

## How it wires together
- `gateway` reads `litellm.config.yaml`; `os.environ/ANTHROPIC_API_KEY` and
  `os.environ/LITELLM_MASTER_KEY` come from `.env`.
- `dev` sets `OPENAI_URL=http://gateway:4000/v1` and `OPENAI_API_KEY=$LITELLM_MASTER_KEY`
  — exactly how a drydock space will point ForgeCode at the gateway.
- `extra_hosts: host.docker.internal:host-gateway` lets the gateway reach a
  host-run Ollama on Linux (where it isn't automatic).

## Troubleshooting
- `dock/local` fails → Ollama not running/pulled on the host, or host-gateway
  mapping missing. Try `--network host` as an alternative on Linux.
- `dock/reason` fails → check `ANTHROPIC_API_KEY` and the model name in
  `litellm.config.yaml` (use one you have access to).
- ForgeCode env path is marked deprecated; if `OPENAI_URL` stops working, switch
  to `forge provider login` for an OpenAI-compatible provider (design doc 11/M0).
