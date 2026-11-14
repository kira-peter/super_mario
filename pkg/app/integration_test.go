package app

import (
	"testing"

	"super-mario/pkg/engine"
	"super-mario/pkg/game"
	"super-mario/pkg/level"
)

// headless builds a world on the real 1-1 without touching a terminal.
func headless(t *testing.T) *game.World {
	t.Helper()
	lv, err := level.LoadEmbedded("1-1")
	if err != nil {
		t.Fatalf("load 1-1: %v", err)
	}
	sm := engine.NewStateMachine()
	sm.StartRun()
	return game.NewWorld(lv, &game.Default, sm, 3)
}

// TestLevelIsTraversable walks a scripted player from the spawn to the
// flagpole. It catches the class of bug unit tests miss: a level that is
// individually well-formed but has a jump no one can actually make.
func TestLevelIsTraversable(t *testing.T) {
	w := headless(t)

	// Keep the viewport realistic so camera-driven activation is exercised.
	w.ViewW = engine.Tile.Mul(10)
	w.ViewH = engine.Tile.Mul(7)

	in := engine.NewInput(nil)
	w.Input = in

	reachedFlag := false
	for step := 0; step < 60*120; step++ { // up to two minutes of game time
		// Simple but honest bot: run right, jump when something blocks the
		// path ahead or when a gap is detected.
		in.KeyDown(engine.KeyRight)
		in.KeyDown(engine.KeyShift)

		if shouldJump(w) {
			in.KeyDown(engine.KeySpace)
		} else {
			in.KeyUp(engine.KeySpace)
		}

		w.Step()

		if w.SM.Current() == engine.StateFlagpole || w.SM.Current() == engine.StateResults {
			reachedFlag = true
			break
		}
		if w.Player.Dead {
			break
		}
	}

	if !reachedFlag {
		t.Fatalf("player never reached the flagpole: x=%v dead=%v state=%v",
			w.Player.Pos.X, w.Player.Dead, w.SM.Current())
	}
}

// tileOf converts a Fixed world coordinate to a tile index.
//
// Fixed raw units are 1/256 of a game unit and a tile is 16 game units, so the
// divisor is Tile and the result is already a tile count; calling Int() on top
// of that would divide by 256 a second time.
func tileOf(f engine.Fixed) int {
	return int(f) / int(engine.Tile)
}

// shouldJump is a deliberately dumb bot: jump at walls and at ledges. It is not
// meant to play well, only to prove the level is passable without teleporting.
func shouldJump(w *game.World) bool {
	p := w.Player
	if !p.Grounded {
		// Keep holding while rising, which produces the tall jump a gap needs.
		return p.Vel.Y < 0
	}

	// The row the feet occupy. Bottom() is exclusive, so subtract one first;
	// using Bottom() directly reports the air row and makes the bot think
	// every tile ahead is a wall.
	footRow := tileOf(p.Bottom() - 1)
	aheadCol := tileOf(p.Right()) + 1

	// A wall directly ahead at torso height.
	if w.Level.SolidAt(aheadCol, footRow-1) || w.Level.SolidAt(aheadCol, footRow-2) {
		return true
	}

	// No floor directly ahead: commit to the jump right at the edge.
	//
	// Looking further than one tile makes the bot jump early, which wastes the
	// arc on flat ground and lands it in the middle of the gap. One tile ahead
	// is both what a player does and what the physics can carry.
	edgeCol := tileOf(p.Right()) + 1
	if !w.Level.SolidAt(edgeCol, footRow+1) {
		return true
	}

	// An enemy ahead, within reach of the jump. The faster the bot is going, the
	// sooner it has to jump to clear it, so the distance grows with speed: two
	// tiles at rest, plus the ground covered in the time it takes the jump to
	// rise over an enemy. A fixed two tiles was tuned to the old, slower
	// acceleration and jumped too late once running got quicker.
	reach := engine.Tile.Mul(2) + p.Vel.X.Abs()*12
	for _, e := range w.Enemies {
		if e.Removed || e.Squashed > 0 {
			continue
		}
		if e.Pos.X > p.Pos.X && e.Pos.X-p.Pos.X < reach {
			return true
		}
	}
	return false
}

// TestBlocksAndItemsAreReachable verifies the level's authored content behaves:
// the mushroom block really grows the player when hit.
func TestMushroomBlockGrowsPlayer(t *testing.T) {
	w := headless(t)

	// Find the mushroom block in the level data.
	var mx, my int
	found := false
	for _, c := range w.Level.Contents {
		if c.Kind == level.BlockMushroom {
			mx, my = c.X, c.Y
			found = true
			break
		}
	}
	if !found {
		t.Fatal("1-1 has no mushroom block")
	}

	// Stand under it and jump into it.
	w.Player.Pos.X = engine.Fixed(mx)*engine.Tile + 2
	w.Player.Pos.Y = engine.Fixed(my+1)*engine.Tile + 2
	w.Player.Vel.Y = engine.FromFloat(-4)
	w.Player.Grounded = false

	for i := 0; i < 40; i++ {
		w.Step()
	}

	if w.Level.TileAt(mx, my).Kind == level.KindQuestion {
		t.Error("mushroom block was not consumed")
	}
	if len(w.Items) == 0 {
		t.Fatal("no item was spawned")
	}
	if w.Items[0].Kind != game.ItemMushroom {
		t.Errorf("spawned %v, want a mushroom", w.Items[0].Kind)
	}
}

// TestCameraStaysInsideLevel ensures the camera never shows the void.
func TestCameraStaysInsideLevel(t *testing.T) {
	w := headless(t)
	w.ViewW = engine.Tile.Mul(10)
	w.ViewH = engine.Tile.Mul(7)

	in := engine.NewInput(nil)
	w.Input = in
	for i := 0; i < 60*60; i++ {
		in.KeyDown(engine.KeyRight)
		in.KeyDown(engine.KeyShift)
		w.Step()
		if w.CameraX < 0 {
			t.Fatalf("camera went negative: %v", w.CameraX)
		}
		maxX := engine.Fixed(w.Level.Width)*engine.Tile - w.ViewW
		if w.CameraX > maxX {
			t.Fatalf("camera passed the right edge: %v > %v", w.CameraX, maxX)
		}
		if w.CameraY < 0 {
			t.Fatalf("camera went above the level: %v", w.CameraY)
		}
	}
}
