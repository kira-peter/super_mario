package game

import (
	"testing"

	"super-mario/pkg/engine"
)

// airRun builds a world, runs the player up to speed, jumps, and then lets the
// caller steer in the air. It returns the world and the speed at take-off.
func airRun(t *testing.T, run bool) (*World, *engine.Input, engine.Fixed) {
	t.Helper()
	w := newTestWorld(t)
	in := engine.NewInput(nil)
	w.Input = in

	for i := 0; i < 200; i++ {
		in.KeyDown(engine.KeyRight)
		if run {
			in.KeyDown(engine.KeyShift)
		}
		w.Step()
	}
	in.KeyDown(engine.KeySpace)
	w.Step()
	in.KeyUp(engine.KeyRight)
	in.KeyUp(engine.KeyShift)
	if w.Player.Grounded {
		t.Fatal("setup: the jump did not leave the ground")
	}
	return w, in, w.Player.Vel.X
}

// flyUntilLanding steps the world with a per-step hook until the player lands,
// and returns how many steps that took.
func flyUntilLanding(w *World, hook func(step int)) int {
	n := 0
	for ; n < 200 && !w.Player.Grounded; n++ {
		if hook != nil {
			hook(n)
		}
		w.Step()
	}
	return n
}

// TestAirSteeringMovesTheLandingPoint is the first rule: after take-off the
// player can still steer, and it changes where they come down.
func TestAirSteeringMovesTheLandingPoint(t *testing.T) {
	landing := func(key engine.Key) engine.Fixed {
		w, in, _ := airRun(t, true)
		start := w.Player.Pos.X
		flyUntilLanding(w, func(int) {
			if key != 0 {
				in.KeyDown(key)
			}
		})
		return w.Player.Pos.X - start
	}
	coast, back, forward := landing(0), landing(engine.KeyLeft), landing(engine.KeyRight)

	if !(back < coast) {
		t.Errorf("holding back did not shorten the jump: back %.1f tiles, coast %.1f",
			float64(back)/float64(engine.Tile), float64(coast)/float64(engine.Tile))
	}
	if !(forward >= coast) {
		t.Errorf("holding forward shortened the jump: forward %.1f, coast %.1f",
			float64(forward)/float64(engine.Tile), float64(coast)/float64(engine.Tile))
	}
	// The correction has to be big enough to matter: at least half a tile.
	if coast-back < engine.Tile/2 {
		t.Errorf("steering back moved the landing by only %.2f tiles; too little to correct a jump",
			float64(coast-back)/float64(engine.Tile))
	}
}

// TestAirSteeringIsNotInstant is the second rule: reversing in mid-air has
// inertia. After a full-speed jump, holding the opposite direction must bleed
// speed away gradually, and must not stop or reverse the player at once.
func TestAirSteeringIsNotInstant(t *testing.T) {
	w, in, takeoff := airRun(t, true)
	if takeoff < w.T.RunMax*9/10 {
		t.Fatalf("setup: take-off speed %d is not a sprint (run max %d)", takeoff, w.T.RunMax)
	}

	prev := w.Player.Vel.X
	var biggestDrop engine.Fixed
	reversed := false
	flyUntilLanding(w, func(step int) {
		in.KeyDown(engine.KeyLeft)
		if step > 0 {
			if d := prev - w.Player.Vel.X; d > biggestDrop {
				biggestDrop = d
			}
		}
		prev = w.Player.Vel.X
		if w.Player.Vel.X < 0 {
			reversed = true
		}
	})

	if reversed {
		t.Error("a full-speed jump reversed direction in the air; the momentum was not respected")
	}
	// No single step may shed more than the air acceleration allows.
	if biggestDrop > w.T.AirAccel+1 {
		t.Errorf("speed dropped %.3f px/f in one step, more than the air acceleration %.3f",
			float64(biggestDrop)/256, float64(w.T.AirAccel)/256)
	}
	// And it must have slowed: steering has to do something.
	if w.Player.Vel.X >= takeoff {
		t.Errorf("holding back left the speed at %d (took off at %d)", w.Player.Vel.X, takeoff)
	}
}

// TestAirControlIsWeakerThanGroundControl checks the steering really is a
// "fine adjustment": slower than on the ground.
func TestAirControlIsWeakerThanGroundControl(t *testing.T) {
	if Default.AirAccel >= Default.WalkAccel {
		t.Errorf("air acceleration %d is not below ground acceleration %d", Default.AirAccel, Default.WalkAccel)
	}
	if Default.AirAccel <= 0 {
		t.Error("air acceleration is zero: the player could not steer at all")
	}
}

// TestJumpMomentumIsKept is the third rule: with no input the horizontal speed
// at take-off is what the player lands with.
func TestJumpMomentumIsKept(t *testing.T) {
	for _, run := range []bool{true, false} {
		w, _, takeoff := airRun(t, run)
		flyUntilLanding(w, nil)
		got := w.Player.Vel.X
		if diff := takeoff - got; diff < 0 || diff > 2 {
			t.Errorf("run=%v: took off at %d, landed at %d; momentum was not kept (lost %d)",
				run, takeoff, got, diff)
		}
	}
}

// TestHoldingForwardInTheAirDoesNotExceedTheCap checks steering cannot be used
// to go faster than the player's own top speed.
func TestHoldingForwardInTheAirDoesNotExceedTheCap(t *testing.T) {
	w, in, _ := airRun(t, false)
	flyUntilLanding(w, func(int) { in.KeyDown(engine.KeyRight) })
	if w.Player.Vel.X > w.T.WalkMax {
		t.Errorf("held forward in the air and reached %d, above the walking cap %d",
			w.Player.Vel.X, w.T.WalkMax)
	}
}

// TestAirSteeringKeepsFacing checks reversing in the air does not turn Mario
// around until he is actually moving the other way.
func TestAirSteeringKeepsFacing(t *testing.T) {
	w, in, _ := airRun(t, true)
	turnedWhileStillGoingRight := false
	flyUntilLanding(w, func(int) {
		in.KeyDown(engine.KeyLeft)
		if w.Player.Facing < 0 && w.Player.Vel.X > 0 {
			turnedWhileStillGoingRight = true
		}
	})
	if turnedWhileStillGoingRight {
		t.Error("Mario turned to face left while still travelling right")
	}
}

// TestFacingTurnsImmediatelyWhenStartingFromRest checks the facing rule does
// not make a standing start feel laggy: at zero speed there is no direction of
// travel to follow, so Mario faces the key at once.
func TestFacingTurnsImmediatelyWhenStartingFromRest(t *testing.T) {
	w := newTestWorld(t)
	in := engine.NewInput(nil)
	w.Input = in

	in.KeyDown(engine.KeyLeft)
	w.Step()
	if w.Player.Facing != -1 {
		t.Errorf("facing = %d one step after pressing left from rest, want -1", w.Player.Facing)
	}
	in.KeyUp(engine.KeyLeft)
	for i := 0; i < 120; i++ {
		w.Step()
	}
	in.KeyDown(engine.KeyRight)
	w.Step()
	if w.Player.Facing != 1 {
		t.Errorf("facing = %d one step after pressing right from rest, want 1", w.Player.Facing)
	}
}
