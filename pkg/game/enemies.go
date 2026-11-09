package game

import (
	"super-mario/pkg/engine"
	"super-mario/pkg/level"
)

// EnemyKind distinguishes the enemy types.
type EnemyKind int

const (
	// EnemyGoomba walks and turns at walls.
	EnemyGoomba EnemyKind = iota
	// EnemyKoopa walks, and can be stomped into a shell.
	EnemyKoopa
)

// KoopaState is the two-state behaviour of a Koopa.
type KoopaState int

const (
	// KoopaWalking is the default.
	KoopaWalking KoopaState = iota
	// KoopaShell is a stationary shell.
	KoopaShell
	// KoopaSliding is a kicked shell.
	KoopaSliding
)

// Enemy is a Goomba or a Koopa.
type Enemy struct {
	Body
	Kind EnemyKind

	// Shell is only meaningful for Koopas.
	Shell KoopaState

	// Squashed counts down the flattened frames after a stomp.
	Squashed int
	// Removed marks an enemy that should be dropped from the world.
	Removed bool
	// Sliding direction is carried by Vel.X for shells.
}

// NewGoomba returns a walking Goomba at a cell.
func NewGoomba(col, row int) *Enemy {
	e := &Enemy{Kind: EnemyGoomba}
	e.W = engine.FromFloat(14)
	e.H = engine.FromFloat(14)
	e.Facing = -1
	e.Vel.X = -Default.EnemyWalkSpeed
	e.Active = false
	e.placeOnCell(col, row)
	return e
}

// NewKoopa returns a walking Koopa at a cell.
func NewKoopa(col, row int) *Enemy {
	e := &Enemy{Kind: EnemyKoopa, Shell: KoopaWalking}
	e.W = engine.FromFloat(13)
	e.H = engine.FromFloat(22)
	e.Facing = -1
	e.Vel.X = -Default.EnemyWalkSpeed
	e.Active = false
	e.placeOnCell(col, row)
	return e
}

// placeOnCell puts the body's feet on the bottom of the given cell row.
func (e *Enemy) placeOnCell(col, row int) {
	e.Pos.X = engine.Fixed(col)*engine.Tile + (engine.Tile-e.W)/2
	e.Pos.Y = engine.Fixed(row+1)*engine.Tile - e.H
}

// update advances an enemy one step.
func (e *Enemy) update(lv *level.Level, t *Tuning) {
	if e.Squashed > 0 {
		e.Squashed--
		if e.Squashed == 0 {
			e.Removed = true
		}
		return
	}
	if e.Removed {
		return
	}

	// Gravity always applies, so walking off a ledge falls naturally.
	e.Vel.Y += t.GravityFall
	if e.Vel.Y > t.MaxFallSpeed {
		e.Vel.Y = t.MaxFallSpeed
	}

	c := moveBody(lv, &e.Body, false)
	if c.HitLeft || c.HitRight {
		// Bounce off the wall and keep the same speed magnitude.
		speed := e.walkSpeed(t)
		if c.HitLeft {
			e.Vel.X = speed
			e.Facing = 1
		} else {
			e.Vel.X = -speed
			e.Facing = -1
		}
	}
	e.Grounded = c.Grounded

	// A sliding shell that fell below the level is gone.
	if e.Pos.Y > engine.Fixed(lv.Height)*engine.Tile {
		e.Removed = true
	}
}

// walkSpeed returns the horizontal speed this enemy should travel at.
func (e *Enemy) walkSpeed(t *Tuning) engine.Fixed {
	if e.Shell == KoopaSliding {
		return t.ShellSpeed
	}
	return t.EnemyWalkSpeed
}

// stomp handles a player landing on this enemy from above. It reports the score
// awarded, and whether the player should bounce.
func (e *Enemy) stomp(t *Tuning) (score int, bounce bool) {
	switch e.Kind {
	case EnemyGoomba:
		e.Squashed = 30
		e.Vel = engine.Vec2{}
		return 100, true
	case EnemyKoopa:
		switch e.Shell {
		case KoopaWalking:
			e.toShell()
			return 100, true
		case KoopaShell:
			// Standing on a shell kicks it away from the player.
			return 400, true
		case KoopaSliding:
			e.Vel.X = 0
			e.Shell = KoopaShell
			return 400, true
		}
	}
	return 0, false
}

// toShell shrinks a walking Koopa into a stationary shell.
func (e *Enemy) toShell() {
	e.Shell = KoopaShell
	e.Vel.X = 0
	oldBottom := e.Bottom()
	e.H = engine.FromFloat(14)
	e.Pos.Y = oldBottom - e.H
}

// kick sends a stationary shell sliding, away from the given point.
func (e *Enemy) kick(fromX engine.Fixed, t *Tuning) {
	if e.Kind != EnemyKoopa || e.Shell != KoopaShell {
		return
	}
	dir := 1
	if fromX > e.CenterX() {
		dir = -1
	}
	e.Shell = KoopaSliding
	e.Vel.X = engine.Fixed(dir) * t.ShellSpeed
	e.Facing = dir
}

// stompScore returns the escalating combo score for the n-th consecutive stomp.
func stompScore(n int) int {
	table := []int{100, 200, 400, 800, 1000, 2000, 4000, 8000}
	if n <= 0 {
		return table[0]
	}
	if n > len(table) {
		n = len(table)
	}
	return table[n-1]
}
