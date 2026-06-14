package gen

import (
	"fmt"
	"strings"
	"testing"

	"github.com/reroute-retake/drydock/internal/config"
	"github.com/reroute-retake/drydock/internal/paths"
	"github.com/reroute-retake/drydock/internal/ports"
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
		`callbacks: ["drydock_logger.instance"]`,
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
		"drydock_logger.py:/etc/litellm/drydock_logger.py",
		`DRYDOCK_TELEMETRY_DIR: "/telemetry"`,
		`"8080:8080"`,
		`"9090:9090"`,
		"host.docker.internal:host-gateway",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("compose missing %q\n---\n%s", want, out)
		}
	}
	if gw := fmt.Sprintf(`"%d:4000"`, ports.GatewayPort("payments")); !strings.Contains(out, gw) {
		t.Fatalf("compose missing gateway port mapping %q\n---\n%s", gw, out)
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

func TestLiteLLMLogger(t *testing.T) {
	src := LiteLLMLogger()
	for _, want := range []string{"CustomLogger", "events.jsonl", "llm_call", "instance = DrydockLogger()"} {
		if !strings.Contains(src, want) {
			t.Fatalf("logger source missing %q", want)
		}
	}
}

func TestDockerfileForgePin(t *testing.T) {
	m := manifest()
	if strings.Contains(Dockerfile(m), "FORGE_VERSION=") {
		t.Fatal("unpinned manifest should not set FORGE_VERSION")
	}
	m.Forge = config.Forge{Version: "v1.2.3"}
	if !strings.Contains(Dockerfile(m), "FORGE_VERSION=v1.2.3") {
		t.Fatalf("expected forge pin in Dockerfile:\n%s", Dockerfile(m))
	}
}

func TestComposeNoDevPorts(t *testing.T) {
	m := manifest()
	m.Ports = nil
	out, err := ComposeFile(m, paths.For("x"))
	if err != nil {
		t.Fatal(err)
	}
	// The gateway always publishes a port, but no dev-server mapping should appear.
	if strings.Contains(out, `"8080:8080"`) {
		t.Fatalf("expected no dev port mapping when Ports is empty\n%s", out)
	}
	if gw := fmt.Sprintf(`"%d:4000"`, ports.GatewayPort(m.Space)); !strings.Contains(out, gw) {
		t.Fatalf("gateway mapping %q should still be present\n%s", gw, out)
	}
}
