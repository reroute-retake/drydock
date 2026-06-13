package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestParseChecksums(t *testing.T) {
	cs := "abc123  dock_linux_amd64.tar.gz\nDEF456  dock_linux_arm64.tar.gz\n"
	m := parseChecksums([]byte(cs))
	if m["dock_linux_amd64.tar.gz"] != "abc123" {
		t.Fatalf("amd64=%q", m["dock_linux_amd64.tar.gz"])
	}
	if m["dock_linux_arm64.tar.gz"] != "def456" { // lower-cased
		t.Fatalf("arm64=%q", m["dock_linux_arm64.tar.gz"])
	}
}

func TestPickAssets(t *testing.T) {
	assets := []Asset{
		{Name: "dock_linux_amd64.tar.gz", URL: "u1"},
		{Name: "dock_linux_arm64.tar.gz", URL: "u2"},
		{Name: "checksums.txt", URL: "u3"},
		{Name: "dock_linux_amd64.tar.gz.sbom.json", URL: "u4"},
	}
	tb, cs, err := pickAssets(assets, "linux", "amd64")
	if err != nil || tb.URL != "u1" || cs.URL != "u3" {
		t.Fatalf("pick: %v %q %q", err, tb.URL, cs.URL)
	}
	if _, _, err := pickAssets(assets, "linux", "riscv64"); err == nil {
		t.Fatal("expected error for missing arch")
	}
}

func TestSameVersion(t *testing.T) {
	if !sameVersion("v0.1.0", "0.1.0") {
		t.Fatal("v0.1.0 should equal 0.1.0")
	}
	if sameVersion("v0.2.0", "0.1.0") {
		t.Fatal("different versions must not be equal")
	}
	if sameVersion("", "") {
		t.Fatal("empty versions must not be considered equal")
	}
}

func TestVerifySHA256(t *testing.T) {
	data := []byte("hello drydock")
	sum := sha256.Sum256(data)
	if err := verifySHA256(data, hex.EncodeToString(sum[:])); err != nil {
		t.Fatalf("should match: %v", err)
	}
	if err := verifySHA256(data, "deadbeef"); err == nil {
		t.Fatal("should mismatch")
	}
}

func makeTarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func TestExtractBinary(t *testing.T) {
	tg := makeTarGz(t, "dock", []byte("BINARY"))
	got, err := extractBinary(tg, "dock")
	if err != nil || string(got) != "BINARY" {
		t.Fatalf("extract: %v %q", err, got)
	}
	if _, err := extractBinary(tg, "nope"); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestAtomicReplace(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "dock")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := atomicReplace(target, []byte("new"), 0o755); err != nil {
		t.Fatalf("replace: %v", err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "new" {
		t.Fatalf("content=%q", got)
	}
	if fi, _ := os.Stat(target); fi.Mode().Perm() != 0o755 {
		t.Fatalf("mode=%v", fi.Mode().Perm())
	}
	// No leftover temp files in the dir.
	ents, _ := os.ReadDir(dir)
	if len(ents) != 1 {
		t.Fatalf("expected 1 file, got %d", len(ents))
	}
}
