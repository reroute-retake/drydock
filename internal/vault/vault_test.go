package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArchiveCopiesTreeAndSkipsEnv(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "works", "PAY-1")
	_ = os.MkdirAll(filepath.Join(src, "docs"), 0o755)
	_ = os.WriteFile(filepath.Join(src, "01-analysis.md"), []byte("analysis"), 0o644)
	_ = os.WriteFile(filepath.Join(src, "docs", "guide.md"), []byte("doc"), 0o644)
	_ = os.WriteFile(filepath.Join(src, ".env"), []byte("SECRET=x"), 0o600)

	vaultDir := filepath.Join(root, "vault")
	inbox, err := Archive(src, vaultDir, "PAY-1")
	if err != nil {
		t.Fatal(err)
	}
	if inbox != filepath.Join(vaultDir, "inbox", "PAY-1") {
		t.Fatalf("inbox=%s", inbox)
	}
	if b, _ := os.ReadFile(filepath.Join(inbox, "01-analysis.md")); string(b) != "analysis" {
		t.Fatalf("analysis not copied: %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(inbox, "docs", "guide.md")); string(b) != "doc" {
		t.Fatalf("nested doc not copied: %q", b)
	}
	if _, err := os.Stat(filepath.Join(inbox, ".env")); !os.IsNotExist(err) {
		t.Fatal(".env must NOT be copied into the vault")
	}
}

func TestIngestHookDetection(t *testing.T) {
	vaultDir := t.TempDir()
	if _, ok := IngestHook(vaultDir); ok {
		t.Fatal("no hook should be detected yet")
	}
	bin := filepath.Join(vaultDir, "bin")
	_ = os.MkdirAll(bin, 0o755)
	hook := filepath.Join(bin, "ingest")
	_ = os.WriteFile(hook, []byte("#!/bin/sh\n"), 0o755)
	if got, ok := IngestHook(vaultDir); !ok || got != hook {
		t.Fatalf("hook not detected: %q %v", got, ok)
	}
}
