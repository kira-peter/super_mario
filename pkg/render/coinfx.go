package render

import (
	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/game"
	"super-mario/pkg/render/theme"
)

// CoinCounterCol is the screen column of the coin icon in the status line,
// where flying coins head for and the "+1" appears. It is the rune offset of
// "ⓒ" in the text HUDRow writes.
const CoinCounterCol = 15

// coinTarget is where a flying coin lands, in canvas pixels: the coin counter,
// which sits on the status line just above the top of the picture.
func (d drawer) coinTarget() (int, int) {
	return CoinCounterCol, -1
}

// coins draws the coins popping out of blocks and flying to the counter.
//
// The counter is at the top of the screen, so a coin rises to it. It starts
// slowly and speeds up as it goes, which reads as being pulled in rather than
// sliding along a rail.
func (d drawer) coins(w *game.World) {
	k := d.castScale()
	pal := func(l byte) (tcell.Color, bool) {
		switch l {
		case '$':
			return d.s.T.Coin, true
		case '*':
			return d.s.T.CoinHi, true
		case 'c':
			return d.s.T.CoinShade, true
		}
		return 0, false
	}
	coin := art(terrain4.Coin[0][:])
	size := 4 * k
	tx, ty := d.coinTarget()

	for _, c := range w.CoinFX {
		if c.Fly < 0 {
			continue // waiting for its turn
		}
		sx := WorldToArtScaled(c.Pos.X, d.px) - d.camX
		sy := WorldToArtScaled(c.Pos.Y, d.px) - d.camY
		x, y := sx, sy
		if c.Pop == 0 {
			p := c.Progress()
			e := p * p
			x = sx + int(float64(tx-sx)*e)
			y = sy + int(float64(ty-sy)*e)
		}
		d.c.blitArt(coin, x-size/2, y-size/2, k, false, pal)
	}
}

// CoinPlusStyle is how the "+1" at the coin counter looks at an age, and
// whether it shows at all.
//
// A terminal cannot draw text larger, so "growing" is done with weight: it
// starts dim, pops to bold and bright, holds, and then fades back into the sky
// colour until it is gone.
func CoinPlusStyle(t theme.Tokens, age int) (tcell.Style, bool) {
	if age >= game.CoinPlusSteps {
		return tcell.StyleDefault, false
	}
	green := t.Cast.CoinPlus
	st := tcell.StyleDefault.Background(t.Sky)
	const pop, hold = 6, 24
	switch {
	case age < pop:
		return st.Foreground(blend(t.Sky, green, float64(age+1)/pop)), true
	case age < hold:
		return st.Foreground(green).Bold(true), true
	default:
		f := 1 - float64(age-hold)/float64(game.CoinPlusSteps-hold)
		return st.Foreground(blend(t.Sky, green, f)), true
	}
}

// blend mixes two colours, f of the way from a to b. A colour without RGB
// (a monochrome terminal's default) gives b, so the text shows rather than
// disappearing.
func blend(a, b tcell.Color, f float64) tcell.Color {
	ar, ag, ab := a.RGB()
	br, bg, bb := b.RGB()
	if ar < 0 || br < 0 {
		return b
	}
	mix := func(x, y int32) int32 { return x + int32(float64(y-x)*f) }
	return tcell.NewRGBColor(mix(ar, br), mix(ag, bg), mix(ab, bb))
}
