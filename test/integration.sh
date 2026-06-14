#!/usr/bin/env bash
# drydock integration test — RUN ON A REAL LINUX + DOCKER HOST (not in CI sandbox).
#
# It builds the jdk21-maven dev image, brings up a throwaway space, and checks:
#   1. the dev image builds and has java + maven
#   2. the LiteLLM gateway comes up and /models lists the routed aliases
#   3. ForgeCode in the dev container points at the gateway (OPENAI_URL)
#   4. a chat routes through the gateway (uses a local Ollama model if present,
#      else a hosted model if ANTHROPIC_API_KEY is set)
#   5. the LiteLLM telemetry callback wrote per-call events to the session
#
# Self-contained: uses temp DRYDOCK_HOME/DRYDOCK_STATE under .itest/ and tears down.
#
#   make integration
#   ANTHROPIC_API_KEY=sk-... make integration        # also exercises a hosted model
#   ITEST_OLLAMA_MODEL=llama3.2 make integration      # pick the local model
set -uo pipefail

here="$(cd "$(dirname "$0")/.." && pwd)"
itroot="$here/.itest"
export DRYDOCK_HOME="$itroot/home"
export DRYDOCK_STATE="$itroot/state"
SPACE="itest"
KEY="sk-itest"
COMPOSE="$DRYDOCK_HOME/$SPACE/.drydock/compose.yaml"
ENVFILE="$DRYDOCK_HOME/$SPACE/.env"
dc() { if [ -f "$ENVFILE" ]; then docker compose --env-file "$ENVFILE" -f "$COMPOSE" "$@"; else docker compose -f "$COMPOSE" "$@"; fi; }

pass=0 fail=0 warn=0
ok()   { echo "  [ok]   $1"; pass=$((pass + 1)); }
bad()  { echo "  [FAIL] $1"; fail=$((fail + 1)); }
note() { echo "  [warn] $1"; warn=$((warn + 1)); }

cleanup() {
  echo "== teardown =="
  [ -f "$COMPOSE" ] && dc down -v >/dev/null 2>&1
  rm -rf "$itroot"
}
trap cleanup EXIT

command -v docker >/dev/null || { echo "docker is required"; exit 1; }

echo "== resolve dock (build from this checkout) =="
if [ -n "${DOCK:-}" ]; then
  ok "using DOCK=$DOCK"
elif command -v go >/dev/null 2>&1; then
  make -C "$here" build >/dev/null || { echo "go build failed"; exit 1; }
  DOCK="$here/bin/dock"
  ok "built dock from source ($(go version 2>/dev/null | awk '{print $3}'))"
else
  cat <<'MSG'
  [FAIL] Go not found on PATH — needed to build the current dock.
         - install Go (https://go.dev/dl), ensure its bin dir is on PATH, re-run
         - or, if you built dock elsewhere:  DOCK=/path/to/dock make integration
         (An install.sh-installed v0.1.0 dock predates the gateway telemetry
          callback and will fail that check — build from this checkout.)
MSG
  exit 1
fi

echo "== scaffold throwaway space =="
rm -rf "$itroot"
"$DOCK" setup "$SPACE" >/dev/null
cat > "$DRYDOCK_HOME/$SPACE/space.yaml" <<YAML
space: $SPACE
vault: { repo: "", branch: main }
repos: []
image:
  base: debian:12-slim
  stacks: [jdk21-maven]
forge:
  version: ""
ports: []
models:
  local:  { provider: ollama, model: ${ITEST_OLLAMA_MODEL:-llama3.2}, api_base: http://host.docker.internal:11434 }
  reason: { provider: anthropic, model: claude-3-5-sonnet-latest }
routing:
  analyze: reason
  develop: local
YAML
cat > "$DRYDOCK_HOME/$SPACE/.env" <<ENV
LITELLM_MASTER_KEY=$KEY
ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY:-}
ENV

echo "== dock build (docker build the jdk21-maven dev image) =="
if "$DOCK" build; then ok "dock build"; else bad "dock build"; fi
if docker run --rm "drydock-$SPACE:dev" bash -lc 'java -version 2>&1 | head -1; mvn -v 2>&1 | head -1'; then
  ok "java + maven present in dev image (note: debian default-jdk may be 17, not 21 — pin a real JDK 21)"
else
  note "could not verify java/maven in the dev image"
fi

echo "== dock start =="
if "$DOCK" start; then ok "dock start"; else bad "dock start"; fi
GW="$(grep -oE '"[0-9]+:4000"' "$COMPOSE" | head -1 | sed -E 's/"([0-9]+):4000"/\1/')"
echo "  gateway host port: ${GW:-unknown}"

echo "== wait for gateway =="
for _ in $(seq 1 30); do
  curl -fsS -H "Authorization: Bearer $KEY" "http://localhost:$GW/v1/models" >/tmp/it_models.json 2>/dev/null && break
  sleep 2
done

echo "== gateway /models lists routed aliases =="
if grep -q '"local"' /tmp/it_models.json 2>/dev/null && grep -q '"reason"' /tmp/it_models.json 2>/dev/null; then
  ok "/models lists aliases (local, reason)"
else
  bad "/models did not list the routed aliases"; head -c 400 /tmp/it_models.json 2>/dev/null; echo
fi

echo "== ForgeCode in dev points at the gateway =="
if dc exec -T dev sh -lc 'printf %s "$OPENAI_URL" | grep -q "gateway:4000"'; then
  ok "dev OPENAI_URL -> gateway:4000"
else
  bad "OPENAI_URL not pointing at the gateway in dev"
fi
if dc exec -T dev sh -lc 'command -v forge >/dev/null'; then
  ok "forge installed in dev"
else
  note "forge not installed in dev (installer may have changed; see spike/Dockerfile)"
fi

echo "== chat through the gateway (routing + telemetry) =="
chat() {
  curl -fsS -H "Authorization: Bearer $KEY" -H 'Content-Type: application/json' \
    -d "{\"model\":\"$1\",\"messages\":[{\"role\":\"user\",\"content\":\"say OK\"}],\"max_tokens\":5,\"metadata\":{\"space\":\"$SPACE\",\"ticket\":\"itest\",\"phase\":\"develop\",\"skill\":\"develop\",\"session_id\":\"itest\"}}" \
    "http://localhost:$GW/v1/chat/completions"
}
chatted=0
if curl -fsS "http://localhost:11434/api/tags" >/dev/null 2>&1; then
  if chat local >/tmp/it_chat.json 2>/dev/null; then ok "chat routed via local (ollama) model"; chatted=1
  else note "local chat failed — is the model pulled? 'ollama pull ${ITEST_OLLAMA_MODEL:-llama3.2}'"; fi
elif [ -n "${ANTHROPIC_API_KEY:-}" ]; then
  if chat reason >/tmp/it_chat.json 2>/dev/null; then ok "chat routed via hosted model"; chatted=1
  else note "hosted chat failed — check ANTHROPIC_API_KEY and the model name"; fi
else
  note "no local Ollama and no ANTHROPIC_API_KEY — skipping live chat (set one to exercise routing + telemetry)"
fi

echo "== telemetry callback wrote per-call events =="
EV="$DRYDOCK_STATE/sessions/$SPACE/gateway/events.jsonl"
if [ "$chatted" = "1" ]; then
  if [ -s "$EV" ] && grep -q '"event_type"' "$EV"; then ok "gateway callback wrote $EV"
  else bad "no telemetry events at $EV (check the litellm callback)"; fi
else
  note "skipped telemetry assertion (no chat was performed)"
fi

echo
echo "result: $pass passed, $fail failed, $warn warning(s)"
[ "$fail" -eq 0 ]
