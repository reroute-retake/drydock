// Package lockfile records the resolved versions for a space so a build is
// reproducible and auditable (design P9). It captures what is configured at
// build time (dock + ForgeCode versions, base image, MCP pins, stacks); it does
// not reach the network to resolve digests — pin those in the manifest.
package lockfile

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/reroute-retake/drydock/internal/config"
)

// Lock is the content of .drydock/lock.yaml.
type Lock struct {
	GeneratedAt  time.Time         `yaml:"generated_at"`
	DockVersion  string            `yaml:"dock_version"`
	ImageBase    string            `yaml:"image_base"`
	ForgeVersion string            `yaml:"forge_version"`
	Stacks       []string          `yaml:"stacks,omitempty"`
	MCP          map[string]string `yaml:"mcp,omitempty"` // server name -> pin ("unpinned" if none)
}

// Build derives a Lock from a manifest and the current dock version.
func Build(m *config.Manifest, dockVersion string) Lock {
	fv := m.Forge.Version
	if fv == "" {
		fv = "latest"
	}
	var mcp map[string]string
	if len(m.MCP) > 0 {
		mcp = make(map[string]string, len(m.MCP))
		for _, s := range m.MCP {
			if s.Pin != "" {
				mcp[s.Name] = s.Pin
			} else {
				mcp[s.Name] = "unpinned"
			}
		}
	}
	return Lock{
		GeneratedAt:  time.Now().UTC(),
		DockVersion:  dockVersion,
		ImageBase:    m.Image.Base,
		ForgeVersion: fv,
		Stacks:       m.Image.Stacks,
		MCP:          mcp,
	}
}

// Save writes the lock as YAML.
func (l Lock) Save(path string) error {
	b, err := yaml.Marshal(l)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
