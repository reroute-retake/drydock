package telemetry

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// ReadEvents reads an events.jsonl file into a slice of Events. Malformed lines
// are skipped so a partial file still yields usable data.
func ReadEvents(path string) ([]Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var evs []Event
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		b := sc.Bytes()
		if len(b) == 0 {
			continue
		}
		var e Event
		if json.Unmarshal(b, &e) == nil {
			evs = append(evs, e)
		}
	}
	return evs, sc.Err()
}

// FindEventFiles returns the sorted paths of every events.jsonl under root.
func FindEventFiles(root string) []string {
	var out []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && d.Name() == "events.jsonl" {
			out = append(out, p)
		}
		return nil
	})
	sort.Strings(out)
	return out
}

// PhaseStat aggregates telemetry for a single lifecycle phase.
type PhaseStat struct {
	Phase     string  `json:"phase"`
	Events    int     `json:"events"`
	Failures  int     `json:"failures"`
	ToolCalls int     `json:"tool_calls"`
	LLMCalls  int     `json:"llm_calls"`
	TokensIn  int     `json:"tokens_in"`
	TokensOut int     `json:"tokens_out"`
	CostUSD   float64 `json:"cost_usd"`
}

// Summary is an aggregate over a set of events — the input shape `retrospect`
// reasons about.
type Summary struct {
	Sessions  int                   `json:"sessions"`
	Events    int                   `json:"events"`
	Failures  int                   `json:"failures"`
	TokensIn  int                   `json:"tokens_in"`
	TokensOut int                   `json:"tokens_out"`
	CostUSD   float64               `json:"cost_usd"`
	ByPhase   map[string]*PhaseStat `json:"by_phase"`
}

// Summarize rolls up events into a Summary (overall + per-phase).
func Summarize(evs []Event) Summary {
	s := Summary{ByPhase: map[string]*PhaseStat{}}
	for _, e := range evs {
		s.Events++
		ph := e.Phase
		if ph == "" {
			ph = "(none)"
		}
		st := s.ByPhase[ph]
		if st == nil {
			st = &PhaseStat{Phase: ph}
			s.ByPhase[ph] = st
		}
		st.Events++
		switch e.Type {
		case Failure:
			s.Failures++
			st.Failures++
		case ToolCall:
			st.ToolCalls++
		case LLMCall:
			st.LLMCalls++
		}
		s.TokensIn += e.TokensIn
		s.TokensOut += e.TokensOut
		s.CostUSD += e.CostUSD
		st.TokensIn += e.TokensIn
		st.TokensOut += e.TokensOut
		st.CostUSD += e.CostUSD
	}
	return s
}
