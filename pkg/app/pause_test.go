package app

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/engine"
)

func keyEvent(k tcell.Key, r rune) *tcell.EventKey {
	return tcell.NewEventKey(k, r, tcell.ModNone)
}

// TestEscapeOpensTheMenuInsteadOfFreezing is the regression test for the report
// that pressing Esc "froze" the game.
//
// Esc used to switch the state machine to paused and nothing else: no menu was
// drawn, and no code read any key while paused, so there was no way back.
func TestEscapeOpensTheMenuInsteadOfFreezing(t *testing.T) {
	a := newHeadlessApp(t)

	a.handleEvent(keyEvent(tcell.KeyEscape, 0))

	if !a.sm.Is(engine.StatePaused) {
		t.Fatalf("state = %v after Esc, want paused", a.sm.Current())
	}
	if a.menu == nil {
		t.Fatal("Esc paused the game but no menu was opened; the player has no way to resume")
	}
}

// TestEscapeAgainResumes checks the way back out.
func TestEscapeAgainResumes(t *testing.T) {
	a := newHeadlessApp(t)
	a.handleEvent(keyEvent(tcell.KeyEscape, 0))
	a.handleEvent(keyEvent(tcell.KeyEscape, 0))

	if !a.sm.Is(engine.StatePlaying) {
		t.Errorf("state = %v after Esc twice, want playing", a.sm.Current())
	}
	if a.menu != nil {
		t.Error("the menu stayed open after resuming")
	}
}

// TestPauseFreezesTheWorld checks nothing advances behind the menu.
func TestPauseFreezesTheWorld(t *testing.T) {
	a := newHeadlessApp(t)
	a.handleEvent(keyEvent(tcell.KeyEscape, 0))

	x, tm := a.world.Player.Pos.X, a.world.TimeLeft
	for i := 0; i < 200; i++ {
		a.stepOnce()
	}
	if a.world.Player.Pos.X != x {
		t.Error("the player moved while paused")
	}
	if a.world.TimeLeft != tm {
		t.Error("the timer ran while paused")
	}
}

// TestSpaceDoesNotJumpBehindTheMenu checks the menu owns the keyboard, so a key
// the game binds to jump does not leak through.
func TestSpaceDoesNotJumpBehindTheMenu(t *testing.T) {
	a := newHeadlessApp(t)
	a.handleEvent(keyEvent(tcell.KeyEscape, 0))
	a.menu.cursor = 3 // park on "quit" so Space activating it would be loud
	a.menu.cursor = 0 // resume

	a.handleEvent(keyEvent(tcell.KeyRune, ' '))

	// Space on "continue game" resumes; it must not also have queued a jump.
	if a.input.Held(engine.ActionJump) {
		t.Error("Space leaked into the game as a jump while the menu was open")
	}
}

// TestMenuNavigationWraps checks the cursor moves and wraps.
func TestMenuNavigationWraps(t *testing.T) {
	a := newHeadlessApp(t)
	a.handleEvent(keyEvent(tcell.KeyEscape, 0))
	n := len(a.menu.items)

	a.handleEvent(keyEvent(tcell.KeyUp, 0))
	if a.menu.cursor != n-1 {
		t.Errorf("cursor = %d after Up from the top, want it to wrap to %d", a.menu.cursor, n-1)
	}
	a.handleEvent(keyEvent(tcell.KeyDown, 0))
	if a.menu.cursor != 0 {
		t.Errorf("cursor = %d after Down from the bottom, want it to wrap to 0", a.menu.cursor)
	}
}

// TestMenuQuitExits checks the quit entry ends the game.
func TestMenuQuitExits(t *testing.T) {
	a := newHeadlessApp(t)
	a.handleEvent(keyEvent(tcell.KeyEscape, 0))
	a.menu.cursor = len(a.menu.items) - 1
	a.handleEvent(keyEvent(tcell.KeyEnter, 0))

	if !a.quit {
		t.Error("choosing quit did not end the game")
	}
}

// TestMenuRestartStartsAFreshRun checks restart resets score and lives.
func TestMenuRestartStartsAFreshRun(t *testing.T) {
	a := newHeadlessApp(t)
	a.world.Score.Points = 5000
	a.world.Score.Lives = 1
	a.handleEvent(keyEvent(tcell.KeyEscape, 0))
	a.menu.cursor = 1
	a.handleEvent(keyEvent(tcell.KeyEnter, 0))

	if a.world.Score.Points != 0 {
		t.Errorf("score = %d after restart, want 0", a.world.Score.Points)
	}
	if a.world.Score.Lives != 3 {
		t.Errorf("lives = %d after restart, want 3", a.world.Score.Lives)
	}
	if !a.sm.Is(engine.StatePlaying) {
		t.Errorf("state = %v after restart, want playing", a.sm.Current())
	}
}

// TestHeldDirectionDoesNotCarryThroughThePause checks that a direction held
// when the menu opened does not run Mario forward on resume.
func TestHeldDirectionDoesNotCarryThroughThePause(t *testing.T) {
	a := newHeadlessApp(t)
	a.handleKeyDown(engine.KeyRight)
	a.handleEvent(keyEvent(tcell.KeyEscape, 0))
	a.handleEvent(keyEvent(tcell.KeyEscape, 0))

	if a.input.Held(engine.ActionRight) {
		t.Error("the right key was still held after resuming")
	}
}

// TestPKeyAlsoPauses keeps the original binding working.
func TestPKeyAlsoPauses(t *testing.T) {
	a := newHeadlessApp(t)
	a.handleEvent(keyEvent(tcell.KeyRune, 'p'))
	if !a.sm.Is(engine.StatePaused) || a.menu == nil {
		t.Errorf("P did not open the pause menu (state %v)", a.sm.Current())
	}
}

// TestCtrlCQuitsEvenWhilePaused guards the one key that must always work.
func TestCtrlCQuitsEvenWhilePaused(t *testing.T) {
	a := newHeadlessApp(t)
	a.handleEvent(keyEvent(tcell.KeyEscape, 0))
	a.handleEvent(keyEvent(tcell.KeyCtrlC, 0))
	if !a.quit {
		t.Error("Ctrl-C did not quit from the pause menu")
	}
}

// TestMenuLabelsStartInTheSameColumn is the regression test for a selected
// entry drawing to the right of the unselected ones.
//
// Each row used to be centred separately, and the selected row carried a
// different prefix, so its label landed in a different column.
func TestMenuLabelsStartInTheSameColumn(t *testing.T) {
	a := newHeadlessApp(t)
	a.handleEvent(keyEvent(tcell.KeyEscape, 0))

	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Skipf("no simulation screen: %v", err)
	}
	defer sim.Fini()
	sim.SetSize(100, 30)
	a.screen = sim

	// A wide rune takes two cells and the second holds no character of its
	// own, so the label is matched on each rune's first cell, stepping by the
	// rune's display width.
	firstLabelCol := func(row int, label string) int {
		w, _ := sim.Size()
		want := []rune(label)
		for x := 0; x < w; x++ {
			cx, ok := x, true
			for _, r := range want {
				got, _, _, _ := sim.GetContent(cx, row)
				if got != r {
					ok = false
					break
				}
				if r > 0x1100 {
					cx += 2
				} else {
					cx++
				}
			}
			if ok {
				return x
			}
		}
		return -1
	}

	cols := map[int]string{}
	for cursor := range a.menu.items {
		a.menu.cursor = cursor
		sim.Clear()
		a.drawPauseMenu(100, 30)
		sim.Show()

		// Find the row the label was drawn on, then its first column.
		for row := 0; row < 30; row++ {
			if c := firstLabelCol(row, a.menu.items[cursor].label); c >= 0 {
				cols[cursor] = a.menu.items[cursor].label
				_ = c
				break
			}
		}
	}

	// Collect each label's start column with the cursor on it and with the
	// cursor elsewhere; they must agree.
	start := func(cursor, item int) int {
		a.menu.cursor = cursor
		sim.Clear()
		a.drawPauseMenu(100, 30)
		sim.Show()
		for row := 0; row < 30; row++ {
			if c := firstLabelCol(row, a.menu.items[item].label); c >= 0 {
				return c
			}
		}
		return -1
	}
	for item := range a.menu.items {
		selected := start(item, item)
		other := start((item+1)%len(a.menu.items), item)
		if selected < 0 || other < 0 {
			t.Fatalf("item %d (%q) was not drawn", item, a.menu.items[item].label)
		}
		if selected != other {
			t.Errorf("%q starts at column %d when selected but %d when not; the highlight shifts it",
				a.menu.items[item].label, selected, other)
		}
	}
}

// TestMenuIsGoneTheFrameAfterResuming is the regression test for the pause
// window lingering for a moment after returning to the game.
//
// The game layer only writes cells that differ from the previous frame, and the
// menu is drawn straight onto the screen, outside that comparison. Cells under
// the menu looked unchanged to the game layer, so nothing repainted them on
// resume and the menu stayed until something else forced a full redraw.
func TestMenuIsGoneTheFrameAfterResuming(t *testing.T) {
	a := newHeadlessApp(t)
	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Skipf("no simulation screen: %v", err)
	}
	defer sim.Fini()
	sim.SetSize(100, 30)
	a.screen = sim

	// Settle a few frames so the previous-frame record is populated.
	for i := 0; i < 3; i++ {
		a.stepOnce()
		_ = a.draw()
	}
	clean := snapshotScreen(sim)

	a.handleEvent(keyEvent(tcell.KeyEscape, 0))
	_ = a.draw() // menu on screen
	a.handleEvent(keyEvent(tcell.KeyEscape, 0))
	_ = a.draw() // the very next frame after resuming

	after := snapshotScreen(sim)
	if diff := countDiff(clean, after); diff != 0 {
		t.Errorf("%d cells still differ from the game picture one frame after resuming; the menu is lingering", diff)
	}
}

// snapshotScreen copies every cell's rune and style.
func snapshotScreen(s tcell.SimulationScreen) []string {
	w, h := s.Size()
	out := make([]string, 0, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, _, st, _ := s.GetContent(x, y)
			fg, bg, _ := st.Decompose()
			out = append(out, string(r)+fg.String()+bg.String())
		}
	}
	return out
}

func countDiff(a, b []string) int {
	n := 0
	for i := range a {
		if a[i] != b[i] {
			n++
		}
	}
	return n
}
