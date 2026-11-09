// Package engine provides the deterministic simulation primitives: fixed-point
// arithmetic, the fixed-step clock, the input sampler and the run state machine.
//
// Everything in this package is deterministic. No function here reads the wall
// clock, and no value is a float. Design decision 1 and 2 in design.md explain
// why: identical inputs must produce identical results on every machine.
package engine

// Fixed is a signed 24.8 fixed-point number.
//
// One game unit (one original pixel) is Sub units, and one tile is 16 game
// units, so Tile is the number of Fixed steps in one tile. Keeping a 256x
// sub-unit resolution means the smallest acceleration constant in the original
// game (about 0.037 px/frame^2) still has integer precision.
type Fixed int32

// Sub is the number of Fixed steps in one game unit.
const Sub = 256

// Unit is one game unit.
const Unit Fixed = Sub

// TileSize is the number of game units in one tile.
const TileSize = 16

// Tile is the number of Fixed steps in one tile.
const Tile Fixed = TileSize * Sub

// FromFloat converts a game-unit count to Fixed, rounding toward zero.
//
// It exists so tuning constants can be written the way the original game
// documented them (pixels per frame) while still landing on integers. It runs
// only at package initialisation.
func FromFloat(v float64) Fixed {
	return Fixed(v * Sub)
}

// Int returns the whole game units, truncated toward zero.
func (f Fixed) Int() int { return int(f) / Sub }

// Floor returns the whole game units, rounded down.
func (f Fixed) Floor() int {
	return int(f&^(Sub-1)) / Sub
}

// Sub returns the sub-unit remainder in [0, Sub).
func (f Fixed) SubUnit() int {
	return int(f & (Sub - 1))
}

// Mul returns f scaled by a plain integer.
func (f Fixed) Mul(n int) Fixed { return f * Fixed(n) }

// Div returns f divided by a plain integer.
func (f Fixed) Div(n int) Fixed { return f / Fixed(n) }

// Abs returns the magnitude.
func (f Fixed) Abs() Fixed {
	if f < 0 {
		return -f
	}
	return f
}

// Sign returns -1, 0 or 1.
func (f Fixed) Sign() int {
	switch {
	case f < 0:
		return -1
	case f > 0:
		return 1
	}
	return 0
}

// Clamp limits f to [lo, hi].
func (f Fixed) Clamp(lo, hi Fixed) Fixed {
	if f < lo {
		return lo
	}
	if f > hi {
		return hi
	}
	return f
}

// MulRatio returns f * num / den using 64-bit intermediates so the constants
// stay exact. Used by variable-jump scaling and acceleration tables.
func MulRatio(f Fixed, num, den int) Fixed {
	return Fixed(int64(f) * int64(num) / int64(den))
}

// Vec2 is a position or velocity in fixed-point world space.
type Vec2 struct {
	X, Y Fixed
}

// Add returns the componentwise sum.
func (v Vec2) Add(o Vec2) Vec2 { return Vec2{v.X + o.X, v.Y + o.Y} }

// Sub returns the componentwise difference.
func (v Vec2) Sub(o Vec2) Vec2 { return Vec2{v.X - o.X, v.Y - o.Y} }

// Rect is an axis-aligned box in fixed-point world space.
//
// X and Y are the top-left corner. Like the original game, Y grows downward.
type Rect struct {
	X, Y, W, H Fixed
}

// Right returns the exclusive right edge.
func (r Rect) Right() Fixed { return r.X + r.W }

// Bottom returns the exclusive bottom edge.
func (r Rect) Bottom() Fixed { return r.Y + r.H }

// Overlaps reports whether two rects share any area. Touching edges do not
// count, which is what makes "standing exactly on a tile" resolve cleanly.
func (r Rect) Overlaps(o Rect) bool {
	return r.X < o.Right() && o.X < r.Right() && r.Y < o.Bottom() && o.Y < r.Bottom()
}

// Contains reports whether a point lies inside the rect.
func (r Rect) Contains(p Vec2) bool {
	return p.X >= r.X && p.X < r.Right() && p.Y >= r.Y && p.Y < r.Bottom()
}
