package render

import (
	"fmt"

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

	tilesW := int(w.ViewW / engine.Tile)
	tilesH := int(w.ViewH / engine.Tile)
	if tilesW <= 0 {
		tilesW = 10
	}
	if tilesH <= 0 {
		tilesH = 7
	}
	artW := tilesW * colsPerTile
	artH := tilesH * colsPerTile
	c := NewCanvas(artW, artH)
	c.Clear(s.T.Sky)

	camX := WorldToArtScaled(w.CameraX, colsPerTile)
	camY := WorldToArtScaled(w.CameraY, colsPerTile)

	d := drawer{c: c, s: s, px: colsPerTile, set: set,
		camX: camX, camY: camY, artW: artW, artH: artH}

	d.terrain(w)
	d.items(w)
	d.enemies(w)
	d.player(w)
	d.particles(w)
	return c
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
		d.c.FillRect(dx, dy, p, p, t.Sky)
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
		d.c.FillRect(dx+p/2-1, dy, 2, p, t.HudMuted)
	case 'T':
		d.c.FillRect(dx+p/4, dy+p/8, p/2, p/2, t.HudText)
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
		dx := WorldToArtScaled(it.Pos.X, d.px) - d.camX
		dy := WorldToArtScaled(it.Pos.Y, d.px) - d.camY
		switch it.Kind {
		case game.ItemMushroom:
			d.draw(d.set.Mushroom, dx, dy)
		case game.ItemStar:
			d.draw(d.set.Star, dx, dy)
		}
	}
}

func (d drawer) enemies(w *game.World) {
	for _, e := range w.Enemies {
		dx := WorldToArtScaled(e.Pos.X, d.px) - d.camX
		dy := WorldToArtScaled(e.Pos.Y, d.px) - d.camY
		if dx < -d.px*2 || dx > d.artW {
			continue
		}
		if e.Squashed > 0 {
			if e.Kind == game.EnemyGoomba {
				d.draw(d.set.GoombaFlat, dx, dy)
			} else {
				d.draw(d.set.Shell, dx, dy)
			}
			continue
		}
		switch e.Kind {
		case game.EnemyGoomba:
			d.draw(d.set.Goomba, dx, dy)
		case game.EnemyKoopa:
			if e.Shell == game.KoopaWalking {
				d.draw(d.set.Koopa, dx, dy)
			} else {
				d.draw(d.set.Shell, dx, dy)
			}
		}
	}
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

	// Blinking: star form and the grace period after a hit.
	if p.Form == game.FormStar && (w.StepCount()/3)%2 == 0 {
		return
	}
	if p.GrowFlash > 0 && (p.GrowFlash/2)%2 == 0 {
		return
	}

	big := p.Form != game.FormSmall && !p.Dead
	pose := HeroStand
	if !p.Grounded && !p.Dead {
		pose = HeroJump
	}

	phases := WheelPhases(big)
	phase := 0
	if !p.Dead {
		phase = p.WheelPhase(phases)
	}
	hero := Hero(big, pose, phase)
	if p.Facing < 0 {
		hero = hero.Mirrored()
	}

	// Scale from the design resolution to the current zoom.
	scaleNum, scaleDen := d.px, heroDesignPx
	sw := hero.Sprite.W * scaleNum / scaleDen
	sh := hero.Sprite.H * scaleNum / scaleDen
	if sw < 1 {
		sw = 1
	}
	if sh < 1 {
		sh = 1
	}

	// Horizontal anchor: the middle of the head, not of the whole drawing.
	//
	// The body is not symmetrical: the back hand is tucked in and the front hand
	// sticks out, so the middle of the drawing moves when he turns round. If
	// that were the anchor, he would hop a pixel every time he changed
	// direction. The head is always centred, so it stays put and the arms swing
	// round it.
	headMid := (hero.Width * scaleNum / scaleDen) / 2
	centreX := WorldToArtScaled(p.Pos.X+p.W/2, d.px) - d.camX

	// Vertical anchor: the wheel rests on the ground line, which is the bottom of
	// the collision box rounded to the nearest pixel, not truncated. The box's
	// bottom edge is rarely on a whole pixel (fifteen game units is three and
	// three-quarter pixels at this scale), and truncating floats him a pixel
	// above the floor.
	bottomY := roundArt(p.Pos.Y+p.H, d.px) - d.camY

	d.c.DrawSized(hero.Sprite, centreX-headMid, bottomY-sh, sw, sh, d.s.T)
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

// HUDRow renders the status line.
func HUDRow(w *game.World, width int) string {
	s := fmt.Sprintf("MARIO %06d   ⓒ×%02d", w.Score.Points, w.Score.Coins)
	right := fmt.Sprintf("WORLD %s   TIME %03d   ♥×%d", w.Level.World, w.TimeLeft, w.Score.Lives)
	gap := width - len([]rune(s)) - len([]rune(right))
	if gap < 1 {
		gap = 1
	}
	out := s
	for i := 0; i < gap; i++ {
		out += " "
	}
	return out + right
}
