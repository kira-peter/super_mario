package render

import (
	"math"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/game"
)

// The death and the return, drawn pixel by pixel.
//
// When he dies he hops up and hangs, fading to grey and trembling. Then the
// drawing breaks apart: every pixel of it becomes a block of its own and flies
// off like the pieces of a smashed brick, up and out, then down under gravity.
// After the camera has glided back to the start, the pixels come back the
// other way: each one flies in from somewhere round the edge of the screen and
// they meet in mid-air, where he is whole again before he drops to the ground.
//
// Every pixel's path comes from a hash of its place in the drawing, worked out
// from the step count alone, so there is nothing to store between frames and a
// replay draws the same pieces.

// heroPixel is one pixel of his drawing, relative to the drawing's top-left.
type heroPixel struct {
	x, y int
	col  tcell.Color
}

// heroPixels lists the drawn pixels of a sprite at its drawn size.
func (d drawer) heroPixels(s Sprite, w, h int, tint func(tcell.Color) tcell.Color) []heroPixel {
	var out []heroPixel
	for dy := 0; dy < h; dy++ {
		sy := dy * s.H / h
		row := s.Rows[sy]
		for dx := 0; dx < w; dx++ {
			sx := dx * s.W / w
			if sx >= len(row) || row[sx] == SlotNone {
				continue
			}
			c, ok := SlotColor(d.s.T, row[sx])
			if !ok {
				continue
			}
			tc, ok := c.(tcell.Color)
			if !ok {
				continue
			}
			if tint != nil {
				tc = tint(tc)
			}
			out = append(out, heroPixel{dx, dy, tc})
		}
	}
	return out
}

// pixelHash spreads a pixel's place into random-looking numbers in 0..1.
func pixelHash(x, y, salt int) (a, b, c float64) {
	h := uint32(x*73856093) ^ uint32(y*19349663) ^ uint32(salt*83492791)
	h ^= h >> 13
	h *= 0x5bd1e995
	h ^= h >> 15
	return float64(h&0x3ff) / 1023, float64(h>>10&0x3ff) / 1023, float64(h>>20&0x3ff) / 1023
}

// greyFade blends a colour toward its grey by f (0 none, 1 all grey).
func greyFade(c tcell.Color, f float64) tcell.Color {
	if f <= 0 {
		return c
	}
	if f >= 1 {
		return greyOf(c)
	}
	return blend(c, greyOf(c), f)
}

// deadHero draws him through the death: the hop, the hang turning grey with a
// shiver, then the burst into flying pixels.
func (d drawer) deadHero(w *game.World, hero HeroDrawing, x, y, sw, sh int) {
	if w.Player.Melting {
		d.meltingHero(w, hero, x, y, sw, sh)
		return
	}
	t := w.Player.DeathTimer()
	switch {
	case t < game.DeathRise:
		d.c.DrawSized(hero.Sprite, x, y, sw, sh, d.s.T)
		return
	case t < game.DeathBurst:
		// Hanging at the top: the colour drains away, and in the last moments
		// he trembles before he breaks.
		f := float64(t-game.DeathRise) / float64(game.DeathHang)
		tint := func(c tcell.Color) tcell.Color { return greyFade(c, f*1.4) }
		if left := game.DeathBurst - t; left < 12 && left%2 == 0 {
			x += 1 - 2*(left/2%2)
		}
		d.c.DrawTinted(hero.Sprite, x, y, sw, sh, d.s.T, tint, false)
		return
	}

	// Burst: each pixel a block flying off like a piece of smashed brick.
	f := float64(t - game.DeathBurst)
	if f > game.DeathScatter {
		return
	}
	k := float64(d.px) / heroDesignPx // art pixels per design pixel
	size := max(1, int(k+0.5))
	cx, cy := float64(sw)/2, float64(sh)/2
	gravity := 0.045 * k
	for _, p := range d.heroPixels(hero.Sprite, sw, sh, greyOf) {
		a, b, c := pixelHash(p.x, p.y, 1)
		// Out from the middle of him, mostly upward, with some randomness.
		ox, oy := float64(p.x)-cx, float64(p.y)-cy
		vx := (ox*0.09 + (a-0.5)*1.4) * k
		vy := (oy*0.06 - 1.1 - b*1.3) * k
		px := float64(x+p.x) + vx*f
		py := float64(y+p.y) + vy*f + gravity*f*f
		// The pieces darken a little as they fall away.
		col := p.col
		if f > game.DeathScatter*0.6 && c < 0.5 {
			col = blend(col, d.skyAt(int(py), w).(tcell.Color), 0.5)
		}
		d.c.FillRect(int(math.Round(px)), int(math.Round(py)), size, size, col)
	}
}

// meltingHero is a death by poison. For the first stretch he stands there
// turning a sick green and swaying; then he melts: each pixel lets go in turn,
// the top of him first, and drips slowly down, darkening into a puddle on the
// ground that seeps away.
func (d drawer) meltingHero(w *game.World, hero HeroDrawing, x, y, sw, sh int) {
	t := w.Player.DeathTimer()
	sick := sickTint(game.PoisonSteps)
	if t < game.DeathBurst {
		// Sickening: green, swaying from side to side, slumping a little.
		sway := int(math.Round(math.Sin(float64(t)*0.35) * 1.2))
		d.c.DrawTinted(hero.Sprite, x+sway, y, sw, sh, d.s.T, sick, false)
		return
	}
	m := float64(t - game.DeathBurst)
	if m > game.MeltSteps {
		return
	}
	k := float64(d.px) / heroDesignPx
	size := max(1, int(k+0.5))
	floor := float64(y + sh) // the ground he stood on
	for _, p := range d.heroPixels(hero.Sprite, sw, sh, sick) {
		a, b, _ := pixelHash(p.x, p.y, 4)
		// Top rows let go first; each pixel a little sooner or later.
		release := float64(p.y)/float64(sh)*18 + a*14
		if m < release {
			// Still in place, but sagging and darkening toward a puddle.
			sag := (m / (release + 1)) * 1.5 * k
			col := blend(p.col, poisonDeep, m/(release+20))
			d.c.FillRect(x+p.x, y+p.y+int(sag), size, size, col)
			continue
		}
		// Dripping: slow, as if thick, speeding up only a little.
		f := m - release
		py := float64(y+p.y) + 0.10*k*f + 0.0035*k*f*f
		px := float64(x+p.x) + math.Sin(f*0.15+b*6)*0.6*k
		col := blend(p.col, poisonDeep, math.Min(1, f/40))
		if py >= floor-float64(size) {
			// On the ground: spread into a puddle that seeps away.
			spread := (b - 0.5) * math.Min(f, 40) * 0.12 * k
			px = float64(x+p.x) + spread
			py = floor - float64(size)
			gone := (m - (release + 30)) / (game.MeltSteps - release - 30)
			if gone > 0 {
				if gone >= 1 || b < gone {
					continue
				}
				col = blend(col, d.skyAt(int(py), w).(tcell.Color), gone)
			}
		}
		d.c.FillRect(int(math.Round(px)), int(math.Round(py)), size, size, col)
	}
}

// gatheringHero draws the pixels flying in from all round the screen and
// meeting in the drawing, in full colour, where he hangs in mid-air.
func (d drawer) gatheringHero(w *game.World, hero HeroDrawing, x, y, sw, sh int) {
	f := w.RespawnGathering()
	n := float64(game.RespawnGather)
	k := float64(d.px) / heroDesignPx
	size := max(1, int(k+0.5))
	for _, p := range d.heroPixels(hero.Sprite, sw, sh, nil) {
		a, b, c := pixelHash(p.x, p.y, 2)
		// Each pixel sets off a little after the one before, from a point on a
		// ring round the screen, so they arrive as a swirl rather than at once.
		delay := c * n * 0.35
		u := (float64(f) - delay) / (n - delay)
		if u < 0 {
			continue
		}
		if u > 1 {
			u = 1
		}
		// Ease out: fast from far away, slowing as it settles into place.
		e := 1 - math.Pow(1-u, 3)
		angle := a * 2 * math.Pi
		dist := (float64(d.artW+d.artH) * 0.4) * (0.6 + 0.4*b)
		sx := float64(x+p.x) + math.Cos(angle)*dist
		sy := float64(y+p.y) + math.Sin(angle)*dist
		// Curve the path a little so it spirals in.
		curl := math.Sin(u*math.Pi) * dist * 0.25
		px := sx + (float64(x+p.x)-sx)*e - math.Sin(angle)*curl
		py := sy + (float64(y+p.y)-sy)*e + math.Cos(angle)*curl
		col := p.col
		if u < 1 {
			// Bright while in flight, its own colour once it has landed.
			col = blend(col, d.s.T.Cast.Cloud, 0.45*(1-e))
		}
		d.c.FillRect(int(math.Round(px)), int(math.Round(py)), size, size, col)
	}
}
