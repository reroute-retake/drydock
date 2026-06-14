// Package proposal stores human-gated, versioned skill-change proposals emitted
// by the retrospect/hindsight skills. Nothing is ever auto-applied (principle
// P6, constraint C9): a proposal is created as "pending"; a human applies it by
// running `dock skill apply`, which writes the proposed SKILL.md into place.
package proposal

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	metaFile    = "proposal.yaml"
	contentFile = "SKILL.md"

	StatusPending  = "pending"
	StatusApplied  = "applied"
	StatusRejected = "rejected"
)

// Proposal is a proposed change to a skill's SKILL.md.
type Proposal struct {
	ID        string    `yaml:"id"`
	Skill     string    `yaml:"skill"`
	Source    string    `yaml:"source"` // retrospect | hindsight
	Rationale string    `yaml:"rationale"`
	Status    string    `yaml:"status"`
	CreatedAt time.Time `yaml:"created_at"`
}

// Create writes a new pending proposal (metadata + the proposed SKILL.md body)
// under dir/<id>/ and returns it.
func Create(dir, skill, source, rationale, content string) (Proposal, error) {
	if skill == "" || content == "" {
		return Proposal{}, fmt.Errorf("proposal needs a skill and content")
	}
	id := time.Now().UTC().Format("20060102T150405Z") + "-" + skill
	pdir := filepath.Join(dir, id)
	if err := os.MkdirAll(pdir, 0o755); err != nil {
		return Proposal{}, err
	}
	p := Proposal{ID: id, Skill: skill, Source: source, Rationale: rationale, Status: StatusPending, CreatedAt: time.Now().UTC()}
	if err := writeMeta(pdir, p); err != nil {
		return Proposal{}, err
	}
	if err := os.WriteFile(filepath.Join(pdir, contentFile), []byte(content), 0o644); err != nil {
		return Proposal{}, err
	}
	return p, nil
}

func writeMeta(pdir string, p Proposal) error {
	b, err := yaml.Marshal(p)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(pdir, metaFile), b, 0o644)
}

// List returns all proposals under dir, oldest first.
func List(dir string) ([]Proposal, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ps []Proposal
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		if p, _, err := Load(dir, e.Name()); err == nil {
			ps = append(ps, p)
		}
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].ID < ps[j].ID })
	return ps, nil
}

// Load returns a proposal's metadata and its proposed SKILL.md content.
func Load(dir, id string) (Proposal, string, error) {
	pdir := filepath.Join(dir, id)
	b, err := os.ReadFile(filepath.Join(pdir, metaFile))
	if err != nil {
		return Proposal{}, "", err
	}
	var p Proposal
	if err := yaml.Unmarshal(b, &p); err != nil {
		return Proposal{}, "", err
	}
	c, err := os.ReadFile(filepath.Join(pdir, contentFile))
	if err != nil {
		return Proposal{}, "", err
	}
	return p, string(c), nil
}

// Apply writes the proposed content to targetSkillFile and marks the proposal
// applied. Running this is the human approval gate.
func Apply(dir, id, targetSkillFile string) error {
	p, content, err := Load(dir, id)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(targetSkillFile), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(targetSkillFile, []byte(content), 0o644); err != nil {
		return err
	}
	p.Status = StatusApplied
	return writeMeta(filepath.Join(dir, id), p)
}

// SetStatus updates a proposal's status (e.g. to reject it).
func SetStatus(dir, id, status string) error {
	p, _, err := Load(dir, id)
	if err != nil {
		return err
	}
	p.Status = status
	return writeMeta(filepath.Join(dir, id), p)
}
