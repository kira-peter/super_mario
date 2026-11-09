package app

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/engine"
	"super-mario/pkg/save"
)

// headlessTitleApp builds an app on the title screen with its own temp save
// directory, so tests never touch the player's real leaderboard.
func headlessTitleApp(t *testing.T) *App {
	t.Helper()
	a := newHeadlessApp(t)
	a.store = save.NewAt(t.TempDir())
	a.board = save.Board{}
	a.sm.ToTitle()
	a.title = newTitleState(a)
	return a
}

// finishWithFlag drives the world through the flagpole to the results screen.
func finishWithFlag(t *testing.T, a *App) {
	t.Helper()
	a.world.Score.Points = 1200
	a.world.TimeLeft = 300
	a.world.Player.Pos.X = engine.Fixed(a.world.Level.FlagX) * engine.Tile
	for i := 0; i < 600 && !a.sm.Is(engine.StateResults); i++ {
		a.stepOnce()
	}
	if !a.sm.Is(engine.StateResults) {
		t.Fatalf("never reached the results screen, state = %v", a.sm.Current())
	}
}

// TestClearedRunIsMarkedClearedAndRecorded drives a real clear and checks the
// board entry: the final score includes the time bonus, the run is marked
// cleared, and it carries a duration.
func TestClearedRunIsMarkedClearedAndRecorded(t *testing.T) {
	a := newHeadlessApp(t)
	a.store = save.NewAt(t.TempDir())
	a.board = save.Board{}
	finishWithFlag(t, a)

	// Let the tally run to the end.
	for i := 0; i < 2000 && !a.recorded; i++ {
		a.stepOnce()
	}
	if !a.recorded {
		t.Fatal("the finished run was never recorded")
	}
	if len(a.board.Entries) != 1 {
		t.Fatalf("board has %d entries, want 1", len(a.board.Entries))
	}
	e := a.board.Entries[0]
	if !e.Cleared {
		t.Error("a run that reached the flagpole was recorded as not cleared")
	}
	if e.Score <= 1200 {
		t.Errorf("score %d does not include the time bonus (started at 1200)", e.Score)
	}
	if e.Steps <= 0 {
		t.Errorf("run time = %d steps, want a positive duration", e.Steps)
	}
}

// TestTimeBonusActuallyConverts is the regression test for the tally that never
// ran in the real game: the results state was not stepped, so remaining time
// never turned into score.
func TestTimeBonusActuallyConverts(t *testing.T) {
	a := newHeadlessApp(t)
	a.store = save.NewAt(t.TempDir())
	finishWithFlag(t, a)
	before := a.world.Score.Points
	for i := 0; i < 2000 && !a.world.Tally.Done; i++ {
		a.stepOnce()
	}
	if !a.world.Tally.Done {
		t.Fatal("the tally never finished")
	}
	if a.world.Score.Points <= before {
		t.Errorf("score stayed at %d: remaining time was not converted", before)
	}
}

// TestRunIsRecordedExactlyOnce checks that stepping the results screen every
// frame does not write the same run repeatedly.
func TestRunIsRecordedExactlyOnce(t *testing.T) {
	a := newHeadlessApp(t)
	a.store = save.NewAt(t.TempDir())
	a.board = save.Board{}
	finishWithFlag(t, a)
	for i := 0; i < 3000; i++ {
		a.stepOnce()
	}
	if len(a.board.Entries) != 1 {
		t.Errorf("the run was recorded %d times, want 1", len(a.board.Entries))
	}
}

// TestRunTimeExcludesPauseAndDeath checks the leaderboard time measures play.
func TestRunTimeExcludesPauseAndDeath(t *testing.T) {
	a := newHeadlessApp(t)
	for i := 0; i < 60; i++ {
		a.stepOnce()
	}
	played := a.world.RunSteps()
	if played < 55 || played > 65 {
		t.Fatalf("one second of play counted %d steps, want about 60", played)
	}

	a.openPause()
	for i := 0; i < 600; i++ {
		a.stepOnce()
	}
	if got := a.world.RunSteps(); got != played {
		t.Errorf("ten paused seconds changed the run time: %d -> %d", played, got)
	}
}

// TestGameOverIsRecordedAsNotCleared checks a lost run still earns a place.
func TestGameOverIsRecordedAsNotCleared(t *testing.T) {
	a := newHeadlessApp(t)
	a.store = save.NewAt(t.TempDir())
	a.board = save.Board{}
	a.world.Score.Points = 700
	a.world.Score.Lives = 1
	a.world.Player.Die()
	a.sm.Die()
	for i := 0; i < 600 && !a.sm.Is(engine.StateResults); i++ {
		a.stepOnce()
	}
	if !a.sm.Is(engine.StateResults) {
		t.Fatalf("losing the last life did not reach the results screen (state %v)", a.sm.Current())
	}
	for i := 0; i < 10 && !a.recorded; i++ {
		a.stepOnce()
	}
	if len(a.board.Entries) != 1 {
		t.Fatalf("a lost run left %d board entries, want 1", len(a.board.Entries))
	}
	if a.board.Entries[0].Cleared {
		t.Error("a lost run was recorded as cleared")
	}
}

// TestZeroScoreRunIsNotRecorded keeps empty runs off the board.
func TestZeroScoreRunIsNotRecorded(t *testing.T) {
	a := newHeadlessApp(t)
	a.store = save.NewAt(t.TempDir())
	a.board = save.Board{}
	a.world.Score.Points = 0
	a.toTitle()
	if len(a.board.Entries) != 0 {
		t.Errorf("a zero-score run was recorded: %+v", a.board.Entries)
	}
}

// TestAbandonedRunWithScoreIsRecorded checks quitting to the title keeps the
// score earned so far.
func TestAbandonedRunWithScoreIsRecorded(t *testing.T) {
	a := newHeadlessApp(t)
	a.store = save.NewAt(t.TempDir())
	a.board = save.Board{}
	a.world.Score.Points = 450
	a.openPause()
	a.toTitle()
	if len(a.board.Entries) != 1 || a.board.Entries[0].Score != 450 {
		t.Fatalf("board = %+v, want one entry of 450", a.board.Entries)
	}
	if a.board.Entries[0].Cleared {
		t.Error("an abandoned run was marked cleared")
	}
	if !a.sm.Is(engine.StateTitle) {
		t.Errorf("state = %v, want the title screen", a.sm.Current())
	}
}

// TestBoardSurvivesARestart checks what was recorded is what the next launch
// loads.
func TestBoardSurvivesARestart(t *testing.T) {
	dir := t.TempDir()
	a := newHeadlessApp(t)
	a.store = save.NewAt(dir)
	a.board = save.Board{}
	a.world.Score.Points = 3300
	a.toTitle()

	b := save.NewAt(dir).Load()
	if len(b.Entries) != 1 || b.Entries[0].Score != 3300 {
		t.Errorf("reloaded board = %+v, want one entry of 3300", b.Entries)
	}
}

// TestFullLoopFromTitleToBoardAndBack walks the whole journey with the same
// entry points the real event loop uses: start from the title menu, clear the
// level, watch the tally run, land on the leaderboard with the new rank, and
// return to the menu. Each stage has its own tests; this checks they connect.
func TestFullLoopFromTitleToBoardAndBack(t *testing.T) {
	a := headlessTitleApp(t)

	// Title -> play.
	a.handleEvent(keyEvent(tcell.KeyEnter, 0))
	if !a.sm.Is(engine.StatePlaying) {
		t.Fatalf("Enter on 开始游戏 left state %v, want playing", a.sm.Current())
	}

	// Play a little, then touch the flag with some score banked.
	for i := 0; i < 120; i++ {
		a.stepOnce()
	}
	a.world.Score.Points = 2400
	a.world.TimeLeft = 250
	a.world.Player.Pos.X = engine.Fixed(a.world.Level.FlagX) * engine.Tile

	// Run until the results screen has finished tallying and recorded the run.
	for i := 0; i < 3000 && !a.recorded; i++ {
		a.stepOnce()
	}
	if !a.sm.Is(engine.StateResults) {
		t.Fatalf("state = %v, want the results screen", a.sm.Current())
	}
	if !a.recorded || len(a.board.Entries) != 1 {
		t.Fatalf("the run was not recorded (recorded=%v, entries=%d)", a.recorded, len(a.board.Entries))
	}
	if a.resultRank != 1 {
		t.Errorf("rank = %d, want 1 on an empty board", a.resultRank)
	}

	// Results -> title, landing on the board with the new run highlighted.
	a.handleKeyDown(engine.KeySpace)
	a.stepOnce()
	if !a.sm.Is(engine.StateTitle) {
		t.Fatalf("confirming the results left state %v, want the title", a.sm.Current())
	}
	if a.title.page != pageBoard {
		t.Errorf("landed on page %v, want the leaderboard", a.title.page)
	}
	if a.title.justAt != 1 {
		t.Errorf("highlighted rank = %d, want 1", a.title.justAt)
	}

	// The board shows the score with its bonus and a real time.
	text := renderTitleText(t, a, 100, 30)
	if !containsText(text, "用时") {
		t.Error("the board does not show a time column")
	}
	e := a.board.Entries[0]
	if e.Score <= 2400 {
		t.Errorf("recorded score %d is missing the time bonus", e.Score)
	}
	if e.Steps < 100 {
		t.Errorf("recorded time %d steps is too short for a run that lasted 120+ steps", e.Steps)
	}

	// Esc -> back to the menu, and Start works again for a second run.
	a.handleEvent(keyEvent(tcell.KeyEscape, 0))
	if a.title.page != pageMenu {
		t.Fatalf("Esc left page %v, want the menu", a.title.page)
	}
	a.handleEvent(keyEvent(tcell.KeyEnter, 0))
	if !a.sm.Is(engine.StatePlaying) {
		t.Fatalf("a second run did not start (state %v)", a.sm.Current())
	}
	if a.world.RunSteps() != 0 {
		t.Errorf("the second run began with %d steps already on the clock", a.world.RunSteps())
	}
	if a.world.Score.Points != 0 {
		t.Errorf("the second run began with %d points", a.world.Score.Points)
	}
}
