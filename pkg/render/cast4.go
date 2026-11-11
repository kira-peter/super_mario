package render

import (
	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/game"
	"super-mario/pkg/render/theme"
)

// The cast at four pixels a tile: mushrooms, Goombas, Koopas and the flag.
//
// These were drawn by hand at the default size and chosen in
// preview/sprites-options.html; the letters are the ones the preview used.
// At larger zooms the same drawing is scaled up by a whole number, the way the
// hero is, so a pixel stays a crisp block and every zoom shows the same art.
type art []string

// width is the widest row.
func (a art) width() int {
	w := 0
	for _, r := range a {
		if len(r) > w {
			w = len(r)
		}
	}
	return w
}

// palette resolves one letter of a drawing to a colour; false leaves the pixel
// unpainted.
type palette func(letter byte) (tcell.Color, bool)

// cast4 holds every character drawing. All of them face left, the way enemies
// walk by default; facing right is a mirror image.
var cast4 = struct {
	// Mushroom is shared by every pickup. The cap colour is what tells them
	// apart: C is the cap, c its shaded rim, W the spot, S the stem.
	Mushroom art

	// Goomba walks with its feet apart, then together. T body, W eyes, f feet.
	Goomba     [2]art
	GoombaFlat art

	// Koopa: Y skin, G shell, L shell pattern, W the white rim under the shell.
	Koopa [2]art
	Shell art

	// The flag: P pole, N ball, n its shade; the cloth is X dark, w light.
	Pole, Ball, Cloth art
}{
	Mushroom: art{".CC.", "CWCC", "cccc", ".SS."},

	Goomba: [2]art{
		{".TTT.", "TWTWT", "TTTTT", "ff.ff"},
		{".TTT.", "TWTWT", "TTTTT", ".f.f."},
	},
	GoombaFlat: art{"TWTWT", "fffff"},

	Koopa: [2]art{
		{"YY.GG.", "YYGLGL", ".YWWWW", "..Y..Y"},
		{"YY.GG.", "YYGLGL", ".YWWWW", "...YY."},
	},
	Shell: art{".GG.", "GLGL", "WWWW"},

	Pole:  art{".P..", ".P..", ".P..", ".P.."},
	Ball:  art{"....", "....", ".N..", "NNn."},
	Cloth: art{"wXwXw", "XwXwX", "wXwXw"},
}

// poleColumn is where the shaft sits inside its tile, so the cloth can hang
// from it.
const poleColumn = 1

// blitArt paints a drawing with its top-left at (x, y), each pixel scaled to a
// k-by-k block, mirrored when flip is set.
func (c *Canvas) blitArt(a art, x, y, k int, flip bool, pal palette) {
	w := a.width()
	for r, row := range a {
		for col := 0; col < len(row); col++ {
			clr, ok := pal(row[col])
			if !ok {
				continue
			}
			cx := col
			if flip {
				cx = w - 1 - col
			}
			c.FillRect(x+cx*k, y+r*k, k, k, clr)
		}
	}
}

// mushroomPalette colours the shared mushroom for one kind of pickup.
func mushroomPalette(t theme.Tokens, kind game.ItemKind) palette {
	c := t.Cast
	cap, rim := c.ShroomRed, c.ShroomRedRim
	switch kind {
	case game.ItemStar:
		cap, rim = c.ShroomBlue, c.ShroomBlueRim
	case game.ItemOneUp:
		cap, rim = c.ShroomGreen, c.ShroomGreenRim
	case game.ItemPoison:
		cap, rim = c.ShroomPurple, c.ShroomPurpleRim
	case game.ItemGold:
		cap, rim = c.ShroomGold, c.ShroomGoldRim
	}
	return func(l byte) (tcell.Color, bool) {
		switch l {
		case 'C':
			return cap, true
		case 'c':
			return rim, true
		case 'W':
			return c.ShroomSpot, true
		case 'S':
			return c.ShroomStem, true
		}
		return 0, false
	}
}

func goombaPalette(t theme.Tokens) palette {
	c := t.Cast
	return func(l byte) (tcell.Color, bool) {
		switch l {
		case 'T':
			return c.GoombaBody, true
		case 'W':
			return c.GoombaEye, true
		case 'f':
			return c.GoombaFeet, true
		}
		return 0, false
	}
}

func koopaPalette(t theme.Tokens) palette {
	c := t.Cast
	return func(l byte) (tcell.Color, bool) {
		switch l {
		case 'Y':
			return c.KoopaSkin, true
		case 'G':
			return c.KoopaShell, true
		case 'L':
			return c.KoopaShellHi, true
		case 'W':
			return c.KoopaRim, true
		}
		return 0, false
	}
}

func flagPalette(t theme.Tokens) palette {
	c := t.Cast
	return func(l byte) (tcell.Color, bool) {
		switch l {
		case 'P':
			return c.FlagPole, true
		case 'N':
			return c.FlagBall, true
		case 'n':
			return c.FlagBallShade, true
		case 'X':
			return c.FlagDark, true
		case 'w':
			return c.FlagLight, true
		}
		return 0, false
	}
}

// walkFrame picks the walking frame. It counts simulation steps, like the
// terrain animations, so the pace does not depend on the frame rate.
func walkFrame(step int) int { return (step / 16) & 1 }

// castScale is how many canvas pixels one pixel of the drawing covers.
func (d drawer) castScale() int {
	k := d.px / heroDesignPx
	if k < 1 {
		k = 1
	}
	return k
}

// standOn draws a drawing centred on a body and resting on its bottom edge.
func (d drawer) standOn(a art, b *game.Body, flip bool, pal palette) {
	k := d.castScale()
	centreX := WorldToArtScaled(b.Pos.X+b.W/2, d.px) - d.camX
	bottomY := roundArt(b.Pos.Y+b.H, d.px) - d.camY
	d.c.blitArt(a, centreX-a.width()*k/2, bottomY-len(a)*k, k, flip, pal)
}
