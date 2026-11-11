package game

import (
	"testing"

	"super-mario/pkg/engine"
)

// cameraWorld is a test world with a ten-tile viewport and the player placed
// mid-level, with the camera settled on him.
func cameraWorld(t *testing.T) *World {
	t.Helper()
	w := newTestWorld(t)
	w.SetViewport(engine.Tile*10, engine.Tile*8)
	w.Player.Pos.X = engine.Tile * 40
	w.updateCamera(true)
	return w
}

// TestWalkingBackScrollsTheCamera is the regression test for a camera that
// never came back: walking left past the left third of the screen must pull
// the view with the player.
func TestWalkingBackScrollsTheCamera(t *testing.T) {
	w := cameraWorld(t)
	start := w.CameraX
	w.Player.Pos.X -= engine.Tile * 6
	w.updateCamera(false)
	if w.CameraX >= start {
		t.Fatalf("walked six tiles back and the camera stayed at %d", w.CameraX)
	}
	left := w.Player.CenterX() - w.CameraX
	if want := w.ViewW / CameraBackFraction; left != want {
		t.Errorf("player is %d from the left edge, want %d (the scroll-back line)", left, want)
	}
}

// TestSmallStepsBackDoNotMoveTheCamera checks the dead zone: between the
// scroll-back line and the middle, the view stays put.
func TestSmallStepsBackDoNotMoveTheCamera(t *testing.T) {
	w := cameraWorld(t)
	start := w.CameraX
	w.Player.Pos.X -= engine.Tile
	w.updateCamera(false)
	if w.CameraX != start {
		t.Errorf("one step back moved the camera %d -> %d", start, w.CameraX)
	}
}

// TestCameraStopsAtTheLevelStart checks walking all the way back does not show
// anything left of column zero.
func TestCameraStopsAtTheLevelStart(t *testing.T) {
	w := cameraWorld(t)
	w.Player.Pos.X = engine.Tile
	w.updateCamera(false)
	if w.CameraX != 0 {
		t.Errorf("camera at %d at the level start, want 0", w.CameraX)
	}
}
