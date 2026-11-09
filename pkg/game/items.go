package game

import (
	"super-mario/pkg/engine"
	"super-mario/pkg/level"
)

// ItemKind is a collectible that moves on its own.
type ItemKind int

const (
	// ItemMushroom makes Mario big.
	ItemMushroom ItemKind = iota
	// ItemStar makes Mario invincible.
	ItemStar
)

// Item is a mushroom or a star that has emerged from a block.
type Item struct {
	Body
	Kind ItemKind
	// Emerging counts the rise out of the block; while > 0 the item cannot
	// be collected (spec blocks-and-items / 蘑菇从方块升起).
	Emerging int
	Removed  bool

	// blockRow is where the item should finish rising.
	blockRow int
}

// emergeSteps is how long an item takes to rise out of its block.
const emergeSteps = 16

// NewItem returns an item sitting inside a block at a cell.
func NewItem(kind ItemKind, col, row int) *Item {
	it := &Item{Kind: kind, Emerging: emergeSteps}
	it.W = engine.FromFloat(14)
	it.H = engine.FromFloat(14)
	it.Active = true
	it.Pos.X = engine.Fixed(col)*engine.Tile + (engine.Tile-it.W)/2
	it.Pos.Y = engine.Fixed(row) * engine.Tile
	return it
}

// update advances an item.
func (it *Item) update(lv *level.Level, t *Tuning) {
	if it.Removed {
		return
	}

	if it.Emerging > 0 {
		it.Emerging--
		it.Pos.Y -= engine.Tile.Div(emergeSteps)
		if it.Emerging == 0 {
			// Settle on top of the block it came from.
			it.Pos.Y = engine.Fixed(it.blockRow)*engine.Tile - it.H
			if it.Kind == ItemMushroom {
				it.Vel.X = t.ItemWalkSpeed
				it.Facing = 1
			} else {
				it.Vel.X = t.ItemWalkSpeed
				it.Vel.Y = engine.FromFloat(-3)
				it.Facing = 1
			}
		}
		return
	}

	if it.Kind == ItemStar {
		it.Vel.Y += t.ItemGravity
		if it.Vel.Y > t.MaxFallSpeed {
			it.Vel.Y = t.MaxFallSpeed
		}
	} else {
		it.Vel.Y += t.GravityFall
		if it.Vel.Y > t.MaxFallSpeed {
			it.Vel.Y = t.MaxFallSpeed
		}
	}

	c := moveBody(lv, &it.Body, false)
	it.Grounded = c.Grounded
	if c.HitLeft || c.HitRight {
		it.Vel.X = -it.Vel.X
		it.Facing = -it.Facing
	}
	if it.Kind == ItemStar && c.HitCeiling {
		it.Vel.Y = engine.FromFloat(2)
	}
	if it.Kind == ItemStar && c.Grounded {
		it.Vel.Y = engine.FromFloat(-3)
	}
	if it.Pos.Y > engine.Fixed(lv.Height)*engine.Tile {
		it.Removed = true
	}
}

// blockRow is set by the world when the item is created, so the rise animation
// knows where to land.
func (it *Item) setOrigin(col, row int) {
	it.blockRow = row
	it.Pos.X = engine.Fixed(col)*engine.Tile + (engine.Tile-it.W)/2
	it.Pos.Y = engine.Fixed(row) * engine.Tile
}

// Particle is a decorative fragment with no collision.
type Particle struct {
	Pos  engine.Vec2
	Vel  engine.Vec2
	Life int
	Kind byte // 'b' brick shard, 's' score text
	Text string
}

// BumpAnim is the short visual hop of a block that was hit.
//
// It is purely visual: the collision cell never moves (spec blocks-and-items /
// 顶击动画期间仍可碰撞).
type BumpAnim struct {
	Col, Row int
	Remain   int
}

// bumpFrames is the length of a block bump animation.
const bumpFrames = 8

// BlockBumpOffset returns the vertical art-pixel offset for a bump, 0 when done.
func BlockBumpOffset(remain int) int {
	if remain <= 0 || remain > bumpFrames {
		return 0
	}
	// Up for the first half, back down for the second.
	half := bumpFrames / 2
	if remain > half {
		return (bumpFrames - remain) * 1
	}
	return remain * 1
}
