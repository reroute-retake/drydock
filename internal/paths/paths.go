// Package paths resolves host-side locations for spaces and session telemetry.
//
// Layout (overridable via env for tests / non-standard hosts):
//
//	$DRYDOCK_HOME/<space>/{repos,vault,works,.drydock}   (default ~/Documents)
//	$DRYDOCK_STATE/{active,sessions/...}                 (default ~/.drydock)
package paths

import (
	"os"
	"path/filepath"
)

// Home is the base dir that holds spaces (default ~/Documents).
func Home() string {
	if v := os.Getenv("DRYDOCK_HOME"); v != "" {
		return v
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, "Documents")
}

// StateHome holds drydock's own state: the active-space pointer and sessions.
func StateHome() string {
	if v := os.Getenv("DRYDOCK_STATE"); v != "" {
		return v
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".drydock")
}

// Space holds the resolved directories for one space.
type Space struct {
	Name    string
	Root    string
	Repos   string
	Vault   string
	Works   string
	Drydock string // generated artifacts: compose.yaml, litellm.config.yaml
}

// For returns the Space layout for a given space name.
func For(name string) Space {
	root := filepath.Join(Home(), name)
	return Space{
		Name:    name,
		Root:    root,
		Repos:   filepath.Join(root, "repos"),
		Vault:   filepath.Join(root, "vault"),
		Works:   filepath.Join(root, "works"),
		Drydock: filepath.Join(root, ".drydock"),
	}
}

// Manifest is the path to the space's space.yaml.
func (s Space) Manifest() string { return filepath.Join(s.Root, "space.yaml") }

// Env is the path to the space's gitignored .env.
func (s Space) Env() string { return filepath.Join(s.Root, ".env") }

// Compose is the generated docker-compose path.
func (s Space) Compose() string { return filepath.Join(s.Drydock, "compose.yaml") }

// LiteLLM is the generated gateway config path.
func (s Space) LiteLLM() string { return filepath.Join(s.Drydock, "litellm.config.yaml") }

// SessionDir is the host telemetry dir for a (ticket, session) under a space.
func SessionDir(space, ticket, session string) string {
	return filepath.Join(StateHome(), "sessions", space, ticket, session)
}
