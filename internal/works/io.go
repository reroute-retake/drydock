package works

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// defaultGate returns the human-approval gate policy for a phase (design 10).
func defaultGate(p Phase) GateKind {
	switch p {
	case Analyze, Grill, Plan, Ship:
		return GateHuman
	default:
		return GateNone
	}
}

// New builds a fresh ticket state: every phase pending, analyze current.
func New(space, ticket string) *State {
	ph := make(map[Phase]PhaseState, len(Order))
	for _, p := range Order {
		ph[p] = PhaseState{Phase: p, Status: StatusPending, Gate: defaultGate(p)}
	}
	return &State{Ticket: ticket, Space: space, Current: Analyze, TaskBackend: "prose", Phases: ph}
}

// Load reads a state.yaml.
func Load(path string) (*State, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s State
	if err := yaml.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if s.Phases == nil {
		s.Phases = map[Phase]PhaseState{}
	}
	return &s, nil
}

// Save writes a state.yaml.
func (s *State) Save(path string) error {
	b, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

const ticketTemplate = "# %s\n\n> Describe the task here. This file is the input to the `analyze` phase.\n"

// Scaffold ensures the works/<ticket>/ layout (dir, docs/, artifacts/, ticket.md,
// state.yaml) and returns its state. It never overwrites an existing ticket.md
// or state.yaml (idempotent).
func Scaffold(dir, space, ticket string) (*State, error) {
	for _, d := range []string{dir, filepath.Join(dir, "docs"), filepath.Join(dir, "artifacts")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	if tk := filepath.Join(dir, "ticket.md"); !exists(tk) {
		if err := os.WriteFile(tk, []byte(fmt.Sprintf(ticketTemplate, ticket)), 0o644); err != nil {
			return nil, err
		}
	}
	statePath := filepath.Join(dir, "state.yaml")
	if !exists(statePath) {
		st := New(space, ticket)
		if err := st.Save(statePath); err != nil {
			return nil, err
		}
		return st, nil
	}
	return Load(statePath)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Mark records a phase's status (plus optional model/note/artifacts). Starting a
// phase (in_progress) is gated: its predecessor must be satisfied (constraint
// C1). Skipping requires a justification note (design R3/D7).
func (s *State) Mark(p Phase, status, model, note string, artifacts ...string) error {
	ps, ok := s.Phases[p]
	if !ok {
		return fmt.Errorf("unknown phase %q", p)
	}
	if status == StatusInProgress && !s.CanEnter(p) {
		return fmt.Errorf("cannot start %q: a prior phase is not complete (gate C1)", p)
	}
	if status == StatusSkipped && note == "" {
		return fmt.Errorf("skipping %q requires a --note justification", p)
	}
	ps.Status = status
	if model != "" {
		ps.Model = model
	}
	if note != "" {
		ps.Note = note
	}
	ps.Artifacts = addUniq(ps.Artifacts, artifacts...)
	s.Phases[p] = ps
	if status == StatusInProgress {
		s.Current = p
	}
	return nil
}

func addUniq(s []string, vs ...string) []string {
	for _, v := range vs {
		if v == "" {
			continue
		}
		found := false
		for _, x := range s {
			if x == v {
				found = true
				break
			}
		}
		if !found {
			s = append(s, v)
		}
	}
	return s
}
