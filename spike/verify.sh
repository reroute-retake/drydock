#!/usr/bin/env bash
# M0 spike verification. Run inside the dev container:
#   docker compose exec dev bash /usr/local/bin/verify.sh
# Proves the gateway is reachable and routes to a hosted AND a local model.
set -uo pipefail

GW="${OPENAI_URL:-http://gateway:4000/v1}"
KEY="${OPENAI_API_KEY:-sk-drydock-local}"
pass=0
fail=0
ok()  { echo "  [ok]   $1"; pass=$((pass + 1)); }
bad() { echo "  [FAIL] $1"; fail=$((fail + 1)); }

echo "== gateway: $GW =="
if curl -fsS -H "Authorization: Bearer $KEY" "$GW/models" >/tmp/models.json 2>/dev/null; then
  ok "GET /models"
  jq -r '.data[].id' /tmp/models.json 2>/dev/null | sed 's/^/        - /'
else
  bad "GET /models (is the gateway up? docker compose logs gateway)"
fi

chat() { # $1 = model alias
  curl -fsS -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" \
    -d "{\"model\":\"$1\",\"messages\":[{\"role\":\"user\",\"content\":\"reply with the word OK\"}],\"max_tokens\":5}" \
    "$GW/chat/completions"
}

echo "== hosted model: dock/reason =="
if chat "dock/reason" >/tmp/hosted.json 2>/dev/null; then
  ok "chat dock/reason -> $(jq -r '.choices[0].message.content' /tmp/hosted.json 2>/dev/null | tr -d '\n')"
else
  bad "chat dock/reason (check ANTHROPIC_API_KEY in spike/.env)"
fi

echo "== local model: dock/local =="
if chat "dock/local" >/tmp/local.json 2>/dev/null; then
  ok "chat dock/local -> $(jq -r '.choices[0].message.content' /tmp/local.json 2>/dev/null | tr -d '\n')"
else
  bad "chat dock/local (on the host: 'ollama serve' && 'ollama pull llama3.1')"
fi

echo "== ForgeCode =="
if command -v forge >/dev/null 2>&1; then
  ok "forge present: $(forge --version 2>/dev/null | head -1)"
else
  bad "forge not on PATH (re-check the install step in spike/Dockerfile)"
fi

echo
echo "result: ${pass} passed, ${fail} failed"
[ "$fail" -eq 0 ]
