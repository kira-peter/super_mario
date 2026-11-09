package game

import "super-mario/pkg/engine"

// Body is the shared physical state of anything that walks, falls or is pushed
// out of terrain.
//
// Design decision 4: entities are concrete structs that embed a Body rather
// than ECS components. There are fewer than ten entity kinds and none of them
// needs arbitrary property composition, so the interface cost of an ECS buys
// nothing here.
type Body struct {
	Pos engine.Vec2
	Vel engine.Vec2
	W   engine.Fixed
	H   engine.Fixed

	// Facing is -1 for left, +1 for right. It only flips once horizontal
	// velocity has actually changed sign, which is what makes a skid read as
	// a skid rather than a flicker (spec mario-physics / 转向).
	Facing int

	// Grounded is set by the last collision resolve.
	Grounded bool

	// Active is false while the body is asleep off-screen.
	Active bool
}

// Rect returns the body's world-space bounds.
func (b *Body) Rect() engine.Rect {
	return engine.Rect{X: b.Pos.X, Y: b.Pos.Y, W: b.W, H: b.H}
}

// Left returns the left edge.
func (b *Body) Left() engine.Fixed { return b.Pos.X }

// Right returns the exclusive right edge.
func (b *Body) Right() engine.Fixed { return b.Pos.X + b.W }

// Top returns the top edge.
func (b *Body) Top() engine.Fixed { return b.Pos.Y }

// Bottom returns the exclusive bottom edge.
func (b *Body) Bottom() engine.Fixed { return b.Pos.Y + b.H }

// CenterX returns the horizontal centre.
func (b *Body) CenterX() engine.Fixed { return b.Pos.X + b.W/2 }

// Overlaps reports whether two bodies share area.
func (b *Body) Overlaps(o *Body) bool { return b.Rect().Overlaps(o.Rect()) }
