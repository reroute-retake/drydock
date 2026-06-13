package works

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewHasAllPhases(t *testing.T) {
	s := New("payments", "PAY-1")
	if len(s.Phases) != len(Order) {
		t.Fatalf("phases=%d want %d", len(s.Phases), len(Order))
	}
	if s.Phases[Analyze].Gate != GateHuman || s.Phases[Develop].Gate != GateNone {
		t.Fatalf("gate defaults wrong")
	}
	if s.Current != Analyze {
		t.Fatalf("current=%q", s.Current)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.yaml")
	in := New("sp", "T-1")
	_ = in.Mark(Analyze, StatusInProgress, "dock/reason", "")
	_ = in.Mark(Analyze, StatusAgreed, "dock/reason", "", "01-analysis.md")
	if err := in.Save(p); err != nil {
		t.Fatal(err)
	}
	out, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if out.Phases[Analyze].Status != StatusAgreed || out.Phases[Analyze].Model != "dock/reason" {
		t.Fatalf("analyze=%+v", out.Phases[Analyze])
	}
	if got := out.Phases[Analyze].Artifacts; len(got) != 1 || got[0] != "01-analysis.md" {
		t.Fatalf("artifacts=%v", got)
	}
}

func TestScaffoldIdempotent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "PAY-1")
	if _, err := Scaffold(dir, "payments", "PAY-1"); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"ticket.md", "state.yaml", "docs", "artifacts"} {
		if !exists(filepath.Join(dir, f)) {
			t.Fatalf("missing %s", f)
		}
	}
	// Edit ticket.md, re-scaffold, ensure it's preserved.
	tk := filepath.Join(dir, "ticket.md")
	_ = os.WriteFile(tk, []byte("# PAY-1\n\nreal task"), 0o644)
	if _, err := Scaffold(dir, "payments", "PAY-1"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(tk)
	if string(b) != "# PAY-1\n\nreal task" {
		t.Fatalf("ticket.md was clobbered: %q", b)
	}
}

func TestMarkGatingAndSkip(t *testing.T) {
	s := New("sp", "T-1")
	// Cannot start develop before analyze/grill/plan are done.
	if err := s.Mark(Develop, StatusInProgress, "", ""); err == nil {
		t.Fatal("expected gate error starting develop early")
	}
	// Skipping needs a justification.
	if err := s.Mark(Grill, StatusSkipped, "", ""); err == nil {
		t.Fatal("expected error skipping without a note")
	}
	if err := s.Mark(Grill, StatusSkipped, "", "no external assumptions"); err != nil {
		t.Fatalf("skip with note should work: %v", err)
	}
	// analyze agreed + grill skipped => plan may start.
	_ = s.Mark(Analyze, StatusAgreed, "", "")
	if err := s.Mark(Plan, StatusInProgress, "", ""); err != nil {
		t.Fatalf("plan should be enterable: %v", err)
	}
	if s.Current != Plan {
		t.Fatalf("current=%q want plan", s.Current)
	}
}
