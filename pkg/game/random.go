package game

import "super-mario/pkg/level"

// Random block contents (spec random-block-content, design decision 9).
//
// A random block draws its content once at the start of every life and keeps it
// for that life. The draw depends only on the run's seed, which life this is,
// and where the block is. There is no shared random stream that blocks take
// turns pulling from, so adding a block elsewhere, or hitting blocks in a
// different order, cannot change what any other block holds.

// SetSeed chooses the run's seed and redraws the random blocks with it. Call it
// before play starts: right after NewWorld, or before Reset for a new run.
func (w *World) SetSeed(seed int64) {
	w.Seed = seed
	w.drawRandomBlocks()
}

// drawRandomBlocks gives every random block its content for this life.
func (w *World) drawRandomBlocks() {
	for i := range w.Level.Contents {
		c := &w.Level.Contents[i]
		if !c.Random {
			continue
		}
		c.Kind = DrawRandom(w.Seed, w.life, c.X, c.Y, c.Pool)
	}
}

// DrawRandom is the content a random block at (x, y) holds on the given life
// of a run with the given seed. It is exported so the editor's "show the
// draws" preview and the tests use exactly the same rule as the game.
func DrawRandom(seed int64, life, x, y int, pool []level.Choice) level.BlockKind {
	total := 0
	for _, c := range pool {
		total += c.Weight
	}
	if total <= 0 {
		// The loader rejects such a pool; this is only a safe fallback.
		return level.BlockCoin
	}
	h := splitmix64(uint64(seed) ^ uint64(life)*0x9e3779b97f4a7c15 ^ uint64(x)<<20 ^ uint64(y))
	r := int(h % uint64(total))
	for _, c := range pool {
		if r < c.Weight {
			return c.Kind
		}
		r -= c.Weight
	}
	return pool[len(pool)-1].Kind
}

// splitmix64 scrambles a 64-bit value so that inputs differing in one bit give
// unrelated outputs. Seed, life and position are mixed into one number first;
// this step is what keeps neighbouring blocks from drawing alike.
func splitmix64(x uint64) uint64 {
	x += 0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	return x ^ (x >> 31)
}
