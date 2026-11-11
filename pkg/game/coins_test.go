package game

import (
	"testing"

	"super-mario/pkg/engine"
	"super-mario/pkg/level"
)

// coinWorld is a test world with one coin tile placed where the player can walk
// into it.
func coinWorld(t *testing.T) (*World, int, int) {
	t.Helper()
	w := newTestWorld(t)
	cx, cy := w.Level.SpawnX+3, w.Level.SpawnY
	w.Level.SetTile(cx, cy, level.Tile{Kind: level.KindDecor, Glyph: 'o'})
	return w, cx, cy
}

// TestWalkingIntoACoinCollectsIt is the regression test for coins that could be
// seen and never taken: they are tiles, and nothing read them back.
func TestWalkingIntoACoinCollectsIt(t *testing.T) {
	w, cx, cy := coinWorld(t)
	in := engine.NewInput(nil)
	w.Input = in
	coins, points := w.Score.Coins, w.Score.Points

	for i := 0; i < 120 && w.Level.TileAt(cx, cy).Glyph == 'o'; i++ {
		in.KeyDown(engine.KeyRight)
		w.Step()
	}
	if w.Level.TileAt(cx, cy).Glyph == 'o' {
		t.Fatal("walked through the coin and it is still there")
	}
	if w.Score.Coins != coins+1 {
		t.Errorf("coins %d -> %d, want one more", coins, w.Score.Coins)
	}
	if w.Score.Points != points+coinScore {
		t.Errorf("points %d -> %d, want +%d", points, w.Score.Points, coinScore)
	}
}

// TestACoinIsCollectedOnlyOnce checks standing in the coin's cell does not keep
// paying out.
func TestACoinIsCollectedOnlyOnce(t *testing.T) {
	w, cx, cy := coinWorld(t)
	w.Player.Pos.X = engine.Fixed(cx)*engine.Tile + 2
	w.Player.Pos.Y = engine.Fixed(cy+1)*engine.Tile - w.Player.H
	before := w.Score.Coins
	for i := 0; i < 60; i++ {
		w.Step()
	}
	if w.Score.Coins != before+1 {
		t.Errorf("standing on a coin gave %d coins, want exactly 1", w.Score.Coins-before)
	}
}

// TestJumpingThroughACoinCollectsIt checks coins in the air are reachable: the
// ones over the gap in 1-1 can only be taken by jumping.
func TestJumpingThroughACoinCollectsIt(t *testing.T) {
	w := newTestWorld(t)
	cx, cy := w.Level.SpawnX, w.Level.SpawnY-2
	w.Level.SetTile(cx, cy, level.Tile{Kind: level.KindDecor, Glyph: 'o'})
	in := engine.NewInput(nil)
	w.Input = in
	before := w.Score.Coins

	in.KeyDown(engine.KeySpace)
	for i := 0; i < 60; i++ {
		w.Step()
	}
	if w.Score.Coins != before+1 {
		t.Errorf("jumping straight up through a coin gave %d coins, want 1", w.Score.Coins-before)
	}
}

// TestEveryCoinIn11CanBeCollected walks the real level's coin list and checks
// each one is collected when the player is placed on it. A coin that is drawn
// but not collectable is exactly the bug that was reported.
func TestEveryCoinIn11CanBeCollected(t *testing.T) {
	lv, err := level.LoadEmbedded("1-1")
	if err != nil {
		t.Fatal(err)
	}
	var coins [][2]int
	for y := 0; y < lv.Height; y++ {
		for x := 0; x < lv.Width; x++ {
			if lv.TileAt(x, y).Glyph == 'o' {
				coins = append(coins, [2]int{x, y})
			}
		}
	}
	if len(coins) == 0 {
		t.Fatal("1-1 has no coins to test")
	}
	for _, c := range coins {
		sm := engine.NewStateMachine()
		sm.StartRun()
		w := NewWorld(lv, &Default, sm, 3)
		w.Player.Pos.X = engine.Fixed(c[0])*engine.Tile + 2
		w.Player.Pos.Y = engine.Fixed(c[1])*engine.Tile + 1
		w.Player.Vel = engine.Vec2{}
		w.collectCoins()
		if w.Level.TileAt(c[0], c[1]).Glyph == 'o' {
			t.Errorf("the coin at column %d row %d was not collected", c[0], c[1])
		}
		w.Level.Restore()
	}
}

// TestRespawnRestoresTheTerrain is the regression test for a respawn that
// cleared its bookkeeping but left the tiles changed: a used question block
// stayed solid and empty, a smashed brick stayed missing, a taken coin stayed
// gone, for the rest of the run.
func TestRespawnRestoresTheTerrain(t *testing.T) {
	// Load from disk: the snapshot is taken at load, so that is the honest
	// starting point. Change some tiles, respawn, check they are back.
	lv, err := level.LoadEmbedded("1-1")
	if err != nil {
		t.Fatal(err)
	}
	sm := engine.NewStateMachine()
	sm.StartRun()
	w := NewWorld(lv, &Default, sm, 3)

	var q, b, coin [2]int
	foundQ, foundB, foundC := false, false, false
	for y := 0; y < lv.Height; y++ {
		for x := 0; x < lv.Width; x++ {
			switch lv.TileAt(x, y).Glyph {
			case '?':
				if !foundQ {
					q, foundQ = [2]int{x, y}, true
				}
			case 'B':
				if !foundB {
					b, foundB = [2]int{x, y}, true
				}
			case 'o':
				if !foundC {
					coin, foundC = [2]int{x, y}, true
				}
			}
		}
	}
	if !foundQ || !foundB || !foundC {
		t.Fatal("1-1 is missing a question block, a brick or a coin")
	}

	w.Level.SetTile(q[0], q[1], level.Tile{Kind: level.KindSolid, Glyph: '#'})
	w.Level.SetTile(b[0], b[1], level.Tile{Kind: level.KindEmpty, Glyph: ' '})
	w.Level.SetTile(coin[0], coin[1], level.Tile{Kind: level.KindEmpty, Glyph: ' '})
	w.Level.Contents[0].Count = 0

	w.Reset()

	if g := w.Level.TileAt(q[0], q[1]).Glyph; g != '?' {
		t.Errorf("the used question block is %q after respawning, want it back as '?'", string(g))
	}
	if g := w.Level.TileAt(b[0], b[1]).Glyph; g != 'B' {
		t.Errorf("the smashed brick is %q after respawning, want it back", string(g))
	}
	if g := w.Level.TileAt(coin[0], coin[1]).Glyph; g != 'o' {
		t.Errorf("the taken coin is %q after respawning, want it back", string(g))
	}
	if w.Level.Contents[0].Count == 0 {
		t.Error("a block's payload count was not restored")
	}
}
