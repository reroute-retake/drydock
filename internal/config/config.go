// Package config defines the drydock space manifest (space.yaml) and its
// loader/validator. The manifest is the single source of truth for a space
// (design doc P2). Secrets live only in .env, never here (P12/C8).
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Scalar accepts a YAML string OR number and stores its textual form, so
// stack versions like `21` and `1.x` both parse cleanly.
type Scalar string

// UnmarshalYAML captures the raw scalar text regardless of its YAML tag.
func (s *Scalar) UnmarshalYAML(n *yaml.Node) error {
	*s = Scalar(n.Value)
	return nil
}

// Stack describes a repo's tech stack (drives per-stack image layers).
type Stack struct {
	Lang    string `yaml:"lang,omitempty"`
	Version Scalar `yaml:"version,omitempty"`
	Build   string `yaml:"build,omitempty"`
}

// Vault is the space's git-backed knowledge layer.
type Vault struct {
	Repo   string `yaml:"repo"`
	Branch string `yaml:"branch"`
}

// Repo is one source repository in the space.
type Repo struct {
	URL   string `yaml:"url"`
	Stack Stack  `yaml:"stack,omitempty"`
}

// Image is the container image strategy (base + per-stack layers).
type Image struct {
	Base   string   `yaml:"base"`
	Stacks []string `yaml:"stacks"`
}

// MCP is a space-scoped MCP server entry.
type MCP struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type,omitempty"`
	Toolsets []string `yaml:"toolsets,omitempty"`
	Pin      string   `yaml:"pin,omitempty"`
}

// Model maps a role alias to a concrete provider/model (resolved by the gateway).
type Model struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	APIBase  string `yaml:"api_base,omitempty"` // set for local/self-hosted models
}

// Tooling captures per-space, opt-in workflow tooling (design doc 7B).
type Tooling struct {
	Tasks     string   `yaml:"tasks,omitempty"` // prose | beads
	Precommit bool     `yaml:"precommit,omitempty"`
	Scanners  []string `yaml:"scanners,omitempty"`
}

// Manifest is the parsed space.yaml.
type Manifest struct {
	Space         string            `yaml:"space"`
	Vault         Vault             `yaml:"vault"`
	Repos         []Repo            `yaml:"repos"`
	Image         Image             `yaml:"image"`
	Ports         []int             `yaml:"ports,omitempty"`
	GatewayPort   int               `yaml:"gateway_port,omitempty"` // host port for the gateway; 0 = derive per-space
	MCP           []MCP             `yaml:"mcp,omitempty"`
	Models        map[string]Model  `yaml:"models"`
	Routing       map[string]string `yaml:"routing"`
	Tooling       Tooling           `yaml:"tooling,omitempty"`
	PrinciplesRef string            `yaml:"principles_ref,omitempty"`
}

// Load reads and validates a space.yaml.
func Load(path string) (*Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// Validate checks the manifest's internal consistency.
func (m *Manifest) Validate() error {
	var errs []string
	if m.Space == "" {
		errs = append(errs, "space is required")
	}
	if m.Image.Base == "" {
		errs = append(errs, "image.base is required")
	}
	if len(m.Models) == 0 {
		errs = append(errs, "at least one model is required")
	}
	if len(m.Routing) == 0 {
		errs = append(errs, "routing is required")
	}
	for phase, role := range m.Routing {
		if _, ok := m.Models[role]; !ok {
			errs = append(errs, fmt.Sprintf("routing[%s] -> unknown model role %q", phase, role))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("invalid manifest: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ModelForPhase resolves a lifecycle phase to its role and model via routing.
func (m *Manifest) ModelForPhase(phase string) (role string, model Model, ok bool) {
	role, ok = m.Routing[phase]
	if !ok {
		return "", Model{}, false
	}
	model, ok = m.Models[role]
	return role, model, ok
}

// Save marshals the manifest back to YAML at path. Note: this normalizes the
// file (comments in a hand-edited space.yaml are not preserved).
func (m *Manifest) Save(path string) error {
	b, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Template returns a starter space.yaml for `dock setup <space>`.
func Template(space string) []byte {
	return []byte(fmt.Sprintf(starter, space))
}

const starter = `space: %s
vault:
  repo: ""          # git URL of this space's vault (llm-wiki) repo
  branch: main
repos: []            # add with: dock addrepo <git-url>
image:
  base: debian:12-slim
  stacks: []
ports: []            # dev-server ports published at 'dock start'
mcp:
  - { name: github, type: command }
  - { name: context7, type: url }
  - { name: sequential-thinking, type: command }
models:               # roles resolved by the LiteLLM gateway; secrets in .env
  reason: { provider: anthropic, model: REPLACE_ME }
  code:   { provider: anthropic, model: REPLACE_ME }
  cheap:  { provider: openai,    model: REPLACE_ME }
routing:              # review intentionally != develop model (R6)
  analyze: reason
  grill: reason
  plan: reason
  develop: code
  review: reason
  triage: cheap
  test: code
  ship: cheap
  archive: cheap
tooling:
  tasks: prose        # or: beads
`
