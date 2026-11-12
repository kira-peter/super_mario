package render

import (
	"strings"
	"testing"

	"super-mario/pkg/engine"
	"super-mario/pkg/game"
	"super-mario/pkg/render/theme"
)

func rowText(cells []Cell, cols, row int) string {
	r := make([]rune, cols)
	for x := 0; x < cols; x++ {
		r[x] = cells[row*cols+x].Rune
	}
	return string(r)
}

// overlayed draws the world, converts it and writes the floating scores on it.
func overlayed(t *testing.T, w *game.World) ([]Cell, int, int) {
	t.Helper()
	c := (&Scene{T: theme.Ink}).Draw(w, 4)
	rows := c.H / 2
	cells := c.Convert(c.W, rows, true)
	OverlayFloatTexts(cells, c.W, rows, w, 4, theme.Ink)
	return cells, c.W, rows
}

// TestPoleScoreIsWrittenBesideThePole checks the number appears on screen.
func TestPoleScoreIsWrittenBesideThePole(t *testing.T) {
	w := castWorld(t)
	w.CameraX = engine.Fixed(w.Level.FlagX-20) * engine.Tile
	w.CameraY = 0
	w.FlagBonus = &game.FlagBonus{Amount: 5000, Y: engine.Fixed(5) * engine.Tile}
	cells, cols, _ := overlayed(t, w)
	if !strings.Contains(rowText(cells, cols, 10), "5000") {
		t.Errorf("the pole score is not on screen: %q", rowText(cells, cols, 10))
	}
}

// TestStompScoreIsWritten is the regression test for stomp scores that were
// made and never shown.
func TestStompScoreIsWritten(t *testing.T) {
	w := castWorld(t)
	w.CameraX, w.CameraY = 0, 0
	w.Parts = append(w.Parts, &game.Particle{
		Pos:  engine.Vec2{X: engine.Fixed(10) * engine.Tile, Y: engine.Fixed(6) * engine.Tile},
		Life: 30, Kind: 's', Text: "100",
	})
	cells, cols, _ := overlayed(t, w)
	if !strings.Contains(rowText(cells, cols, 12), "100") {
		t.Errorf("the stomp score is not on screen: %q", rowText(cells, cols, 12))
	}
}

// TestNothingIsWrittenWithoutScores keeps the overlay from drawing stray text.
func TestNothingIsWrittenWithoutScores(t *testing.T) {
	w := castWorld(t)
	w.CameraX, w.CameraY = 0, 0
	cells, cols, rows := overlayed(t, w)
	for r := 0; r < rows; r++ {
		if strings.ContainsAny(rowText(cells, cols, r), "0123456789") {
			t.Fatalf("digits on row %d with no score showing", r)
		}
	}
}

// TestFadeOf pins the fade curve: full, then falling to nothing.
func TestFadeOf(t *testing.T) {
	if fadeOf(30, 20) != 1 || fadeOf(0, 20) != 0 || fadeOf(10, 20) != 0.5 {
		t.Error("fadeOf is not full, then linear down to zero")
	}
}
