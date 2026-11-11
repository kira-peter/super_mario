package render

import (
	"super-mario/pkg/game"
	"super-mario/pkg/level"
	"super-mario/pkg/render/theme"
)

// Terrain art for the default four-pixel tile.
//
// At four pixels a tile there is no room for anything but a handful of flat
// cells, so these are drawn by hand as 4x4 patterns rather than scaled down from
// the larger art. The letters are the same ones the design preview used, so a
// pattern reads here exactly as it did when it was chosen.
//
//	r  grass                            R  dark grass
//	d  dark soil
//	b  brick face                       m  mortar between bricks
//	S  stone, lit                       s  stone, flat        t  stone, shaded
//	?  question block gold              o  outline            h  pale glint
//	$  coin                             *  coin glint         c  coin shade
//
// They are looked up through the theme, never by colour value, so every one of
// them follows the active theme and the terminal's colour depth.
type pattern4 [4]string

// terrain4 holds every pattern drawn at the four-pixel size.
var terrain4 = struct {
	// Ground has a strip of turf on its top layer, and plain dark soil on
	// every layer below, where turf would only draw a stripe through the
	// middle of the earth. The turf alternates two patterns by column so a
	// long floor does not read as one repeated stamp.
	GroundTop   [2]pattern4
	GroundBelow pattern4
	// Brick alternates two patterns by column, so the seams do not line up.
	Brick [2]pattern4
	// Stair is a lit, flat and shaded block that reads as a different material
	// from both the earth and the brick.
	Stair pattern4
	// Question pulses between two frames.
	Question [2]pattern4
	// Coin has a shaded body with a glint that comes and goes.
	Coin [2]pattern4
}{
	GroundTop: [2]pattern4{
		{"rrRr", "dddd", "dddd", "dddd"},
		{"rRrr", "dddd", "dddd", "dddd"},
	},
	GroundBelow: pattern4{"dddd", "dddd", "dddd", "dddd"},
	Brick: [2]pattern4{
		{"bbmb", "mmmm", "bmbb", "mmmm"},
		{"bmbb", "mmmm", "bbmb", "mmmm"},
	},
	Stair: pattern4{"SSSS", "Ssst", "Ssst", "tttt"},
	Question: [2]pattern4{
		{"oooo", "o??o", "o??o", "oooo"},
		{"oooo", "ohho", "ohho", "oooo"},
	},
	Coin: [2]pattern4{
		{".$$.", "$*$c", "$$$c", ".cc."},
		{".$$.", "$$$c", "$$$c", ".cc."},
	},
}

// terrainColor resolves a pattern letter to a theme colour. The second result
// is false for an unpainted cell, so the background shows through.
func terrainColor(t theme.Tokens, letter byte) (interface{}, bool) {
	switch letter {
	case 'G':
		return t.GroundEdge, true
	case 'r':
		return t.Cast.Grass, true
	case 'R':
		return t.Cast.GrassDark, true
	case 'd':
		return t.GroundDeep, true
	case 'b':
		return t.Brick, true
	case 'm':
		return t.BrickMortar, true
	case 'S':
		return t.StoneLight, true
	case 's':
		return t.Stone, true
	case 't':
		return t.StoneDark, true
	case '?':
		return t.Question, true
	case 'o':
		return t.Outline, true
	case 'h':
		return t.QuestionGlint, true
	case '$':
		return t.Coin, true
	case '*':
		return t.CoinHi, true
	case 'c':
		return t.CoinShade, true
	}
	return nil, false
}

// blit4 paints a four-by-four pattern with its top-left at (x, y).
func (c *Canvas) blit4(p pattern4, x, y int, t theme.Tokens) {
	for r := 0; r < 4; r++ {
		row := p[r]
		for col := 0; col < 4 && col < len(row); col++ {
			if clr, ok := terrainColor(t, row[col]); ok {
				c.Set(x+col, y+r, clr)
			}
		}
	}
}

// questionFrame picks which of the question block's two frames to show.
//
// It counts simulation steps, not wall-clock time or rendered frames: the
// renderer's frame rate is a setting, and a block that pulsed per rendered frame
// would pulse twice as fast at 60 fps as at 30. Twenty steps a frame is about
// three pulses a second.
func questionFrame(step int) int {
	return (step / 20) & 1
}

// coinFrame picks the coin's glint frame. The column is added so a row of coins
// does not flash in unison.
func coinFrame(step, col int) int {
	return ((step / 24) + col) & 1
}

// tile4 paints one terrain cell with the hand-drawn four-pixel patterns and
// reports whether it handled the glyph. Glyphs it does not know (pipes, the
// flagpole, a used block) are left to the general drawing.
func (d drawer) tile4(tx, ty int, tile level.Tile, dx, dy int, w *game.World) bool {
	t := d.s.T
	step := w.StepCount()

	switch tile.Glyph {
	case 'X':
		// The turf belongs only to the top of the ground. Every layer below it
		// is plain soil, otherwise a stripe of grass runs through the middle
		// of the earth and cuts it into slabs. Which turf pattern a column
		// gets is fixed by its place in the level, so it does not shimmer as
		// the view scrolls.
		if above := w.Level.TileAt(tx, ty-1); above.Glyph == 'X' {
			d.c.blit4(terrain4.GroundBelow, dx, dy, t)
		} else {
			d.c.blit4(terrain4.GroundTop[((tx%2)+2)%2], dx, dy, t)
		}
	case 'B':
		// Alternate by column so the seams of neighbouring bricks do not line
		// up. Which pattern a column gets is fixed by its position in the
		// level, not by where the camera is, or the wall would shimmer as the
		// view scrolls.
		d.c.blit4(terrain4.Brick[((tx%2)+2)%2], dx, dy, t)
	case 'H':
		d.c.blit4(terrain4.Stair, dx, dy, t)
	case '?':
		d.c.blit4(terrain4.Question[questionFrame(step)], dx, dy, t)
	case 'o':
		d.c.blit4(terrain4.Coin[coinFrame(step, tx)], dx, dy, t)
	default:
		return false
	}
	return true
}
