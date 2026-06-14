// Package skillcheck validates a SKILL.md before it is applied, so the
// self-improvement loop can't install a malformed or un-bumped skill (the eval
// gate in front of `dock skill apply`). Checks follow the §10A authoring rules.
package skillcheck

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Levels for an Issue.
const (
	LevelError = "error"
	LevelWarn  = "warn"
)

// Issue is a single validation finding.
type Issue struct {
	Level string
	Msg   string
}

// SkillDoc is the parsed, checkable shape of a SKILL.md.
type SkillDoc struct {
	Name        string
	Description string
	Version     string
	BodyLines   int
}

var kebab = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Parse splits a SKILL.md into frontmatter fields and a body line count.
func Parse(content string) (SkillDoc, error) {
	s := strings.TrimLeft(content, " \t\r\n")
	if !strings.HasPrefix(s, "---") {
		return SkillDoc{}, errors.New("missing YAML frontmatter (--- ... ---)")
	}
	lines := strings.Split(s, "\n")
	var fm, body []string
	closed := false
	for i := 1; i < len(lines); i++ {
		ln := strings.TrimRight(lines[i], "\r")
		if !closed && ln == "---" {
			closed = true
			continue
		}
		if closed {
			body = append(body, lines[i])
		} else {
			fm = append(fm, lines[i])
		}
	}
	if !closed {
		return SkillDoc{}, errors.New("unterminated frontmatter (no closing ---)")
	}
	var f struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Metadata    struct {
			Version string `yaml:"version"`
		} `yaml:"metadata"`
	}
	if err := yaml.Unmarshal([]byte(strings.Join(fm, "\n")), &f); err != nil {
		return SkillDoc{}, fmt.Errorf("invalid frontmatter YAML: %w", err)
	}
	n := len(body)
	for n > 0 && strings.TrimSpace(body[n-1]) == "" {
		n--
	}
	return SkillDoc{Name: f.Name, Description: f.Description, Version: f.Metadata.Version, BodyLines: n}, nil
}

// Check validates a SKILL.md against the §10A rules. dirName is the skill's
// directory name (the canonical skill name); current is the currently-installed
// skill doc (or nil if this is a new skill) — used to require a version bump.
func Check(content, dirName string, current *SkillDoc) []Issue {
	doc, err := Parse(content)
	if err != nil {
		return []Issue{{LevelError, err.Error()}}
	}
	var is []Issue

	switch {
	case doc.Name == "":
		is = append(is, Issue{LevelError, "name is required"})
	case !kebab.MatchString(doc.Name):
		is = append(is, Issue{LevelError, fmt.Sprintf("name %q must be kebab-case ([a-z0-9-])", doc.Name)})
	case dirName != "" && doc.Name != dirName:
		is = append(is, Issue{LevelError, fmt.Sprintf("name %q must match the skill directory %q", doc.Name, dirName)})
	}

	switch {
	case doc.Description == "":
		is = append(is, Issue{LevelError, "description is required"})
	case len(doc.Description) > 1024:
		is = append(is, Issue{LevelError, fmt.Sprintf("description is %d chars (max 1024)", len(doc.Description))})
	case len(doc.Description) < 30:
		is = append(is, Issue{LevelWarn, "description is very short — state what it does, when to trigger, and a negative trigger"})
	case len(doc.Description) > 400:
		is = append(is, Issue{LevelWarn, fmt.Sprintf("description is %d chars; ~100-200 is ideal", len(doc.Description))})
	}

	if doc.BodyLines > 500 {
		is = append(is, Issue{LevelError, fmt.Sprintf("body is %d lines (max 500 — move detail to references/)", doc.BodyLines)})
	}

	if doc.Version == "" {
		is = append(is, Issue{LevelError, "metadata.version is required"})
	} else if current != nil && current.Version != "" && !versionGreater(doc.Version, current.Version) {
		is = append(is, Issue{LevelError, fmt.Sprintf("metadata.version %q must be greater than the current %q", doc.Version, current.Version)})
	}

	return is
}

// HasErrors reports whether any issue is an error (vs. a warning).
func HasErrors(issues []Issue) bool {
	for _, i := range issues {
		if i.Level == LevelError {
			return true
		}
	}
	return false
}

func versionGreater(a, b string) bool {
	pa, oka := parseVer(a)
	pb, okb := parseVer(b)
	if !oka || !okb {
		return a != b // unparseable: require at least a change
	}
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		if x != y {
			return x > y
		}
	}
	return false
}

func parseVer(s string) ([]int, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if s == "" {
		return nil, false
	}
	parts := strings.Split(s, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, false
		}
		out = append(out, n)
	}
	return out, true
}
