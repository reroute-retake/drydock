package telemetry

import (
	"path/filepath"
	"testing"
)

func TestReadEventsAndSummarize(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(dir)
	if err != nil {
		t.Fatal(err)
	}
	_ = w.Append(Event{Phase: "develop", Type: ToolCall})
	_ = w.Append(Event{Phase: "develop", Type: LLMCall, TokensIn: 100, TokensOut: 40, CostUSD: 0.01})
	_ = w.Append(Event{Phase: "review", Type: Failure})

	evs, err := ReadEvents(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 3 {
		t.Fatalf("events=%d", len(evs))
	}
	s := Summarize(evs)
	if s.Events != 3 || s.Failures != 1 {
		t.Fatalf("summary totals: %+v", s)
	}
	if s.ByPhase["develop"].ToolCalls != 1 || s.ByPhase["develop"].LLMCalls != 1 {
		t.Fatalf("develop stats: %+v", s.ByPhase["develop"])
	}
	if s.ByPhase["develop"].TokensIn != 100 || s.ByPhase["review"].Failures != 1 {
		t.Fatalf("by-phase wrong: %+v", s.ByPhase)
	}
}

func TestFindEventFiles(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "PAY-1", "sess1")
	w, _ := NewWriter(sub)
	_ = w.Append(Event{Type: PhaseStart})
	got := FindEventFiles(root)
	if len(got) != 1 || filepath.Base(got[0]) != "events.jsonl" {
		t.Fatalf("find=%v", got)
	}
}
