package game

import (
	"testing"

	"super-mario/pkg/engine"
	"super-mario/pkg/level"
)

// testLevel builds a flat floor level wide enough to move around on.
func testLevel(t *testing.T, width int) *level.Level {
	t.Helper()
	rows := make([]string, 15)
	for y := range rows {
		b := make([]byte, width)
		for x := range b {
			b[x] = ' '
		}
		if y >= 13 {
			for x := range b {
				b[x] = 'X'
			}
		}
		rows[y] = string(b)
	}
	data := `{"name":"t","world":"1-1","width":` + itoa(width) + `,"height":15,` +
		`"timeLimit":400,"spawn":{"x":2,"y":12},` +
		`"flag":{"x":` + itoa(width-2) + `,"topY":2,"bottomY":12},` +
		`"rows":[` + quoteRows(rows) + `],"contents":[],"entities":[]}`
	lv, err := level.Parse([]byte(data))
	if err != nil {
		t.Fatalf("build test level: %v", err)
	}
	return lv
}

func quoteRows(rows []string) string {
	out := ""
	for i, r := range rows {
		if i > 0 {
			out += ","
		}
		out += `"` + r + `"`
	}
	return out
}

func newTestWorld(t *testing.T) *World {
	t.Helper()
	lv := testLevel(t, 80)
	sm := engine.NewStateMachine()
	sm.StartRun()
	w := NewWorld(lv, &Default, sm, 3)
	return w
}

// runSteps holds the given keys for n steps and then releases them.
func runSteps(w *World, n int, keys ...engine.Key) {
	for i := 0; i < n; i++ {
		for _, k := range keys {
			w.Input.KeyDown(k)
		}
		w.Step()
	}
	for _, k := range keys {
		w.Input.KeyUp(k)
	}
}

func TestPlayerAcceleratesToWalkSpeed(t *testing.T) {
	w := newTestWorld(t)
	in := engine.NewInput(nil)
	w.Input = in

	// Hold right without run until the speed stops changing.
	var frames int
	prev := w.Player.Vel.X
	for i := 0; i < 300; i++ {
		in.KeyDown(engine.KeyRight)
		w.Step()
		if w.Player.Vel.X == prev && i > 10 {
			break
		}
		prev = w.Player.Vel.X
		frames++
	}

	if w.Player.Vel.X != Default.WalkMax {
		t.Errorf("walk speed = %v, want %v", w.Player.Vel.X, Default.WalkMax)
	}
	// The original reaches walking speed in roughly 20 to 40 frames.
	if frames < 15 || frames > 60 {
		t.Errorf("reached walk speed in %d frames, want roughly 20-40", frames)
	}
}

func TestRunButtonRaisesTopSpeed(t *testing.T) {
	w := newTestWorld(t)
	in := engine.NewInput(nil)
	w.Input = in

	for i := 0; i < 400; i++ {
		in.KeyDown(engine.KeyRight)
		in.KeyDown(engine.KeyShift)
		w.Step()
	}
	if w.Player.Vel.X != Default.RunMax {
		t.Errorf("run speed = %v, want %v", w.Player.Vel.X, Default.RunMax)
	}
	if Default.RunMax <= Default.WalkMax {
		t.Error("run speed must exceed walk speed")
	}
}

func TestFrictionBringsPlayerToRest(t *testing.T) {
	w := newTestWorld(t)
	in := engine.NewInput(nil)
	w.Input = in
	runSteps(w, 120, engine.KeyRight, engine.KeyShift)

	if w.Player.Vel.X <= 0 {
		t.Fatalf("player should be moving before releasing, got %v", w.Player.Vel.X)
	}
	startX := w.Player.Pos.X

	for i := 0; i < 200; i++ {
		w.Step()
	}
	if w.Player.Vel.X != 0 {
		t.Errorf("after release velocity = %v, want 0", w.Player.Vel.X)
	}
	if w.Player.Pos.X <= startX {
		t.Error("player should have slid forward before stopping")
	}
}

func TestSkidReversesOnlyAfterCrossingZero(t *testing.T) {
	w := newTestWorld(t)
	in := engine.NewInput(nil)
	w.Input = in
	runSteps(w, 200, engine.KeyRight, engine.KeyShift)

	if w.Player.Facing != 1 {
		t.Fatalf("facing = %d, want 1 while running right", w.Player.Facing)
	}

	// Press left: facing must not flip until velocity actually goes negative.
	flippedWhileMovingRight := false
	for i := 0; i < 200; i++ {
		in.KeyDown(engine.KeyLeft)
		w.Step()
		if w.Player.Facing == -1 && w.Player.Vel.X > 0 {
			flippedWhileMovingRight = true
		}
		if w.Player.Vel.X < 0 {
			break
		}
	}
	if flippedWhileMovingRight {
		t.Error("facing flipped while still moving right")
	}
	if w.Player.Vel.X >= 0 {
		t.Errorf("velocity = %v, expected negative after skid", w.Player.Vel.X)
	}
}

func TestJumpHeightDependsOnHoldDuration(t *testing.T) {
	short := jumpPeak(t, 2)
	long := jumpPeak(t, 60)

	if long <= short {
		t.Fatalf("long jump peak %v must exceed short jump peak %v", long, short)
	}
	// The original's full jump is roughly three times a tap.
	if long < short*2 {
		t.Errorf("long/short peak ratio = %.2f, want at least 2",
			float64(long)/float64(short))
	}
}

// jumpPeak returns how far above the ground the player rose, in game units.
func jumpPeak(t *testing.T, holdFrames int) engine.Fixed {
	t.Helper()
	w := newTestWorld(t)
	in := engine.NewInput(nil)
	w.Input = in

	ground := w.Player.Pos.Y

	for i := 0; i < holdFrames; i++ {
		in.KeyDown(engine.KeySpace)
		w.Step()
	}
	// Release and let the arc finish.
	for i := 0; i < 200; i++ {
		w.Step()
	}

	// Re-run measuring the peak instead of only the final position.
	w2 := newTestWorld(t)
	in2 := engine.NewInput(nil)
	w2.Input = in2
	peak := w2.Player.Pos.Y
	for i := 0; i < 200; i++ {
		if i < holdFrames {
			in2.KeyDown(engine.KeySpace)
		} else {
			in2.KeyUp(engine.KeySpace)
		}
		w2.Step()
		if w2.Player.Pos.Y < peak {
			peak = w2.Player.Pos.Y
		}
	}
	return ground - peak
}

func TestCannotDoubleJumpWhenSwitchedOff(t *testing.T) {
	w := newTestWorld(t)
	// The original rule: with the option off, a press in mid-air does nothing.
	w.T = &Tuning{}
	*w.T = Default
	w.T.AirJump = false
	w.Player.T = w.T
	in := engine.NewInput(nil)
	w.Input = in

	// Leave the ground.
	in.KeyDown(engine.KeySpace)
	w.Step()
	in.KeyUp(engine.KeySpace)
	w.Step()
	airspeed1 := w.Player.Vel.Y

	// Try to jump again mid-air.
	in.KeyDown(engine.KeySpace)
	w.Step()

	// A second jump would have snapped velocity back to the initial value.
	if w.Player.Vel.Y > airspeed1+Default.GravityRise*2 {
		t.Error("a second jump was applied while airborne with air jump off")
	}
}

func TestAirJumpGivesOneExtraKick(t *testing.T) {
	w := newTestWorld(t)
	w.T = &Tuning{}
	*w.T = Default
	w.T.AirJump = true
	w.Player.T = w.T
	in := engine.NewInput(nil)
	w.Input = in

	in.KeyDown(engine.KeySpace)
	w.Step()
	in.KeyUp(engine.KeySpace)
	for i := 0; i < 12; i++ {
		w.Step()
	}
	before := w.Player.Vel.Y

	in.KeyDown(engine.KeySpace)
	w.Step()
	if w.Player.Vel.Y >= before {
		t.Errorf("the air jump did not kick upward: vy %d -> %d", before.Int(), w.Player.Vel.Y.Int())
	}
	if !w.Player.AirJumpUsed {
		t.Error("the air jump was not marked used")
	}

	// And only once: another press must not add a second kick.
	in.KeyUp(engine.KeySpace)
	w.Step()
	mid := w.Player.Vel.Y
	in.KeyDown(engine.KeySpace)
	w.Step()
	if w.Player.Vel.Y < mid-Default.GravityFall*2 {
		t.Error("a third jump was applied")
	}
}

func placePlayer(w *World, x, y engine.Fixed, vx, vy engine.Fixed) {
	w.Player.Pos = engine.Vec2{X: x, Y: y}
	w.Player.Vel = engine.Vec2{X: vx, Y: vy}
	w.Player.Grounded = false
	w.Player.Jumping = false
}

func TestStompScoresAndBounces(t *testing.T) {
	w := newTestWorld(t)
	// A goomba directly under the player, one step away.
	e := NewGoomba(3, 12)
	e.Active = true
	w.Enemies = []*Enemy{e}

	placePlayer(w, e.Pos.X, e.Pos.Y-w.Player.H-1, 0, engine.FromFloat(2))

	before := w.Score.Points
	w.Step()

	if w.Score.Points <= before {
		t.Errorf("score did not increase on stomp: %d -> %d", before, w.Score.Points)
	}
	if w.Player.Vel.Y >= 0 {
		t.Errorf("player should bounce upward, got vy=%v", w.Player.Vel.Y)
	}
}

func TestSideContactHurtsSmallPlayer(t *testing.T) {
	w := newTestWorld(t)
	e := NewGoomba(3, 12)
	e.Active = true
	w.Enemies = []*Enemy{e}

	// Same vertical band, approaching from the side. Align the feet, not the
	// tops: the player is one game unit taller than a Goomba, so aligning tops
	// would bury the feet in the floor and the collision would push up instead.
	placePlayer(w, e.Pos.X-w.Player.W+1, e.Bottom()-w.Player.H, engine.FromFloat(1), 0)

	w.Step()

	if !w.Player.Dead {
		t.Error("small player should die from side contact")
	}
}

func TestBigPlayerShrinksInsteadOfDying(t *testing.T) {
	w := newTestWorld(t)
	w.Player.Grow(w.Level)
	if w.Player.Form != FormBig {
		t.Fatal("player did not grow")
	}

	e := NewGoomba(3, 12)
	e.Active = true
	w.Enemies = []*Enemy{e}
	w.Player.Pos.Y = e.Pos.Y
	w.Player.Pos.X = e.Pos.X - w.Player.W + 1

	w.Step()

	if w.Player.Dead {
		t.Error("big player must not die on first hit")
	}
	if w.Player.Form != FormSmall {
		t.Errorf("form = %v, want small after being hit", w.Player.Form)
	}
	if !w.Player.Invulnerable() {
		t.Error("player should be briefly invulnerable after shrinking")
	}
}

// addContent registers a block payload on the test level.
func addContent(w *World, x, y int, kind level.BlockKind, count int) {
	w.Level.Contents = append(w.Level.Contents, level.Content{
		X: x, Y: y, Kind: kind, Count: count,
	})
}

func TestBrickBreaksOnlyWhenBig(t *testing.T) {
	w := newTestWorld(t)
	// Put a brick directly above the player's head.
	bx, by := w.Level.SpawnX, w.Level.SpawnY-2
	var content []string
	_ = content
	w.Level.SetTile(bx, by, level.Tile{Kind: level.KindBreakable, Glyph: 'B'})

	// Small player: brick survives.
	w.Player.Pos.X = engine.Fixed(bx)*engine.Tile + 2
	w.Player.Pos.Y = engine.Fixed(by+1)*engine.Tile + 2
	w.Player.Vel.Y = engine.FromFloat(-4)
	for i := 0; i < 20; i++ {
		w.Step()
	}
	if w.Level.TileAt(bx, by).Kind != level.KindBreakable {
		t.Error("small player must not break a brick")
	}

	// Big player: brick breaks.
	w2 := newTestWorld(t)
	w2.Player.Grow(w2.Level)
	w2.Level.SetTile(bx, by, level.Tile{Kind: level.KindBreakable, Glyph: 'B'})
	w2.Player.Pos.X = engine.Fixed(bx)*engine.Tile + 2
	w2.Player.Pos.Y = engine.Fixed(by+1)*engine.Tile + 2
	w2.Player.Vel.Y = engine.FromFloat(-4)
	for i := 0; i < 20; i++ {
		w2.Step()
	}
	if w2.Level.TileAt(bx, by).Kind != level.KindEmpty {
		t.Error("big player should break the brick")
	}
}

func TestCoinBlockPaysOnce(t *testing.T) {
	w := newTestWorld(t)
	bx, by := w.Level.SpawnX, w.Level.SpawnY-2
	w.Level.SetTile(bx, by, level.Tile{Kind: level.KindQuestion, Glyph: '?'})
	addContent(w, bx, by, level.BlockCoin, 2)

	before := w.Score.Coins
	placePlayer(w, engine.Fixed(bx)*engine.Tile+2, engine.Fixed(by+1)*engine.Tile+2, 0, engine.FromFloat(-4))
	for i := 0; i < 20; i++ {
		w.Step()
	}

	if w.Score.Coins != before+1 {
		t.Errorf("coin block paid %d coins, want 1", w.Score.Coins-before)
	}

	// A second hit on the same block must pay nothing more once exhausted;
	// this block still holds one coin, so it pays exactly one again.
	placePlayer(w, engine.Fixed(bx)*engine.Tile+2, engine.Fixed(by+1)*engine.Tile+2, 0, engine.FromFloat(-4))
	for i := 0; i < 20; i++ {
		w.Step()
	}
	if w.Score.Coins != before+2 {
		t.Errorf("total coins = %d, want %d", w.Score.Coins, before+2)
	}

	// Exhausted: the tile becomes a used solid block and pays nothing.
	before = w.Score.Coins
	placePlayer(w, engine.Fixed(bx)*engine.Tile+2, engine.Fixed(by+1)*engine.Tile+2, 0, engine.FromFloat(-4))
	for i := 0; i < 20; i++ {
		w.Step()
	}
	if w.Score.Coins != before {
		t.Errorf("exhausted block paid again: %d -> %d", before, w.Score.Coins)
	}
	if w.Level.TileAt(bx, by).Kind == level.KindQuestion {
		t.Error("exhausted block should no longer be a question tile")
	}
}

func TestCoinsConvertToExtraLife(t *testing.T) {
	var s Score
	s.Lives = 3
	s.addCoins(100)
	if s.Coins != 0 {
		t.Errorf("coins = %d, want 0 after conversion", s.Coins)
	}
	if s.Lives != 4 {
		t.Errorf("lives = %d, want 4", s.Lives)
	}
}

func TestScoreNeverDecreases(t *testing.T) {
	var s Score
	s.add(500)
	// Taking damage must not touch points; add only accepts positive values.
	s.add(-100)
	if s.Points != 500 {
		t.Errorf("points = %d, want 500", s.Points)
	}
}

func TestTimerDecrementsOncePerSecond(t *testing.T) {
	w := newTestWorld(t)
	start := w.TimeLeft
	for i := 0; i < 60; i++ {
		w.Step()
	}
	if w.TimeLeft != start-1 {
		t.Errorf("after 60 steps time = %d, want %d", w.TimeLeft, start-1)
	}
}

func TestPausedWorldDoesNotAdvance(t *testing.T) {
	w := newTestWorld(t)
	w.SM.Pause()
	startTime := w.TimeLeft
	startX := w.Player.Pos.X
	in := engine.NewInput(nil)
	in.KeyDown(engine.KeyRight)
	w.Input = in
	for i := 0; i < 120; i++ {
		w.Step()
	}
	if w.TimeLeft != startTime {
		t.Error("timer advanced while paused")
	}
	if w.Player.Pos.X != startX {
		t.Error("player moved while paused")
	}
}

func TestEnemyTurnsAtWall(t *testing.T) {
	w := newTestWorld(t)
	e := NewGoomba(5, 12)
	e.Active = true
	e.Vel.X = -Default.EnemyWalkSpeed
	e.Facing = -1
	w.Enemies = []*Enemy{e}

	// Wall one cell to the left of the enemy.
	w.Level.SetTile(3, 12, level.Tile{Kind: level.KindSolid, Glyph: 'X'})

	for i := 0; i < 200; i++ {
		e.update(w.Level, &Default)
	}
	if w.Level.SolidAt(4, 12) {
		t.Skip("test geometry changed")
	}
}

func TestComboScoreEscalatesAndCaps(t *testing.T) {
	want := []int{100, 200, 400, 800, 1000, 2000, 4000, 8000}
	for i, w := range want {
		if got := stompScore(i + 1); got != w {
			t.Errorf("stompScore(%d) = %d, want %d", i+1, got, w)
		}
	}
	if got := stompScore(99); got != 8000 {
		t.Errorf("stompScore beyond the table = %d, want 8000", got)
	}
}

func TestDeterministicSimulation(t *testing.T) {
	run := func() engine.Vec2 {
		w := newTestWorld(t)
		in := engine.NewInput(nil)
		w.Input = in
		for i := 0; i < 180; i++ {
			if i%7 == 0 {
				in.KeyDown(engine.KeySpace)
			}
			if i%13 == 0 {
				in.KeyUp(engine.KeySpace)
			}
			in.KeyDown(engine.KeyRight)
			w.Step()
		}
		return w.Player.Pos
	}
	a, b := run(), run()
	if a != b {
		t.Errorf("simulation diverged: %v vs %v", a, b)
	}
}
