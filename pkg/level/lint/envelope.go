package lint

import (
	"sync"

	"super-mario/pkg/engine"
	"super-mario/pkg/game"
)

// Envelope says which landing spots a jump can reach, relative to where it took
// off, in whole tiles. It is worked out by playing real jumps with the game's
// own physics (design decision 5): change the tuning and the envelope changes
// with it.
type Envelope struct {
	// Land[dx] is the highest step up, in tiles, that a jump can land on dx
	// columns ahead. Lower steps at the same distance are reachable too: the
	// player can always ease off and drop further. A column absent from Land
	// cannot be reached at all.
	Land map[int]int
	// MaxRise is the highest a jump climbs, in tiles.
	MaxRise int
}

// CanLand reports whether a jump can land dx columns ahead (either way) on
// ground dh tiles higher than the take-off (negative for lower).
func (e Envelope) CanLand(dx, dh int) bool {
	if dx < 0 {
		dx = -dx
	}
	best, ok := e.Land[dx]
	return ok && dh <= best
}

// Reach returns the furthest column a jump can land on at the take-off height.
func (e Envelope) Reach() int {
	far := 0
	for dx, h := range e.Land {
		if h >= 0 && dx > far {
			far = dx
		}
	}
	return far
}

var (
	envMu    sync.Mutex
	envCache = map[game.Tuning]Envelope{}
)

// EnvelopeFor returns the jump envelope for a tuning, cached per tuning.
func EnvelopeFor(t *game.Tuning) Envelope {
	envMu.Lock()
	defer envMu.Unlock()
	if e, ok := envCache[*t]; ok {
		return e
	}
	e := buildEnvelope(t)
	envCache[*t] = e
	return e
}

// buildEnvelope tries jumps from a standstill, from a walk and from a full run,
// each with a range of button hold times, and records for every column ahead the
// highest point at which some jump is coming down through it: a player there,
// falling, can land on anything that high or lower.
func buildEnvelope(t *game.Tuning) Envelope {
	e := Envelope{Land: map[int]int{}}
	tile := engine.Tile
	record := func(dx, h int) {
		if old, ok := e.Land[dx]; !ok || h > old {
			e.Land[dx] = h
		}
	}
	// Standing still or walking off an edge reaches the next column down at
	// any depth; that is just falling.
	record(0, 0)
	record(1, 0)

	for _, run := range []bool{false, true} {
		for _, runup := range []int{0, 6, 20, 60} {
			for _, hold := range []int{1, 4, 8, 14, 20, 30, 60} {
				path := game.JumpTrajectory(t, run, runup, hold)
				for i, s := range path {
					rise := int(s.DY / tile)
					if rise > e.MaxRise {
						e.MaxRise = rise
					}
					// Only the descent can land.
					if i == 0 || s.DY >= path[i-1].DY {
						continue
					}
					// The column under the body's centre. The body is a
					// little narrower than a tile and starts centred in
					// its own, so the centre's offset in tiles is the
					// displacement rounded to the nearest tile.
					dx := int((s.DX + tile/2) / tile)
					// A landing is on top of a block whose top is at a
					// whole tile; the body is still above a block dh
					// high while its feet are at least that high.
					h := int(s.DY / tile)
					if s.DY < 0 {
						h = -int((-s.DY + tile - 1) / tile)
					}
					record(dx, h)
				}
			}
		}
	}
	// Anything reachable at a height is reachable lower down at the same
	// column (ease off the button, or steer back). Make the table say so for
	// columns the coarse sampling skipped between two reachable ones.
	far := 0
	for dx := range e.Land {
		if dx > far {
			far = dx
		}
	}
	for dx := far - 1; dx >= 0; dx-- {
		if _, ok := e.Land[dx]; !ok {
			e.Land[dx] = e.Land[dx+1]
		}
	}
	return e
}
