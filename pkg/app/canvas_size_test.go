package app

import "testing"

// TestCanvasMatchesTheTerminalPixelForPixel is the regression test for bricks
// drawing at different heights. The canvas was rounded down to whole tiles, so
// it came out a row or two short of the terminal, and stretching it back
// duplicated those rows: any brick that fell on one drew a pixel taller.
//
// A mismatch of even one row means some row is drawn twice, so this checks
// equality, across terminal sizes that land on fractional tile counts.
func TestCanvasMatchesTheTerminalPixelForPixel(t *testing.T) {
	for _, zoom := range []int{4, 8, 12, 16} {
		for _, size := range [][2]int{{120, 39}, {100, 29}, {80, 23}, {157, 47}} {
			a := newHeadlessApp(t)
			a.colsPerTile = zoom
			cols, rows := size[0], size[1]
			a.applyViewport(cols, rows)
			c := a.scene.Draw(a.world, a.colsPerTile)

			if c.W != cols || c.H != rows*2 {
				t.Errorf("zoom %d, terminal %dx%d: canvas is %dx%d, want %dx%d (one art pixel per terminal pixel)",
					zoom, cols, rows, c.W, c.H, cols, rows*2)
			}
		}
	}
}
