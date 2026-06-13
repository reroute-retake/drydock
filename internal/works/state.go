// Package works models the per-ticket artifact contract and lifecycle state
// machine. The works/<ticket>/ folder IS the state; state.yaml is its index.
//
// The ordered phase sequence is the engineer-facing interface (design doc P1);
// the implementation behind each phase may change without changing this order.
// Transitions are gated (constraint C1) and the graph allows back-edges
// (re-entering an earlier phase, e.g. review -> develop) and skip-edges (the
// lightweight path) with recorded justification.
package works

// Phase is a single lifecycle phase.
type Phase string

const (
	Analyze Phase = "analyze"
	Grill   Phase = "grill"
	Plan    Phase = "plan"
	Develop Phase = "develop"
	Review  Phase = "review"
	Triage  Phase = "triage"
	Test    Phase = "test"
	Ship    Phase = "ship"
	Archive Phase = "archive"
)

// Order is the canonical happy-path sequence.
var Order = []Phase{Analyze, Grill, Plan, Develop, Review, Triage, Test, Ship, Archive}

// GateKind describes whether advancing past a phase needs human approval.
type GateKind string

const (
	GateNone  GateKind = "none"
	GateHuman GateKind = "human"
)

// Status values a phase may hold.
const (
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusAgreed     = "agreed" // analyze/grill/plan reach "agreed" at their gate
	StatusDone       = "done"
	StatusSkipped    = "skipped"
)

// PhaseState records the status of a single phase within a ticket.
type PhaseState struct {
	Phase        Phase    `json:"phase" yaml:"phase"`
	Status       string   `json:"status" yaml:"status"`
	Gate         GateKind `json:"gate" yaml:"gate"`
	Model        string   `json:"model,omitempty" yaml:"model,omitempty"`
	ModelVersion string   `json:"model_version,omitempty" yaml:"model_version,omitempty"`
	Artifacts    []string `json:"artifacts,omitempty" yaml:"artifacts,omitempty"`
	Note         string   `json:"note,omitempty" yaml:"note,omitempty"` // e.g. justification when skipped
}

// State is the content of works/<ticket>/state.yaml.
type State struct {
	Ticket      string               `json:"ticket" yaml:"ticket"`
	Space       string               `json:"space" yaml:"space"`
	Current     Phase                `json:"current" yaml:"current"`
	Lightweight bool                 `json:"lightweight,omitempty" yaml:"lightweight,omitempty"`
	TaskBackend string               `json:"task_backend,omitempty" yaml:"task_backend,omitempty"` // prose | beads
	Phases      map[Phase]PhaseState `json:"phases" yaml:"phases"`
}

func index(p Phase) int {
	for i, q := range Order {
		if q == p {
			return i
		}
	}
	return -1
}

// satisfied reports whether a phase status counts as "complete enough" to let
// the next phase begin.
func satisfied(status string) bool {
	switch status {
	case StatusDone, StatusAgreed, StatusSkipped:
		return true
	default:
		return false
	}
}

// CanEnter reports whether phase p may start given the current state: its
// immediate predecessor in Order must be satisfied (C1). The first phase, and
// re-entry of an already-recorded earlier phase (a back-edge), are allowed.
func (s *State) CanEnter(p Phase) bool {
	i := index(p)
	if i <= 0 {
		return true
	}
	if _, seen := s.Phases[p]; seen {
		return true // back-edge into a phase we've already touched
	}
	prev, ok := s.Phases[Order[i-1]]
	return ok && satisfied(prev.Status)
}

// Next returns the phase following Current in Order, or "" at the end.
func (s *State) Next() Phase {
	i := index(s.Current)
	if i < 0 || i+1 >= len(Order) {
		return ""
	}
	return Order[i+1]
}
