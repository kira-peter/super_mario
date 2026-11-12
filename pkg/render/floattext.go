package render

import (
	"strconv"

	"super-mario/pkg/engine"
	"super-mario/pkg/game"
	"super-mario/pkg/render/theme"
)

// Floating scores: the points that rise from a stomped enemy, and the pole score
// that rises beside the flagpole.
//
// Text cannot be drawn into the pixel canvas, so it is written onto the cells
// after the canvas is converted. It is written into the cell slice, before the
// frame is compared with the last one, so the redraw logic sees the text move
// and clears the old one like any other change.

// floatRise is how many game units a pole score rises per step.
const floatRise = 1

// floatFade is how many of the last steps a score spends fading out.
const floatFade = 20

// OverlayFloatTexts writes the world's floating scores onto converted cells.
// The canvas is one pixel per column and two per row, so a canvas pixel maps
// straight to a cell.
func OverlayFloatTexts(cells []Cell, cols, rows int, w *game.World, px int, t theme.Tokens) {
	camX := WorldToArtScaled(w.CameraX, px)
	camY := WorldToArtScaled(w.CameraY, px)

	put := func(pos engine.Vec2, text string, fade float64) {
		x := WorldToArtScaled(pos.X, px) - camX - len(text)/2
		row := (WorldToArtScaled(pos.Y, px) - camY) / 2
		if row < 0 || row >= rows {
			return
		}
		clr := blend(t.Sky, t.HudText, fade)
		for i, r := range text {
			col := x + i
			if col < 0 || col >= cols {
				continue
			}
			cell := &cells[row*cols+col]
			cell.Rune = r
			cell.FG = clr
		}
	}

	for _, p := range w.Parts {
		if p.Kind != 's' {
			continue
		}
		put(p.Pos, p.Text, fadeOf(p.Life, floatFade))
	}
	if b := w.FlagBonus; b != nil {
		pos := engine.Vec2{
			X: engine.Fixed(w.Level.FlagX+1)*engine.Tile + engine.Tile/2,
			Y: b.Y - engine.Fixed(b.Age*floatRise),
		}
		put(pos, strconv.Itoa(b.Amount), fadeOf(game.FlagBonusSteps-b.Age, floatFade))
	}
}

// fadeOf is 1 until the last n steps of a life, then falls to 0.
func fadeOf(remaining, n int) float64 {
	if remaining >= n {
		return 1
	}
	if remaining <= 0 {
		return 0
	}
	return float64(remaining) / float64(n)
}
