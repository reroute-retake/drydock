package skillcheck

import (
	"strings"
	"testing"
)

const valid = `---
name: analyze
description: >-
  Brainstorms and researches a new ticket into a written analysis. Use when
  starting a ticket; do NOT use to verify assumptions (grill) or plan tasks.
metadata:
  version: "0.2"
---
# Analyze
body
`

func TestParse(t *testing.T) {
	d, err := Parse(valid)
	if err != nil {
		t.Fatal(err)
	}
	if d.Name != "analyze" || d.Version != "0.2" {
		t.Fatalf("doc=%+v", d)
	}
	if d.BodyLines == 0 {
		t.Fatal("body should have lines")
	}
}

func TestParseMissingFrontmatter(t *testing.T) {
	if _, err := Parse("# no frontmatter\n"); err == nil {
		t.Fatal("expected error")
	}
}

func TestCheckCleanWithBump(t *testing.T) {
	cur := &SkillDoc{Name: "analyze", Version: "0.1"}
	is := Check(valid, "analyze", cur)
	if HasErrors(is) {
		t.Fatalf("expected no errors, got %+v", is)
	}
}

func TestCheckNameMismatch(t *testing.T) {
	is := Check(valid, "grill", nil)
	if !HasErrors(is) {
		t.Fatal("expected name-mismatch error")
	}
}

func TestCheckVersionNotBumped(t *testing.T) {
	cur := &SkillDoc{Name: "analyze", Version: "0.2"}
	if !HasErrors(Check(valid, "analyze", cur)) {
		t.Fatal("expected version-not-bumped error (0.2 vs 0.2)")
	}
	cur = &SkillDoc{Name: "analyze", Version: "0.3"}
	if !HasErrors(Check(valid, "analyze", cur)) {
		t.Fatal("expected error (0.2 < 0.3)")
	}
}

func TestCheckMissingVersion(t *testing.T) {
	noVer := "---\nname: analyze\ndescription: " + strings.Repeat("x", 60) + "\n---\nbody\n"
	if !HasErrors(Check(noVer, "analyze", nil)) {
		t.Fatal("expected missing-version error")
	}
}

func TestCheckBodyTooLong(t *testing.T) {
	big := "---\nname: analyze\ndescription: " + strings.Repeat("x", 60) + "\nmetadata:\n  version: \"0.2\"\n---\n" + strings.Repeat("line\n", 501)
	if !HasErrors(Check(big, "analyze", nil)) {
		t.Fatal("expected body-too-long error")
	}
}

func TestVersionGreater(t *testing.T) {
	if !versionGreater("0.2", "0.1") || !versionGreater("1.0", "0.9") || versionGreater("0.2", "0.2") {
		t.Fatal("versionGreater logic wrong")
	}
}
