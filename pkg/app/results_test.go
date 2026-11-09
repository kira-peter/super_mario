package app

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/engine"
	"super-mario/pkg/game"
	"super-mario/pkg/save"
)

// strictRows reads the screen the way a terminal shows it: a wide rune takes
// two cells and the cell after it is skipped. Unlike containsText it does not
// forgive missing characters, so it can see a character that was overwritten.
func strictRows(sim tcell.SimulationScreen) []string {
	w, h := sim.Size()
	rows := make([]string, h)
	for y := 0; y < h; y++ {
		var b strings.Builder
		for x := 0; x < w; {
			r, _, _, width := sim.GetContent(x, y)
			if r == 0 {
				r = ' '
			}
			b.WriteRune(r)
			if width < 1 {
				width = 1
			}
			x += width
		}
		rows[y] = b.String()
	}
	return rows
}

func resultsApp(t *testing.T, cleared bool) (*App, tcell.SimulationScreen) {
	t.Helper()
	a := newHeadlessApp(t)
	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Skipf("no simulation screen: %v", err)
	}
	t.Cleanup(sim.Fini)
	sim.SetSize(90, 24)
	a.screen = sim

	a.world.Score.Points = 17550
	a.world.Tally = game.Tally{Done: true, Base: 4600, Units: 259, Awarded: 12950}
	a.resultCleared = cleared
	a.resultRank = 1
	// A previous best exists, so rank 1 means beating it rather than a first
	// entry; a first entry is a different message with its own test.
	a.resultHadPrev = true
	a.resultPrevBest = save.Entry{Score: 15240}
	_ = a.sm.TouchFlag()
	_ = a.sm.ShowResults()
	if !a.sm.Is(engine.StateResults) {
		t.Fatalf("setup: state = %v", a.sm.Current())
	}
	return a, sim
}

// hasRowLoose is hasRow that ignores spaces. An ambiguous-width character such
// as "×" or "★" is two cells wide here, and the spacing the layout puts around
// it is not part of what is being checked.
func hasRowLoose(rows []string, want string) bool {
	squashed := strings.ReplaceAll(want, " ", "")
	for _, r := range rows {
		if strings.Contains(strings.ReplaceAll(r, " ", ""), squashed) {
			return true
		}
	}
	return false
}

func hasRow(rows []string, want string) bool {
	for _, r := range rows {
		if strings.Contains(r, want) {
			return true
		}
	}
	return false
}

// TestResultsScreenTextIsIntact is the regression test for the results screen
// losing characters: "分数" showed as "分", "时间奖励" as "时奖", "新纪录" as
// "新录", "空格 继续" as "空 继".
//
// Text was written one cell per character, so each wide character's second
// cell was overwritten by the next character.
func TestResultsScreenTextIsIntact(t *testing.T) {
	a, sim := resultsApp(t, true)
	a.drawResults(90, 24)
	sim.Show()
	rows := strictRows(sim)

	for _, want := range []string{"基础得分", "时间奖励", "总  分", "用  时", "排  名", "新纪录", "空格 继续", "4,600", "+12,950", "17,550", "× 259"} {
		if !hasRowLoose(rows, want) {
			t.Errorf("the results screen does not show %q intact", want)
		}
	}
}

// TestResultsValuesLineUp checks the numbers share one right edge, so the
// block reads as a table and not as lines that happen to be centred.
func TestResultsValuesLineUp(t *testing.T) {
	a, sim := resultsApp(t, true)
	a.drawResults(90, 24)
	sim.Show()

	w, h := sim.Size()
	edge := func(label string) int {
		for y := 0; y < h; y++ {
			var row strings.Builder
			for x := 0; x < w; {
				r, _, _, width := sim.GetContent(x, y)
				if r == 0 {
					r = ' '
				}
				row.WriteRune(r)
				if width < 1 {
					width = 1
				}
				x += width
			}
			if !strings.Contains(row.String(), label) {
				continue
			}
			last := -1
			for x := 0; x < w; x++ {
				if r, _, _, _ := sim.GetContent(x, y); r != ' ' && r != 0 {
					last = x
				}
			}
			return last
		}
		return -1
	}

	score, timeRow, bonus := edge("总  分"), edge("用  时"), edge("时间奖励")
	if score < 0 || timeRow < 0 || bonus < 0 {
		t.Fatalf("rows missing: score=%d time=%d bonus=%d", score, timeRow, bonus)
	}
	if score != timeRow || score != bonus {
		t.Errorf("value columns end at %d, %d and %d; they should share one right edge",
			score, timeRow, bonus)
	}
}

// TestGameOverScreenDoesNotClaimABonus checks a lost run reads as one.
func TestGameOverScreenDoesNotClaimABonus(t *testing.T) {
	a, sim := resultsApp(t, false)
	a.world.Tally = game.Tally{}
	a.resultRank = 0
	a.drawResults(90, 24)
	sim.Show()
	rows := strictRows(sim)

	if hasRow(rows, "时间奖励") {
		t.Error("a run that did not clear shows a time bonus")
	}
	if hasRow(rows, "CLEAR") {
		t.Error("a lost run is headed as a clear")
	}
	if !hasRowLoose(rows, "GAMEOVER") {
		t.Error("a lost run has no game-over heading")
	}
}

// TestDisplayWidthMatchesWhatTheScreenDraws checks the layout's idea of a
// string's width against the screen library's own.
//
// Ambiguous-width characters (arrows, middle dots, box-drawing, stars) take one
// cell or two depending on the locale, and on a Chinese-language system they
// take two. So the right expectation is not a fixed number: it is whatever the
// screen actually uses, because layout and drawing must agree.
func TestDisplayWidthMatchesWhatTheScreenDraws(t *testing.T) {
	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Skipf("no simulation screen: %v", err)
	}
	defer sim.Fini()
	sim.SetSize(40, 3)

	for _, s := range []string{"abc", "分数", "空格 继续", "↑↓", "·", "─", "★", "× 259"} {
		sim.Clear()
		end := writeRunes(sim, 0, 0, s, tcell.StyleDefault)
		if want := displayWidth([]rune(s)); end != want {
			t.Errorf("%q: writeRunes advanced %d cells but displayWidth says %d", s, end, want)
		}
		// Every character must be where the running width says it is, and
		// none may have been overwritten by its neighbour.
		x := 0
		for _, r := range s {
			got, _, _, _ := sim.GetContent(x, 0)
			if got != r {
				t.Errorf("%q: expected %q at column %d, found %q (a neighbour overwrote it)",
					s, string(r), x, string(got))
			}
			x += runeWidth(r)
		}
	}
}

// TestFirstEntryIsNotCalledARecord checks an empty board says so, instead of
// claiming a record against nothing.
func TestFirstEntryIsNotCalledARecord(t *testing.T) {
	a, sim := resultsApp(t, true)
	a.resultHadPrev = false
	a.drawResults(90, 24)
	sim.Show()
	rows := strictRows(sim)
	if hasRowLoose(rows, "新纪录") {
		t.Error("a first-ever entry was called a new record")
	}
	if !hasRowLoose(rows, "首次上榜") {
		t.Error("a first-ever entry has no first-entry message")
	}
}

// TestResultsAddUp checks the card is arithmetic the player can verify: the
// base score plus the bonus is the total.
func TestResultsAddUp(t *testing.T) {
	a, sim := resultsApp(t, true)
	a.drawResults(90, 24)
	sim.Show()
	rows := strictRows(sim)
	if !hasRow(rows, "4,600") || !hasRow(rows, "+12,950") || !hasRow(rows, "17,550") {
		t.Fatalf("breakdown rows missing:\n%s", strings.Join(rows, "\n"))
	}
	if 4600+12950 != 17550 {
		t.Fatal("test data does not add up")
	}
}

// TestRankLineIsHeldBackWhileTheBonusIsCounting stops the rank flickering
// between values while the time bonus is still converting.
func TestRankLineIsHeldBackWhileTheBonusIsCounting(t *testing.T) {
	a, sim := resultsApp(t, true)
	a.world.Tally.Done = false
	a.drawResults(90, 24)
	sim.Show()
	rows := strictRows(sim)
	if hasRow(rows, "排  名") {
		t.Error("the rank is shown before the tally has settled")
	}
	if hasRow(rows, "空格 继续") {
		t.Error("the continue prompt is shown while the bonus is still counting")
	}
}

// TestNewRecordShowsTheMargin checks beating the old best says by how much.
func TestNewRecordShowsTheMargin(t *testing.T) {
	a, sim := resultsApp(t, true)
	a.resultHadPrev = true
	a.resultPrevBest = save.Entry{Score: 15240}
	a.drawResults(90, 24)
	sim.Show()
	if rows := strictRows(sim); !hasRow(rows, "+2,310") {
		t.Errorf("the margin over the old record (+2,310) is not shown:\n%s", strings.Join(rows, "\n"))
	}
}

// TestSecondPlaceShowsTheGapToFirst checks a run that did not win says how far
// off it was.
func TestSecondPlaceShowsTheGapToFirst(t *testing.T) {
	a, sim := resultsApp(t, true)
	a.resultRank = 2
	a.resultHadPrev = true
	a.resultPrevBest = save.Entry{Score: 20000}
	a.drawResults(90, 24)
	sim.Show()
	rows := strictRows(sim)
	if !hasRow(rows, "第 2 名") || !hasRow(rows, "-2,450") {
		t.Errorf("rank and gap to first are not both shown:\n%s", strings.Join(rows, "\n"))
	}
}

// TestGroupDigits pins the number formatting.
func TestGroupDigits(t *testing.T) {
	for in, want := range map[int]string{0: "0", 999: "999", 1000: "1,000", 17550: "17,550", 1234567: "1,234,567", -4800: "-4,800"} {
		if got := groupDigits(in); got != want {
			t.Errorf("groupDigits(%d) = %q, want %q", in, got, want)
		}
	}
	if signed(5) != "+5" || signed(-5) != "-5" || signed(0) != "+0" {
		t.Errorf("signed gave %q %q %q", signed(5), signed(-5), signed(0))
	}
}
