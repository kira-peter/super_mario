package app

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/engine"
	"super-mario/pkg/save"
)

// TestTitleStartsOnTheMenuWithStartSelected checks the default is the thing
// most people want.
func TestTitleStartsOnTheMenuWithStartSelected(t *testing.T) {
	a := headlessTitleApp(t)
	if a.title.page != pageMenu {
		t.Errorf("title page = %v, want the main menu", a.title.page)
	}
	if got := a.title.menu.items[a.title.menu.cursor].label; got != "开始游戏" {
		t.Errorf("default selection = %q, want 开始游戏", got)
	}
}

// TestEnterStartsTheGame replaces the old "press space to start" behaviour.
func TestEnterStartsTheGame(t *testing.T) {
	a := headlessTitleApp(t)
	a.handleEvent(keyEvent(tcell.KeyEnter, 0))
	if !a.sm.Is(engine.StatePlaying) {
		t.Errorf("state = %v after Enter on 开始游戏, want playing", a.sm.Current())
	}
}

// TestSpaceActivatesTheSelectedEntry keeps the old muscle memory working.
func TestSpaceActivatesTheSelectedEntry(t *testing.T) {
	a := headlessTitleApp(t)
	a.handleEvent(keyEvent(tcell.KeyRune, ' '))
	if !a.sm.Is(engine.StatePlaying) {
		t.Errorf("state = %v after Space on 开始游戏, want playing", a.sm.Current())
	}
}

// TestSpaceDoesNotJumpTheMomentTheRunStarts is the reason the title has its own
// key routing: Space is both "confirm" on the title and "jump" in play, and a
// press that starts the run must not also be the first jump.
func TestSpaceDoesNotJumpTheMomentTheRunStarts(t *testing.T) {
	a := headlessTitleApp(t)
	a.handleEvent(keyEvent(tcell.KeyRune, ' '))
	for i := 0; i < 3; i++ {
		a.stepOnce()
	}
	if a.world.Player.Jumping {
		t.Error("the Space that started the run also made Mario jump")
	}
}

func menuIndex(t *testing.T, a *App, label string) int {
	t.Helper()
	for i, it := range a.title.menu.items {
		if it.label == label {
			return i
		}
	}
	t.Fatalf("no menu entry %q", label)
	return -1
}

// TestMenuEntriesOpenTheirPages checks every entry reaches its page and Esc
// comes back.
func TestMenuEntriesOpenTheirPages(t *testing.T) {
	for label, page := range map[string]titlePage{"排行榜": pageBoard, "操作说明": pageHelp} {
		a := headlessTitleApp(t)
		a.title.menu.cursor = menuIndex(t, a, label)
		a.handleEvent(keyEvent(tcell.KeyEnter, 0))
		if a.title.page != page {
			t.Errorf("%s opened page %v, want %v", label, a.title.page, page)
		}
		a.handleEvent(keyEvent(tcell.KeyEscape, 0))
		if a.title.page != pageMenu {
			t.Errorf("%s: Esc did not return to the menu", label)
		}
		if !a.sm.Is(engine.StateTitle) {
			t.Errorf("%s: left the title screen", label)
		}
	}
}

// TestQuitEntryExits checks the exit entry and Esc on the main menu both quit.
func TestQuitEntryExits(t *testing.T) {
	a := headlessTitleApp(t)
	a.title.menu.cursor = menuIndex(t, a, "退出")
	a.handleEvent(keyEvent(tcell.KeyEnter, 0))
	if !a.quit {
		t.Error("the 退出 entry did not quit")
	}

	b := headlessTitleApp(t)
	b.handleEvent(keyEvent(tcell.KeyEscape, 0))
	if !b.quit {
		t.Error("Esc on the main menu did not quit")
	}
}

// TestEscOnASubPageDoesNotQuit guards against a back key being a quit key.
func TestEscOnASubPageDoesNotQuit(t *testing.T) {
	a := headlessTitleApp(t)
	a.openTitlePage(pageBoard)
	a.handleEvent(keyEvent(tcell.KeyEscape, 0))
	if a.quit {
		t.Error("Esc on the leaderboard quit the game instead of going back")
	}
}

// TestTitleShowsNoHighScore is the requirement that the best score lives on the
// leaderboard, not on the title.
func TestTitleShowsNoHighScore(t *testing.T) {
	a := headlessTitleApp(t)
	a.board.Add(save.Entry{Score: 424242, Steps: 100, Level: "1-1", Cleared: true})
	text := renderTitleText(t, a, 100, 34)
	for _, banned := range []string{"424242", "HIGH", "最高分"} {
		if containsText(text, banned) {
			t.Errorf("the title screen shows %q; the best score belongs on the leaderboard", banned)
		}
	}
}

// TestBoardShowsTimeForEachRun is the requirement that the board lists the
// time alongside the score.
func TestBoardShowsTimeForEachRun(t *testing.T) {
	a := headlessTitleApp(t)
	a.board.Add(save.Entry{Score: 5000, Steps: 4994, Level: "1-1", Cleared: true}) // 1:23.2
	a.board.Add(save.Entry{Score: 3000, Steps: 3600, Level: "1-1", Cleared: true}) // 1:00.0
	a.openTitlePage(pageBoard)

	text := renderTitleText(t, a, 100, 34)
	for _, want := range []string{"005000", "1:23.2", "003000", "1:00.0", "用时"} {
		if !containsText(text, want) {
			t.Errorf("the leaderboard does not show %q", want)
		}
	}
}

// TestBoardEntryWithoutATimeShowsADash covers a score migrated from an old
// save, which never recorded a duration.
func TestBoardEntryWithoutATimeShowsADash(t *testing.T) {
	a := headlessTitleApp(t)
	a.board.Add(save.Entry{Score: 777, Level: "1-1"})
	a.openTitlePage(pageBoard)
	text := renderTitleText(t, a, 100, 34)
	if containsText(text, "0:00.0") {
		t.Error("a missing time was shown as 0:00.0, which reads as a real instant run")
	}
	if !containsText(text, "--") {
		t.Error("a missing time should show a dash")
	}
}

// TestEmptyBoardSaysSo checks the empty state is a message, not a blank page.
func TestEmptyBoardSaysSo(t *testing.T) {
	a := headlessTitleApp(t)
	a.openTitlePage(pageBoard)
	if text := renderTitleText(t, a, 100, 34); !containsText(text, "还没有成绩") {
		t.Error("an empty leaderboard shows nothing to explain itself")
	}
}

// TestLogoSpellsBothWords checks the dot-matrix font has every letter it
// needs, so a typo in a glyph table fails here rather than as a gap on screen.
func TestLogoSpellsBothWords(t *testing.T) {
	for _, word := range []string{"SUPER", "MARIO"} {
		for _, r := range word {
			if _, ok := glyphs[r]; !ok {
				t.Errorf("the logo font has no glyph for %q", string(r))
			}
		}
	}
	for r, g := range glyphs {
		for y, row := range g {
			if len(row) != 5 {
				t.Errorf("glyph %q row %d is %d dots wide, want 5", string(r), y, len(row))
			}
		}
	}
	top, bottom := LogoSprites()
	if top.H != bottom.H || top.W != bottom.W {
		t.Errorf("the two words differ in size (%dx%d vs %dx%d); they would not stack evenly",
			top.W, top.H, bottom.W, bottom.H)
	}
}

// TestNarrowTerminalFallsBackFromTheLogo checks a small window shows plain text
// instead of a logo that spills off the edge.
func TestNarrowTerminalFallsBackFromTheLogo(t *testing.T) {
	a := headlessTitleApp(t)
	text := renderTitleText(t, a, 30, 24)
	if !containsText(text, "S U P E R") {
		t.Error("a narrow terminal should fall back to spaced letters")
	}
}

// TestTitleMenuLabelsDoNotShiftWhenSelected is the same guard the pause menu
// has: a selected entry must start in the same column as when it is not.
func TestTitleMenuLabelsDoNotShiftWhenSelected(t *testing.T) {
	a := headlessTitleApp(t)
	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Skipf("no simulation screen: %v", err)
	}
	defer sim.Fini()
	sim.SetSize(70, 26)
	a.screen = sim

	// The column of a label's first character, found by scanning the menu rows.
	colOf := func(first rune) int {
		for y := 0; y < 26; y++ {
			for x := 0; x < 70; x++ {
				if r, _, _, _ := sim.GetContent(x, y); r == first {
					return x
				}
			}
		}
		return -1
	}
	draw := func(cursor int) {
		a.title.menu.cursor = cursor
		sim.Clear()
		a.drawTitleScreen(70, 26)
		sim.Show()
	}

	n := len(a.title.menu.items)
	for i, it := range a.title.menu.items {
		first := []rune(it.label)[0]
		draw(i)
		selected := colOf(first)
		draw((i + 1) % n)
		other := colOf(first)
		if selected < 0 || other < 0 {
			t.Fatalf("%q was not drawn", it.label)
		}
		if selected != other {
			t.Errorf("%q starts at column %d when selected and %d when not", it.label, selected, other)
		}
	}
}
