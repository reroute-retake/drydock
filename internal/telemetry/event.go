// Package telemetry defines the drydock session event schema and a JSONL writer.
//
// Events are the machine-readable record of a session that the `retrospect`
// skill consumes. The normalized model/token/cost fields are intended to be
// populated from the LiteLLM gateway's per-call logging (see design doc 7.4),
// so comparisons across models and providers are apples-to-apples.
//
// Invariant: secrets must NEVER be written into any event field (constraint C8).
package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// EventType enumerates the kinds of events recorded in events.jsonl.
type EventType string

const (
	PhaseStart EventType = "phase_start"
	PhaseEnd   EventType = "phase_end"
	ToolCall   EventType = "tool_call"
	Gate       EventType = "gate"
	Failure    EventType = "failure"
	LLMCall    EventType = "llm_call"
)

// Event is one line in events.jsonl.
type Event struct {
	TS           time.Time         `json:"ts"`
	SessionID    string            `json:"session_id"`
	Space        string            `json:"space"`
	Ticket       string            `json:"ticket"`
	Phase        string            `json:"phase,omitempty"`
	Skill        string            `json:"skill,omitempty"`
	Type         EventType         `json:"event_type"`
	Tool         string            `json:"tool,omitempty"`
	DurationMS   int64             `json:"duration_ms,omitempty"`
	Status       string            `json:"status,omitempty"` // ok | error | blocked
	Error        string            `json:"error,omitempty"`
	Model        string            `json:"model,omitempty"`
	ModelVersion string            `json:"model_version,omitempty"`
	TokensIn     int               `json:"tokens_in,omitempty"`
	TokensOut    int               `json:"tokens_out,omitempty"`
	CostUSD      float64           `json:"cost_usd,omitempty"`
	Meta         map[string]string `json:"meta,omitempty"`
}

// Writer appends events to <sessionDir>/events.jsonl, safe for concurrent use.
type Writer struct {
	mu   sync.Mutex
	path string
}

// NewWriter creates (if needed) the session directory and returns a Writer.
func NewWriter(sessionDir string) (*Writer, error) {
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return nil, err
	}
	return &Writer{path: filepath.Join(sessionDir, "events.jsonl")}, nil
}

// Append writes one event as a single JSON line. TS is stamped if unset.
func (w *Writer) Append(e Event) error {
	if e.TS.IsZero() {
		e.TS = time.Now().UTC()
	}
	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	f, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}
