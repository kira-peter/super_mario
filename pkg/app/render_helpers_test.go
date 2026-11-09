package app

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// renderTitleText draws the title area onto a simulated screen and returns
// every row as text, so tests can assert what the player actually sees.
func renderTitleText(t *testing.T, a *App, w, h int) []string {
	t.Helper()
	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Skipf("no simulation screen: %v", err)
	}
	t.Cleanup(sim.Fini)
	sim.SetSize(w, h)
	a.screen = sim
	a.drawTitleScreen(w, h)
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

// containsText reports whether any rendered row contains the text. A wide rune
// leaves a blank in the cell after it, so rows are also compared with spaces
// removed.
func containsText(rows []string, want string) bool {
	squashed := strings.ReplaceAll(want, " ", "")
	for _, r := range rows {
		if strings.Contains(r, want) || strings.Contains(strings.ReplaceAll(r, " ", ""), squashed) {
			return true
		}
	}
	return false
}
