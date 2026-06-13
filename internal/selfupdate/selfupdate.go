// Package selfupdate replaces the running dock binary with the latest GitHub
// release. It is stdlib-only (no third-party deps) and consumes GoReleaser's
// artifacts: dock_<os>_<arch>.tar.gz plus a single checksums.txt.
//
// Flow: resolve release -> pick platform asset -> download tarball + checksums
// -> verify SHA256 -> extract the binary -> atomically replace the current
// executable (write a temp file in the same dir, then rename). A non-writable
// target (root-owned install) yields a clear, actionable error.
package selfupdate

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const apiBase = "https://api.github.com"

// Asset is one release artifact.
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

// Release is the subset of the GitHub release payload we use.
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Options configures a self-update run.
type Options struct {
	Owner, Repo string
	Current     string // current version (from internal/version)
	Version     string // target tag; "" means latest
	Force       bool   // update even if Current == latest tag
	HTTPClient  *http.Client
}

// normVer drops a leading "v" so a tag (v0.1.0) and a GoReleaser-injected
// version (0.1.0) compare equal.
func normVer(s string) string { return strings.TrimPrefix(strings.TrimSpace(s), "v") }

func sameVersion(a, b string) bool { return normVer(a) != "" && normVer(a) == normVer(b) }

// pickAssets selects the platform tarball and the checksums file.
func pickAssets(assets []Asset, goos, goarch string) (tarball, checksums Asset, err error) {
	want := fmt.Sprintf("dock_%s_%s.tar.gz", goos, goarch)
	for _, a := range assets {
		switch a.Name {
		case want:
			tarball = a
		case "checksums.txt":
			checksums = a
		}
	}
	if tarball.URL == "" {
		return tarball, checksums, fmt.Errorf("no release asset named %q", want)
	}
	if checksums.URL == "" {
		return tarball, checksums, errors.New("release is missing checksums.txt")
	}
	return tarball, checksums, nil
}

// parseChecksums parses GoReleaser's checksums.txt lines: "<sha256>  <filename>".
func parseChecksums(data []byte) map[string]string {
	out := map[string]string{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		if f := strings.Fields(sc.Text()); len(f) == 2 {
			out[f[1]] = strings.ToLower(f[0])
		}
	}
	return out
}

func verifySHA256(data []byte, wantHex string) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, wantHex) {
		return fmt.Errorf("checksum mismatch: got %s, want %s", got, wantHex)
	}
	return nil
}

// extractBinary returns the bytes of the named regular file from a .tar.gz.
func extractBinary(targz []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(targz))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag == tar.TypeReg && filepath.Base(h.Name) == name {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("binary %q not found in archive", name)
}

// atomicReplace writes data to a temp file in target's directory, then renames
// over target. A permission error is rewrapped with actionable guidance.
func atomicReplace(target string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, ".dock-new-*")
	if err != nil {
		if os.IsPermission(err) {
			return notWritable(target, err)
		}
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // harmless no-op once renamed
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return err
	}
	if err := os.Rename(tmpName, target); err != nil {
		if os.IsPermission(err) {
			return notWritable(target, err)
		}
		return err
	}
	return nil
}

func notWritable(target string, err error) error {
	return fmt.Errorf("cannot replace %s (%v)\n"+
		"  the binary is not writable by this user — re-run with sudo, or install into a\n"+
		"  user-owned dir (e.g. ~/.local/bin) via install.sh and update that copy", target, err)
}

func (o Options) get(url, accept string) ([]byte, error) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "dock-self-update")
	req.Header.Set("Accept", accept)
	resp, err := o.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// Run performs the update and returns the new version, or "" if already current.
func Run(o Options) (string, error) {
	if o.HTTPClient == nil {
		o.HTTPClient = &http.Client{Timeout: 60 * time.Second}
	}
	relURL := fmt.Sprintf("%s/repos/%s/%s/releases/latest", apiBase, o.Owner, o.Repo)
	if o.Version != "" {
		relURL = fmt.Sprintf("%s/repos/%s/%s/releases/tags/%s", apiBase, o.Owner, o.Repo, o.Version)
	}
	meta, err := o.get(relURL, "application/vnd.github+json")
	if err != nil {
		return "", err
	}
	var rel Release
	if err := json.Unmarshal(meta, &rel); err != nil {
		return "", err
	}
	if rel.TagName == "" {
		return "", errors.New("release has no tag")
	}
	if !o.Force && sameVersion(rel.TagName, o.Current) {
		return "", nil // already up to date
	}
	tarball, checksums, err := pickAssets(rel.Assets, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", err
	}
	tgz, err := o.get(tarball.URL, "application/octet-stream")
	if err != nil {
		return "", err
	}
	cs, err := o.get(checksums.URL, "application/octet-stream")
	if err != nil {
		return "", err
	}
	want, ok := parseChecksums(cs)[tarball.Name]
	if !ok {
		return "", fmt.Errorf("checksums.txt has no entry for %s", tarball.Name)
	}
	if err := verifySHA256(tgz, want); err != nil {
		return "", err
	}
	bin, err := extractBinary(tgz, "dock")
	if err != nil {
		return "", err
	}
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(self); err == nil {
		self = resolved
	}
	if err := atomicReplace(self, bin, 0o755); err != nil {
		return "", err
	}
	return rel.TagName, nil
}
