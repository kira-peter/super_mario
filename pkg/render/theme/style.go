package theme

import (
	"math"

	"github.com/gdamore/tcell/v2"
	"github.com/lucasb-eyer/go-colorful"

	"super-mario/pkg/level"
)

// Applying a level's style (spec level-appearance).
//
// A level picks one main colour per element. Everything else an element is
// drawn with, its lit edge, its shade, the mortar of a brick or the soil under
// the turf, is worked out from that one colour here, at fixed lightness steps in
// OKLCH. Working in OKLCH keeps the steps looking the same size whatever the
// hue: in RGB, darkening yellow and darkening blue by the same amount do not
// look like the same change (design decision 8).

// SkyFill is what the renderer paints behind everything: one colour, or a
// gradient from the top of the level to the horizon.
type SkyFill struct {
	Top, Bottom tcell.Color
	// bands, when set, is the gradient already reduced to what a 256-colour
	// terminal can show: a run of palette colours from top to bottom. At reads
	// from it instead of blending.
	bands []tcell.Color
}

// Gradient reports whether the two ends differ.
func (s SkyFill) Gradient() bool { return s.Top != s.Bottom }

// At returns the sky colour at a fraction of the way down, 0 at the top and 1
// at the bottom.
func (s SkyFill) At(f float64) tcell.Color {
	if !s.Gradient() {
		return s.Top
	}
	if f < 0 {
		f = 0
	}
	if f > 1 {
		f = 1
	}
	if n := len(s.bands); n > 0 {
		i := int(f * float64(n-1))
		return s.bands[i]
	}
	if f == 0 {
		return s.Top
	}
	if f == 1 {
		return s.Bottom
	}
	return fromColorful(toColorful(s.Top).BlendOkLab(toColorful(s.Bottom), f).Clamped())
}

// Apply returns the palette for a level: base with the level's style on top.
//
// Only the tokens the scenery is drawn with change. The HUD, the menus, Mario,
// the enemies and the pickups keep their theme colours, so a level's look never
// reaches past its own picture. A zero style returns base untouched, which is
// what keeps old levels pixel-identical.
//
// c is what the terminal can show. On a mono terminal the style is ignored; on
// a 256-colour one the sky is reduced to bands and any scenery colour that
// would come out as the same palette entry as the sky is moved off it.
func Apply(base Tokens, s level.Style, c Capacity) (Tokens, SkyFill) {
	t := base
	sky := SkyFill{Top: base.Sky, Bottom: base.Sky}
	if s.IsZero() || !base.Full || c == Mono {
		// A mono terminal ignores level colours entirely; its greys are chosen
		// for structure, and a level colour could only make them worse.
		return t, sky
	}
	r := s.Resolved()

	if r.Sky != nil {
		sky = SkyFill{Top: rgb(r.Sky.Top), Bottom: rgb(r.Sky.Bottom)}
		// The flat token is the horizon: it is what sits directly behind the
		// terrain, the pipes' insets and every sprite's transparent edge.
		t.Sky = sky.Bottom
		t.Cast.Cloud, t.Cast.CloudShade = cloudsFor(sky)
	}
	if r.Ground != nil {
		g := toColorful(rgb(*r.Ground))
		t.Ground = fromColorful(g)
		t.GroundDeep = shift(g, -0.10, 1.0)
		t.GroundEdge = shift(g, +0.10, 1.0)
	}
	if r.Turf != nil {
		g := toColorful(rgb(*r.Turf))
		t.Cast.Grass = fromColorful(g)
		t.Cast.GrassDark = shift(g, -0.12, 1.0)
	}
	if r.Brick != nil {
		b := toColorful(rgb(*r.Brick))
		t.Brick = fromColorful(b)
		t.BrickEdge = shift(b, +0.12, 1.0)
		t.BrickMortar = shift(b, -0.16, 0.8)
	}
	if r.Stair != nil {
		st := toColorful(rgb(*r.Stair))
		t.Stone = fromColorful(st)
		t.StoneLight = shift(st, +0.12, 0.9)
		t.StoneDark = shift(st, -0.14, 1.0)
	}
	if r.Pipe != nil {
		p := toColorful(rgb(*r.Pipe))
		t.Pipe = fromColorful(p)
		t.PipeHi = shift(p, +0.14, 1.0)
		t.PipeLo = shift(p, -0.10, 1.0)
	}
	if c == Color256 {
		sky = bandSky(sky)
		separateFromSky(&t, sky)
	}
	return t, sky
}

// --- 256 colours ---------------------------------------------------------

// skyBands is how finely a gradient is sampled before it is reduced to the
// palette. Neighbouring samples that land on the same palette entry merge, so
// this only bounds how many bands there can be.
const skyBands = 48

// palette256 is the terminal's 256 colours, the same list tcell fits to.
var palette256 = func() []tcell.Color {
	p := make([]tcell.Color, 256)
	for i := range p {
		p[i] = tcell.PaletteColor(i)
	}
	return p
}()

// Quantize256 is the colour a 256-colour terminal will actually show for c.
func Quantize256(c tcell.Color) tcell.Color {
	if c == tcell.ColorDefault || !c.Valid() {
		return c
	}
	if !c.IsRGB() {
		return c
	}
	return tcell.FindColor(c, palette256)
}

// bandSky reduces a gradient to palette colours. Each sample takes its nearest
// palette colour, except that a band may never step back the way the gradient
// came: the nearest colour to a slightly lighter blue can be a darker teal, and
// a sky that went dark-light-dark-light would read as stripes, not a sky.
func bandSky(s SkyFill) SkyFill {
	if !s.Gradient() {
		q := Quantize256(s.Top)
		return SkyFill{Top: q, Bottom: q}
	}
	rising := Luminance(s.Bottom) >= Luminance(s.Top)
	bands := make([]tcell.Color, skyBands)
	for i := range bands {
		f := float64(i) / float64(skyBands-1)
		q := Quantize256(s.At(f))
		if i > 0 {
			prev := bands[i-1]
			if rising && Luminance(q) < Luminance(prev) || !rising && Luminance(q) > Luminance(prev) {
				q = prev
			}
		}
		bands[i] = q
	}
	out := SkyFill{Top: bands[0], Bottom: bands[len(bands)-1], bands: bands}
	if out.Top == out.Bottom {
		// The whole gradient fell into one palette colour.
		out.bands = nil
	}
	return out
}

// skyColours lists every colour the sky can show.
func skyColours(s SkyFill) []tcell.Color {
	if len(s.bands) == 0 {
		return []tcell.Color{s.Top, s.Bottom}
	}
	return s.bands
}

// separateFromSky moves any scenery colour that a 256-colour terminal would
// show as the same palette entry as some part of the sky. Without this a pale
// blue brick on a pale blue sky can come out as one flat colour, and the brick
// simply vanishes.
func separateFromSky(t *Tokens, sky SkyFill) {
	skies := skyColours(sky)
	clash := func(c tcell.Color) bool {
		q := Quantize256(c)
		for _, s := range skies {
			if q == s {
				return true
			}
		}
		return false
	}
	fix := func(c *tcell.Color) {
		if !clash(*c) {
			return
		}
		*c = nearestApart(*c, skies)
	}
	for _, c := range []*tcell.Color{
		&t.Ground, &t.GroundEdge, &t.GroundDeep,
		&t.Brick, &t.BrickEdge, &t.BrickMortar,
		&t.Stone, &t.StoneLight, &t.StoneDark,
		&t.Pipe, &t.PipeHi, &t.PipeLo,
		&t.Cast.Grass, &t.Cast.GrassDark,
	} {
		fix(c)
	}
}

// nearestApart is the palette colour nearest to c that is none of avoid,
// preferring to move away from the sky in lightness so the element stays the
// same shade relative to its surroundings.
func nearestApart(c tcell.Color, avoid []tcell.Color) tcell.Color {
	banned := map[tcell.Color]bool{}
	var skyL float64
	for _, a := range avoid {
		banned[a] = true
		skyL += Luminance(a)
	}
	skyL /= float64(len(avoid))
	away := Luminance(c) >= skyL // lighter than the sky: stay lighter

	src := toColorful(c)
	best, bestD := tcell.ColorDefault, math.Inf(1)
	for _, p := range palette256 {
		if banned[p] {
			continue
		}
		if (Luminance(p) >= skyL) != away {
			continue
		}
		if d := src.DistanceCIE76(toColorful(p)); d < bestD {
			best, bestD = p, d
		}
	}
	if best == tcell.ColorDefault {
		// Nothing on the preferred side (the sky is pure white or black):
		// take the nearest on either side.
		for _, p := range palette256 {
			if banned[p] {
				continue
			}
			if d := src.DistanceCIE76(toColorful(p)); d < bestD {
				best, bestD = p, d
			}
		}
	}
	return best
}

// cloudsFor picks cloud colours that stay visible on this sky. On a dark sky
// clouds are white, as they always were; on a light one pure white disappears,
// so they turn a pale grey with a darker underside.
func cloudsFor(s SkyFill) (tcell.Color, tcell.Color) {
	// The clouds sit in the upper half, so judge by the colour there.
	behind := toColorful(s.At(0.3))
	l, _, _ := behind.OkLch()
	if l < 0.72 {
		return rgb(0xffffff), rgb(0xcfdcef)
	}
	return shift(behind, -0.12, 0.25), shift(behind, -0.22, 0.25)
}

// shift moves a colour's lightness by dl and scales its chroma, in OKLCH.
func shift(c colorful.Color, dl, chroma float64) tcell.Color {
	l, ch, h := c.OkLch()
	l = math.Max(0.05, math.Min(0.97, l+dl))
	return fromColorful(colorful.OkLch(l, ch*chroma, h).Clamped())
}

func rgb(v level.RGB) tcell.Color { return tcell.NewHexColor(int32(v & 0xffffff)) }

func toColorful(c tcell.Color) colorful.Color {
	r, g, b := c.RGB()
	return colorful.Color{R: float64(r) / 255, G: float64(g) / 255, B: float64(b) / 255}
}

func fromColorful(c colorful.Color) tcell.Color {
	r, g, b := c.Clamped().RGB255()
	return tcell.NewRGBColor(int32(r), int32(g), int32(b))
}

// Luminance is the WCAG relative luminance of a colour, 0 for black and 1 for
// white.
func Luminance(c tcell.Color) float64 {
	r, g, b := c.RGB()
	lin := func(v int32) float64 {
		s := float64(v) / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

// Contrast is the WCAG contrast ratio between two colours, from 1 (the same)
// to 21 (black on white).
func Contrast(a, b tcell.Color) float64 {
	la, lb := Luminance(a), Luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// Distinctness is how different two colours look, as CIEDE2000 on a 0-100
// scale: 0 is identical, about 2 is the smallest difference most people
// notice side by side, and 40 or more is unmistakable.
//
// It is the measure for "can a sprite be told from the sky behind it". WCAG
// contrast only compares lightness, and would call a gold coin on a pale blue
// sky unreadable when hue alone makes it leap out.
func Distinctness(a, b tcell.Color) float64 {
	return toColorful(a).DistanceCIEDE2000(toColorful(b)) * 100
}
