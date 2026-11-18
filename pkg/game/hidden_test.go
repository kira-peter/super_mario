package game

import (
	"testing"

	"super-mario/pkg/engine"
	"super-mario/pkg/level"
)

// hiddenWorld puts a hidden block in the air above the floor, close enough to
// the spawn that a jump reaches it.
func hiddenWorld(t *testing.T) (*World, int, int) {
	t.Helper()
	w := newTestWorld(t)
	cx, cy := w.Level.SpawnX+1, w.Level.SpawnY-4
	w.Level.Contents = append(w.Level.Contents, level.Content{
		X: cx, Y: cy, Kind: level.BlockCoin, Count: 1, Hidden: true,
	})
	return w, cx, cy
}

// jumpUntilTop bumps the player up into the cell at (cx, cy) by holding jump
// while standing under it. It returns false if the head never reached the row.
func jumpUntilTop(w *World, in *engine.Input, cx, cy int) bool {
	in.KeyDown(engine.KeyRight)
	for i := 0; i < 6; i++ {
		w.Step()
	}
	in.KeyUp(engine.KeyRight)
	for i := 0; i < 60; i++ {
		in.KeyDown(engine.KeySpace)
		w.Step()
		if w.Player.LastHit.HitCeiling && w.Player.LastHit.CeilingX == cx && w.Player.LastHit.CeilingY == cy {
			in.KeyUp(engine.KeySpace)
			return true
		}
	}
	in.KeyUp(engine.KeySpace)
	return false
}

// TestHiddenBlockStopsTheHead is the regression test for hidden blocks that
// could not be found at all: the tile grid says air there, so the vertical
// collision skipped the cell and the head passed straight through it.
func TestHiddenBlockStopsTheHead(t *testing.T) {
	w, cx, cy := hiddenWorld(t)
	in := engine.NewInput(nil)
	w.Input = in

	if !jumpUntilTop(w, in, cx, cy) {
		t.Fatalf("the head never hit the hidden block at (%d,%d); player at %v,%v",
			cx, cy, w.Player.Pos.X, w.Player.Pos.Y)
	}
}

// TestBumpingAHiddenBlockRevealsAndPays checks the whole interaction: the bump
// turns the block into a question tile and gives out its payload.
func TestBumpingAHiddenBlockRevealsAndPays(t *testing.T) {
	w, cx, cy := hiddenWorld(t)
	in := engine.NewInput(nil)
	w.Input = in
	coins, points := w.Score.Coins, w.Score.Points

	if !jumpUntilTop(w, in, cx, cy) {
		t.Fatalf("the head never hit the hidden block at (%d,%d)", cx, cy)
	}
	w.Step() // handleBump runs on the step after the contact

	// A one-coin block is used up in the same bump, exactly like a question
	// block: the reveal and the payout happen together.
	if got := w.Level.TileAt(cx, cy).Glyph; got != '#' {
		t.Errorf("tile at the hidden block = %q after the bump, want '#' (used)", string(got))
	}
	if w.Score.Coins != coins+1 || w.Score.Points != points+coinScore {
		t.Errorf("coins %d -> %d, points %d -> %d; want +1 coin, +%d points",
			coins, w.Score.Coins, points, w.Score.Points, coinScore)
	}
}

// TestAHiddenBlockIsNotSolidFromAbove guards the other half: it is air until
// found, so walking over it must not make the player stand on nothing.
func TestAHiddenBlockIsNotSolidFromAbove(t *testing.T) {
	w := newTestWorld(t)
	in := engine.NewInput(nil)
	w.Input = in
	cx, cy := w.Level.SpawnX+6, w.Level.SpawnY-2
	w.Level.SetTile(cx, cy, level.Tile{Kind: level.KindEmpty, Glyph: ' '})
	w.Level.Contents = append(w.Level.Contents, level.Content{
		X: cx, Y: cy, Kind: level.BlockCoin, Count: 1, Hidden: true,
	})

	for i := 0; i < 120; i++ {
		in.KeyDown(engine.KeyRight)
		w.Step()
	}
	if !w.Player.Grounded {
		t.Fatal("the player is not standing on the floor")
	}
	if row := int(w.Player.Pos.Y+engine.Tile) / int(engine.Tile); row < cy {
		t.Errorf("the player was propped up by the hidden block: bottom row %d, block at %d", row, cy)
	}
}
