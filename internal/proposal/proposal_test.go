package proposal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProposalLifecycle(t *testing.T) {
	dir := t.TempDir()
	p, err := Create(dir, "analyze", "hindsight", "too many review comments traced to vague analysis", "---\nname: analyze\n---\nnew body\n")
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != StatusPending {
		t.Fatalf("status=%q", p.Status)
	}

	ps, err := List(dir)
	if err != nil || len(ps) != 1 {
		t.Fatalf("list: %v len=%d", err, len(ps))
	}

	loaded, content, err := Load(dir, p.ID)
	if err != nil || loaded.Skill != "analyze" || content == "" {
		t.Fatalf("load: %v %+v %q", err, loaded, content)
	}

	// Apply is the human gate: it writes the SKILL.md and flips status.
	target := filepath.Join(t.TempDir(), "skills", "analyze", "SKILL.md")
	if err := Apply(dir, p.ID, target); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(target); string(b) != content {
		t.Fatalf("applied content mismatch: %q", b)
	}
	applied, _, _ := Load(dir, p.ID)
	if applied.Status != StatusApplied {
		t.Fatalf("status=%q want applied", applied.Status)
	}
}

func TestCreateRequiresContent(t *testing.T) {
	if _, err := Create(t.TempDir(), "analyze", "retrospect", "r", ""); err == nil {
		t.Fatal("expected error with empty content")
	}
}

func TestListEmpty(t *testing.T) {
	ps, err := List(filepath.Join(t.TempDir(), "nope"))
	if err != nil || ps != nil {
		t.Fatalf("empty list: %v %v", err, ps)
	}
}
