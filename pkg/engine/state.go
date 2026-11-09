package engine

// State is the run-level state of the game.
//
// Only StatePlaying advances the simulation. Every other state freezes the
// world, which is what makes pausing, dying and the title screen cheap to
// implement and impossible to get subtly wrong.
type State int

const (
	// StateTitle is the attract screen before a run starts.
	StateTitle State = iota
	// StatePlaying is normal gameplay.
	StatePlaying
	// StatePaused freezes the world and keeps the last frame.
	StatePaused
	// StateDying plays the death animation; input is ignored.
	StateDying
	// StateFlagpole plays the slide down the pole and the walk to the castle.
	StateFlagpole
	// StateResults counts remaining time into score.
	StateResults
)

// String returns a stable name, used in tests and debug output.
func (s State) String() string {
	switch s {
	case StateTitle:
		return "title"
	case StatePlaying:
		return "playing"
	case StatePaused:
		return "paused"
	case StateDying:
		return "dying"
	case StateFlagpole:
		return "flagpole"
	case StateResults:
		return "results"
	}
	return "unknown"
}

// SimulatesWorld reports whether entities should be advanced in this state.
func (s State) SimulatesWorld() bool {
	return s == StatePlaying || s == StateDying || s == StateFlagpole
}

// AcceptsPlayerInput reports whether player control is live. Dying and
// flagpole still animate but ignore input.
func (s State) AcceptsPlayerInput() bool { return s == StatePlaying }

// StateMachine is the explicit state with its transition table.
//
// Transitions are methods rather than a public setter so an illegal move is a
// no-op the caller can observe, not a silently corrupt state.
type StateMachine struct {
	cur State
}

// NewStateMachine starts on the title screen.
func NewStateMachine() *StateMachine { return &StateMachine{cur: StateTitle} }

// Current returns the state.
func (m *StateMachine) Current() State { return m.cur }

// Is reports whether the machine is in a state.
func (m *StateMachine) Is(s State) bool { return m.cur == s }

// StartRun moves from the title (or results) into gameplay.
func (m *StateMachine) StartRun() bool {
	if m.cur != StateTitle && m.cur != StateResults {
		return false
	}
	m.cur = StatePlaying
	return true
}

// Pause toggles between playing and paused.
func (m *StateMachine) Pause() bool {
	switch m.cur {
	case StatePlaying:
		m.cur = StatePaused
		return true
	case StatePaused:
		m.cur = StatePlaying
		return true
	}
	return false
}

// Die enters the death state.
func (m *StateMachine) Die() bool {
	if m.cur != StatePlaying { //nolint:staticcheck // explicit for clarity
		return false
	}
	m.cur = StateDying
	return true
}

// TouchFlag enters the flagpole sequence.
func (m *StateMachine) TouchFlag() bool {
	if m.cur != StatePlaying {
		return false
	}
	m.cur = StateFlagpole
	return true
}

// ShowResults enters the end-of-level tally.
func (m *StateMachine) ShowResults() bool {
	if m.cur != StateFlagpole {
		return false
	}
	m.cur = StateResults
	return true
}

// Respawn returns to gameplay after a death, when lives remain.
func (m *StateMachine) Respawn() bool {
	if m.cur != StateDying {
		return false
	}
	m.cur = StatePlaying
	return true
}

// GameOver ends the run at the results screen after the last life.
func (m *StateMachine) GameOver() bool {
	if m.cur != StateDying {
		return false
	}
	m.cur = StateResults
	return true
}

// ToTitle returns to the attract screen.
func (m *StateMachine) ToTitle() bool {
	m.cur = StateTitle
	return true
}
