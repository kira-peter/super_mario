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
// The collision box is narrower than the artwork so a player can slip through
// a one-tile gap. The renderer fits the artwork under solid ceilings.
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
	// deathY0 is where he stood when he died; the hop starts there.
	deathY0 engine.Fixed

	// GrowFlash counts the frames of the grow/shrink flash.
	GrowFlash int
	// A mushroom collected in a one-tile passage takes effect when there is room.
	growPending bool
	// Shrinking counts down the shrink animation after a hit, during which the
	// world holds still: he flickers between big and small, as in the original.
	Shrinking int
	// DeathFacing is the way he faced when he died; he tumbles the other way.
	DeathFacing int
	// Respawning counts down the drop from the sky at the start of a new life.
	Respawning int
	// Poisoned counts down the sick look after a poison mushroom.
	Poisoned int
	// Melting marks a death by poison: instead of breaking apart he sickens
	// where he stands, then melts, his pixels dripping slowly down.
	Melting bool

	// trail is where he stood over the last few steps, newest last, for the
	// star's afterimages. trailN is how many entries are filled.
	trail  [trailLen]engine.Vec2
	trailN int

	// SlideDown is set while he is on the pole.
	SlideDown bool
	// Hidden is set once he has walked into the castle: he is not drawn.
	Hidden bool

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
func (p *Player) applyForm(lv *level.Level, f Form) bool {
	size := SmallSize
	if f != FormSmall {
		size = BigSize
	}
	r := engine.Rect{X: p.Pos.X, Y: p.Bottom() - size.Y, W: size.X, H: size.Y}
	if r.Y < p.Top() && overlapsSolid(lv, r) {
		x0, x1 := tileSpan(r.X, r.Right())
		y0, y1 := tileSpan(r.Y, p.Top())
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				if lv.SolidAt(x, y) {
					r.Y = engine.Fixed(y+1) * engine.Tile
				}
			}
		}
		if overlapsSolid(lv, r) {
			return false
		}
	}
	p.Pos.Y, p.W, p.H = r.Y, r.W, r.H
	if f == FormStar {
		f = FormBig // star power is not a size; see Star
	}
	p.Form = f
	return true
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
		p.growPending = !p.applyForm(lv, FormBig)
		if p.growPending {
			return false
		}
		p.GrowFlash = 16
		return true
	}
	return false
}

// Star makes the player invincible for the given number of steps. It does
// not change his size: a small hero stays small under the star and after it.
func (p *Player) Star(steps int) {
	p.StarSec = steps
}

// HasStar reports whether star power is running.
func (p *Player) HasStar() bool { return p.StarSec > 0 }

// IsBig reports whether he is the big size (and so can smash bricks).
func (p *Player) IsBig() bool { return p.Form != FormSmall }

// Hurt applies damage. A big player shrinks and becomes briefly immune; a small
// player dies.
func (p *Player) Hurt(lv *level.Level) (died bool) {
	if p.HasStar() || p.invuln > 0 {
		return false
	}
	if p.Form != FormSmall {
		p.applyForm(lv, FormSmall)
		p.invuln = playerInvulnSteps
		p.Shrinking = ShrinkSteps
		return false
	}
	p.Die()
	return true
}

// ShrinkSteps is how long the shrink animation holds the world still.
const ShrinkSteps = 48

// Death and respawn timing, in steps.
const (
	// DeathRise is the hop up when he dies.
	DeathRise = 20
	// DeathHang is the moment he hangs at the top, turning grey.
	DeathHang = 30
	// DeathBurst is when he breaks apart into pixels.
	DeathBurst = DeathRise + DeathHang
	// DeathScatter is how long the pixels fly apart, like smashed bricks.
	DeathScatter = 60
	// DeathPan is how long the camera takes to glide back to the start.
	DeathPan = 80
	// DeathTotal is the whole sequence; the new life starts after it.
	DeathTotal = DeathBurst + DeathScatter + DeathPan

	// RespawnGather is how long the pixels take to fly in and assemble him
	// in mid-air above the start.
	RespawnGather = 60
	// RespawnFall is the drop from there onto the ground.
	RespawnFall = 28
	// RespawnDrop is the whole arrival.
	RespawnDrop = RespawnGather + RespawnFall
)

// deathRiseHeight is how high the death hop goes.
const deathRiseHeight = engine.Tile * 2

// MeltSteps is how long a poisoned hero takes to melt away.
const MeltSteps = 110

// DeathScatterEnd is the step the pieces are gone and the camera may glide
// back: after the burst for a normal death, after the melt for poison.
func (p *Player) DeathScatterEnd() int {
	if p.Melting {
		return DeathBurst + MeltSteps
	}
	return DeathBurst + DeathScatter
}

// Die starts the death animation.
func (p *Player) Die() {
	p.Dead = true
	p.deathTimer = 0
	p.Melting = false
	p.growPending = false
	p.deathY0 = p.Pos.Y
	p.DeathFacing = p.Facing
	p.Vel = engine.Vec2{}
	p.Form = FormSmall
	p.StarSec = 0
	p.Shrinking = 0
	// Keep the feet where they were when a big hero dies small.
	bottom := p.Pos.Y + p.H
	p.W, p.H = SmallSize.X, SmallSize.Y
	p.Pos.Y = bottom - p.H
	p.deathY0 = p.Pos.Y
}

// DeathTimer is how far into the death sequence he is, in steps.
func (p *Player) DeathTimer() int { return p.deathTimer }

// trailLen is how many past positions are kept for the afterimages.
const trailLen = 12

// recordTrail notes where he is this step.
func (p *Player) recordTrail() {
	copy(p.trail[:], p.trail[1:])
	p.trail[trailLen-1] = p.Pos
	if p.trailN < trailLen {
		p.trailN++
	}
}

// PastPos is where he was some steps ago, or false when that is not known.
func (p *Player) PastPos(steps int) (engine.Vec2, bool) {
	if steps <= 0 || steps > p.trailN {
		return engine.Vec2{}, false
	}
	return p.trail[trailLen-1-steps], true
}

// Invulnerable reports whether contact should be ignored.
func (p *Player) Invulnerable() bool { return p.invuln > 0 || p.HasStar() }

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
	p.recordTrail()

	if p.GrowFlash > 0 {
		p.GrowFlash--
	}
	if p.invuln > 0 {
		p.invuln--
	}
	if p.StarSec > 0 {
		p.StarSec--
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

// advanceWheel turns the wheel by a distance travelled outside the normal
// movement code, such as the walk to the castle.
func (p *Player) advanceWheel(d engine.Fixed) { p.wheelDist += d.Abs() }

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
	if p.HasStar() {
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

// updateDeath plays his part of the death: a hop up, easing to a stop, where
// he hangs. The renderer does the rest (grey, then the burst into pixels).
func (p *Player) updateDeath() {
	p.deathTimer++
	if p.Melting {
		return // he sickens where he stands; no hop
	}
	if t := p.deathTimer; t <= DeathRise {
		n := DeathRise
		p.Pos.Y = p.deathY0 - engine.MulRatio(deathRiseHeight, n*n-(n-t)*(n-t), n*n)
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
