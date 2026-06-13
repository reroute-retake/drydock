package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEventRoundTrip(t *testing.T) {
	in := Event{
		TS:        time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC),
		SessionID: "s-1",
		Space:     "payments",
		Ticket:    "PAY-123",
		Phase:     "develop",
		Skill:     "develop",
		Type:      LLMCall,
		Model:     "dock/code",
		TokensIn:  100,
		TokensOut: 50,
		CostUSD:   0.0123,
		Meta:      map[string]string{"task": "bd-a1b2"},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out Event
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Ticket != in.Ticket || out.Type != in.Type || out.TokensIn != 100 {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}

func TestWriterAppend(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(dir)
	if err != nil {
		t.Fatalf("new writer: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := w.Append(Event{SessionID: "s", Space: "sp", Ticket: "T-1", Type: ToolCall}); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	data, err := os.ReadFile(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	lines := 0
	for _, c := range data {
		if c == '\n' {
			lines++
		}
	}
	if lines != 3 {
		t.Fatalf("expected 3 JSONL lines, got %d", lines)
	}
}
