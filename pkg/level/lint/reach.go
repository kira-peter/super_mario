package lint

import (
	"super-mario/pkg/game"
	"super-mario/pkg/level"
)

// Reach checks that the flag can be reached from the start, and when it cannot,
// points at the gap or wall where the way forward runs out.
//
// The search works on "standing spots": open cells with something solid
// underneath. From each it follows every jump the envelope allows, to the left
// as well as the right, since a level may need a step back. Enemies, items and
// shells are left out: they move, and they can be stomped for height, so they
// would only make the answer less certain. That, and the mid-air second jump
// being a single extra boost the envelope cannot place exactly, is why these
// are warnings and why their wording says "may".
func Reach(lv *level.Level, t *game.Tuning) []Issue {
	env := EnvelopeFor(t)
	type spot struct{ x, y int }

	// Every standing spot in each column, top to bottom.
	cols := make([][]int, lv.Width)
	for x := 0; x < lv.Width; x++ {
		for y := 0; y < lv.Height-1; y++ {
			if standable(lv, x, y) {
				cols[x] = append(cols[x], y)
			}
		}
	}

	start := spot{lv.SpawnX, lv.SpawnY}
	if !standable(lv, start.x, start.y) {
		// Fall straight down to whatever is below the spawn.
		found := false
		for _, y := range cols[start.x] {
			if y >= start.y {
				start.y, found = y, true
				break
			}
		}
		if !found {
			return nil // spawnChecks already reports this
		}
	}

	seen := map[spot]bool{start: true}
	queue := []spot{start}
	furthest := start
	reach := env.Reach() + 4
	for len(queue) > 0 {
		s := queue[0]
		queue = queue[1:]
		if s.x > furthest.x {
			furthest = s
		}
		for dx := -reach; dx <= reach; dx++ {
			x := s.x + dx
			if x < 0 || x >= lv.Width {
				continue
			}
			for _, y := range cols[x] {
				n := spot{x, y}
				if seen[n] {
					continue
				}
				dh := s.y - y // up is positive
				if !env.CanLand(dx, dh) || blocked(lv, s.x, s.y, x, y, env) {
					continue
				}
				seen[n] = true
				queue = append(queue, n)
			}
		}
	}

	// The flag is reached by touching its pole anywhere, so standing within a
	// jump of it, at a height the pole spans, is enough.
	for s := range seen {
		d := lv.FlagX - s.x
		if d < 0 {
			d = -d
		}
		if d <= 1 || (env.CanLand(d, 0) && s.y >= lv.FlagTop-env.MaxRise) {
			return nil
		}
	}

	return []Issue{explainStop(lv, furthest.x, furthest.y, env)}
}

// blocked reports whether a wall stands between two spots that the jump cannot
// get over. A column in between blocks the jump when it is solid all the way
// from the lower of the two standing heights up past the highest point the
// feet can reach; a column with an opening the body fits through does not.
func blocked(lv *level.Level, x0, y0, x1, y1 int, env Envelope) bool {
	step := 1
	if x1 < x0 {
		step = -1
	}
	// Rows count down from the top, so the peak is the smallest row the
	// feet reach, and the floor of the move is the larger of the two rows.
	peak := y0 - env.MaxRise
	floor := max(y0, y1)
	for x := x0 + step; x != x1; x += step {
		// The body is a tile tall when small: it gets through a column if
		// some row between the peak and the floor is open.
		open := false
		for y := max(peak, 0); y <= floor; y++ {
			if !lv.SolidAt(x, y) {
				open = true
				break
			}
		}
		if !open {
			return true
		}
	}
	return false
}

// explainStop says why progress ends at a spot: a gap ahead that is too wide,
// or a wall ahead that is too high.
func explainStop(lv *level.Level, x, y int, env Envelope) Issue {
	// Walk forward from the last good spot to find what is there.
	for nx := x + 1; nx < lv.Width && nx <= x+env.Reach()+6; nx++ {
		if !lv.SolidAt(nx, y+1) && !lv.SolidAt(nx, y) {
			// Measure the gap: open columns until something to stand on.
			end := nx
			for end < lv.Width && len(standingIn(lv, end)) == 0 {
				end++
			}
			return withArgs(warn(nx, y+1, "gap-too-wide",
				"这个坑可能跳不过去：宽 %d 格，按当前手感奔跑起跳最远约 %d 格；从出生点出发可能到不了旗杆",
				end-nx, env.Reach()), "w", end-nx, "reach", env.Reach())
		}
		if lv.SolidAt(nx, y) {
			h := 0
			for yy := y; yy >= 0 && lv.SolidAt(nx, yy); yy-- {
				h++
			}
			return withArgs(warn(nx, y, "wall-too-high",
				"这里的障碍可能跳不上去：高 %d 格，满跳最高约 %d 格；从出生点出发可能到不了旗杆",
				h, env.MaxRise), "h", h, "rise", env.MaxRise)
		}
	}
	return withArgs(warn(lv.FlagX, lv.FlagBot, "flag-unreachable",
		"从出生点出发可能到不了旗杆（最远走到第 %d 列）", x), "x", x)
}

func standingIn(lv *level.Level, x int) []int {
	var ys []int
	for y := 0; y < lv.Height-1; y++ {
		if standable(lv, x, y) {
			ys = append(ys, y)
		}
	}
	return ys
}
