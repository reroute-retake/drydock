// Package version exposes build-time version metadata.
//
// Values are injected at build time via -ldflags, e.g.:
//
//	go build -ldflags "-X github.com/reroute-retake/drydock/internal/version.Version=$(git describe --tags --dirty)" ./cmd/dock
package version

import (
	"fmt"
	"runtime"
)

// Set via -ldflags at build time; defaults are used for `go run`/dev builds.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String returns a human-readable, single-line version banner.
func String() string {
	return fmt.Sprintf("dock %s (commit %s, built %s, %s/%s)",
		Version, Commit, Date, runtime.GOOS, runtime.GOARCH)
}
