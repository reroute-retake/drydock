package config

import (
	"os"
	"path/filepath"
	"testing"
)

const sample = `space: payments
vault: { repo: "git@h:o/v.git", branch: main }
repos:
  - url: git@h:o/core.git
    stack: { lang: java, version: 21, build: maven }
image: { base: drydock/base:1.x, stacks: [jdk21-maven] }
ports: [8080]
models:
  reason: { provider: anthropic, model: m-reason }
  code:   { provider: anthropic, model: m-code }
  local:  { provider: ollama, model: llama3.1, api_base: http://host.docker.internal:11434 }
routing:
  analyze: reason
  develop: code
  test: local
`

func writeTmp(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "space.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadValid(t *testing.T) {
	m, err := Load(writeTmp(t, sample))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if m.Space != "payments" {
		t.Fatalf("space=%q", m.Space)
	}
	// Scalar handles an int version.
	if got := string(m.Repos[0].Stack.Version); got != "21" {
		t.Fatalf("version=%q want 21", got)
	}
	role, mdl, ok := m.ModelForPhase("develop")
	if !ok || role != "code" || mdl.Model != "m-code" {
		t.Fatalf("develop routing wrong: %q %+v %v", role, mdl, ok)
	}
	if _, lm, _ := m.ModelForPhase("test"); lm.APIBase == "" {
		t.Fatalf("local model should carry api_base")
	}
}

func TestValidateCatchesBadRouting(t *testing.T) {
	bad := `space: x
image: { base: b }
models: { code: { provider: anthropic, model: m } }
routing: { develop: nope }
`
	if _, err := Load(writeTmp(t, bad)); err == nil {
		t.Fatal("expected validation error for unknown model role")
	}
}

func TestTemplateParses(t *testing.T) {
	p := filepath.Join(t.TempDir(), "space.yaml")
	if err := os.WriteFile(p, Template("demo"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := Load(p)
	if err != nil {
		t.Fatalf("template should load: %v", err)
	}
	if m.Space != "demo" {
		t.Fatalf("space=%q", m.Space)
	}
}
