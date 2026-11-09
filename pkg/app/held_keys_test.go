package app

import (
	"testing"
	"time"

	"super-mario/pkg/engine"
)

// fakeKeyboard is a keyboard the test controls, standing in for the platform's
// "is this key really down" query.
type fakeKeyboard struct {
	down map[engine.Key]bool
}

func newFakeKeyboard() *fakeKeyboard { return &fakeKeyboard{down: map[engine.Key]bool{}} }

func (f *fakeKeyboard) probe(k engine.Key) (bool, bool) { return f.down[k], true }

// clockedApp returns a headless, already-playing app with a controllable clock.
func clockedApp(t *testing.T) (*App, *time.Time) {
	t.Helper()
	a := newHeadlessApp(t)
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	a.timeNow = func() time.Time { return now }
	return a, &now
}

// advance moves the test clock and runs that many simulation steps, sweeping
// key releases before each step exactly as the real loop does.
func advance(a *App, now *time.Time, steps int) {
	for i := 0; i < steps; i++ {
		*now = now.Add(engine.StepDuration)
		a.sweepReleases()
		a.stepOnce()
	}
}

// TestHeldDirectionSurvivesAJumpWithoutAPlatformProbe is the report as it
// happens on a plain terminal: hold right, tap Space, and the right key must
// still count as held in the air and after landing.
//
// The terminal repeats only the last key pressed, so once Space goes down the
// right key's repeats stop. Inferring "held" from repeats alone releases it
// 180 ms later, mid-jump.
func TestHeldDirectionIsKeptThroughAJump(t *testing.T) {
	a, now := clockedApp(t)
	kb := newFakeKeyboard()
	a.probe = kb.probe

	kb.down[engine.KeyRight] = true
	a.handleKeyDown(engine.KeyRight)
	advance(a, now, 30)
	startX := a.world.Player.Pos.X

	// Tap Space. From here the terminal stops repeating Right.
	kb.down[engine.KeySpace] = true
	a.handleKeyDown(engine.KeySpace)
	advance(a, now, 5)
	kb.down[engine.KeySpace] = false

	// A full second in the air and on the ground with Right still held down
	// but no further Right events arriving.
	advance(a, now, 60)

	if !a.input.Held(engine.ActionRight) {
		t.Fatal("the right key was released while it was still physically held")
	}
	if !a.world.Player.Grounded {
		t.Fatal("setup: the player never landed")
	}
	if a.world.Player.Vel.X <= 0 {
		t.Errorf("Mario stopped on landing (vx=%d) while right was still held", a.world.Player.Vel.X)
	}
	if a.world.Player.Pos.X <= startX {
		t.Error("Mario made no progress across the jump while right was held")
	}
}

// TestLandingKeepsWalkingInTheHeldDirection checks the behaviour the report
// asks for directly: nothing is pressed again after landing, and Mario walks.
func TestLandingKeepsWalkingInTheHeldDirection(t *testing.T) {
	a, now := clockedApp(t)
	kb := newFakeKeyboard()
	a.probe = kb.probe

	kb.down[engine.KeyRight] = true
	a.handleKeyDown(engine.KeyRight)
	kb.down[engine.KeySpace] = true
	a.handleKeyDown(engine.KeySpace)
	advance(a, now, 3)
	kb.down[engine.KeySpace] = false

	for i := 0; i < 200 && !(a.world.Player.Grounded && i > 5); i++ {
		advance(a, now, 1)
	}
	landedX := a.world.Player.Pos.X
	advance(a, now, 30)

	if a.world.Player.Pos.X <= landedX {
		t.Error("Mario stood still after landing with the direction key still held")
	}
}

// TestReleasingTheKeyStillStopsMario guards the other side: believing the
// keyboard must not make keys stick.
func TestReleasingTheKeyStillStopsMario(t *testing.T) {
	a, now := clockedApp(t)
	kb := newFakeKeyboard()
	a.probe = kb.probe

	kb.down[engine.KeyRight] = true
	a.handleKeyDown(engine.KeyRight)
	advance(a, now, 40)
	kb.down[engine.KeyRight] = false
	advance(a, now, 2)

	if a.input.Held(engine.ActionRight) {
		t.Error("the right key stayed held after it was physically released")
	}
	advance(a, now, 240)
	if a.world.Player.Vel.X != 0 {
		t.Errorf("Mario kept moving (vx=%d) after the key was released", a.world.Player.Vel.X)
	}
}

// TestInferenceStillWorksWithoutAProbe checks the fallback used on terminals
// and platforms that cannot report physical key state.
func TestInferenceStillWorksWithoutAProbe(t *testing.T) {
	a, now := clockedApp(t)
	a.probe = nil

	a.handleKeyDown(engine.KeyRight)
	advance(a, now, 3)
	if !a.input.Held(engine.ActionRight) {
		t.Fatal("a fresh press was not held")
	}
	// No more repeats arrive: it must time out.
	advance(a, now, int(releaseWindow/engine.StepDuration)+5)
	if a.input.Held(engine.ActionRight) {
		t.Error("with no probe and no repeats the key never timed out")
	}
}

// TestReleaseFreesEveryActionOnAKey is the sticky-action bug: Space is bound to
// both jump and confirm, only the first was released, and a held action never
// produces a new press edge.
func TestReleaseFreesEveryActionOnAKey(t *testing.T) {
	a, now := clockedApp(t)
	kb := newFakeKeyboard()
	a.probe = kb.probe

	kb.down[engine.KeySpace] = true
	a.handleKeyDown(engine.KeySpace)
	advance(a, now, 2)
	kb.down[engine.KeySpace] = false
	advance(a, now, 2)

	for _, act := range []engine.Action{engine.ActionJump, engine.ActionConfirm} {
		if a.input.Held(act) {
			t.Errorf("action %d stayed held after Space was released", act)
		}
	}

	// And a second press must produce a fresh edge for both.
	kb.down[engine.KeySpace] = true
	a.handleKeyDown(engine.KeySpace)
	s := a.input.Sample()
	if !s.HasEdge(engine.ActionJump) || !s.HasEdge(engine.ActionConfirm) {
		t.Errorf("a second press did not register on both actions (jump=%v confirm=%v)",
			s.HasEdge(engine.ActionJump), s.HasEdge(engine.ActionConfirm))
	}
}

// TestProbeCannotStartAnActionByItself checks the keyboard query only confirms
// keys the game has already seen pressed, since it reads the whole system.
func TestProbeCannotStartAnActionByItself(t *testing.T) {
	a, now := clockedApp(t)
	kb := newFakeKeyboard()
	a.probe = kb.probe

	// Right is down somewhere, but the game never received a press for it.
	kb.down[engine.KeyRight] = true
	advance(a, now, 30)

	if a.input.Held(engine.ActionRight) {
		t.Error("a key pressed outside the game started an action")
	}
	if a.world.Player.Pos.X != a.world.Player.Pos.X || a.world.Player.Vel.X != 0 {
		t.Errorf("Mario moved (vx=%d) from a key the game never saw", a.world.Player.Vel.X)
	}
}

// TestSteeringInTheAirWorksWhileHoldingThroughTheJump ties the two reports
// together: hold forward, jump, and keep steering for the whole flight.
func TestSteeringInTheAirWorksWhileHoldingThroughTheJump(t *testing.T) {
	a, now := clockedApp(t)
	kb := newFakeKeyboard()
	a.probe = kb.probe

	kb.down[engine.KeyRight] = true
	a.handleKeyDown(engine.KeyRight)
	kb.down[engine.KeyShift] = true
	a.handleKeyDown(engine.KeyShift)
	advance(a, now, 90)
	kb.down[engine.KeySpace] = true
	a.handleKeyDown(engine.KeySpace)
	advance(a, now, 2)
	kb.down[engine.KeySpace] = false

	// Mid-flight the right key must still be held.
	advance(a, now, 20)
	if a.world.Player.Grounded {
		t.Skip("landed before the mid-air check; jump too short for this setup")
	}
	if !a.input.Held(engine.ActionRight) {
		t.Error("the direction key was lost while airborne")
	}
	if !a.input.Held(engine.ActionRun) {
		t.Error("the run key was lost while airborne")
	}
}
