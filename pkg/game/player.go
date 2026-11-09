package game

import (
	"super-mario/pkg/engine"
	"super-mario/pkg/level"
)

// Form is Mario's size / power state.
type Form int

const (
	// FormSmall is the default.
	FormSmall Form = iota
	// FormBig breaks bricks and survives one hit.
	FormBig
	// FormStar is invincible for a limited time.
	FormStar
)

// SmallSize and BigSize are collision boxes in game units.
//
// The art is 16x16 small and 16x32 big, but the collision box is deliberately
// narrower than the sprite so a player can slip through a one-tile gap that
// looks passable. That is the original behaviour and it matters more than
// sprite-accurate boxes.
var (
	SmallSize = engine.Vec2{X: engine.FromFloat(12), Y: engine.FromFloat(15)}
	BigSize   = engine.Vec2{X: engine.FromFloat(12), Y: engine.FromFloat(30)}
)

// Player is Mario.
type Player struct {
	Body
	T       *Tuning
	Form    Form
	StarSec int // remaining star time in simulation steps

	// Jumping is true between leaving the ground and landing.
	Jumping bool
	// JumpHeld tracks the jump button for variable-height jumps.
	JumpHeld bool
	// invuln is the post-hit grace period, in steps.
	invuln int
	// Dead marks a player in the death animation.
	Dead bool
	// deathTimer counts the death animation.
	deathTimer int

	// GrowFlash counts the frames of the grow/shrink flash.
	GrowFlash int

	// SlideDown is set during the flagpole sequence.
	SlideDown bool

	// LastHit is the collision result of the most recent step, so the world
	// can react to a ceiling hit without re-deriving it.
	LastHit Collision

	// StompChainOpen stays true while the player is airborne after a stomp,
	// so consecutive air stomps escalate the score.
	StompChainOpen bool

	// AirJumpUsed marks that the optional mid-air jump has been spent; it
	// resets on landing so each stint off the ground allows at most one.
	AirJumpUsed bool

	// wheelDist is how far the wheel under the hero has rolled, in fixed-point
	// game units, always counted forward whichever way he is facing. The
	// renderer turns it into a frame; keeping the number here, advanced by the
	// simulation, means the wheel follows the ground actually passing under it
	// and does not depend on how often the screen is redrawn.
	wheelDist engine.Fixed
}

// playerInvulnSteps is how long the player is immune after shrinking.
const playerInvulnSteps = 120

// NewPlayer returns a small Mario standing at a tile cell.
func NewPlayer(t *Tuning, col, row int) *Player {
	p := &Player{T: t, Form: FormSmall}
	p.W, p.H = SmallSize.X, SmallSize.Y
	p.Facing = 1
	p.Active = true
	p.placeAtCell(col, row)
	return p
}

// placeAtCell puts the player's feet on top of the given cell's row, centred.
func (p *Player) placeAtCell(col, row int) {
	p.Pos.X = engine.Fixed(col)*engine.Tile + (engine.Tile-p.W)/2
	p.Pos.Y = engine.Fixed(row+1)*engine.Tile - p.H
	p.Vel = engine.Vec2{}
	p.Grounded = true
	p.Jumping = false
}

// applyForm resizes the collision box, keeping the feet planted.
//
// Design decision: growth prefers to expand upward and falls back to expanding
// downward, so a player under a low ceiling is never pushed into a tile
// (spec mario-physics / 头顶空间不足时变大).
func (p *Player) applyForm(lv *level.Level, f Form) {
	oldBottom := p.Bottom()
	switch f {
	case FormSmall:
		p.W, p.H = SmallSize.X, SmallSize.Y
	case FormBig, FormStar:
		p.W, p.H = BigSize.X, BigSize.Y
	}
	p.Pos.Y = oldBottom - p.H

	// If growing put the head inside a tile, try shifting down instead.
	if f != FormSmall && overlapsSolid(lv, p.Rect()) {
		p.Pos.Y = oldBottom - p.H
		if overlapsSolid(lv, p.Rect()) {
			// Still stuck: fall back to the previous size so the player is
			// never trapped inside geometry.
			if f == FormStar {
				p.Form = FormStar
				return
			}
		}
	}
	p.Form = f
}

// overlapsSolid reports whether a rect intersects any solid cell.
//
// The cell range comes from tileSpan, not from Floor(): Floor gives game units,
// and a tile is sixteen of them.
func overlapsSolid(lv *level.Level, r engine.Rect) bool {
	x0, x1 := tileSpan(r.X, r.Right())
	y0, y1 := tileSpan(r.Y, r.Bottom())
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if lv.SolidAt(x, y) {
				return true
			}
		}
	}
	return false
}

// Grow switches to big form, or consumes the mushroom for score if already big.
func (p *Player) Grow(lv *level.Level) bool {
	if p.Form == FormSmall {
		p.applyForm(lv, FormBig)
		p.GrowFlash = 16
		return true
	}
	return false
}

// Star makes the player invincible for the given number of steps.
func (p *Player) Star(steps int) {
	p.Form = FormStar
	p.StarSec = steps
}

// Hurt applies damage. A big player shrinks and becomes briefly immune; a small
// player dies.
func (p *Player) Hurt(lv *level.Level) (died bool) {
	if p.Form == FormStar || p.invuln > 0 {
		return false
	}
	if p.Form == FormBig {
		p.applyForm(lv, FormSmall)
		p.invuln = playerInvulnSteps
		p.GrowFlash = 16
		return false
	}
	p.Die()
	return true
}

// Die starts the death animation.
func (p *Player) Die() {
	p.Dead = true
	p.deathTimer = 0
	p.Vel = engine.Vec2{X: 0, Y: engine.FromFloat(-4)}
	p.Form = FormSmall
	p.W, p.H = SmallSize.X, SmallSize.Y
}

// Invulnerable reports whether contact should be ignored.
func (p *Player) Invulnerable() bool { return p.invuln > 0 || p.Form == FormStar }

// update advances the player one simulation step.
//
// Order matters: horizontal intent, then the jump edge, then gravity, then the
// collision resolve. Doing the jump edge before gravity is what makes the first
// frame of a jump exactly the tuned initial velocity.
func (p *Player) update(lv *level.Level, in engine.Pressed) {
	if p.Dead {
		p.updateDeath()
		return
	}
	if p.SlideDown {
		p.updateSlide(lv)
		return
	}

	if p.GrowFlash > 0 {
		p.GrowFlash--
	}
	if p.invuln > 0 {
		p.invuln--
	}
	if p.StarSec > 0 {
		p.StarSec--
		if p.StarSec == 0 && p.Form == FormStar {
			p.Form = FormBig
			p.applyForm(lv, FormBig)
		}
	}

	p.horizontal(lv, in)
	p.vertical(lv, in)

	before := p.Pos.X
	c := moveBody(lv, &p.Body, false)
	p.LastHit = c
	p.Grounded = c.Grounded
	if c.Grounded && p.Vel.Y >= 0 {
		p.Jumping = false
	}

	// The wheel turns with the ground under it. It is measured from the
	// distance actually moved, so pushing against a wall does not spin it, and
	// it holds still in the air, where nothing is touching it.
	if p.Grounded {
		moved := p.Pos.X - before
		p.wheelDist += moved.Abs()
	}
}

// unitsPerSpritePixel is how many game units one pixel of the hero's drawing
// stands for. The hero is drawn at four pixels to a tile and a tile is sixteen
// game units, so a pixel is four of them. This is a property of the drawing,
// not of the zoom level: a larger zoom enlarges the picture, it does not make
// the wheel a different size.
const unitsPerSpritePixel = 4

// WheelPhase returns which frame of the wheel to show, in [0, phases).
//
// One full turn covers the wheel's circumference. The small wheel is two
// pixels across, so its rim is about six pixels; the big one is three across,
// about nine. Dividing by the number of frames in a turn gives how far he
// travels per frame.
//
// Mixing up pixels and game units here makes the wheel spin four times too
// fast, like a propeller: at a full run Mario covers about four tiles a second.
func (p *Player) WheelPhase(phases int) int {
	if phases <= 0 {
		return 0
	}
	rimPixels := 6
	if p.Form != FormSmall {
		rimPixels = 9
	}
	circumference := engine.Fixed(rimPixels*unitsPerSpritePixel) * engine.Unit
	perFrame := circumference / engine.Fixed(phases)
	if perFrame <= 0 {
		return 0
	}
	return int(p.wheelDist/perFrame) % phases
}

// WheelRolledLeft reports whether the hero is facing left, which is when the
// wheel has to turn the other way. The renderer mirrors the whole figure for
// that, so this only exists to keep the rule in one place.
func (p *Player) WheelRolledLeft() bool { return p.Facing < 0 }

// horizontal applies acceleration, friction and skidding.
func (p *Player) horizontal(lv *level.Level, in engine.Pressed) {
	dir := 0
	if in.HasHeld(engine.ActionLeft) {
		dir--
	}
	if in.HasHeld(engine.ActionRight) {
		dir++
	}
	run := in.HasHeld(engine.ActionRun)

	// Star form runs a little faster; the original gives a speed boost.
	maxSpeed := p.T.WalkMax
	if run {
		maxSpeed = p.T.RunMax
	}
	if p.Form == FormStar {
		maxSpeed = maxSpeed + maxSpeed.Div(4)
	}

	if dir == 0 {
		// No input: friction on the ground, barely any drag in the air.
		if p.Grounded {
			p.Vel.X = decelerate(p.Vel.X, p.T.Friction)
		} else {
			p.Vel.X = decelerate(p.Vel.X, p.T.AirDrag)
		}
		return
	}

	// Turning against momentum uses the skid deceleration first.
	if dir*p.Vel.X.Sign() < 0 && p.Grounded {
		p.Vel.X = decelerate(p.Vel.X, p.T.SkidDecel)
		// Facing flips only once the velocity has actually crossed zero.
		if p.Vel.X.Sign() == dir || p.Vel.X == 0 {
			p.Facing = dir
		}
		return
	}

	accel := p.T.WalkAccel
	if !p.Grounded {
		accel = p.T.AirAccel
	} else if p.Vel.X.Abs() >= p.T.WalkMax {
		accel = p.T.RunAccel
	}

	if p.Vel.X.Abs() < maxSpeed {
		p.Vel.X += engine.Fixed(dir) * accel
		if p.Vel.X.Abs() > maxSpeed {
			p.Vel.X = engine.Fixed(dir) * maxSpeed
		}
	} else if p.Vel.X.Sign() != dir {
		// Above the cap and moving the other way: only decelerate.
		p.Vel.X = decelerate(p.Vel.X, accel)
	}

	// Facing follows the direction of travel, not the key. Holding the opposite
	// direction in mid-air only slows Mario; he is still moving the old way, so
	// he keeps facing it until his speed actually crosses zero. Turning on the
	// key press instead would have him glide backwards through the air.
	switch s := p.Vel.X.Sign(); {
	case s != 0:
		p.Facing = s
	default:
		p.Facing = dir
	}
}

// vertical applies the jump edge and gravity.
func (p *Player) vertical(lv *level.Level, in engine.Pressed) {
	switch {
	case p.Grounded && in.HasEdge(engine.ActionJump):
		p.Vel.Y = p.T.JumpVelocityFor(p.Vel.X)
		p.Jumping = true
		p.Grounded = false
		p.JumpHeld = true
		p.AirJumpUsed = false

	case p.T.AirJump && !p.Grounded && !p.AirJumpUsed && in.HasEdge(engine.ActionJump):
		// Optional second jump. Off by default: the original has none, and
		// the only legitimate way to rise while airborne is to stomp an
		// enemy, which the world already handles.
		p.Vel.Y = p.T.AirJumpVelocity
		p.JumpHeld = true
		p.AirJumpUsed = true

	case p.Jumping && p.JumpHeld && !in.HasHeld(engine.ActionJump):
		// Releasing mid-rise cuts the arc short. This is the short hop.
		p.JumpHeld = false
		if p.Vel.Y < 0 {
			p.Vel.Y = engine.MulRatio(p.Vel.Y, int(p.T.JumpCutFactor), int(engine.Unit))
		}
	}

	if p.Grounded && p.Vel.Y >= 0 {
		p.Vel.Y = p.T.GroundStickGravity
		return
	}

	g := p.T.GravityFor(p.Vel.Y)
	p.Vel.Y += g
	if p.Vel.Y > p.T.MaxFallSpeed {
		p.Vel.Y = p.T.MaxFallSpeed
	}
}

// updateDeath plays the death arc: rise, fall, then leave the screen.
func (p *Player) updateDeath() {
	p.deathTimer++
	if p.deathTimer < 30 {
		p.Vel.Y += p.T.GravityRise.Div(2)
		p.Pos.Y += p.Vel.Y
	}
}

// updateSlide moves the player down the flagpole.
func (p *Player) updateSlide(lv *level.Level) {
	if p.Pos.Y+p.H < engine.Fixed(lv.FlagBot+1)*engine.Tile {
		p.Pos.Y += engine.FromFloat(2)
	}
}

// decelerate moves v toward zero by amount, without overshooting.
func decelerate(v, amount engine.Fixed) engine.Fixed {
	if v > 0 {
		v -= amount
		if v < 0 {
			return 0
		}
		return v
	}
	if v < 0 {
		v += amount
		if v > 0 {
			return 0
		}
		return v
	}
	return 0
}
