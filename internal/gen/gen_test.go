package gen

import (
	"strings"
	"testing"

	"github.com/reroute-retake/drydock/internal/config"
	"github.com/reroute-retake/drydock/internal/paths"
)

func manifest() *config.Manifest {
	return &config.Manifest{
		Space: "payments",
		Image: config.Image{Base: "drydock/base:1.x"},
		Ports: []int{8080, 9090},
		Models: map[string]config.Model{
			"reason": {Provider: "anthropic", Model: "m-reason"},
			"local":  {Provider: "ollama", Model: "llama3.1", APIBase: "http://host.docker.internal:11434"},
		},
		Routing: map[string]string{"analyze": "reason", "test": "local"},
	}
}

func TestLiteLLMConfig(t *testing.T) {
	out, err := LiteLLMConfig(manifest())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"model_name: reason",
		"model: anthropic/m-reason",
		"api_key: os.environ/ANTHROPIC_API_KEY",
		"model_name: local",
		"api_base: http://host.docker.internal:11434",
		"master_key: os.environ/LITELLM_MASTER_KEY",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("litellm config missing %q\n---\n%s", want, out)
		}
	}
	// A local model must NOT get an api_key line.
	if strings.Contains(out, "OLLAMA_API_KEY") {
		t.Fatalf("local model should not request an api_key:\n%s", out)
	}
}

func TestComposeFile(t *testing.T) {
	sp := paths.For("payments")
	out, err := ComposeFile(manifest(), sp)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"name: drydock-payments",
		`image: "drydock-payments:dev"`,
		"dockerfile: Dockerfile",
		`OPENAI_URL: "http://gateway:4000/v1"`,
		"/workspace/repos",
		"/workspace/vault",
		"/workspace/works",
		"/workspace/telemetry:ro",
		`"8080:8080"`,
		`"9090:9090"`,
		"host.docker.internal:host-gateway",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("compose missing %q\n---\n%s", want, out)
		}
	}
}

func TestDockerfile(t *testing.T) {
	m := manifest()
	m.Image = config.Image{Base: "debian:12-slim", Stacks: []string{"jdk21-maven", "node", "bogus"}}
	out := Dockerfile(m)
	for _, want := range []string{
		"FROM debian:12-slim",
		"forgecode.dev/cli",
		"maven",
		"nodejs npm",
		"# stack: bogus",
		"no known layer",
		"WORKDIR /workspace",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("dockerfile missing %q\n%s", want, out)
		}
	}
}

func TestComposeNoPorts(t *testing.T) {
	m := manifest()
	m.Ports = nil
	out, err := ComposeFile(m, paths.For("x"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "ports:") {
		t.Fatalf("expected no ports: block when Ports is empty\n%s", out)
	}
}
