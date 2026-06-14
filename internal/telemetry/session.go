package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Session binds a Writer to a session directory and stamps space/ticket/id onto
// every event, so phases and tool calls land in one events.jsonl that
// `retrospect` can analyze.
type Session struct {
	ID     string
	Space  string
	Ticket string
	Dir    string
	w      *Writer
}

// StartSession creates the session dir, writes session.json (including any
// extra metadata such as resolved versions, for reproducibility), and records a
// phase_start marker. The host session dir is the input to `retrospect`.
func StartSession(dir, space, ticket, id string, extra map[string]string) (*Session, error) {
	w, err := NewWriter(dir)
	if err != nil {
		return nil, err
	}
	s := &Session{ID: id, Space: space, Ticket: ticket, Dir: dir, w: w}
	meta := map[string]string{
		"session_id": id,
		"space":      space,
		"ticket":     ticket,
		"started_at": time.Now().UTC().Format(time.RFC3339),
	}
	for k, v := range extra {
		meta[k] = v
	}
	if b, err := json.MarshalIndent(meta, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(dir, "session.json"), b, 0o644)
	}
	ev := map[string]string{"note": "session start"}
	for k, v := range extra {
		ev[k] = v
	}
	return s, s.Log(Event{Type: PhaseStart, Meta: ev})
}

// Log records an event, filling in the session's identifiers.
func (s *Session) Log(e Event) error {
	e.SessionID, e.Space, e.Ticket = s.ID, s.Space, s.Ticket
	return s.w.Append(e)
}
