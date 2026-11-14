package game

import "super-mario/pkg/engine"

// Tuning is every number that decides how Mario feels.
//
// Design decision 7: these are not values that were tuned by eye. Each one is
// derived from the original game's per-frame pixel behaviour at 16px tiles and
// 60 Hz, and the comment above each value says what that behaviour is. If a
// future change wants a different feel, it changes this struct, not the
// physics code, and the comment tells it what it is departing from.
//
// Units: engine.Fixed, where engine.Unit is one original pixel and one tile is
// 16 units. "px/f" means original pixels per frame, "px/f^2" per frame squared.
type Tuning struct {
	// WalkMax is the top speed while the run button is not held.
	// Original: 0x00130 = 1.5625 px/f  (93.75 px/s)
	WalkMax engine.Fixed

	// RunMax is the top speed while the run button is held.
	// Original: 0x01900 = 2.5 px/f  (150 px/s)
	RunMax engine.Fixed

	// WalkAccel is the ground acceleration below walking speed.
	// Original: 0x00098 = 0.037109375 px/f^2
	WalkAccel engine.Fixed

	// RunAccel is the ground acceleration above walking speed.
	// Original: 0x000E4 = 0.0556640625 px/f^2
	RunAccel engine.Fixed

	// Friction decelerates the player when no direction is held.
	// Original: 0x000D0 = 0.05078125 px/f^2
	Friction engine.Fixed

	// SkidDecel is the sharper deceleration while turning against momentum.
	// Original: 0x001A0 = 0.1015625 px/f^2
	SkidDecel engine.Fixed

	// AirAccel controls horizontal control in the air.
	// Original: about half of RunAccel.
	AirAccel engine.Fixed

	// AirDrag is how much horizontal speed bleeds away per step while airborne
	// with no direction held.
	//
	// Zero: a jump carries its horizontal momentum to the landing untouched, and
	// only the player's own input changes it. That is what makes a running jump
	// a commitment. An earlier default of 0.0156 cost a sprint jump over a third
	// of its speed in mid-air, which is air resistance the game never had.
	AirDrag engine.Fixed

	// JumpVelocity is the initial upward speed of a standing jump.
	//
	// Measured peak height by simulation: 5.0 clears 4.01 tiles, 5.5 clears
	// 4.87 and 6.0 clears 5.81. The tallest pipe in 1-1 is four tiles, so 5.0
	// has no margin at all and 6.0 makes the jump feel floaty. 5.5 is the
	// point where the pipe is clear but the arc still reads as a jump.
	JumpVelocity engine.Fixed

	// JumpVelocityRunBonus is added when jumping at running speed, which is
	// what makes a running jump higher and reach further.
	JumpVelocityRunBonus engine.Fixed

	// JumpCutFactor scales upward velocity the frame the jump button is
	// released. This is the source of the short-hop feel.
	// Original: upward velocity is clamped to a much smaller value.
	JumpCutFactor engine.Fixed

	// GravityRise applies while ascending.
	// Original: 0x001E0 = 0.1875 px/f^2 (lighter, the classic float)
	GravityRise engine.Fixed

	// GravityFall applies while descending.
	// Original: 0x00700 = 0.4375 px/f^2
	GravityFall engine.Fixed

	// MaxFallSpeed caps downward speed so the player cannot tunnel.
	// Original: 4.5 px/f
	MaxFallSpeed engine.Fixed

	// GroundStickGravity is applied while grounded to keep contact with the
	// floor without letting the player sink.
	GroundStickGravity engine.Fixed

	// EnemyWalkSpeed is the constant walking speed of a Goomba.
	// Original: 0.5 px/f
	EnemyWalkSpeed engine.Fixed

	// ShellSpeed is the sliding speed of a kicked Koopa shell.
	// Original: about 3 px/f
	ShellSpeed engine.Fixed

	// ItemWalkSpeed is the walking speed of a mushroom.
	// Original: 1 px/f
	ItemWalkSpeed engine.Fixed

	// StompBounce is the upward velocity granted after stomping an enemy.
	// Original: -4 px/f or the current jump velocity, whichever is larger.
	StompBounce engine.Fixed

	// StompBounceHi is used while holding jump during a stomp.
	StompBounceHi engine.Fixed

	// AirJump lets the player jump once more while airborne.
	//
	// On by default. The original game has no mid-air jump, so this is a
	// deliberate departure that makes the game easier; -noairjump restores the
	// original rule.
	AirJump bool

	// AirJumpVelocity is the upward speed of the mid-air jump. It is a little
	// weaker than a standing jump, so the second hop assists a jump rather than
	// replacing it.
	AirJumpVelocity engine.Fixed

	// Gravity applied to loose items and particles.
	ItemGravity engine.Fixed
}

// Default is the tuning that matches the original game's feel where a terminal
// allows it.
//
// The top speeds and the jump's peak height are the original's. How fast Mario
// gets going, stops and turns is not: those are several times quicker. A
// terminal sends no key-up and starts auto-repeat only after half a second, so
// input reaches the game later and leaves it later than a pad's would; with the
// original's gentle acceleration on top of that, walking felt like wading. The
// jump keeps its height but rises faster, so it spends about an eighth less
// time in the air.
var Default = Tuning{
	WalkMax: engine.FromFloat(1.5625),
	RunMax:  engine.FromFloat(2.5),
	// Walk speed in nine frames, run speed in nine more: 0.15 s and 0.3 s,
	// where the original takes 0.72 s and 1 s.
	WalkAccel: engine.FromFloat(0.17361),
	RunAccel:  engine.FromFloat(0.10417),
	// Letting go at walk speed slides under half a tile, where the original
	// slides one and a half.
	Friction: engine.FromFloat(0.171),
	// A run turns round in ten frames, 0.17 s, where the original takes 0.42 s.
	SkidDecel: engine.FromFloat(0.25),
	// Air control reaches walk speed in about 36 frames, where the original
	// takes 56. It was first left near the original, and the air felt dead next
	// to the quickened ground. It is still well under the ground figure, so a
	// jump is a commitment: a full sprint jump stays up about 54 frames, and
	// steered hard back for all of them it sheds 54 x 0.044 = 2.38 of its 2.5
	// px/f, so it still lands moving forward, as in the original; it cannot be
	// turned round in the air. Anything above about 0.046 would let it.
	AirAccel: engine.FromFloat(0.044),
	AirDrag:  0,

	// The peak is unchanged at about 4.9 tiles, which clears the tallest
	// pipe. The jump starts faster and rises against heavier gravity, so it
	// spends 46 frames in the air instead of 52. Rise gravity stays well under
	// fall gravity, which keeps the light hang at the top of the arc.
	JumpVelocity:         engine.FromFloat(-6.989),
	JumpVelocityRunBonus: engine.FromFloat(-1.017),
	JumpCutFactor:        engine.FromFloat(0.25),

	GravityRise:        engine.FromFloat(0.30),
	GravityFall:        engine.FromFloat(0.4375),
	MaxFallSpeed:       engine.FromFloat(4.5),
	GroundStickGravity: engine.FromFloat(0.0625),

	EnemyWalkSpeed: engine.FromFloat(0.5),
	ShellSpeed:     engine.FromFloat(3.0),
	ItemWalkSpeed:  engine.FromFloat(1.0),
	ItemGravity:    engine.FromFloat(0.25),

	// Scaled with the jump, so a bounce reaches the same height as before.
	StompBounce:   engine.FromFloat(-5.083),
	StompBounceHi: engine.FromFloat(-6.353),

	AirJump:         true,
	AirJumpVelocity: engine.FromFloat(-6.353),
}

// JumpVelocityFor returns the initial vertical speed for a jump started at the
// given horizontal speed. The interpolation is linear from rest to RunMax, which
// reproduces the original's small but real bonus for running jumps.
func (t Tuning) JumpVelocityFor(speed engine.Fixed) engine.Fixed {
	s := speed.Abs()
	if s > t.RunMax {
		s = t.RunMax
	}
	// bonus = JumpVelocityRunBonus * (s / RunMax)
	bonus := engine.MulRatio(t.JumpVelocityRunBonus, int(s), int(t.RunMax))
	return t.JumpVelocity + bonus
}

// GravityFor returns the gravity to apply for the current vertical direction.
func (t Tuning) GravityFor(vy engine.Fixed) engine.Fixed {
	if vy < 0 {
		return t.GravityRise
	}
	return t.GravityFall
}
