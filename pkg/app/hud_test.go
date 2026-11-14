package app

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// renderGameText draws a game frame onto a simulated screen and returns every
// row as text.
func renderGameText(t *testing.T, a *App, w, h int) []string {
	t.Helper()
	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Skipf("no simulation screen: %v", err)
	}
	t.Cleanup(sim.Fini)
	sim.SetSize(w, h)
	a.screen = sim
	a.drawGame(w, h)
	sim.Show()

	rows := make([]string, h)
	for y := 0; y < h; y++ {
		line := make([]rune, 0, w)
		for x := 0; x < w; x++ {
			r, _, _, _ := sim.GetContent(x, y)
			if r == 0 {
				r = ' '
			}
			line = append(line, r)
		}
		rows[y] = string(line)
	}
	return rows
}

// TestHudIsAtTheTop checks the status line is on the first row, as in the
// original, and nowhere else on the screen.
func TestHudIsAtTheTop(t *testing.T) {
	a := newHeadlessApp(t)
	rows := renderGameText(t, a, 120, 40)
	if !strings.Contains(rows[0], "MARIO") || !strings.Contains(rows[0], "WORLD") {
		t.Fatalf("the top row is not the status line: %q", rows[0])
	}
	for y := 1; y < len(rows); y++ {
		if strings.Contains(rows[y], "MARIO") {
			t.Errorf("the status line also appears on row %d", y)
		}
	}
}

// TestPictureStartsUnderTheHud checks the game picture is drawn below the
// status line, not under it: the bottom row, which used to hold the status
// line, is now picture.
func TestPictureStartsUnderTheHud(t *testing.T) {
	a := newHeadlessApp(t)
	rows := renderGameText(t, a, 120, 40)
	if strings.Contains(rows[len(rows)-1], "MARIO") {
		t.Error("the status line is still on the bottom row")
	}
	// The ground is at the bottom of the picture; with the status line on top,
	// the bottom row is ground, drawn as blocks, not text.
	if strings.TrimSpace(rows[len(rows)-1]) == "" && strings.TrimSpace(rows[len(rows)-2]) == "" {
		// Ground cells are spaces with a background colour, so text cannot show
		// it; check the background instead.
		sim := a.screen.(tcell.SimulationScreen)
		_, _, st, _ := sim.GetContent(5, len(rows)-1)
		_, bg, _ := st.Decompose()
		if bg == a.theme.Sky {
			t.Error("the bottom row is sky; the picture does not reach the bottom of the screen")
		}
	}
}
