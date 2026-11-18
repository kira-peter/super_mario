package game

import (
	"testing"

	"super-mario/pkg/engine"
	"super-mario/pkg/level"
)

func TestZeroPlaytestOptionsChangeNothing(t *testing.T) {
	a := newTestWorld(t)
	b := newTestWorld(t)
	b.SetPlaytest(PlaytestOptions{})
	for i := 0; i < 120; i++ {
		a.Input.KeyDown(engine.KeyRight)
		b.Input.KeyDown(engine.KeyRight)
		a.Step()
		b.Step()
	}
	if a.Player.Pos != b.Player.Pos || a.Player.Form != b.Player.Form {
		t.Error("zero playtest options changed the simulation")
	}
}

func TestStartBig(t *testing.T) {
	w := newTestWorld(t)
	w.SetPlaytest(PlaytestOptions{StartBig: true})
	if w.Player.Form != FormBig {
		t.Fatalf("form = %v, want big", w.Player.Form)
	}
	// Still standing on the floor, not sunk into it.
	if !w.Player.Grounded || w.Player.Bottom() != engine.Fixed(13)*engine.Tile {
		t.Errorf("big Mario's feet are at %d, want on the floor", w.Player.Bottom())
	}
	// And again after a respawn.
	w.Player.Form = FormSmall
	w.Reset()
	if w.Player.Form != FormBig {
		t.Error("the next life did not start big")
	}
}

func TestInvincibleSurvivesEnemies(t *testing.T) {
	w := newTestWorld(t)
	w.SetPlaytest(PlaytestOptions{Invincible: true})
	w.hurtPlayer()
	if w.Player.Dead || w.SM.Is(engine.StateDying) {
		t.Error("an invincible small Mario died to a hit")
	}
}

func TestInvincibleSurvivesThePit(t *testing.T) {
	lv := testLevel(t, 80)
	for x := 10; x < 20; x++ {
		lv.SetTile(x, 13, level.Tile{Kind: level.KindEmpty, Glyph: ' '})
		lv.SetTile(x, 14, level.Tile{Kind: level.KindEmpty, Glyph: ' '})
	}
	sm := engine.NewStateMachine()
	sm.StartRun()
	w := NewWorld(lv, &Default, sm, 3)
	w.SetPlaytest(PlaytestOptions{Invincible: true})
	// Run straight into the pit, then let go once rescued.
	rescued := false
	for i := 0; i < 400; i++ {
		if !rescued {
			w.Input.KeyDown(engine.KeyRight)
		} else {
			w.Input.KeyUp(engine.KeyRight)
		}
		before := w.Player.Pos.Y
		w.Step()
		if before > engine.Fixed(13)*engine.Tile && w.Player.Pos.Y < before {
			rescued = true
		}
	}
	if w.SM.Is(engine.StateDying) || w.Player.Dead {
		t.Fatal("an invincible Mario died in a pit")
	}
	if !rescued {
		t.Fatal("Mario never fell in, so the test proves nothing")
	}
	if !w.Player.Grounded || w.Player.Pos.Y > engine.Fixed(13)*engine.Tile {
		t.Errorf("after the rescue Mario is at y=%d, grounded=%v; want back on the floor", w.Player.Pos.Y, w.Player.Grounded)
	}
}

func TestInvincibleOutlastsTheClock(t *testing.T) {
	w := newTestWorld(t)
	w.SetPlaytest(PlaytestOptions{Invincible: true})
	w.TimeLeft = 1
	for i := 0; i < 180; i++ {
		w.Step()
	}
	if w.SM.Is(engine.StateDying) {
		t.Error("the clock killed an invincible Mario")
	}
}

func TestWithoutInvincibleThePitKills(t *testing.T) {
	w := newTestWorld(t)
	w.Player.Pos.Y = engine.Fixed(w.Level.Height+1) * engine.Tile
	w.Step()
	if !w.SM.Is(engine.StateDying) {
		t.Error("falling out of the level no longer kills in the normal game")
	}
}

func TestStartAtColumn(t *testing.T) {
	w := newTestWorld(t)
	// A three-high stack at column 30: the start goes on top of it.
	for y := 10; y <= 12; y++ {
		w.Level.SetTile(30, y, level.Tile{Kind: level.KindSolid, Glyph: 'H'})
	}
	if !w.StartAtColumn(30) {
		t.Fatal("column 30 has a top to stand on")
	}
	if got := tileIndex(w.Player.CenterX()); got != 30 {
		t.Errorf("player is in column %d, want 30", got)
	}
	if w.Player.Bottom() != engine.Fixed(10)*engine.Tile {
		t.Errorf("player's feet at %d, want on top of the stack", w.Player.Bottom())
	}
	if w.CameraX == 0 {
		t.Error("the camera did not follow to column 30")
	}
}

func TestStartAtColumnOverAPit(t *testing.T) {
	w := newTestWorld(t)
	for _, y := range []int{13, 14} {
		w.Level.SetTile(40, y, level.Tile{Kind: level.KindEmpty, Glyph: ' '})
	}
	spawn := w.Player.Pos
	if w.StartAtColumn(40) {
		t.Fatal("a pit column was accepted as a start")
	}
	if w.Player.Pos != spawn {
		t.Error("a refused start still moved the player")
	}
}

func TestStartAtColumnKeepsBigOutOfLowGaps(t *testing.T) {
	w := newTestWorld(t)
	w.SetPlaytest(PlaytestOptions{StartBig: true})
	// A one-high gap at row 9 over a block at row 10, and open floor below.
	w.Level.SetTile(25, 8, level.Tile{Kind: level.KindSolid, Glyph: 'H'})
	w.Level.SetTile(25, 10, level.Tile{Kind: level.KindSolid, Glyph: 'H'})
	if !w.StartAtColumn(25) {
		t.Fatal("no start found")
	}
	if overlapsSolid(w.Level, w.Player.Rect()) {
		t.Error("big Mario was placed inside the terrain")
	}
}

func TestTraceRecordsJumpsAndDeath(t *testing.T) {
	w := newTestWorld(t)
	var tr Trace
	for i := 0; i < 60; i++ {
		w.Input.KeyDown(engine.KeyRight)
		if i == 10 {
			w.Input.KeyDown(engine.KeySpace)
		}
		w.Step()
		tr.Record(w)
	}
	jumps := 0
	for _, p := range tr.Points() {
		if p.Jump {
			jumps++
		}
	}
	if jumps != 1 {
		t.Errorf("recorded %d jumps, want 1", jumps)
	}
	w.killPlayer()
	w.Step()
	tr.Record(w)
	if tr.Death == nil {
		t.Error("the death was not recorded")
	}
}

func TestTraceKeepsTheLastMinute(t *testing.T) {
	w := newTestWorld(t)
	var tr Trace
	for i := 0; i < MaxTrace+500; i++ {
		w.Input.KeyDown(engine.KeyRight)
		w.Step()
		tr.Record(w)
	}
	pts := tr.Points()
	if len(pts) != MaxTrace || tr.Dropped != 500 {
		t.Fatalf("kept %d points, dropped %d", len(pts), tr.Dropped)
	}
	// Oldest first, and the newest is where the player is now.
	if pts[len(pts)-1].X != w.Player.CenterX() {
		t.Error("the newest point is not last")
	}
	for i := 1; i < len(pts); i++ {
		if pts[i].X < pts[i-1].X {
			t.Fatalf("points out of order at %d", i)
		}
	}
}
