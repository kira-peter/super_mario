package render

import (
	"super-mario/pkg/engine"
	"super-mario/pkg/render/theme"
)

// Canvas is a virtual pixel buffer in art pixels.
//
// Design decision 6: the game draws into this buffer, then Convert turns the
// buffer into terminal cells once. Nothing writes characters directly, which is
// what preserves a single, predictable draw order.
type Canvas struct {
	W, H int
	// Px holds a token colour per art pixel.
	Px []interface{}
}

// NewCanvas returns an empty buffer of the given art size.
func NewCanvas(w, h int) *Canvas {
	return &Canvas{W: w, H: h, Px: make([]interface{}, w*h)}
}

// Clear fills the canvas with one colour.
func (c *Canvas) Clear(col interface{}) {
	for i := range c.Px {
		c.Px[i] = col
	}
}

// Set paints one pixel, clipped to the buffer.
func (c *Canvas) Set(x, y int, col interface{}) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	c.Px[y*c.W+x] = col
}

// FillRect paints an axis-aligned rectangle in art pixels.
func (c *Canvas) FillRect(x, y, w, h int, col interface{}) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			c.Set(xx, yy, col)
		}
	}
}

// Draw blits a sprite with its top-left at (x, y). Pixels whose slot is empty
// are skipped, so sprites composite over what is already there.
func (c *Canvas) Draw(s Sprite, x, y int, t theme.Tokens) {
	for sy := 0; sy < s.H; sy++ {
		row := s.Rows[sy]
		for sx := 0; sx < s.W && sx < len(row); sx++ {
			slot := row[sx]
			if slot == SlotNone {
				continue
			}
			col, ok := SlotColor(t, slot)
			if !ok {
				continue
			}
			c.Set(x+sx, y+sy, col)
		}
	}
}

// DrawSized blits a sprite scaled to a target art size.
//
// A sprite is authored at its set's resolution, but a canvas may be at another
// (the twelve-column view uses the sixteen-pixel art). Drawing it unscaled
// would make Mario a third of a tile wider than the tile grid he collides with,
// so every entity is drawn through this when the two differ. Sampling is
// nearest-neighbour: at these sizes it keeps the pixel-art edges crisp, and
// smoothing a hat into a grey blur would lose the silhouette.
func (c *Canvas) DrawSized(s Sprite, x, y, w, h int, t theme.Tokens) {
	if (w == s.W && h == s.H) || w <= 0 || h <= 0 {
		c.Draw(s, x, y, t)
		return
	}
	for dy := 0; dy < h; dy++ {
		sy := dy * s.H / h
		if sy >= s.H {
			sy = s.H - 1
		}
		row := s.Rows[sy]
		for dx := 0; dx < w; dx++ {
			sx := dx * s.W / w
			if sx >= len(row) {
				continue
			}
			slot := row[sx]
			if slot == SlotNone {
				continue
			}
			col, ok := SlotColor(t, slot)
			if !ok {
				continue
			}
			c.Set(x+dx, y+dy, col)
		}
	}
}

// Cell is one terminal cell's rendered appearance.
type Cell struct {
	Rune   rune
	FG, BG interface{}
}

// Convert turns the art buffer into terminal cells.
//
// Two art pixels share one cell vertically: the upper becomes the foreground of
// a half-block glyph and the lower the background. That doubling is what makes
// the terminal's 1:2 character cells behave like roughly square pixels
// (spec terminal-renderer / 纵向分辨率翻倍).
func (c *Canvas) Convert(cols, rows int, unicode bool) []Cell {
	out := make([]Cell, cols*rows)
	vrows := rows * 2
	for cy := 0; cy < rows; cy++ {
		for cx := 0; cx < cols; cx++ {
			top := c.sample(cx, cy*2, cols, vrows)
			bot := c.sample(cx, cy*2+1, cols, vrows)

			var cell Cell
			switch {
			case top == nil && bot == nil:
				cell = Cell{Rune: ' ', FG: nil, BG: nil}
			case top == nil:
				cell = Cell{Rune: ' ', BG: bot}
			case bot == nil:
				cell = Cell{Rune: ' ', BG: top}
			case sameColor(top, bot):
				cell = Cell{Rune: ' ', BG: top}
			default:
				if unicode {
					cell = Cell{Rune: '▀', FG: top, BG: bot}
				} else {
					// ASCII fallback: one glyph per cell, upper pixel wins.
					// Vertical resolution halves and the caller accepts it.
					cell = Cell{Rune: asciiFor(top, bot), FG: top, BG: bot}
				}
			}
			out[cy*cols+cx] = cell
		}
	}
	return out
}

// sample maps an output cell coordinate onto the art buffer.
//
// Each output cell covers a small region of art pixels. Taking the most common
// non-empty colour in that region (rather than the single pixel under the cell
// centre) keeps thin one-pixel features — an eye, a coin highlight, the right
// edge of a sprite — from vanishing when the art is downsampled from 16 to 12
// columns per tile.
// sample maps an output cell coordinate onto the art buffer.
//
// Each output cell covers a small region of art pixels. Taking the most common
// non-empty colour in that region (rather than the single pixel under the cell
// centre) keeps thin one-pixel features — an eye, the edge of a sprite — from
// vanishing when art is downsampled.
//
// The tally is a fixed-size stack array rather than a map: this runs once per
// output cell, a few thousand times per frame, and allocating a map each time
// was the single largest cost in the render loop. The region is at most a few
// pixels wide and tall, so a dozen slots is plenty; past that the oldest entry
// is overwritten, which only matters for pathological scales.
func (c *Canvas) sample(cx, vy, cols, vrows int) interface{} {
	if cols <= 0 || vrows <= 0 || c.W == 0 || c.H == 0 {
		return nil
	}
	gx0 := cx * c.W / cols
	gx1 := (cx + 1) * c.W / cols
	gy0 := vy * c.H / vrows
	gy1 := (vy + 1) * c.H / vrows
	if gx1 <= gx0 {
		gx1 = gx0 + 1
	}
	if gy1 <= gy0 {
		gy1 = gy0 + 1
	}

	var (
		keys  [12]interface{}
		count [12]int
		n     int
	)
	best := interface{}(nil)
	bestN := 0

	for gy := gy0; gy < gy1; gy++ {
		if gy < 0 || gy >= c.H {
			continue
		}
		rowBase := gy * c.W
		for gx := gx0; gx < gx1; gx++ {
			if gx < 0 || gx >= c.W {
				continue
			}
			px := c.Px[rowBase+gx]
			if px == nil {
				continue
			}
			found := false
			for i := 0; i < n; i++ {
				if keys[i] == px {
					count[i]++
					if count[i] > bestN {
						bestN = count[i]
						best = px
					}
					found = true
					break
				}
			}
			if !found {
				slot := n
				if slot >= len(keys) {
					slot = len(keys) - 1
				} else {
					n++
				}
				keys[slot] = px
				count[slot] = 1
				if 1 > bestN {
					bestN = 1
					best = px
				}
			}
		}
	}
	return best
}

// sameColor compares two pixel values. Colours are tcell.Color values, so a
// plain equality is correct and cheap.
func sameColor(a, b interface{}) bool { return a == b }

// asciiFor picks an ASCII glyph that suggests the density of the two pixels.
func asciiFor(top, bot interface{}) rune {
	switch {
	case top == nil && bot == nil:
		return ' '
	case top == nil:
		return '.'
	case bot == nil:
		return '='
	}
	return '#'
}

// WorldToArt converts a world-space position to art pixels at the given
// art-pixels-per-tile scale.
func WorldToArt(v engine.Fixed, artPerTile int) int {
	return int(v) * artPerTile / int(engine.Tile)
}
