package game

import (
	"testing"

	"super-mario/pkg/engine"
)

// rolledPhases runs the player for n steps with the given keys held and returns
// how many wheel frames went by, counting every change of frame.
func rolledPhases(t *testing.T, w *World, in *engine.Input, steps int, keys ...engine.Key) int {
	t.Helper()
	phases := 4
	if w.Player.Form != FormSmall {
		phases = 8
	}
	last := w.Player.WheelPhase(phases)
	changes := 0
	for i := 0; i < steps; i++ {
		for _, k := range keys {
			in.KeyDown(k)
		}
		w.Step()
		if p := w.Player.WheelPhase(phases); p != last {
			changes++
			last = p
		}
	}
	return changes
}

// TestWheelTurnsWhileWalking checks the wheel rolls on the ground.
func TestWheelTurnsWhileWalking(t *testing.T) {
	w := newTestWorld(t)
	in := engine.NewInput(nil)
	w.Input = in
	if got := rolledPhases(t, w, in, 60, engine.KeyRight); got < 3 {
		t.Errorf("the wheel changed frame %d times in a second of walking, want it turning", got)
	}
}

// TestWheelStandsStillWhenStanding checks a stationary hero's wheel does not
// creep.
func TestWheelStandsStillWhenStanding(t *testing.T) {
	w := newTestWorld(t)
	in := engine.NewInput(nil)
	w.Input = in
	if got := rolledPhases(t, w, in, 120); got != 0 {
		t.Errorf("the wheel changed frame %d times while standing still", got)
	}
}

// TestWheelFreezesInTheAir is the rule that nothing is touching the wheel while
// he is airborne, so it must not turn.
func TestWheelFreezesInTheAir(t *testing.T) {
	w := newTestWorld(t)
	in := engine.NewInput(nil)
	w.Input = in

	// Get up to speed, then jump and sample the wheel through the flight.
	for i := 0; i < 90; i++ {
		in.KeyDown(engine.KeyRight)
		in.KeyDown(engine.KeyShift)
		w.Step()
	}
	in.KeyDown(engine.KeySpace)
	w.Step()
	if w.Player.Grounded {
		t.Fatal("setup: the jump did not leave the ground")
	}
	before := w.Player.WheelPhase(4)
	moved := w.Player.Pos.X
	for i := 0; i < 40 && !w.Player.Grounded; i++ {
		in.KeyDown(engine.KeyRight)
		in.KeyDown(engine.KeyShift)
		w.Step()
		if w.Player.Grounded {
			break
		}
		if got := w.Player.WheelPhase(4); got != before {
			t.Fatalf("the wheel turned in the air (frame %d -> %d)", before, got)
		}
	}
	if w.Player.Pos.X == moved {
		t.Fatal("setup: he did not travel during the jump, so this proves nothing")
	}
}

// TestWheelDoesNotSpinAgainstAWall checks the wheel follows distance actually
// covered. Pushing into a wall covers none.
func TestWheelDoesNotSpinAgainstAWall(t *testing.T) {
	w := newTestWorld(t)
	in := engine.NewInput(nil)
	w.Input = in

	// Walk left until he is stopped by the level's edge.
	for i := 0; i < 300; i++ {
		in.KeyDown(engine.KeyLeft)
		w.Step()
	}
	phase := w.Player.WheelPhase(4)
	for i := 0; i < 120; i++ {
		in.KeyDown(engine.KeyLeft)
		w.Step()
	}
	if got := w.Player.WheelPhase(4); got != phase {
		t.Errorf("the wheel kept turning against the wall (frame %d -> %d)", phase, got)
	}
}

// TestWheelRollsTheSameWayWhicheverDirectionHeTravels checks the phase counts
// distance, not signed distance: the renderer decides the turning direction by
// mirroring, so the phase itself must always advance forwards.
func TestWheelRollsTheSameWayWhicheverDirectionHeTravels(t *testing.T) {
	run := func(key engine.Key) int {
		w := newTestWorld(t)
		// Start in the middle so there is room to walk either way.
		w.Player.Pos.X = engine.Fixed(30) * engine.Tile
		in := engine.NewInput(nil)
		w.Input = in
		return rolledPhases(t, w, in, 60, key)
	}
	right, left := run(engine.KeyRight), run(engine.KeyLeft)
	if left < right-1 || left > right+1 {
		t.Errorf("rolling left changed frame %d times and right %d; they should match", left, right)
	}
}

// TestWheelTurnsInStepWithTheGround is the regression test for a units mistake:
// the rim was first computed in sprite pixels but compared against game units,
// which made the wheel turn four times too fast.
//
// At a full run he covers 150 game units a second. A small wheel's rim is six
// sprite pixels, four game units each, so 24 units: a little over six turns a
// second. The check is against that arithmetic, not against a feel for what
// looks right, so a units slip in either direction fails it.
func TestWheelTurnsInStepWithTheGround(t *testing.T) {
	w := newTestWorld(t)
	in := engine.NewInput(nil)
	w.Input = in

	for i := 0; i < 120; i++ { // reach a steady run
		in.KeyDown(engine.KeyRight)
		in.KeyDown(engine.KeyShift)
		w.Step()
	}
	startX := w.Player.Pos.X
	frames := rolledPhases(t, w, in, 60, engine.KeyRight, engine.KeyShift)
	travelled := float64(w.Player.Pos.X-startX) / float64(engine.Unit)

	want := travelled / 24 * 4 // turns = distance / rim, four frames per turn
	if float64(frames) < want-2 || float64(frames) > want+2 {
		t.Errorf("the wheel changed frame %d times over %.0f units; rolling that far should be about %.0f",
			frames, travelled, want)
	}
}

// TestBigWheelTurnsMoreSlowlyThanSmall checks the bigger wheel covers more
// ground per turn.
func TestBigWheelTurnsMoreSlowlyThanSmall(t *testing.T) {
	turns := func(big bool) float64 {
		w := newTestWorld(t)
		if big {
			w.Player.Grow(w.Level)
		}
		in := engine.NewInput(nil)
		w.Input = in
		for i := 0; i < 120; i++ {
			in.KeyDown(engine.KeyRight)
			in.KeyDown(engine.KeyShift)
			w.Step()
		}
		phases := 4
		if big {
			phases = 8
		}
		return float64(rolledPhases(t, w, in, 60, engine.KeyRight, engine.KeyShift)) / float64(phases)
	}
	small, big := turns(false), turns(true)
	if !(big < small) {
		t.Errorf("the big wheel turned %.2f times a second, not fewer than the small one's %.2f", big, small)
	}
}

// TestWheelPhaseIsDeterministic checks the same inputs always give the same
// frame, so replays and redraw rates cannot disagree about it.
func TestWheelPhaseIsDeterministic(t *testing.T) {
	run := func() int {
		w := newTestWorld(t)
		in := engine.NewInput(nil)
		w.Input = in
		for i := 0; i < 150; i++ {
			in.KeyDown(engine.KeyRight)
			w.Step()
		}
		return w.Player.WheelPhase(8)
	}
	if a, b := run(), run(); a != b {
		t.Errorf("the wheel ended on frame %d one time and %d the next", a, b)
	}
}
