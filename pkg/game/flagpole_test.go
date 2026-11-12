package game

import (
	"testing"

	"super-mario/pkg/engine"
)

// grabPole puts the player against the pole at a height, in tiles from the top
// of the level, and steps once so the pole is grabbed.
func grabPole(t *testing.T, w *World, row int) {
	t.Helper()
	w.Player.Pos.X = engine.Fixed(w.Level.FlagX) * engine.Tile
	w.Player.Pos.Y = engine.Fixed(row) * engine.Tile
	w.Step()
	if !w.Player.SlideDown {
		t.Fatal("the pole was not grabbed")
	}
}

// stepUntil steps the world until done reports true, failing after max steps.
func stepUntil(t *testing.T, w *World, max int, what string, done func() bool) {
	t.Helper()
	for i := 0; i < max; i++ {
		if done() {
			return
		}
		w.Step()
	}
	t.Fatalf("never reached: %s", what)
}

// TestFlagCatchesUpThenSlidesWithHim checks the order asked for: the flag
// first races down to his height while he hangs on the pole, then the two
// slide down together at the same speed.
func TestFlagCatchesUpThenSlidesWithHim(t *testing.T) {
	w := newTestWorld(t)
	grabPole(t, w, w.Level.FlagTop+4)
	if w.flagPhase != flagCatchUp {
		t.Fatalf("phase %v right after the grab, want the flag catching up", w.flagPhase)
	}
	grabY := w.Player.Pos.Y
	target := grabY - engine.Fixed(w.Level.FlagTop+1)*engine.Tile

	steps := 0
	for w.flagPhase == flagCatchUp && steps < 60 {
		w.Step()
		steps++
	}
	if w.flagPhase != flagSliding {
		t.Fatal("the flag never caught up with him")
	}
	if w.Player.Pos.Y != grabY {
		t.Errorf("he moved from %v to %v while the flag was catching up, want him to hang still", grabY, w.Player.Pos.Y)
	}
	if w.FlagDrop != target {
		t.Errorf("flag stopped at %v, want level with him at %v", w.FlagDrop, target)
	}
	// Faster than the slide: the drop is 48 units, and at the slide speed it
	// would take 24 steps.
	if steps > 10 {
		t.Errorf("the flag took %d steps to catch up, want it quick", steps)
	}

	dropBefore, yBefore := w.FlagDrop, w.Player.Pos.Y
	for i := 0; i < 10; i++ {
		w.Step()
	}
	if w.FlagDrop-dropBefore != w.Player.Pos.Y-yBefore {
		t.Errorf("the flag moved %v while he moved %v, want them sliding together",
			w.FlagDrop-dropBefore, w.Player.Pos.Y-yBefore)
	}
	if w.Player.Pos.Y <= yBefore {
		t.Error("neither of them slid down after the catch-up")
	}
}

// TestFlagSlidesDownWithMario checks the cloth drops while he slides.
func TestFlagSlidesDownWithMario(t *testing.T) {
	w := newTestWorld(t)
	grabPole(t, w, w.Level.FlagTop+1)
	start := w.FlagDrop
	for i := 0; i < 20; i++ {
		w.Step()
	}
	if w.FlagDrop <= start {
		t.Errorf("the flag did not move while he slid: %v -> %v", start, w.FlagDrop)
	}
}

// TestFlagReachesTheBottomWithHim checks both end up at the foot of the pole.
func TestFlagReachesTheBottomWithHim(t *testing.T) {
	w := newTestWorld(t)
	grabPole(t, w, w.Level.FlagTop+1)
	stepUntil(t, w, 2000, "walking to the castle", func() bool { return w.flagPhase == flagWalking })
	floor := engine.Fixed(w.Level.FlagBot+1) * engine.Tile
	if w.Player.Pos.Y+w.Player.H != floor {
		t.Errorf("he is at %v, want his feet on the bottom of the pole at %v", w.Player.Pos.Y+w.Player.H, floor)
	}
	maxDrop := floor - engine.FromFloat(flagClothHeight) - engine.Fixed(w.Level.FlagTop+1)*engine.Tile
	if w.FlagDrop != maxDrop {
		t.Errorf("flag dropped %v, want %v", w.FlagDrop, maxDrop)
	}
}

// TestWalksToTheCastleThenResults checks the order: slide, walk, vanish, and
// only then the results screen.
func TestWalksToTheCastleThenResults(t *testing.T) {
	w := newTestWorld(t)
	grabPole(t, w, w.Level.FlagTop+1)
	stepUntil(t, w, 2000, "walking", func() bool { return w.flagPhase == flagWalking })
	if w.SM.Current() != engine.StateFlagpole {
		t.Fatalf("state %v while walking, want the flagpole sequence", w.SM.Current())
	}
	if w.Player.SlideDown {
		t.Error("still stuck to the pole while walking")
	}
	if w.Player.Pos.X <= engine.Fixed(w.Level.FlagX)*engine.Tile {
		t.Error("he did not step round to the far side of the pole")
	}
	stepUntil(t, w, 2000, "inside the castle", func() bool { return w.Player.Hidden })
	if w.SM.Current() != engine.StateFlagpole {
		t.Error("the results started before the pause after he went in")
	}
	stepUntil(t, w, 2000, "results", func() bool { return w.SM.Current() == engine.StateResults })
}

// TestClearedStaysSetToTheResults is the regression test for the results screen
// reading a game over once the slide ended.
func TestClearedStaysSetToTheResults(t *testing.T) {
	w := newTestWorld(t)
	if w.Cleared {
		t.Fatal("cleared before touching the pole")
	}
	grabPole(t, w, w.Level.FlagTop+1)
	stepUntil(t, w, 4000, "results", func() bool { return w.SM.Current() == engine.StateResults })
	if !w.Cleared {
		t.Error("the level is not marked cleared at the results screen")
	}
	w.Reset()
	if w.Cleared || w.Player.Hidden || w.FlagDrop != 0 || w.FlagBonus != nil {
		t.Error("a respawn left the end-of-level state behind")
	}
}

// TestPoleScoreTiers pins the score table: higher is worth more, every tier
// is reachable, and the pole score is banked when it is grabbed.
func TestPoleScoreTiers(t *testing.T) {
	lv := testLevel(t, 40)
	top := flagpoleScore(engine.Fixed(lv.FlagTop)*engine.Tile, lv)
	bottom := flagpoleScore(engine.Fixed(lv.FlagBot)*engine.Tile, lv)
	if top != 5000 {
		t.Errorf("top of the pole is worth %d, want 5000", top)
	}
	if bottom >= top {
		t.Errorf("the bottom (%d) is worth as much as the top (%d)", bottom, top)
	}
	seen := map[int]bool{}
	for y := lv.FlagTop; y <= lv.FlagBot; y++ {
		seen[flagpoleScore(engine.Fixed(y)*engine.Tile, lv)] = true
	}
	if len(seen) < len(flagTiers) {
		t.Errorf("only %d of %d score tiers can be reached", len(seen), len(flagTiers)+1)
	}

	w := newTestWorld(t)
	before := w.Score.Points
	grabPole(t, w, w.Level.FlagTop+1)
	if w.FlagBonus == nil || w.Score.Points != before+w.FlagBonus.Amount {
		t.Errorf("pole score not banked: points %d -> %d, bonus %+v", before, w.Score.Points, w.FlagBonus)
	}
}

// TestPoleScoreFloatsThenGoes checks the number shows for its time only.
func TestPoleScoreFloatsThenGoes(t *testing.T) {
	w := newTestWorld(t)
	grabPole(t, w, w.Level.FlagTop+1)
	if w.FlagBonus == nil {
		t.Fatal("no score showing after the grab")
	}
	for i := 0; i < FlagBonusSteps; i++ {
		w.Step()
	}
	if w.FlagBonus != nil {
		t.Error("the pole score is still showing after its time")
	}
}
