package app

import (
	"testing"

	"super-mario/pkg/engine"
)

// TestViewportIsSquareAtEveryZoom is the regression test for a bug that made
// shrinking the game shrink only one axis.
//
// Terminal cells are about twice as tall as they are wide, and the renderer
// stacks two art pixels into each cell, so a square art grid needs the same
// number of art pixels per tile on both axes. Dividing rows by a constant while
// dividing columns by the zoom stretched every pixel vertically: the picture got
// thinner but kept its full height, which is exactly what the report described.
func TestViewportIsSquareAtEveryZoom(t *testing.T) {
	for _, zoom := range []int{4, 8, 12, 16} {
		a := newHeadlessApp(t)
		a.colsPerTile = zoom

		cols, rows := 120, 39
		a.applyViewport(cols, rows)

		// The viewport in art pixels: a tile is engine.Tile game units and gets
		// `zoom` art pixels, so the whole window is this many art pixels. It is
		// fractional on purpose, so it is converted rather than truncated to
		// whole tiles first.
		artW := float64(a.world.ViewW) / float64(engine.Tile) * float64(zoom)
		artH := float64(a.world.ViewH) / float64(engine.Tile) * float64(zoom)

		// The terminal presents cols columns of one cell each, and rows cells
		// holding two art pixels apiece.
		termPxW := float64(cols)
		termPxH := float64(rows * 2)
		if d := artW - termPxW; d < -1 || d > 1 {
			t.Errorf("zoom %d: art width %.1f does not match the %.0f terminal columns",
				zoom, artW, termPxW)
		}
		if d := artH - termPxH; d < -1 || d > 1 {
			t.Errorf("zoom %d: art height %.1f does not match the %.0f available art rows",
				zoom, artH, termPxH)
		}

		// Stretching is the symptom; assert it directly.
		if artH > 0 {
			ratio := termPxH / artH
			if ratio > 1.02 {
				t.Errorf("zoom %d: picture is stretched %.2fx vertically", zoom, ratio)
			}
		}
	}
}

// TestSmallerZoomShowsMoreOfTheLevel checks that the zoom actually buys area.
func TestSmallerZoomShowsMoreOfTheLevel(t *testing.T) {
	area := func(zoom int) int {
		a := newHeadlessApp(t)
		a.colsPerTile = zoom
		a.applyViewport(120, 39)
		return int(a.world.ViewW/engine.Tile) * int(a.world.ViewH/engine.Tile)
	}
	prev := 0
	for _, zoom := range []int{16, 12, 8, 4} {
		got := area(zoom)
		if got <= prev {
			t.Errorf("zoom %d shows %d tiles, not more than the previous %d", zoom, got, prev)
		}
		prev = got
	}
}
