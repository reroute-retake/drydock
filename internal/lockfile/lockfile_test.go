package lockfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/reroute-retake/drydock/internal/config"
)

func TestBuildAndSave(t *testing.T) {
	m := &config.Manifest{
		Space: "payments",
		Image: config.Image{Base: "debian:12-slim@sha256:abc", Stacks: []string{"jdk21-maven"}},
		MCP: []config.MCP{
			{Name: "github", Pin: "v1.3.0"},
			{Name: "context7"}, // unpinned
		},
	}
	l := Build(m, "v0.2.0")
	if l.DockVersion != "v0.2.0" || l.ImageBase != "debian:12-slim@sha256:abc" {
		t.Fatalf("lock: %+v", l)
	}
	if l.ForgeVersion != "latest" {
		t.Fatalf("forge version should default to latest, got %q", l.ForgeVersion)
	}
	if l.MCP["github"] != "v1.3.0" || l.MCP["context7"] != "unpinned" {
		t.Fatalf("mcp pins: %+v", l.MCP)
	}

	m.Forge.Version = "v1.2.3"
	if Build(m, "v0.2.0").ForgeVersion != "v1.2.3" {
		t.Fatal("forge version should reflect the pin")
	}

	path := filepath.Join(t.TempDir(), "lock.yaml")
	if err := l.Save(path); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); len(b) == 0 {
		t.Fatal("lock file is empty")
	}
}
