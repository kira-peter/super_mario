package engine

// Action is a game action, decoupled from physical keys.
//
// Design decision 9: the simulation never sees a key, only an action. That is
// what lets the same actions be bound to several keys and lets a terminal that
// swallows a modifier still be playable.
type Action int

const (
	ActionLeft Action = iota
	ActionRight
	ActionDown
	ActionJump
	ActionRun
	ActionPause
	ActionQuit
	ActionConfirm
	actionCount
)

// Key is a physical key identifier. The input layer maps terminal events onto
// these, so the mapping stays testable without a terminal.
type Key int

const (
	KeyUnknown Key = iota
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeySpace
	KeyShift
	KeyEnter
	KeyEscape
	KeyP
	KeyQ
	KeyX
	KeyZ
	KeyTab
	KeyQuestion
	KeyR
)

// Binding maps actions to the keys that trigger them.
//
// An action may have several keys; a key may trigger only one action per state.
type Binding struct {
	keys map[Action][]Key
}

// DefaultBinding returns the out-of-the-box key map.
//
// Run is bound to both Shift and X because several terminals and multiplexers
// swallow Shift combined with an arrow key.
func DefaultBinding() *Binding {
	return &Binding{keys: map[Action][]Key{
		ActionLeft:    {KeyLeft},
		ActionRight:   {KeyRight},
		ActionDown:    {KeyDown},
		ActionJump:    {KeySpace, KeyZ, KeyUp},
		ActionRun:     {KeyShift, KeyX},
		ActionPause:   {KeyP},
		ActionQuit:    {KeyQ, KeyEscape},
		ActionConfirm: {KeySpace, KeyEnter, KeyZ},
	}}
}

// Keys returns the keys bound to an action.
func (b *Binding) Keys(a Action) []Key { return b.keys[a] }

// Bind replaces the keys for an action. An empty list unbinds it.
func (b *Binding) Bind(a Action, keys ...Key) {
	if len(keys) == 0 {
		delete(b.keys, a)
		return
	}
	b.keys[a] = append([]Key(nil), keys...)
}

// ActionsFor returns every action bound to a key, in binding order.
//
// One key can legitimately serve two actions in different screens: Space is
// jump during play and confirm on the title screen. The caller picks the one
// that makes sense for the current state; refusing the ambiguity here would
// force two different keys for what the player expects to be one.
func (b *Binding) ActionsFor(k Key) []Action {
	var out []Action
	for a := Action(0); a < actionCount; a++ {
		for _, bound := range b.keys[a] {
			if bound == k {
				out = append(out, a)
				break
			}
		}
	}
	return out
}

// ActionFor returns the first action bound to a key, and whether one was found.
func (b *Binding) ActionFor(k Key) (Action, bool) {
	for a := Action(0); a < actionCount; a++ {
		for _, bound := range b.keys[a] {
			if bound == k {
				return a, true
			}
		}
	}
	return 0, false
}

// Pressed is one step's worth of input, frozen so every step inside a single
// rendered frame reads the same values.
//
// Design decision 9: collapsing the event queue into a bit set is what makes
// "two jump presses in one frame count as one" true by construction.
type Pressed struct {
	// Held is the continuous state, one bit per action.
	Held uint32
	// Edge is set on the step an action went down.
	Edge uint32
	// Released is set on the step an action went up.
	Released uint32
}

// HasHeld reports whether an action is currently held.
func (p Pressed) HasHeld(a Action) bool { return p.Held&(1<<uint(a)) != 0 }

// HasEdge reports whether an action was pressed on this step.
func (p Pressed) HasEdge(a Action) bool { return p.Edge&(1<<uint(a)) != 0 }

// HasReleased reports whether an action was released on this step.
func (p Pressed) HasReleased(a Action) bool { return p.Released&(1<<uint(a)) != 0 }

// Input is the event queue plus the sampler.
//
// Callers push terminal events in arrival order; the simulation calls Sample
// once per step and gets a Pressed snapshot. Edges are cleared after the step,
// held bits persist until a release arrives.
type Input struct {
	binding Binding
	held    uint32
	edges   uint32
	release uint32
}

// NewInput returns an input sampler using the given binding.
func NewInput(b *Binding) *Input {
	if b == nil {
		b = DefaultBinding()
	}
	return &Input{binding: *b}
}

// SetBinding replaces the key map. Already-held actions keep their state.
func (i *Input) SetBinding(b *Binding) { i.binding = *b }

// Binding returns a copy of the current key map.
func (i *Input) Binding() Binding { return i.binding }

// KeyDown records a key press. Repeated presses of a held action are ignored so
// the edge fires once.
func (i *Input) KeyDown(k Key) {
	a, ok := i.binding.ActionFor(k)
	if !ok {
		return
	}
	i.PressAction(a)
}

// KeyUp records a key release.
func (i *Input) KeyUp(k Key) {
	a, ok := i.binding.ActionFor(k)
	if !ok {
		return
	}
	i.ReleaseAction(a)
}

// PressAction records a press of one action directly.
//
// It exists because a key can be bound to several actions (Space is jump in
// play and confirm on the title screen) and only the caller knows which one the
// current screen cares about.
func (i *Input) PressAction(a Action) {
	bit := uint32(1) << uint(a)
	if i.held&bit != 0 {
		return
	}
	i.held |= bit
	i.edges |= bit
}

// ReleaseAction records a release of one action directly.
func (i *Input) ReleaseAction(a Action) {
	bit := uint32(1) << uint(a)
	if i.held&bit == 0 {
		return
	}
	i.held &^= bit
	i.release |= bit
}

// Held reports whether an action is currently down.
func (i *Input) Held(a Action) bool { return i.held&(uint32(1)<<uint(a)) != 0 }

// Sample freezes the current state for one simulation step.
//
// The caller must call ConsumeEdges after the step so a press is seen by exactly
// one step.
func (i *Input) Sample() Pressed {
	return Pressed{Held: i.held, Edge: i.edges, Released: i.release}
}

// ConsumeEdges clears the one-step signals. Held bits are untouched.
func (i *Input) ConsumeEdges() {
	i.edges = 0
	i.release = 0
}

// ReleaseAll clears every held action, used when focus is lost or the game is
// paused so keys do not stick down.
func (i *Input) ReleaseAll() {
	i.held = 0
	i.edges = 0
	i.release = 0
}
