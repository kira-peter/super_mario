package engine

import "time"

// StepDuration is one simulation step. Sixty steps per second matches the
// original game, which is what lets the tuning constants be used directly.
const StepDuration = time.Second / 60

// MaxCatchUpSteps caps how many steps one Advance call may run. After the
// terminal was suspended or the process stalled, the wall clock can be seconds
// behind; replaying all of it would fast-forward the game through the level.
// The extra time is dropped instead (design decision 2).
const MaxCatchUpSteps = 5

// Clock turns a wall-clock elapsed time into a number of simulation steps.
//
// It holds no game state and touches no timers, so tests drive it with explicit
// durations and get exactly reproducible results.
type Clock struct {
	acc time.Duration
}

// NewClock returns a clock with no accumulated time.
func NewClock() *Clock { return &Clock{} }

// Advance reports how many whole steps have elapsed for the given delta,
// carrying the remainder to the next call. It never returns more than
// MaxCatchUpSteps; when the backlog exceeds that, the accumulator is cleared so
// the excess is dropped rather than replayed.
func (c *Clock) Advance(delta time.Duration) int {
	if delta < 0 {
		delta = 0
	}
	c.acc += delta
	steps := 0
	for c.acc >= StepDuration && steps < MaxCatchUpSteps {
		c.acc -= StepDuration
		steps++
	}
	if steps == MaxCatchUpSteps && c.acc >= StepDuration {
		c.acc = 0
	}
	return steps
}

// Reset clears any accumulated remainder, used when a new game starts so a
// paused or suspended session does not bleed into the new run.
func (c *Clock) Reset() { c.acc = 0 }

// Remainder returns the un-consumed fraction of a step. Exposed for tests and
// for the renderer's optional interpolation.
func (c *Clock) Remainder() time.Duration { return c.acc }
