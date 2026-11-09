package app

import (
	"testing"

	"super-mario/pkg/engine"
)

// TestRespawnKeepsCameraInPlace is the regression test for the reported bug:
// after dying and respawning, the whole picture sat lower than before.
//
// The camera's resting position is derived from the viewport. If the world
// computes it once with its placeholder viewport and the real size only arrives
// later, the two disagree and the picture shifts — which is exactly what a
// respawn does, since the player is put back at the start.
func TestRespawnKeepsCameraInPlace(t *testing.T) {
	a := newHeadlessApp(t)
	vw := engine.Tile.Mul(20)
	vh := engine.Tile.Mul(12)
	a.world.SetViewport(vw, vh)

	// Settle the camera with the player standing on the spawn tile.
	for i := 0; i < 10; i++ {
		a.stepOnce()
	}
	before := a.world.CameraY
	beforeX := a.world.CameraX

	// Kill the player, then run through the death animation to a respawn.
	a.world.Score.Lives = 3
	a.world.Player.Die()
	a.sm.Die()
	for i := 0; i < 120; i++ {
		a.stepOnce()
		if a.sm.Is(engine.StatePlaying) {
			break
		}
	}
	if !a.sm.Is(engine.StatePlaying) {
		t.Fatalf("did not respawn, state = %v", a.sm.Current())
	}
	for i := 0; i < 10; i++ {
		a.stepOnce()
	}

	if a.world.CameraY != before {
		t.Errorf("camera y moved across a respawn: %d -> %d (one screen low of the other)",
			before, a.world.CameraY)
	}
	if a.world.CameraX != beforeX {
		t.Errorf("camera x moved across a respawn: %d -> %d", beforeX, a.world.CameraX)
	}
}

// TestViewportChangeReanchors checks that the camera lands in the same place
// whether the viewport is set before or after the world settles.
func TestViewportChangeReanchors(t *testing.T) {
	a := newHeadlessApp(t)
	a.stepOnce()
	a.world.SetViewport(engine.Tile.Mul(20), engine.Tile.Mul(12))
	after := a.world.CameraY

	b := newHeadlessApp(t)
	b.world.SetViewport(engine.Tile.Mul(20), engine.Tile.Mul(12))
	b.stepOnce()

	if after != b.world.CameraY {
		t.Errorf("camera y depends on when the viewport was set: %d vs %d",
			after, b.world.CameraY)
	}
}

// TestJumpDoesNotMoveCamera locks in that the view holds still while airborne.
func TestJumpDoesNotMoveCamera(t *testing.T) {
	a := newHeadlessApp(t)
	a.world.SetViewport(engine.Tile.Mul(20), engine.Tile.Mul(12))
	for i := 0; i < 5; i++ {
		a.stepOnce()
	}
	start := a.world.CameraY

	a.input.KeyDown(engine.KeySpace)
	for i := 0; i < 40; i++ {
		a.stepOnce()
		if a.world.CameraY != start {
			t.Fatalf("camera moved during a jump: %d -> %d at step %d",
				start, a.world.CameraY, i)
		}
	}
}
