package game

import (
	"super-mario/pkg/engine"
	"super-mario/pkg/level"
)

// Collision is the result of resolving one entity against the tile grid.
//
// Design decision 3: the two axes are solved separately and reported
// separately, so the caller can tell "landed on a floor" from "hit a wall"
// without inspecting velocities after the fact.
type Collision struct {
	// HitLeft and HitRight report a wall contact on that side.
	HitLeft  bool
	HitRight bool
	// Grounded reports a floor contact.
	Grounded bool
	// HitCeiling reports a ceiling contact.
	HitCeiling bool
	// CeilingX, CeilingY is the cell that was hit from below, valid when
	// HitCeiling is true. It is what a bump acts on.
	CeilingX, CeilingY int
}

// moveBody advances a body by its velocity and resolves each axis in turn.
//
// The sweep uses the rectangle spanned by the motion, not just the destination,
// so a body moving faster than one tile per step still cannot tunnel
// (spec world-collision / 高速移动不穿透).
func moveBody(lv *level.Level, b *Body, ignoreOneWay bool) Collision {
	var c Collision

	b.Pos.X += b.Vel.X
	c.HitLeft, c.HitRight = resolveAxisX(lv, b)
	if c.HitLeft || c.HitRight {
		b.Vel.X = 0
	}

	wasFalling := b.Vel.Y >= 0
	b.Pos.Y += b.Vel.Y
	grounded, ceiling := resolveAxisY(lv, b, ignoreOneWay)
	c.Grounded = grounded
	c.HitCeiling = ceiling != nil
	if ceiling != nil {
		c.CeilingX, c.CeilingY = ceiling.col, ceiling.row
	}
	if (grounded && wasFalling) || ceiling != nil {
		b.Vel.Y = 0
	}
	return c
}

type cell struct{ col, row int }

// tileIndex converts a world-space coordinate to a tile row or column.
//
// On-disk the world is measured in game units, where sixteen make one tile.
// Flooring a Fixed gives game units, not tiles, so every grid lookup has to
// divide by Tile — getting this wrong makes the world read as empty air.
func tileIndex(f engine.Fixed) int {
	v := int(f) / int(engine.Tile)
	if f < 0 && int(f)%int(engine.Tile) != 0 {
		v--
	}
	return v
}

// tileSpan returns the inclusive tile range covered by [lo, hi).
func tileSpan(lo, hi engine.Fixed) (first, last int) {
	first = tileIndex(lo)
	last = tileIndex(hi - 1)
	if last < first {
		last = first
	}
	return first, last
}

// resolveAxisX pushes the body out of any solid cell it now overlaps.
//
// The body's leading edge decides the push direction, which is why a body whose
// velocity was positive is pushed to the left of the tile it entered.
func resolveAxisX(lv *level.Level, b *Body) (hitLeft, hitRight bool) {
	r := b.Rect()
	y0, y1 := tileSpan(r.Y, r.Bottom())

	rightCol := tileIndex(r.Right() - 1)
	for row := y0; row <= y1; row++ {
		if lv.SolidAt(rightCol, row) {
			if b.Vel.X > 0 {
				b.Pos.X = engine.Fixed(rightCol)*engine.Tile - b.W
				hitRight = true
				break
			}
		}
	}
	if hitRight {
		return false, true
	}

	leftCol := tileIndex(r.X)
	for row := y0; row <= y1; row++ {
		if lv.SolidAt(leftCol, row) {
			if b.Vel.X < 0 {
				b.Pos.X = engine.Fixed(leftCol+1) * engine.Tile
				hitLeft = true
				break
			}
		}
	}
	return hitLeft, false
}

// resolveAxisY pushes the body out of solids vertically and reports the floor
// or ceiling it met.
//
// One-way platforms only block a body that is descending onto them: the check
// requires the body's previous bottom to sit at or above the platform's top
// edge, which is what lets the player jump up through the platform from below
// (spec world-collision / 单向平台).
func resolveAxisY(lv *level.Level, b *Body, ignoreOneWay bool) (grounded bool, ceiling *cell) {
	r := b.Rect()
	x0, x1 := tileSpan(r.X, r.Right())

	bottomRow := tileIndex(r.Bottom() - 1)
	prevBottom := r.Bottom() - b.Vel.Y
	for col := x0; col <= x1; col++ {
		t := lv.TileAt(col, bottomRow)
		if !t.Kind.Solid() {
			continue
		}
		if ignoreOneWay && t.Kind == level.KindOneWay {
			continue
		}
		if t.Kind == level.KindOneWay && prevBottom > engine.Fixed(bottomRow)*engine.Tile {
			// The body came from inside or below the platform; pass through.
			continue
		}
		if b.Vel.Y >= 0 {
			b.Pos.Y = engine.Fixed(bottomRow)*engine.Tile - b.H
			return true, nil
		}
	}

	topRow := tileIndex(r.Y)
	for col := x0; col <= x1; col++ {
		t := lv.TileAt(col, topRow)
		if t.Kind.Solid() && !(ignoreOneWay && t.Kind == level.KindOneWay) {
			if b.Vel.Y < 0 {
				b.Pos.Y = engine.Fixed(topRow+1) * engine.Tile
				return false, &cell{col, topRow}
			}
		}
	}
	return false, nil
}

// crushedByTerrain reports whether a body's box is inside solid geometry and
// cannot be pushed out, which is the crush case (spec world-collision /
// 被地形挤压).
//
// It is checked after the move, so a body that merely touched a wall during the
// resolve is not misreported: only genuine overlap that survived both axes
// counts.
func crushedByTerrain(lv *level.Level, b *Body) bool {
	return overlapsSolid(lv, b.Rect())
}

// groundedBelow reports whether a solid cell sits directly under the body's
// feet without moving it. Used for coyote-free jump checks.
func groundedBelow(lv *level.Level, b *Body) bool {
	r := b.Rect()
	row := tileIndex(r.Bottom()-1) + 1
	x0, x1 := tileSpan(r.X, r.Right())
	for col := x0; col <= x1; col++ {
		if lv.SolidAt(col, row) {
			return true
		}
	}
	return false
}
