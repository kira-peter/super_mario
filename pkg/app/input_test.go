package app

import (
	"testing"
	"time"

	"super-mario/pkg/engine"
	"super-mario/pkg/level"
)

// newHeadlessApp builds an app with no terminal, already in gameplay.
func newHeadlessApp(t *testing.T) *App {
	t.Helper()
	lv, err := level.LoadEmbedded("1-1")
	if err != nil {
		t.Fatalf("load 1-1: %v", err)
	}
	a, err := newApp(nil, Options{ThemeName: "ink", FPS: 30, Scale: "small"}, lv)
	if err != nil {
		t.Fatalf("build app: %v", err)
	}
	// Tests about movement need the world actually stepping; the title
	// screen deliberately does not advance it.
	a.sm.StartRun()
	return a
}

// handleEventKeyForTest drives a key press through the same path the event
// loop uses.
func (a *App) handleEventKeyForTest(k engine.Key) { a.handleKeyDown(k) }

// TestWorldSamplesTheAppInput is the regression test for a bug that made the
// game look alive but uncontrollable: the terminal wrote key events into the
// app's input sampler while the world sampled a different one, so the timer
// counted down and Mario never moved.
func TestWorldSamplesTheAppInput(t *testing.T) {
	a := newHeadlessApp(t)

	if a.world.Input != a.input {
		t.Fatal("the world is sampling a different input instance than the terminal writes to; " +
			"keys would never reach the player")
	}
}

// TestPressingRightMovesThePlayer drives the app exactly as the event loop does
// and checks the player actually travels.
func TestPressingRightMovesThePlayer(t *testing.T) {
	a := newHeadlessApp(t)
	startX := a.world.Player.Pos.X

	a.input.KeyDown(engine.KeyRight)
	for i := 0; i < 60; i++ {
		a.sweepReleases()
		a.stepOnce()
	}

	if a.world.Player.Pos.X <= startX {
		t.Fatalf("player did not move right: %v -> %v", startX, a.world.Player.Pos.X)
	}
}

// TestHeldKeyStaysDownBetweenRepeats checks the release window does not cut a
// held key off between terminal repeat events, which would make movement
// stutter.
func TestHeldKeyStaysDownBetweenRepeats(t *testing.T) {
	a := newHeadlessApp(t)

	// Simulate a terminal that repeats every 40ms: record a press, then sweep
	// after less than the release window.
	a.handleEventKeyForTest(engine.KeyRight)
	a.heldAt[engine.KeyRight] = time.Now()

	startX := a.world.Player.Pos.X
	for i := 0; i < 30; i++ {
		// A repeat arrives well inside the window.
		a.heldAt[engine.KeyRight] = time.Now()
		a.sweepReleases()
		a.stepOnce()
	}

	if a.world.Player.Pos.X <= startX {
		t.Error("a continuously repeated key should keep the player moving")
	}
}

// TestKeyIsReleasedAfterPressesStop checks that letting go stops the player.
func TestKeyIsReleasedAfterPressesStop(t *testing.T) {
	a := newHeadlessApp(t)

	a.handleEventKeyForTest(engine.KeyRight)
	// Pretend the last repeat was long ago.
	a.heldAt[engine.KeyRight] = time.Now().Add(-2 * releaseWindow)

	a.sweepReleases()

	if a.input.Sample().HasHeld(engine.ActionRight) {
		t.Error("a key with no recent presses should have been released")
	}
}

// TestJumpEdgeIsSeenByExactlyOneStep guards against a press being consumed by
// two steps, which would allow a mid-air second jump.
func TestJumpEdgeIsSeenByExactlyOneStep(t *testing.T) {
	a := newHeadlessApp(t)

	a.handleEventKeyForTest(engine.KeySpace)
	a.stepOnce()
	if !a.world.Player.Jumping {
		t.Fatal("the jump press did not start a jump")
	}

	// The next step must not see the edge again.
	vy := a.world.Player.Vel.Y
	a.stepOnce()
	if a.world.Player.Vel.Y < vy {
		t.Error("the jump edge was consumed by two steps")
	}
}

// TestPausedGameIgnoresMovement checks the pause state really freezes input.
func TestPausedGameIgnoresMovement(t *testing.T) {
	a := newHeadlessApp(t)
	_ = a.sm.Pause()

	startX := a.world.Player.Pos.X
	a.input.KeyDown(engine.KeyRight)
	for i := 0; i < 60; i++ {
		a.stepOnce()
	}
	if a.world.Player.Pos.X != startX {
		t.Error("the player moved while paused")
	}
}
