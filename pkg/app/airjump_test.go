package app

import (
	"testing"
	"time"

	"super-mario/pkg/engine"
)

// pressAt delivers a key press as if it arrived `ago` after the previous one,
// by back-dating the last-seen timestamp. This is the only way to exercise the
// real terminal timing without sleeping in a test.
func (a *App) pressAfter(k engine.Key, gap time.Duration) {
	if _, ok := a.heldAt[k]; ok {
		a.heldAt[k] = time.Now().Add(-gap)
	}
	a.handleKeyDown(k)
}

// TestDoubleJumpThroughTheRealInputPath is the regression test for a double
// jump that worked in unit tests and not at the keyboard.
//
// A terminal sends no key-up, so the first Space stays "held" until the release
// window passes. Pressing Space again in mid-air produced no press edge, and
// the jump code, which needs an edge, never ran. The earlier test called KeyUp
// and KeyDown directly, which no terminal can do, so it passed regardless.
func TestDoubleJumpThroughTheRealInputPath(t *testing.T) {
	a := newHeadlessApp(t)
	a.world.T.AirJump = true

	// First press: take off.
	a.handleKeyDown(engine.KeySpace)
	for i := 0; i < 12; i++ {
		a.stepOnce()
	}
	if a.world.Player.Grounded {
		t.Fatal("setup: the first press did not leave the ground")
	}
	vyBefore := a.world.Player.Vel.Y

	// Second press, a deliberate tap well after the first: not an auto-repeat.
	a.pressAfter(engine.KeySpace, 150*time.Millisecond)
	a.stepOnce()

	if a.world.Player.Vel.Y >= vyBefore {
		t.Errorf("the second press did nothing: vy %d -> %d (want a fresh upward kick)",
			vyBefore.Int(), a.world.Player.Vel.Y.Int())
	}
	if !a.world.Player.AirJumpUsed {
		t.Error("the mid-air jump was not recorded as used")
	}
}

// TestHeldKeyRepeatIsNotADoubleJump checks the other side of the same line:
// keeping Space down, so the keyboard auto-repeats, must not fire the second
// jump. Without this a long press would burn both jumps at once.
func TestHeldKeyRepeatIsNotADoubleJump(t *testing.T) {
	a := newHeadlessApp(t)
	a.world.T.AirJump = true

	a.handleKeyDown(engine.KeySpace)
	for i := 0; i < 30; i++ {
		// Auto-repeat arrives every ~35 ms, well inside repeatGap.
		a.pressAfter(engine.KeySpace, 35*time.Millisecond)
		a.stepOnce()
	}
	if a.world.Player.AirJumpUsed {
		t.Error("holding the key down spent the mid-air jump")
	}
}

// TestOnlyOneAirJumpPerTakeoff checks the jump is limited to one per stint off
// the ground.
func TestOnlyOneAirJumpPerTakeoff(t *testing.T) {
	a := newHeadlessApp(t)
	a.world.T.AirJump = true

	a.handleKeyDown(engine.KeySpace)
	for i := 0; i < 10; i++ {
		a.stepOnce()
	}
	a.pressAfter(engine.KeySpace, 150*time.Millisecond)
	a.stepOnce()
	vyAfterSecond := a.world.Player.Vel.Y

	// A third press must not add anything.
	for i := 0; i < 5; i++ {
		a.stepOnce()
	}
	vyBefore := a.world.Player.Vel.Y
	a.pressAfter(engine.KeySpace, 150*time.Millisecond)
	a.stepOnce()
	if a.world.Player.Vel.Y < vyBefore-a.world.T.GravityFall*2 {
		t.Errorf("a third jump was applied: vy %d -> %d (second jump left it at %d)",
			vyBefore.Int(), a.world.Player.Vel.Y.Int(), vyAfterSecond.Int())
	}
}

// TestAirJumpResetsOnLanding checks a fresh takeoff gets its own extra jump.
func TestAirJumpResetsOnLanding(t *testing.T) {
	a := newHeadlessApp(t)
	a.world.T.AirJump = true

	a.handleKeyDown(engine.KeySpace)
	for i := 0; i < 10; i++ {
		a.stepOnce()
	}
	a.pressAfter(engine.KeySpace, 150*time.Millisecond)
	a.stepOnce()

	// Let go and wait to land.
	a.heldAt = map[engine.Key]timeStamp{}
	a.input.ReleaseAll()
	for i := 0; i < 200 && !a.world.Player.Grounded; i++ {
		a.stepOnce()
	}
	if !a.world.Player.Grounded {
		t.Fatal("setup: never landed")
	}

	a.handleKeyDown(engine.KeySpace)
	a.stepOnce()
	if a.world.Player.AirJumpUsed {
		t.Error("the mid-air jump was still marked used after landing")
	}
}

// TestNoAirJumpFlagRestoresOriginalRule checks the opt-out really disables it.
func TestNoAirJumpFlagRestoresOriginalRule(t *testing.T) {
	a := newHeadlessApp(t)
	a.world.T.AirJump = false

	a.handleKeyDown(engine.KeySpace)
	for i := 0; i < 12; i++ {
		a.stepOnce()
	}
	vyBefore := a.world.Player.Vel.Y
	a.pressAfter(engine.KeySpace, 150*time.Millisecond)
	a.stepOnce()
	if a.world.Player.Vel.Y < vyBefore-a.world.T.GravityFall*2 {
		t.Error("an air jump happened with the feature switched off")
	}
}
