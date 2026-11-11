package game

import (
	"super-mario/pkg/engine"
	"super-mario/pkg/level"
)

// ItemKind is a collectible that moves on its own.
//
// Every pickup is a mushroom of the same shape; the colour says what it does.
type ItemKind int

const (
	// ItemMushroom is the red mushroom: it makes Mario big.
	ItemMushroom ItemKind = iota
	// ItemStar is the blue mushroom: it makes Mario invincible for a while.
	// It took the star's place, and kept its name so level files still say
	// "star".
	ItemStar
	// ItemOneUp is the green mushroom: one extra life.
	ItemOneUp
	// ItemPoison is the purple mushroom: it hurts, like touching an enemy.
	ItemPoison
	// ItemGold is the gold mushroom: five coins at once.
	ItemGold
)

// Item is a mushroom that has emerged from a block.
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
			// Settle on top of the block it came from and walk off.
			it.Pos.Y = engine.Fixed(it.blockRow)*engine.Tile - it.H
			it.Vel.X = t.ItemWalkSpeed
			it.Facing = 1
		}
		return
	}

	it.Vel.Y += t.GravityFall
	if it.Vel.Y > t.MaxFallSpeed {
		it.Vel.Y = t.MaxFallSpeed
	}

	c := moveBody(lv, &it.Body, false)
	it.Grounded = c.Grounded
	// Turn round at a wall or a pipe. The collision resolve has already zeroed
	// the speed, so negating it would leave the mushroom pressed against the
	// wall for good; the direction comes from which side it hit instead.
	if c.HitLeft {
		it.Vel.X = t.ItemWalkSpeed
		it.Facing = 1
	} else if c.HitRight {
		it.Vel.X = -t.ItemWalkSpeed
		it.Facing = -1
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
