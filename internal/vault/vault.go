// Package vault implements drydock's side of the vault interface (design doc 8):
// archive copies a ticket's works/ artifacts into the vault's inbox/, and the
// (deferred) vault project owns ingestion. drydock optionally invokes an
// executable hook at <vault>/bin/ingest if the vault provides one.
//
// Secrets never travel here: a file named .env is skipped on copy (P12/C8).
package vault

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Archive copies srcTicketDir into <vaultDir>/inbox/<ticket>/ and returns the
// inbox path.
func Archive(srcTicketDir, vaultDir, ticket string) (string, error) {
	dst := filepath.Join(vaultDir, "inbox", ticket)
	if err := copyTree(srcTicketDir, dst); err != nil {
		return "", err
	}
	return dst, nil
}

// IngestHook returns <vaultDir>/bin/ingest if it exists and is executable.
func IngestHook(vaultDir string) (string, bool) {
	p := filepath.Join(vaultDir, "bin", "ingest")
	if fi, err := os.Stat(p); err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
		return p, true
	}
	return "", false
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if d.Name() == ".env" {
			return nil // never copy secrets into the vault
		}
		return copyFile(p, target)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
