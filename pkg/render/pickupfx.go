package render

import (
	"math"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/game"
)

// Pickup effects: the extra life and the poison mushroom.
//
// The extra life: a copy of every pixel of him lifts off, scatters upward and
// gathers above him into a heart, turning red on the way; the finished heart
// then floats to the lives counter at the top right. He carries on meanwhile.
// The poison mushroom: a cloud of purple bubbles boils up round him and he
// turns a sickly green for a moment before it clears.

var (
	heartRed    = tcell.NewRGBColor(0xe8, 0x2a, 0x3c)
	heartShade  = tcell.NewRGBColor(0xb4, 0x1c, 0x2c)
	heartLight  = tcell.NewRGBColor(0xff, 0x9c, 0xa8)
	poisonDeep  = tcell.NewRGBColor(0x6a, 0x2c, 0x9a)
	poisonLight = tcell.NewRGBColor(0xb0, 0x6c, 0xe0)
	sickGreen   = tcell.NewRGBColor(0x8c, 0xb0, 0x4a)
)

// heartShape is the heart his pixels assemble into, small and without an
// outline: x the red, s a darker red shading the lower right, h a single
// highlight on the left lobe.
var heartShape = []string{
	".xx.xx.",
	"xhxxxxx",
	"xxxxxxs",
	".xxxxs.",
	"..xxs..",
	"...s...",
}

// heartCell is one cell of an assembled shape and its colour.
type heartCell struct {
	x, y int
	col  tcell.Color
}

// cellsOf lists a shape's filled cells, colouring each letter with pal.
func cellsOf(shape []string, pal map[byte]tcell.Color) []heartCell {
	var out []heartCell
	for y, row := range shape {
		for x := 0; x < len(row); x++ {
			if c, ok := pal[row[x]]; ok {
				out = append(out, heartCell{x, y, c})
			}
		}
	}
	return out
}

// heartCells lists the heart's cells.
func heartCells() []heartCell {
	return cellsOf(heartShape, map[byte]tcell.Color{'x': heartRed, 's': heartShade, 'h': heartLight})
}

// goldCoinShape is the gold mushroom's coin: the same gold as a coin in the
// level, a size up, flat, with a glint at the top left and the slot down the
// middle of the face.
//
//	$ gold   c shade   * glint   r ring (darker gold)   s slot (darkest)
var goldCoinShape = []string{
	".$$$$.",
	"$**$$$",
	"$*$s$$",
	"$$$s$$",
	"$$$$$$",
	".$$$$.",
}

// goldCells lists the gold coin's cells in the theme's coin colours, the same
// colours the level's coin uses, with the ring and slot a step darker.
func (d drawer) goldCells() []heartCell {
	t := d.s.T
	return cellsOf(goldCoinShape, map[byte]tcell.Color{
		'$': t.Coin,
		'*': t.CoinHi,
		's': t.CoinShade,
	})
}

// effects draws the pickup effects over the scene.
func (d drawer) effects(w *game.World) {
	k := float64(d.px) / heroDesignPx
	size := max(1, int(k+0.5))
	for _, e := range w.Effects {
		switch e.Kind {
		case game.EffectOneUp:
			// The lives counter is on the left of the status line.
			d.assemble(w, e, size, heartShape, heartCells(), heartLight, LivesCounterCol, false)
		case game.EffectGold:
			// The coin counter is near the start of the status line.
			d.assemble(w, e, size, goldCoinShape, d.goldCells(), d.s.T.CoinHi, CoinCounterCol, false)
		case game.EffectPoison:
			cx := float64(WorldToArtScaled(e.Pos.X, d.px) - d.camX)
			cy := float64(WorldToArtScaled(e.Pos.Y, d.px) - d.camY)
			d.poisonMist(w, cx, cy, float64(e.Age)/game.EffectSteps, k, size, e.Age)
		}
	}
}

// Phases of the assembling effects, as fractions of their length: the pixels
// gather, the shape hangs a moment, then a short quick flight to the counter.
const (
	heartGather = 0.52 // the pixels have assembled into the shape
	heartHold   = 0.72 // it has hung there a moment, beating
)

// assemble: his pixels, copied, rise and gather into a shape (the 1UP heart,
// the gold coin) above where he ate the mushroom; the shape beats for a moment
// and then floats to its counter in the status line, at canvas column targetX.
// shadeKeep names the cells that do not light up on the beat; centre says
// whether the shape lands with its middle (true) or its left edge on targetX.
func (d drawer) assemble(w *game.World, e *game.Effect, size int, shape []string, cells []heartCell, light tcell.Color, targetX int, centre bool) {
	t := float64(e.Age) / float64(e.Steps())

	// His drawing as it was, where he ate it in the level: the copy and the
	// shape it gathers into stay put in the scenery, scrolling with it, until
	// the shape flies off to the status line.
	hero := Hero(e.Big, HeroStand, 0)
	if e.Facing < 0 {
		hero = hero.Mirrored()
	}
	sw := max(1, hero.Sprite.W*d.px/heroDesignPx)
	sh := max(1, hero.Sprite.H*d.px/heroDesignPx)
	camX, camY := d.camX, d.camY
	x0 := WorldToArtScaled(e.Pos.X, d.px) - camX - (hero.Width*d.px/heroDesignPx)/2
	y0 := roundArt(e.Pos.Y, d.px) - camY - sh
	pixels := d.heroPixels(hero.Sprite, sw, sh, nil)
	if len(pixels) == 0 {
		return
	}

	// The shape: a few rows above his head. Every cell of it is filled, each
	// fed by one of his pixels (some pixels feed two cells when he has fewer
	// pixels than the shape has cells), so the shape is always whole.
	cell := size
	hw := len(shape[0]) * cell
	hh := len(shape) * cell
	hx := x0 + sw/2 - hw/2
	hy := y0 - hh - 3*size

	// Once whole, the shape flies to its counter, just above the picture. The
	// flight is worked out afresh each frame from where the shape is on the
	// screen now (it scrolls with the level) to the counter, which does not
	// move, so it always lands exactly on the counter whatever the camera did.
	tx := float64(targetX)
	if centre {
		tx -= float64(hw / 2)
	}
	ty := float64(-hh)
	var fx, fy float64
	flying := t > heartHold
	if flying {
		u := (t - heartHold) / (1 - heartHold)
		k := u * u // accelerating: pulled in to the counter
		fx = (tx - float64(hx)) * k
		fy = (ty - float64(hy)) * k
		if u > 0.9 && (e.Age/2)%2 == 0 {
			return // a blink as it reaches the counter
		}
	}
	beat := false
	if t >= heartGather && !flying {
		beat = math.Sin((t-heartGather)/(heartHold-heartGather)*2*math.Pi) > 0.3
	}

	for i, hc := range cells {
		p := pixels[i*len(pixels)/len(cells)]
		gx := float64(hx + hc.x*cell)
		gy := float64(hy + hc.y*cell)
		sx := float64(x0 + p.x)
		sy := float64(y0 + p.y)

		px, py := gx, gy
		col := hc.col
		if t < heartGather {
			// Lift off, scatter up and out, then settle into place.
			a, b, cc := pixelHash(hc.x, hc.y, 5)
			delay := cc * 0.12
			u := math.Min(1, math.Max(0, (t-delay)/(heartGather-delay)))
			ease := 1 - math.Pow(1-u, 3)
			spread := math.Sin(u*math.Pi) * (a - 0.5) * 16 * float64(size)
			lift := math.Sin(u*math.Pi) * (3 + b*5) * float64(size)
			px = sx + (gx-sx)*ease + spread
			py = sy + (gy-sy)*ease - lift
			// Turning from his colour to the shape's as it goes.
			col = blend(p.col, hc.col, math.Min(1, u*1.4))
		} else if beat && hc.col == cells[0].col {
			col = blend(hc.col, light, 0.3)
		}
		px += fx
		py += fy
		d.c.FillRect(int(math.Round(px)), int(math.Round(py)), cell, cell, col)
	}
}

// poisonMist: purple bubbles boiling up round him, swelling and popping.
func (d drawer) poisonMist(w *game.World, cx, cy, t, k float64, size, age int) {
	for i := 0; i < 14; i++ {
		a, b, c := pixelHash(i, 7, 3)
		start := c * 0.4
		u := (t - start) / (1 - start)
		if u <= 0 || u >= 1 {
			continue
		}
		// Out from him in a ring, then up, wobbling.
		ang := a * 2 * math.Pi
		r := (3 + u*9*b) * k
		x := cx + math.Cos(ang)*r + math.Sin(u*9+a*6)*1.2*k
		y := cy + math.Sin(ang)*r*0.6 - u*14*k
		// A bubble swells, then pops into a dot.
		rad := 1
		if u > 0.3 && u < 0.75 {
			rad = 2
		}
		col := poisonLight
		if b < 0.5 {
			col = poisonDeep
		}
		if u > 0.75 {
			col = blend(col, d.skyAt(int(y), w).(tcell.Color), (u-0.75)/0.25)
		}
		d.c.FillRect(int(x)-(rad-1)*size/2, int(y)-(rad-1)*size/2, rad*size, rad*size, col)
	}
}

// sickTint turns his colours toward a sickly green while he is poisoned,
// pulsing, and fading back to normal at the end.
func sickTint(left int) func(tcell.Color) tcell.Color {
	f := 0.55
	if left < 30 {
		f *= float64(left) / 30
	}
	pulse := 0.75 + 0.25*math.Sin(float64(left)*0.4)
	return func(c tcell.Color) tcell.Color { return blend(c, sickGreen, f*pulse) }
}
