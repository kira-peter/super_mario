package render

import (
	"fmt"
	"math"
	"strings"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/engine"
	"super-mario/pkg/game"
	"super-mario/pkg/level"
	"super-mario/pkg/render/theme"
)

// ArtPerTile is the authored resolution of the detailed sprite set.
const ArtPerTile = 16

// Scene draws one frame of the world onto a canvas.
type Scene struct {
	T     theme.Tokens
	Theme string
	// Sky holds the drifting clouds. Nil draws a clear sky, which is what the
	// tests that count pixels want.
	Sky *Sky
	// Fill is the sky behind everything. The zero value paints T.Sky, flat,
	// which is the look of a level with no style.
	Fill theme.SkyFill
	// Transparent leaves the sky unpainted, so only what stands in the level
	// is drawn and every other pixel stays nil. The editor's sprite atlas is
	// cut from scenes drawn this way, so each element looks exactly as the
	// game draws it.
	Transparent bool
	// Stars puts a star field on the sky: set for a level at night.
	Stars bool
}

// ForLevel returns a scene for a level: the theme with the level's own colours
// applied for what the terminal can show. The HUD is not drawn by the scene, so
// it keeps the plain theme.
func ForLevel(base theme.Tokens, themeName string, lv *level.Level, c theme.Capacity) Scene {
	t, fill := theme.Apply(base, lv.Style, c)
	if !fill.Gradient() && fill.Top == base.Sky {
		fill = theme.SkyFill{}
	}
	return Scene{T: t, Theme: themeName, Fill: fill, Stars: lv.Style.Night()}
}

// paintSky fills the canvas with the sky. A gradient runs from the top of the
// level to its bottom, in level space, so it scrolls with the camera instead of
// being stuck to the screen: looking down from a tall stair shows more of the
// horizon colour, as it should.
func (s *Scene) paintSky(c *Canvas, w *game.World, camY, colsPerTile int) {
	if s.Transparent {
		return
	}
	levelH := w.Level.Height * colsPerTile
	if !s.Fill.Gradient() {
		for y := 0; y < c.H; y++ {
			col := s.T.Sky
			if levelH > 0 && camY+y >= levelH {
				col = s.T.GroundDeep
			} else if camY+y < 0 {
				col = s.T.Sky
			}
			for x := 0; x < c.W; x++ {
				c.Px[y*c.W+x] = col
			}
		}
		return
	}
	if levelH <= 1 {
		c.Clear(s.Fill.Bottom)
		return
	}
	var prev interface{}
	var prevRow int = -1
	for y := 0; y < c.H; y++ {
		var col interface{}
		ay := camY + y
		if ay >= levelH {
			col = s.T.GroundDeep
		} else if ay < 0 {
			col = s.Fill.At(0)
		} else {
			col = s.Fill.At(float64(ay) / float64(levelH-1))
		}
		if prevRow >= 0 && col == prev {
			copy(c.Px[y*c.W:(y+1)*c.W], c.Px[prevRow*c.W:(prevRow+1)*c.W])
			continue
		}
		for x := 0; x < c.W; x++ {
			c.Px[y*c.W+x] = col
		}
		prev, prevRow = col, y
	}
}

// skyAt is the sky colour behind an art row of the canvas, for the few places
// that paint "background" on purpose, such as the inset beside a pipe body.
func (d drawer) skyAt(y int, w *game.World) interface{} {
	if !d.s.Fill.Gradient() {
		return d.s.T.Sky
	}
	levelH := w.Level.Height * d.px
	if levelH <= 1 {
		return d.s.Fill.Bottom
	}
	return d.s.Fill.At(float64(d.camY+y) / float64(levelH-1))
}

// Draw renders the world into a canvas sized for the given terminal geometry.
//
// colsPerTile is both the zoom and the art resolution. Terminal cells are about
// twice as tall as they are wide, and Convert stacks two art pixels into one
// cell, so a square-looking tile needs art pixels per tile to equal columns per
// tile. Keeping those two numbers equal is what stops the art from being
// stretched, which is why the sprite set is chosen from the same value.
func (s *Scene) Draw(w *game.World, colsPerTile int) *Canvas {
	if colsPerTile < 4 {
		colsPerTile = 4
	}
	set := SetFor(colsPerTile)

	// The canvas is exactly as many art pixels as the viewport covers, not a
	// whole number of tiles. The viewport is sized to the terminal and is
	// usually a fractional number of tiles (19.5 down on a 40-row terminal).
	// Rounding that down to whole tiles gave a canvas a row or two short, and
	// stretching it back over the terminal duplicated those rows: the bricks
	// they fell in drew taller than the rest.
	// Rounded up: the viewport was itself rounded down when it was converted
	// from terminal pixels (4096 does not divide by 12), and truncating again
	// here would come out one pixel short.
	artW := ceilArt(w.ViewW, colsPerTile)
	artH := ceilArt(w.ViewH, colsPerTile)
	if artW <= 0 {
		artW = 10 * colsPerTile
	}
	if artH <= 0 {
		artH = 7 * colsPerTile
	}
	c := NewCanvas(artW, artH)

	camX := WorldToArtScaled(w.CameraX, colsPerTile)
	camY := WorldToArtScaled(w.CameraY, colsPerTile)
	s.paintSky(c, w, camY, colsPerTile)

	d := drawer{c: c, s: s, px: colsPerTile, set: set,
		camX: camX, camY: camY, artW: artW, artH: artH}

	d.stars(w)
	d.clouds(w)
	d.lavaSurface(w.Level)
	d.terrain(w)
	d.castle(w)
	d.flag(w)
	d.items(w)
	d.enemies(w)
	d.player(w)
	d.effects(w)
	d.particles(w)
	d.coins(w)
	return c
}

// lavaSurface fills the bottom of a lava-themed scene. Terrain is drawn
// over it, so the surface shows through open pits and under bridges.
func (d drawer) lavaSurface(lv *level.Level) {
	if d.s.Transparent || lv.Style.Theme != "lava" {
		return
	}
	top := (lv.Height-1)*d.px - d.camY
	if top >= d.artH || top+d.px <= 0 {
		return
	}
	d.c.FillRect(0, top, d.artW, d.px, d.s.T.Cast.GrassDark)
	d.c.FillRect(0, top, d.artW, max(1, d.px/4), d.s.T.Cast.Grass)
	for x := -d.camX % (2 * d.px); x < d.artW; x += 2 * d.px {
		d.c.FillRect(x, top, max(1, d.px/2), max(1, d.px/4), d.s.T.QuestionGlint)
	}
}

// drawer holds the per-frame state shared by the tile and entity painters.
type drawer struct {
	c    *Canvas
	s    *Scene
	px   int // art pixels per tile
	set  SpriteSet
	camX int
	camY int
	artW int
	artH int
}

// u scales a value expressed in sixteenths of a tile to the current art
// resolution, so every shape can be authored against the detailed grid.
func (d drawer) u(n int) int { return n * d.px / ArtPerTile }

// draw places a sprite at an art position, scaling it from its authored
// resolution to the canvas resolution.
//
// Without this, the twelve-column view would blit sixteen-pixel art into a
// twelve-pixel tile: Mario would draw a third of a tile wider than the box he
// collides with, and walk through gaps he appears to hit.
func (d drawer) draw(sp Sprite, x, y int) {
	w := sp.W * d.px / d.set.Px
	h := sp.H * d.px / d.set.Px
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	d.c.DrawSized(sp, x, y, w, h, d.s.T)
}

// roundArt converts a world-space position to art pixels, rounding to the
// nearest pixel rather than truncating.
func roundArt(v engine.Fixed, artPerTile int) int {
	n := int64(v)*int64(artPerTile) + int64(engine.Tile)/2
	return int(n / int64(engine.Tile))
}

// WorldToArtScaled converts a world-space position to art pixels.
func WorldToArtScaled(v engine.Fixed, artPerTile int) int {
	n := int64(v) * int64(artPerTile) / int64(engine.Tile)
	return int(n)
}

// ceilArt converts a world-space length to art pixels, rounding up.
func ceilArt(v engine.Fixed, artPerTile int) int {
	n := int64(v)*int64(artPerTile) + int64(engine.Tile) - 1
	return int(n / int64(engine.Tile))
}

// terrain paints every visible tile, plus the bump offsets of blocks just hit.
func (d drawer) terrain(w *game.World) {
	x0 := d.camX / d.px
	x1 := (d.camX + d.artW) / d.px
	y0 := d.camY / d.px
	y1 := (d.camY + d.artH) / d.px

	bumps := map[int]int{}
	for _, b := range w.Bumps {
		bumps[b.Row*w.Level.Width+b.Col] = BlockBumpArt(b.Remain, d.px)
	}

	for ty := y0; ty <= y1; ty++ {
		for tx := x0; tx <= x1; tx++ {
			if tx < 0 || ty < 0 || tx >= w.Level.Width || ty >= w.Level.Height {
				continue
			}
			tile := w.Level.TileAt(tx, ty)
			if tile.Kind == level.KindEmpty {
				continue
			}
			dx := tx*d.px - d.camX
			dy := ty*d.px - d.camY - bumps[ty*w.Level.Width+tx]
			d.tile(tx, ty, tile, dx, dy, w)
		}
	}
}

// BlockBumpArt converts a bump's remaining frames into an upward art offset.
func BlockBumpArt(remain, artPerTile int) int {
	if remain <= 0 {
		return 0
	}
	const total = 8
	half := total / 2
	step := artPerTile / 8
	if step < 1 {
		step = 1
	}
	if remain > half {
		return (total - remain) * step
	}
	return remain * step
}

// tile paints one terrain cell at an art-space origin.
func (d drawer) tile(tx, ty int, tile level.Tile, dx, dy int, w *game.World) {
	if d.px == 4 && d.tile4(tx, ty, tile, dx, dy, w) {
		return
	}
	t := d.s.T
	p := d.px
	switch tile.Glyph {
	case 'X':
		// A thin bright crust on a dark body. The crust is what reads as
		// "ground" at a glance; the body below only has to be dark enough to
		// look solid, so it stays thin and leaves the screen to the sky.
		edge := p / 4
		if edge < 1 {
			edge = 1
		}
		d.c.FillRect(dx, dy, p, edge, t.GroundEdge)
		d.c.FillRect(dx, dy+edge, p, p-edge, t.Ground)
	case 'H':
		d.c.FillRect(dx, dy, p, p, t.Ground)
		d.c.FillRect(dx, dy, p, 1, t.GroundEdge)
	case 'B':
		d.c.FillRect(dx, dy, p, p, t.Brick)
		// Mortar: a line across the middle and staggered verticals.
		mid := p / 2
		for x := 0; x < p; x++ {
			d.c.Set(dx+x, dy, t.Outline)
			d.c.Set(dx+x, dy+mid, t.Outline)
		}
		for y := 0; y < mid; y++ {
			d.c.Set(dx, dy+y, t.Outline)
		}
		for y := mid; y < p; y++ {
			d.c.Set(dx+mid, dy+y, t.Outline)
		}
	case '?':
		d.c.FillRect(dx, dy, p, p, t.Outline)
		d.c.FillRect(dx+1, dy+1, p-2, p-2, t.Question)
		// A question mark, sized to the resolution.
		if p >= 12 {
			for y, row := range []string{
				"..###..", ".#...#.", ".....#.",
				"....#..", "...#...", ".......", "...#...",
			} {
				for x := 0; x < len(row); x++ {
					if row[x] == '#' {
						d.c.Set(dx+5+x, dy+4+y, t.Outline)
					}
				}
			}
		} else {
			// Two pixels are all there is room for; keep the block readable
			// by colour and outline rather than trying to draw a glyph.
			d.c.Set(dx+p/2, dy+p/3, t.Outline)
			d.c.Set(dx+p/2, dy+2*p/3, t.Outline)
		}
	case '#':
		d.c.FillRect(dx, dy, p, p, t.Outline)
		d.c.FillRect(dx+1, dy+1, p-2, p-2, t.Used)
	case '[', ']':
		d.c.FillRect(dx, dy, p, p, t.Pipe)
		hi := d.u(3)
		if hi < 1 {
			hi = 1
		}
		if tile.Glyph == '[' {
			d.c.FillRect(dx, dy, hi, p, t.PipeHi)
			d.c.FillRect(dx, dy, 1, p, t.Outline)
		} else {
			d.c.FillRect(dx+p-hi, dy, hi, p, t.PipeLo)
			d.c.FillRect(dx+p-1, dy, 1, p, t.Outline)
		}
		d.c.FillRect(dx, dy, p, 1, t.Outline)
		d.c.FillRect(dx, dy+p-1, p, 1, t.Outline)
	case '{', '}':
		// The body is inset, which is what makes the pipe read as having a rim.
		inset := d.u(2)
		if inset < 1 {
			inset = 1
		}
		if d.s.Transparent {
			// Nothing behind the pipe's inset: leave it clear.
		} else if d.s.Fill.Gradient() {
			for y := 0; y < p; y++ {
				d.c.FillRect(dx, dy+y, p, 1, d.skyAt(dy+y, w))
			}
		} else {
			d.c.FillRect(dx, dy, p, p, t.Sky)
		}
		if tile.Glyph == '{' {
			d.c.FillRect(dx+inset, dy, p-inset, p, t.Pipe)
			d.c.FillRect(dx+inset, dy, d.u(3), p, t.PipeHi)
			d.c.FillRect(dx+inset, dy, 1, p, t.Outline)
		} else {
			d.c.FillRect(dx, dy, p-inset, p, t.Pipe)
			d.c.FillRect(dx+p-inset-d.u(3), dy, d.u(3), p, t.PipeLo)
			d.c.FillRect(dx+p-inset-1, dy, 1, p, t.Outline)
		}
	case 'F':
		d.c.blitArt(cast4.Pole, dx, dy, d.castScale(), false, flagPalette(t))
	case 'T':
		d.c.blitArt(cast4.Ball, dx, dy, d.castScale(), false, flagPalette(t))
	case 'o':
		frame := (int(w.CameraX)/64 + tx) % len(d.set.Coin)
		if frame < 0 {
			frame += len(d.set.Coin)
		}
		d.draw(d.set.Coin[frame], dx, dy)
	}
}

func (d drawer) items(w *game.World) {
	for _, it := range w.Items {
		if it.Removed {
			continue
		}
		d.standOn(cast4.Mushroom, &it.Body, false, mushroomPalette(d.s.T, it.Kind))
	}
}

// enemies draws the Goombas and Koopas. The drawings face left; one walking
// right is mirrored.
func (d drawer) enemies(w *game.World) {
	step := w.StepCount()
	gp, kp := goombaPalette(d.s.T), koopaPalette(d.s.T)
	for _, e := range w.Enemies {
		if e.Removed {
			continue
		}
		dx := WorldToArtScaled(e.Pos.X, d.px) - d.camX
		if dx < -d.px*2 || dx > d.artW {
			continue
		}
		flip := e.Facing > 0
		if e.Knocked {
			// Knocked out: upside down, falling away.
			switch e.Kind {
			case game.EnemyGoomba:
				d.standOnFlipped(cast4.Goomba[0], &e.Body, flip, gp)
			case game.EnemyKoopa:
				d.standOnFlipped(cast4.Shell, &e.Body, false, kp)
			}
			continue
		}
		switch e.Kind {
		case game.EnemyGoomba:
			if e.Squashed > 0 {
				d.standOn(cast4.GoombaFlat, &e.Body, false, gp)
			} else {
				d.standOn(cast4.Goomba[walkFrame(step)], &e.Body, flip, gp)
			}
		case game.EnemyKoopa:
			if e.Shell == game.KoopaWalking {
				d.standOn(cast4.Koopa[walkFrame(step)], &e.Body, flip, kp)
			} else {
				d.standOn(cast4.Shell, &e.Body, false, kp)
			}
		}
	}
}

// flag hangs the cloth off the left of the pole, just under the ball.
func (d drawer) flag(w *game.World) {
	lv := w.Level
	k := d.castScale()
	poleX := lv.FlagX*d.px - d.camX + poleColumn*k
	topY := (lv.FlagTop+1)*d.px - d.camY + WorldToArtScaled(w.FlagDrop, d.px)
	d.c.blitArt(cast4.Cloth, poleX-cast4.Cloth.width()*k, topY, k, false, flagPalette(d.s.T))
}

// heroDesignPx is the resolution the hero was drawn at: four pixels to a tile.
// At any other zoom the drawing is scaled by a whole number where it can be, so
// a pixel of the drawing stays a crisp block instead of being resampled.
const heroDesignPx = 4

// player draws the hero.
//
// He is anchored by his feet and centred on his collision box. The drawing is
// wider than the box (six or seven pixels against four), and the surplus is
// shared equally between the two sides, so the wheel stays under the box and
// the hands reach out past it without shifting where he is.
func (d drawer) player(w *game.World) {
	p := w.Player
	if p.Hidden {
		return
	}
	step := w.StepCount()

	// After a hit: flicker between the big and the small drawing while the
	// world holds still, then a short grace period of faint blinking.
	big := p.Form != game.FormSmall && !p.Dead
	if p.Shrinking > 0 {
		big = (p.Shrinking/4)%2 == 0
	} else if p.Invulnerable() && !p.HasStar() && !p.Dead && (step/2)%3 == 0 {
		return
	}
	if p.GrowFlash > 0 {
		// Growing: flicker small, big, small, big.
		big = (p.GrowFlash/3)%2 == 1
	}

	pose := HeroStand
	if (!p.Grounded || p.Respawning > 0) && !p.Dead {
		pose = HeroJump
	}
	phases := WheelPhases(big)
	phase := 0
	if !p.Dead {
		phase = p.WheelPhase(phases)
	}
	hero := Hero(big, pose, phase)
	facing := p.Facing
	if p.Dead {
		// Dead, he faces the way he was going.
		facing = p.DeathFacing
		hero = Hero(false, HeroJump, 0)
	}
	if facing < 0 {
		hero = hero.Mirrored()
	}

	scaleNum, scaleDen := d.px, heroDesignPx
	sw := max(1, hero.Sprite.W*scaleNum/scaleDen)
	sh := max(1, hero.Sprite.H*scaleNum/scaleDen)
	headMid := (hero.Width * scaleNum / scaleDen) / 2
	centreX := WorldToArtScaled(p.Pos.X+p.W/2, d.px) - d.camX
	bottomY := roundArt(p.Pos.Y+p.H, d.px) - d.camY
	x, y := centreX-headMid, bottomY-sh
	trailHeight := sh
	if !p.Dead && p.Respawning == 0 {
		y = d.fitHeroUnderCeiling(w.Level, p.Pos.Y, x, y, sw)
		sh = max(1, bottomY-y)
	}

	switch {
	case p.Dead:
		d.deadHero(w, hero, x, y, sw, sh)
	case w.RespawnGathering() >= 0:
		d.gatheringHero(w, hero, x, y, sw, sh)
	case p.HasStar():
		// Star power: a trail of fading afterimages behind him, and his colours
		// cycling through the rainbow. It speeds up and fades as it runs out.
		d.starTrail(w, hero, sw, trailHeight, headMid)
		hue := step * 12
		ending := p.StarSec < 120
		if ending {
			hue = step * 6
		}
		tint := func(c tcell.Color) tcell.Color { return rainbowTint(c, hue) }
		if ending && (step/4)%2 == 0 {
			tint = nil
		}
		d.c.DrawTinted(hero.Sprite, x, y, sw, sh, d.s.T, tint, false)
		d.starSparkles(x, y, sw, sh, step)
	default:
		if p.Poisoned > 0 {
			d.c.DrawTinted(hero.Sprite, x, y, sw, sh, d.s.T, sickTint(p.Poisoned), false)
			return
		}
		d.c.DrawSized(hero.Sprite, x, y, sw, sh, d.s.T)
	}
}

// fitHeroUnderCeiling keeps the feet anchored and fits the taller artwork
// below any solid tiles occupying the space above the collision box.
func (d drawer) fitHeroUnderCeiling(lv *level.Level, bodyTop engine.Fixed, x, y, width int) int {
	firstCol := max(0, floorDiv(x+d.camX, d.px))
	lastCol := min(lv.Width-1, floorDiv(x+width-1+d.camX, d.px))
	firstRow := max(0, floorDiv(y+d.camY, d.px))
	rowAboveBody := int(bodyTop/engine.Tile) - 1
	for row := rowAboveBody; row >= firstRow; row-- {
		for col := firstCol; col <= lastCol; col++ {
			if lv.SolidAt(col, row) {
				return max(y, (row+1)*d.px-d.camY)
			}
		}
	}
	return y
}

// starTrail draws a few faint copies of him where he really was a moment ago,
// from the positions the player keeps; standing still they sit under him and
// do not show.
func (d drawer) starTrail(w *game.World, hero HeroDrawing, sw, sh, headMid int) {
	p := w.Player
	for i := 3; i >= 1; i-- {
		past, ok := p.PastPos(i * 3)
		if !ok {
			continue
		}
		cx := WorldToArtScaled(past.X+p.W/2, d.px) - d.camX
		by := roundArt(past.Y+p.H, d.px) - d.camY
		fade := float64(i) * 0.22
		hue := (w.StepCount() - i*4) * 12
		tint := func(c tcell.Color) tcell.Color {
			return blend(rainbowTint(c, hue), d.skyAt(by-sh/2, w).(tcell.Color), 0.35+fade)
		}
		x := cx - headMid
		y := d.fitHeroUnderCeiling(w.Level, past.Y, x, by-sh, sw)
		d.c.DrawTinted(hero.Sprite, x, y, sw, max(1, by-y), d.s.T, tint, false)
	}
}

// starSparkles twinkles a few points of light around him.
func (d drawer) starSparkles(x, y, w, h, step int) {
	for i := 0; i < 3; i++ {
		s := uint32(step/5 + i*7)
		s = s*2654435761 ^ s>>13
		sx := x - 1 + int(s%uint32(w+2))
		sy := y - 1 + int((s>>8)%uint32(h+2))
		c := d.s.T.Cast.Cloud
		if i == 1 {
			c = d.s.T.CoinHi
		}
		d.c.Set(sx, sy, c)
	}
}

// greyOf turns a colour into its grey, a touch darker: the colour of a hero
// who has just lost a life.
func greyOf(c tcell.Color) tcell.Color {
	r, g, b := c.RGB()
	if r < 0 {
		return c
	}
	l := int32(0.30*float64(r)+0.59*float64(g)+0.11*float64(b)) * 4 / 5
	return tcell.NewRGBColor(l, l, l)
}

// rainbowTint shifts a colour round the colour wheel by hue degrees, keeping
// its lightness, so the drawing stays readable while it cycles.
func rainbowTint(c tcell.Color, hue int) tcell.Color {
	r, g, b := c.RGB()
	if r < 0 {
		return c
	}
	h, s, l := rgbToHSL(float64(r)/255, float64(g)/255, float64(b)/255)
	if s < 0.15 {
		// Whites, greys and the outline: add a little of the current hue.
		s = 0.35
	} else if s < 0.7 {
		s = 0.85
	}
	h = math.Mod(h+float64(hue), 360)
	if l < 0.25 {
		l = 0.3
	}
	rr, gg, bb := hslToRGB(h, s, l)
	return tcell.NewRGBColor(int32(rr*255), int32(gg*255), int32(bb*255))
}

func rgbToHSL(r, g, b float64) (h, s, l float64) {
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	l = (mx + mn) / 2
	if mx == mn {
		return 0, 0, l
	}
	d := mx - mn
	if l > 0.5 {
		s = d / (2 - mx - mn)
	} else {
		s = d / (mx + mn)
	}
	switch mx {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	return h * 60, s, l
}

func hslToRGB(h, s, l float64) (r, g, b float64) {
	if s == 0 {
		return l, l, l
	}
	q := l * (1 + s)
	if l >= 0.5 {
		q = l + s - l*s
	}
	p := 2*l - q
	hk := h / 360
	conv := func(t float64) float64 {
		if t < 0 {
			t++
		}
		if t > 1 {
			t--
		}
		switch {
		case t < 1.0/6:
			return p + (q-p)*6*t
		case t < 0.5:
			return q
		case t < 2.0/3:
			return p + (q-p)*(2.0/3-t)*6
		}
		return p
	}
	return conv(hk + 1.0/3), conv(hk), conv(hk - 1.0/3)
}

func (d drawer) particles(w *game.World) {
	sz := d.u(3)
	if sz < 1 {
		sz = 1
	}
	for _, p := range w.Parts {
		if p.Kind != 'b' {
			continue
		}
		dx := WorldToArtScaled(p.Pos.X, d.px) - d.camX
		dy := WorldToArtScaled(p.Pos.Y, d.px) - d.camY
		d.c.FillRect(dx, dy, sz, sz, d.s.T.Brick)
	}
}

// mirror returns a horizontally flipped copy of a sprite.
func mirror(s Sprite) Sprite {
	rows := make([]string, s.H)
	for y := 0; y < s.H; y++ {
		src := []byte(s.Rows[y])
		dst := make([]byte, len(src))
		for i := range src {
			dst[len(src)-1-i] = src[i]
		}
		rows[y] = string(dst)
	}
	return Sprite{W: s.W, H: s.H, Rows: rows}
}

// HUDRow renders the status line. The coin count is the one on screen, which
// catches up as each flying coin lands.
func HUDRow(w *game.World, width int) string {
	// Everything that changes is on the left, in fixed columns with room after
	// each counter for its "+N"; the level's name stands alone on the right.
	//
	//	MARIO 000000   ⓒ×00      ♥×3      TIME 400              WORLD 1-1
	//	0              15        25       34
	s := fmt.Sprintf("MARIO %06d   ⓒ×%02d", w.Score.Points, w.CoinsShown())
	s = padTo(s, LivesCounterCol) + fmt.Sprintf("♥×%d", w.LivesShown())
	s = padTo(s, TimeCol) + fmt.Sprintf("TIME %03d", w.TimeLeft)
	right := "WORLD " + w.Level.World
	gap := width - len([]rune(s)) - len([]rune(right))
	if gap < 2 {
		gap = 2
	}
	return s + strings.Repeat(" ", gap) + right
}

// LivesCounterCol is the screen column of the heart icon in the status line,
// where the 1UP heart flies to and its "+1" appears just after.
const LivesCounterCol = 25

// TimeCol is the screen column the time starts in.
const TimeCol = 34

// padTo pads s with spaces to n runes (or one space, if it is already longer).
func padTo(s string, n int) string {
	l := len([]rune(s))
	if l >= n {
		return s + " "
	}
	return s + strings.Repeat(" ", n-l)
}

// StarCountdown is the star-power countdown for the status line: the text to
// show ("★ 7.3"), and the colour it is in this step, cycling like the hero
// while plenty is left and flashing as it runs out. ok is false when there is
// no star.
func StarCountdown(w *game.World, t theme.Tokens) (text string, col tcell.Color, ok bool) {
	p := w.Player
	if !p.HasStar() {
		return "", 0, false
	}
	secs := float64(p.StarSec) / 60
	text = fmt.Sprintf("★ %.1f", secs)
	step := w.StepCount()
	col = rainbowTint(t.CoinHi, step*12)
	if p.StarSec < 120 && (step/6)%2 == 0 {
		col = t.HudMuted
	}
	return text, col, true
}
