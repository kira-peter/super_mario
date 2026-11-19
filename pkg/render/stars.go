package render

import (
	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/game"
	"super-mario/pkg/level"
)

// Stars come out on a night sky.
//
// Only a level set at night has them: the night theme, or the night or starry
// sky (Scene.Stars, from level.Style.Night). Within that sky, each star also
// checks how dark the sky is where it sits, so a gradient that lightens toward
// the horizon has stars high up that thin out and stop where the sky gets
// light.
//
// Like the clouds, stars hang in the level: they scroll with the scenery and a
// level always has the same stars. Their positions come from a hash of the
// cell they sit in, so there is nothing to store and nothing random per run.
// Every star twinkles at random (see starColour), by the step count, so a
// paused game holds still.

// starCell is the side of the square, in four-pixel-tile art pixels, that holds
// at most one star. With one cell in two holding a star, all in the upper
// third of the sky (starBandOf), twenty-pixel cells leave a sparse sky of about
// two stars a screen, chosen by eye (TestStarsAreSparse).
const starCell = 20

// starDarkLimit is the brightest sky a star still shows on, as relative
// luminance in 0..1. The starry sky runs from 0.03 at the top to 0.15 at the
// horizon, so stars fill the upper sky and grow sparse toward the ground.
const starDarkLimit = 0.16

// starBandOf is the top fraction of the level, as 1/starBandOf, that stars may
// sit in: the upper third. The sky above the level's top edge, which a tall
// screen shows, counts as upper sky too.
const starBandOf = 3

// inStarBand reports whether a level row, in art pixels from the level's top
// (negative above it), is high enough for a star.
func inStarBand(y, levelH int) bool { return y < levelH/starBandOf }

// starHash mixes a cell's coordinates into 32 well-spread bits.
func starHash(x, y int) uint32 {
	h := uint32(x)*0x9e3779b1 ^ uint32(y)*0x85ebca77
	h ^= h >> 15
	h *= 0x2c1b3c6d
	h ^= h >> 12
	h *= 0x297a2d39
	h ^= h >> 15
	return h
}

// luminance is a colour's relative luminance, 0 for black and 1 for white.
func luminance(c tcell.Color) float64 {
	r, g, b := c.RGB()
	if r < 0 {
		return 1 // a default or unknown colour: treat it as light, draw no stars
	}
	return (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 255
}

// SkyStar is one star, at a position in the level measured in pixels of the
// four-pixel-tile drawing (so it is the same at every zoom).
type SkyStar struct {
	X, Y int
	// Dim marks the fainter stars.
	Dim bool
	// Cross marks the rare bright star that gets a small cross at a large zoom.
	Cross bool
	// fade is this star's threshold for a lightening sky: it shows while the
	// sky's luminance, as a fraction of starDarkLimit, is below it.
	fade float64
	hash uint32
}

// candidate returns the star a grid cell could hold, if it holds one.
func candidate(cx, cy int) (SkyStar, bool) {
	h := starHash(cx, cy)
	// About one cell in two holds a star.
	if h%2 != 0 {
		return SkyStar{}, false
	}
	// Where in the cell, kept off the edges so neighbours never touch.
	ox := 1 + int(h>>8&0xf)%(starCell-2)
	oy := 1 + int(h>>12&0xf)%(starCell-2)
	return SkyStar{
		X: cx*starCell + ox, Y: cy*starCell + oy,
		Dim: (h>>4)%3 == 0, Cross: h>>28 == 0,
		fade: float64(h>>20&0xff) / 255, hash: h,
	}, true
}

// visibleOn reports whether a star shows on a sky of the given luminance: only
// on dark sky, and fading out as the sky lightens toward the limit.
func (s SkyStar) visibleOn(lum float64) bool {
	return lum <= starDarkLimit && s.fade >= lum/starDarkLimit
}

// StarField lists every star of a level, for the editor's canvas, which draws
// its own sky. sky gives the sky colour at a fraction of the way down the
// level (0 at the top). It is empty unless the level is set at night.
func StarField(st level.Style, width, height int, sky func(f float64) tcell.Color) []SkyStar {
	if !st.Night() {
		return nil
	}
	var out []SkyStar
	w, h := width*heroDesignPx, height*heroDesignPx
	for cy := 0; cy*starCell < h; cy++ {
		for cx := 0; cx*starCell < w; cx++ {
			s, ok := candidate(cx, cy)
			if !ok || s.X >= w || !inStarBand(s.Y, h) {
				continue
			}
			if !s.visibleOn(luminance(sky(float64(s.Y) / float64(h-1)))) {
				continue
			}
			out = append(out, s)
		}
	}
	return out
}

// stars draws the star field over the sky and behind everything else.
func (d drawer) stars(w *game.World) {
	if !d.s.Stars || d.s.Transparent || !d.s.T.Full {
		return
	}
	k := d.castScale()
	levelH := w.Level.Height * d.px
	levelW := w.Level.Width * d.px

	// Grid cells, in level art pixels at the four-pixel scale, that the
	// screen covers.
	cell := starCell * k
	cx0, cx1 := floorDiv(d.camX, cell), floorDiv(d.camX+d.artW, cell)
	cy0, cy1 := floorDiv(d.camY, cell), floorDiv(d.camY+d.artH, cell)
	step := w.StepCount()

	for cy := cy0; cy <= cy1; cy++ {
		for cx := cx0; cx <= cx1; cx++ {
			s, ok := candidate(cx, cy)
			if !ok {
				continue
			}
			lx, ly := s.X*k, s.Y*k
			if lx < 0 || lx >= levelW || !inStarBand(ly, levelH) {
				continue // over the level's width, in the upper third of the sky
			}
			x, y := lx-d.camX, ly-d.camY
			if x < 0 || y < 0 || x >= d.artW || y >= d.artH {
				continue
			}
			// Only where nothing but sky is drawn yet, and only dark sky.
			bg, ok := d.c.Px[y*d.c.W+x].(tcell.Color)
			if !ok || !s.visibleOn(luminance(bg)) {
				continue
			}
			col, lit, flare := d.starColour(s.hash, step)
			if !lit || col == bg {
				continue // out for this moment of its twinkle
			}
			// One art pixel; a rare bright one flares into a cross at a large
			// zoom, only at its brightest.
			d.c.FillRect(x, y, k, k, col)
			if s.Cross && flare && k >= 2 {
				d.c.FillRect(x-k, y, k, k, d.s.T.Cast.CloudShade)
				d.c.FillRect(x+k, y, k, k, d.s.T.Cast.CloudShade)
				d.c.FillRect(x, y-k, k, k, d.s.T.Cast.CloudShade)
				d.c.FillRect(x, y+k, k, k, d.s.T.Cast.CloudShade)
			}
		}
	}
}

// starColour is a star's colour this step, and whether it shows at all.
//
// Every star twinkles at random. Time is cut into short slices, and in each
// slice the star draws its brightness afresh from a hash of the star and the
// slice: mostly lit, sometimes dimmed, now and then gone for a moment. Each
// star's slices are a different length and start at a different moment, so
// no two stars change together and the sky shimmers rather than blinks. It is
// all a function of the step count, so it is the same on every replay and
// holds still when the game is paused.
func (d drawer) starColour(h uint32, step int) (col tcell.Color, lit, flare bool) {
	bright := d.s.T.Cast.Cloud
	dim := blend(d.s.T.Cast.CloudShade, d.s.T.Sky, 0.35)
	if (h>>4)%3 == 0 {
		// Some stars are fainter to begin with, which gives the sky depth.
		bright = d.s.T.Cast.CloudShade
		dim = blend(bright, d.s.T.Sky, 0.55)
	}
	// A slice of 6 to 13 steps: a tenth to a fifth of a second.
	slice := 6 + int(h>>16)%8
	n := (step + int(h>>3&0xff)) / slice
	r := starHash(int(h), n) % 100
	switch {
	case r < 7: // out for a moment
		return 0, false, false
	case r < 30: // dimmed
		return dim, true, false
	case r < 36: // at its brightest
		return bright, true, true
	}
	return bright, true, false
}

// Twinkle is a star's colour at a simulation step in a scene's palette, for
// the editor's canvas: the same rule the game draws by. ok is false while the
// star is out.
func (s *Scene) Twinkle(st SkyStar, step int) (tcell.Color, bool) {
	c, lit, _ := drawer{s: s}.starColour(st.hash, step)
	return c, lit
}

// floorDiv divides rounding towards minus infinity, so cells to the left of
// or above the level's origin still map to a stable grid.
func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}
