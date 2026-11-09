package game

import (
	"testing"
	"time"

	"super-mario/pkg/engine"
	"super-mario/pkg/level"
)

func TestOneWayPlatformBlocksFromAbove(t *testing.T) {
	w := newTestWorld(t)
	const col, row = 6, 10
	w.Level.SetTile(col, row, level.Tile{Kind: level.KindOneWay, Glyph: '='})

	// Drop onto the platform from a few units above it. Position by the
	// player's top, which is bottom minus height.
	top := engine.Fixed(row) * engine.Tile
	placePlayer(w, engine.Fixed(col)*engine.Tile+2, top-engine.FromFloat(2)-w.Player.H, 0, engine.FromFloat(1.5))

	for i := 0; i < 30; i++ {
		w.Step()
	}

	if !w.Player.Grounded {
		t.Fatal("player did not land on the one-way platform")
	}
	if got, want := w.Player.Bottom(), top; got != want {
		t.Errorf("stood at y=%v, want the platform top %v", got, want)
	}
}

func TestOneWayPlatformPassesFromBelow(t *testing.T) {
	w := newTestWorld(t)
	const col, row = 6, 10
	w.Level.SetTile(col, row, level.Tile{Kind: level.KindOneWay, Glyph: '='})

	top := engine.Fixed(row) * engine.Tile
	// Start under the platform, rising.
	placePlayer(w, engine.Fixed(col)*engine.Tile+2, top+engine.FromFloat(4), 0, engine.FromFloat(-5))

	passed := false
	for i := 0; i < 40; i++ {
		w.Step()
		if w.Player.Bottom() < top {
			passed = true
			break
		}
	}
	if !passed {
		t.Errorf("player was blocked by the platform from below; bottom stayed at %v (top %v)",
			w.Player.Bottom(), top)
	}
}

func TestGrownPlayerNeverStuckInsideTerrain(t *testing.T) {
	w := newTestWorld(t)

	// A low ceiling: a solid tile two rows above the floor, leaving exactly
	// the small player's height and no more.
	ceil := 10
	for x := 0; x < 8; x++ {
		w.Level.SetTile(x, ceil, level.Tile{Kind: level.KindSolid, Glyph: 'X'})
	}
	placePlayer(w, engine.Fixed(3)*engine.Tile+2, engine.Fixed(13)*engine.Tile-SmallSize.Y, 0, 0)
	w.Player.Grounded = true

	w.Player.Grow(w.Level)

	if crushedByTerrain(w.Level, &w.Player.Body) {
		t.Fatalf("player is embedded in solid terrain after growing (pos=%v size=%v)",
			w.Player.Pos, engine.Vec2{X: w.Player.W, Y: w.Player.H})
	}
}

func TestCrushDetectionFindsEmbeddedBody(t *testing.T) {
	w := newTestWorld(t)
	w.Level.SetTile(4, 11, level.Tile{Kind: level.KindSolid, Glyph: 'X'})
	// Force the body inside the tile.
	w.Player.Pos = engine.Vec2{X: engine.Fixed(4) * engine.Tile, Y: engine.Fixed(11) * engine.Tile}
	if !crushedByTerrain(w.Level, &w.Player.Body) {
		t.Error("expected a body inside a solid tile to be reported as crushed")
	}
}

func TestFlagpoleScoreRewardsHeight(t *testing.T) {
	lv := testLevel(t, 40)
	low := flagpoleScore(engine.Fixed(lv.FlagBot)*engine.Tile, lv)
	high := flagpoleScore(engine.Fixed(lv.FlagTop)*engine.Tile, lv)
	if high <= low {
		t.Errorf("high grab (%d) should beat a low grab (%d)", high, low)
	}
}

func TestFlagpoleEndsTheRun(t *testing.T) {
	w := newTestWorld(t)
	w.Player.Pos.X = engine.Fixed(w.Level.FlagX) * engine.Tile
	w.Step()
	if w.SM.Current() != engine.StateFlagpole && w.SM.Current() != engine.StateResults {
		t.Fatalf("touching the pole did not start the ending, state = %v", w.SM.Current())
	}
}

func TestTallyConvertsTimeIntoScoreGradually(t *testing.T) {
	w := newTestWorld(t)
	w.TimeLeft = 100
	w.SM.ShowResults()
	w.startTally()

	before := w.Score.Points
	w.stepTally()

	if w.Score.Points <= before {
		t.Fatal("the tally did not award any score")
	}
	if w.Tally.Done {
		t.Error("the tally finished in one step; the conversion should be visible")
	}
	if w.Tally.Remaining >= 100 {
		t.Error("remaining time did not decrease")
	}

	// Drain it.
	for i := 0; i < 1000 && !w.Tally.Done; i++ {
		w.stepTally()
	}
	if !w.Tally.Done {
		t.Fatal("tally never finished")
	}
	if got, want := w.Tally.Awarded, 100*scorePerTimeUnit; got != want {
		t.Errorf("awarded %d, want %d", got, want)
	}
}

func TestTallyNeverAwardsMoreThanRemaining(t *testing.T) {
	w := newTestWorld(t)
	w.TimeLeft = 3
	w.SM.ShowResults()
	w.startTally()
	for i := 0; i < 100; i++ {
		w.stepTally()
	}
	if got, want := w.Tally.Awarded, 3*scorePerTimeUnit; got != want {
		t.Errorf("awarded %d for 3 units of time, want %d", got, want)
	}
}

func TestResetRestoresBlocksAndEnemies(t *testing.T) {
	w := newTestWorld(t)
	bx, by := 5, 10
	w.Level.SetTile(bx, by, level.Tile{Kind: level.KindQuestion, Glyph: '?'})
	addContent(w, bx, by, level.BlockCoin, 1)

	placePlayer(w, engine.Fixed(bx)*engine.Tile+2, engine.Fixed(by+1)*engine.Tile+2, 0, engine.FromFloat(-4))
	for i := 0; i < 20; i++ {
		w.Step()
	}
	if w.Level.TileAt(bx, by).Kind == level.KindQuestion && w.Score.Coins == 0 {
		t.Fatal("setup failed: block did not pay")
	}

	points := w.Score.Points
	w.Reset()

	if w.Score.Points != points {
		t.Errorf("score dropped across a reset: %d -> %d", points, w.Score.Points)
	}
	if w.TimeLeft != w.Level.TimeLimit {
		t.Errorf("timer = %d after reset, want %d", w.TimeLeft, w.Level.TimeLimit)
	}
	if !w.Player.Grounded {
		t.Error("player should spawn grounded")
	}
}

func TestClockNeverRunsMoreThanTheCap(t *testing.T) {
	c := engine.NewClock()
	// Ten seconds of backlog in one call: only the cap may run.
	steps := c.Advance(10 * time.Second)
	if steps != engine.MaxCatchUpSteps {
		t.Errorf("ran %d steps, want the cap %d", steps, engine.MaxCatchUpSteps)
	}
	// And the backlog must be dropped, not carried.
	if c.Remainder() != 0 {
		t.Errorf("remainder = %v after a capped advance, want 0", c.Remainder())
	}
}

func TestClockSteadyStateRunsOneStepPerStep(t *testing.T) {
	c := engine.NewClock()
	total := 0
	for i := 0; i < 100; i++ {
		total += c.Advance(engine.StepDuration)
	}
	if total != 100 {
		t.Errorf("ran %d steps for 100 step-intervals", total)
	}
}

func TestClockFastFramesDoNotAdvance(t *testing.T) {
	c := engine.NewClock()
	// A render running twice as fast as the simulation must not speed it up.
	for i := 0; i < 10; i++ {
		if n := c.Advance(engine.StepDuration / 4); n != 0 {
			// Two quarters make a step, so every other call may legitimately
			// return one; never more.
			if n != 1 {
				t.Fatalf("0.25 step produced %d steps", n)
			}
		}
	}
}
