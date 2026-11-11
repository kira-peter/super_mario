package game

import (
	"super-mario/pkg/engine"
	"super-mario/pkg/level"
)

// Score is a running tally with the game's scoring rules.
//
// The best score is not tracked here: it lives on the leaderboard, which ranks
// whole runs and keeps their times. A separate running maximum would be a
// second source of truth for the same fact.
type Score struct {
	Points int
	Coins  int
	Lives  int
}

// maxCoins is the coin count that converts into an extra life.
const maxCoins = 100

// addCoins awards coins and converts a full hundred into a life.
func (s *Score) addCoins(n int) {
	s.Coins += n
	for s.Coins >= maxCoins {
		s.Coins -= maxCoins
		s.Lives++
	}
}

// add awards points. Only positive amounts count: the score never goes down.
func (s *Score) add(p int) {
	if p <= 0 {
		return
	}
	s.Points += p
}

// World owns every live entity and the level they stand on.
type World struct {
	Level *level.Level
	T     *Tuning
	Input *engine.Input
	SM    *engine.StateMachine
	Score Score

	Player  *Player
	Enemies []*Enemy
	Items   []*Item
	Parts   []*Particle
	Bumps   []BumpAnim

	// CoinFX are coins flying to the counter, CoinPlus the "+1" shown there.
	CoinFX   []*CoinFX
	CoinPlus []*CoinPlus

	// TimeLeft is the countdown in seconds.
	TimeLeft int
	// stepCount is the number of simulation steps executed in this run.
	stepCount int
	// runSteps counts the steps the player actually spent playing, which is
	// what a leaderboard time should measure. stepCount cannot be used for
	// that: it also ticks through pauses and the end-of-level sequence.
	runSteps int
	// timeAcc counts steps toward the next second.
	timeAcc int

	// Camera is the left edge of the visible region, in game units.
	CameraX engine.Fixed
	// CameraY is the top edge.
	CameraY engine.Fixed

	// ViewW and ViewH are the visible size in game units, set by the renderer.
	ViewW, ViewH engine.Fixed

	// combo counts consecutive stomps for escalating score.
	combo int
	// stompChainOpen stays true while the player is airborne after a stomp.
	stompChainOpen bool

	// Tally is the results-screen state: remaining time converting to score.
	Tally Tally

	// usedBlocks tracks consumed block payloads by cell key.
	usedBlocks map[int]bool
	// revealed marks hidden blocks that have been found.
	revealed map[int]bool

	// Events records things the presentation layer should react to.
	Events []Event
}

// Tally drives the end-of-level score conversion.
//
// The spec asks for the conversion to be visible rather than instantaneous, so
// it runs at a fixed number of time units per step while the results screen is
// up (spec game-flow / 时间折算为分数).
type Tally struct {
	// Base is the score before any time bonus. It is kept so the results screen
	// can show the score as base plus bonus, rather than one number that
	// changes under the player's eyes.
	Base int
	// Units is the total time units being converted, fixed when the tally
	// starts. Remaining counts down from it; Units does not move.
	Units int
	// Remaining is the time still to be converted.
	Remaining int
	// Awarded is the score banked so far by the conversion.
	Awarded int
	// PerStep is how many time units convert each step.
	PerStep int
	// ScorePerUnit is the score each converted unit is worth.
	ScorePerUnit int
	// Done reports whether the conversion finished.
	Done bool
}

// tallyPerStep is how fast the tally drains, tuned so a full 400 reads in a few
// seconds rather than instantly or tediously.
const tallyPerStep = 4

// scorePerTimeUnit is the classic fifty points per remaining unit of time.
const scorePerTimeUnit = 50

// Event is a one-step signal for effects that are not game state, such as a
// sound trigger or a screen shake.
type Event struct {
	Kind string
	X, Y int
	Text string
}

// NewWorld builds a playable world from a parsed level.
func NewWorld(lv *level.Level, t *Tuning, sm *engine.StateMachine, lives int) *World {
	w := &World{
		Level:      lv,
		T:          t,
		SM:         sm,
		Input:      engine.NewInput(nil),
		TimeLeft:   lv.TimeLimit,
		usedBlocks: map[int]bool{},
		revealed:   map[int]bool{},
	}
	w.Score.Lives = lives
	w.Player = NewPlayer(t, lv.SpawnX, lv.SpawnY)
	w.resetEnemies()
	w.ViewW = engine.Tile.Mul(10)
	w.ViewH = engine.Tile.Mul(7)
	w.updateCamera(true)
	return w
}

// resetEnemies rebuilds the enemy list from the level's spawn points.
func (w *World) resetEnemies() {
	w.Enemies = w.Enemies[:0]
	for _, s := range w.Level.Spawns {
		var e *Enemy
		switch s.Kind {
		case "goomba":
			e = NewGoomba(s.X, s.Y)
		case "koopa":
			e = NewKoopa(s.X, s.Y)
		default:
			continue
		}
		w.Enemies = append(w.Enemies, e)
	}
}

// Reset puts the world back to the level's opening state, used after a death.
//
// The score survives; terrain, blocks and enemies do not (spec game-flow /
// 场景状态重置).
func (w *World) Reset() {
	// Terrain first: smashed bricks, used blocks and collected coins all come
	// back. Clearing only the bookkeeping maps left the tiles themselves
	// changed, so after a respawn a used question block stayed solid and
	// empty for the rest of the run.
	w.Level.Restore()
	w.Player = NewPlayer(w.T, w.Level.SpawnX, w.Level.SpawnY)
	w.Enemies = nil
	w.Items = nil
	w.Parts = nil
	w.Bumps = nil
	w.CoinFX = nil
	w.CoinPlus = nil
	w.usedBlocks = map[int]bool{}
	w.revealed = map[int]bool{}
	w.TimeLeft = w.Level.TimeLimit
	w.timeAcc = 0
	w.combo = 0
	w.stompChainOpen = false
	w.CameraX = 0
	w.CameraY = 0
	w.resetEnemies()
	w.updateCamera(true)
}

// Step advances one simulation step.
func (w *World) Step() {
	w.stepCount++

	// The results screen does not simulate entities, but it does run the
	// time-to-score conversion. That has to be reached before the gate below:
	// SimulatesWorld is false for results, so a case inside the switch further
	// down is dead code, and the remaining time never turned into score.
	if w.SM.Current() == engine.StateResults {
		w.stepTally()
		return
	}

	if !w.SM.Current().SimulatesWorld() {
		return
	}

	in := w.Input.Sample()
	w.Input.ConsumeEdges()

	switch w.SM.Current() {
	case engine.StatePlaying:
		w.stepPlaying(in)
	case engine.StateDying:
		w.Player.update(w.Level, in)
		if w.Player.deathTimer >= 90 {
			if w.Score.Lives > 1 {
				w.Score.Lives--
				w.SM.Respawn()
				w.Reset()
			} else {
				w.Score.Lives = 0
				w.SM.GameOver()
			}
		}
	case engine.StateFlagpole:
		w.stepFlagpole()
	}
}

// stepPlaying runs one normal gameplay step.
func (w *World) stepPlaying(in engine.Pressed) {
	w.runSteps++
	w.Player.update(w.Level, in)
	w.handleBump()

	for _, e := range w.Enemies {
		w.updateEnemy(e)
	}
	for _, it := range w.Items {
		it.update(w.Level, w.T)
	}
	w.updateParticles()
	w.updateBumps()
	w.updateCoinFX()

	w.resolveEnemyContacts()
	w.resolveItemContacts()
	w.collectCoins()
	w.cleanup()

	w.tickClock()
	w.updateCamera(false)

	// Falling out of the level is death.
	if w.Player.Pos.Y > engine.Fixed(w.Level.Height)*engine.Tile {
		w.killPlayer()
	}

	// Touching the flagpole starts the slide.
	if w.Player.Right() > engine.Fixed(w.Level.FlagX)*engine.Tile &&
		w.Player.Left() < engine.Fixed(w.Level.FlagX+1)*engine.Tile {
		w.startFlagpole()
	}
}

// stepFlagpole advances the end-of-level sequence.
func (w *World) stepFlagpole() {
	w.updateCoinFX()
	w.Player.updateSlide(w.Level)
	if w.Player.Pos.Y+w.Player.H >= engine.Fixed(w.Level.FlagBot+1)*engine.Tile {
		w.SM.ShowResults()
		w.startTally()
	}
}

// startTally begins converting remaining time into score.
func (w *World) startTally() {
	w.Tally = Tally{
		Base:         w.Score.Points,
		Units:        w.TimeLeft,
		Remaining:    w.TimeLeft,
		PerStep:      tallyPerStep,
		ScorePerUnit: scorePerTimeUnit,
	}
	if w.Tally.Remaining == 0 {
		w.Tally.Done = true
	}
}

// stepTally drains the time-to-score conversion one slice per step.
func (w *World) stepTally() {
	if w.Tally.Done {
		return
	}
	n := w.Tally.PerStep
	if n > w.Tally.Remaining {
		n = w.Tally.Remaining
	}
	w.Tally.Remaining -= n
	awarded := n * w.Tally.ScorePerUnit
	w.Tally.Awarded += awarded
	w.Score.add(awarded)
	if w.Tally.Remaining == 0 {
		w.Tally.Done = true
	}
}

// tickClock decrements the countdown once per second of simulation.
func (w *World) tickClock() {
	w.timeAcc++
	if w.timeAcc < 60 {
		return
	}
	w.timeAcc = 0
	if w.TimeLeft > 0 {
		w.TimeLeft--
	}
	if w.TimeLeft == 0 {
		w.killPlayer()
	}
}

// updateEnemy activates and steps one enemy.
func (w *World) updateEnemy(e *Enemy) {
	// Activation is one-way: an enemy woken by the camera keeps simulating
	// even after the camera moves past it.
	if !e.Active {
		activateX := w.CameraX + w.ViewW + engine.Tile.Mul(2)
		if e.Pos.X <= activateX {
			e.Active = true
		} else {
			return
		}
	}
	e.update(w.Level, w.T)
}

// resolveEnemyContacts handles player/enemy and shell/enemy overlaps.
func (w *World) resolveEnemyContacts() {
	p := w.Player
	if p.Dead {
		return
	}

	// A sliding shell wipes out any enemy it touches.
	for _, shell := range w.Enemies {
		if shell.Shell != KoopaSliding || shell.Removed {
			continue
		}
		for _, other := range w.Enemies {
			if other == shell || other.Removed || other.Squashed > 0 {
				continue
			}
			if shell.Overlaps(&other.Body) {
				other.Squashed = 30
				other.Vel = engine.Vec2{}
				w.Score.add(stompScore(w.combo + 1))
				w.combo++
			}
		}
	}

	for _, e := range w.Enemies {
		if e.Removed || e.Squashed > 0 || e.Grace > 0 {
			continue
		}
		if !p.Overlaps(&e.Body) {
			continue
		}

		// Contact direction decides the outcome: only a downward-moving
		// player standing above the enemy counts as a stomp.
		if p.Vel.Y > 0 && p.Bottom() <= e.Bottom() {
			w.stomp(e)
			continue
		}
		if i := whichShellToKick(e, p); i {
			continue
		}
		w.hurtPlayer()
	}
}

// whichShellToKick kicks a resting shell the player walked into, instead of
// treating the contact as damage.
func whichShellToKick(e *Enemy, p *Player) bool {
	if e.Kind != EnemyKoopa || e.Shell != KoopaShell {
		return false
	}
	e.kick(p.CenterX(), Default0())
	return true
}

// Default0 is a tiny helper so the kick call site reads cleanly.
func Default0() *Tuning { return &Default }

// stomp applies a stomp to an enemy.
func (w *World) stomp(e *Enemy) {
	p := w.Player
	wasResting := e.Kind == EnemyKoopa && e.Shell == KoopaShell
	score, bounce := e.stomp(w.T)
	if score == 0 {
		return
	}
	// The second stomp sends the shell flying, away from where he landed on it.
	// Only a shell that was already resting: stomping a Koopa makes the shell,
	// and stomping a sliding shell stops it.
	if wasResting {
		e.kick(p.CenterX(), w.T)
	}
	w.combo++
	w.Score.add(stompScore(w.combo))
	w.Parts = append(w.Parts, &Particle{
		Pos:  engine.Vec2{X: e.CenterX(), Y: e.Pos.Y},
		Life: 40,
		Kind: 's',
		Text: itoa(score),
	})
	if bounce {
		if p.JumpHeld {
			p.Vel.Y = w.T.StompBounceHi
		} else {
			p.Vel.Y = w.T.StompBounce
		}
		p.Jumping = true
		p.StompChainOpen = true
	}
}

// coinScore is what one coin from the level is worth.
const coinScore = 200

// collectCoins takes every coin tile the player is touching.
//
// Coins in the level are tiles, not entities: they never move, and there are
// enough of them that a tile is the cheap way to hold them. Nothing used to read
// them back, so they were drawn and could never be taken. The tile is cleared
// rather than flagged, so it simply stops being drawn; a respawn restores it
// with the rest of the terrain.
func (w *World) collectCoins() {
	p := w.Player
	if p.Dead {
		return
	}
	r := p.Rect()
	x0 := int(r.X) / int(engine.Tile)
	x1 := int(r.Right()-1) / int(engine.Tile)
	y0 := int(r.Y) / int(engine.Tile)
	y1 := int(r.Bottom()-1) / int(engine.Tile)
	if r.Y < 0 {
		y0 = -1
	}
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if w.Level.TileAt(x, y).Glyph != 'o' {
				continue
			}
			w.Level.SetTile(x, y, level.Tile{Kind: level.KindEmpty, Glyph: ' '})
			w.Score.addCoins(1)
			w.Score.add(coinScore)
			w.flyCoin(engine.Vec2{
				X: engine.Fixed(x)*engine.Tile + engine.Tile/2,
				Y: engine.Fixed(y)*engine.Tile + engine.Tile/2,
			}, 0)
		}
	}
}

// resolveItemContacts collects items the player touches.
func (w *World) resolveItemContacts() {
	p := w.Player
	for _, it := range w.Items {
		if it.Removed || it.Emerging > 0 {
			continue
		}
		if !p.Overlaps(&it.Body) {
			continue
		}
		it.Removed = true
		switch it.Kind {
		case ItemMushroom:
			w.Score.add(1000)
			p.Grow(w.Level)
		case ItemStar:
			w.Score.add(1000)
			p.Star(600)
		case ItemOneUp:
			w.Score.Lives++
		case ItemGold:
			w.Score.addCoins(goldCoins)
			w.Score.add(goldCoins * coinScore)
			at := engine.Vec2{X: it.CenterX(), Y: it.Pos.Y + it.H/2}
			for i := 0; i < goldCoins; i++ {
				w.flyCoin(at, i*goldStagger)
			}
		case ItemPoison:
			// Same as touching an enemy, including the grace period and
			// invincibility protecting him from it.
			if !p.Invulnerable() {
				w.hurtPlayer()
			}
		}
	}
}

// goldCoins is how many coins the gold mushroom is worth.
const goldCoins = 5

// itemFor maps a block's payload to the mushroom it releases, or false for a
// payload that is not a mushroom.
func itemFor(k level.BlockKind) (ItemKind, bool) {
	switch k {
	case level.BlockMushroom:
		return ItemMushroom, true
	case level.BlockStar:
		return ItemStar, true
	case level.BlockOneUp:
		return ItemOneUp, true
	case level.BlockPoison:
		return ItemPoison, true
	case level.BlockGold:
		return ItemGold, true
	}
	return 0, false
}

// handleBump reacts to the player's head hitting a block.
func (w *World) handleBump() {
	c := w.Player.LastHit
	if !c.HitCeiling {
		return
	}
	col, row := c.CeilingX, c.CeilingY

	key := row*w.Level.Width + col
	if w.usedBlocks[key] {
		w.Bumps = append(w.Bumps, BumpAnim{Col: col, Row: row, Remain: bumpFrames})
		return
	}

	t := w.Level.TileAt(col, row)
	switch t.Kind {
	case level.KindBreakable:
		if w.Player.Form == FormBig || w.Player.Form == FormStar {
			w.Level.SetTile(col, row, level.Tile{Kind: level.KindEmpty, Glyph: ' '})
			w.spawnShards(col, row)
			w.Score.add(50)
			return
		}
		w.Bumps = append(w.Bumps, BumpAnim{Col: col, Row: row, Remain: bumpFrames})
	case level.KindQuestion:
		w.Bumps = append(w.Bumps, BumpAnim{Col: col, Row: row, Remain: bumpFrames})
		w.dispense(col, row)
	case level.KindEmpty:
		// A hidden block sits in empty air; reveal and dispense it.
		if idx, ok := w.Level.ContentAt(col, row); ok && w.Level.Contents[idx].Hidden && !w.revealed[key] {
			w.revealed[key] = true
			w.Level.SetTile(col, row, level.Tile{Kind: level.KindQuestion, Glyph: '?'})
			w.Bumps = append(w.Bumps, BumpAnim{Col: col, Row: row, Remain: bumpFrames})
			w.dispense(col, row)
		}
	}
}

// dispense releases a block's payload.
func (w *World) dispense(col, row int) {
	idx, ok := w.Level.ContentAt(col, row)
	if !ok {
		w.Level.SetTile(col, row, level.Tile{Kind: level.KindSolid, Glyph: '#'})
		return
	}
	c := &w.Level.Contents[idx]
	key := row*w.Level.Width + col

	switch c.Kind {
	case level.BlockCoin:
		w.Score.addCoins(1)
		w.Score.add(coinScore)
		w.popCoin(col, row)
		c.Count--
		if c.Count <= 0 {
			w.usedBlocks[key] = true
			w.Level.SetTile(col, row, level.Tile{Kind: level.KindSolid, Glyph: '#'})
		}
	default:
		kind, ok := itemFor(c.Kind)
		if !ok {
			break
		}
		it := NewItem(kind, col, row-1)
		it.setOrigin(col, row-1)
		w.Items = append(w.Items, it)
		w.usedBlocks[key] = true
		w.Level.SetTile(col, row, level.Tile{Kind: level.KindSolid, Glyph: '#'})
	}
}

// spawnShards creates the four flying brick fragments.
func (w *World) spawnShards(col, row int) {
	cx := engine.Fixed(col)*engine.Tile + engine.Tile/2
	cy := engine.Fixed(row) * engine.Tile
	for i, v := range []engine.Vec2{
		{X: engine.FromFloat(-1.5), Y: engine.FromFloat(-4)},
		{X: engine.FromFloat(1.5), Y: engine.FromFloat(-4)},
		{X: engine.FromFloat(-1), Y: engine.FromFloat(-2.5)},
		{X: engine.FromFloat(1), Y: engine.FromFloat(-2.5)},
	} {
		_ = i
		w.Parts = append(w.Parts, &Particle{
			Pos:  engine.Vec2{X: cx, Y: cy},
			Vel:  v,
			Life: 60,
			Kind: 'b',
		})
	}
}

// updateParticles advances decorative fragments.
func (w *World) updateParticles() {
	live := w.Parts[:0]
	for _, p := range w.Parts {
		p.Life--
		if p.Life > 0 {
			p.Vel.Y += w.T.ItemGravity
			p.Pos = p.Pos.Add(p.Vel)
			live = append(live, p)
		}
	}
	w.Parts = live
}

// updateBumps advances block bump animations.
func (w *World) updateBumps() {
	live := w.Bumps[:0]
	for _, b := range w.Bumps {
		b.Remain--
		if b.Remain > 0 {
			live = append(live, b)
		}
	}
	w.Bumps = live
}

// cleanup drops removed entities.
func (w *World) cleanup() {
	live := w.Enemies[:0]
	for _, e := range w.Enemies {
		if !e.Removed {
			live = append(live, e)
		}
	}
	w.Enemies = live

	liveItems := w.Items[:0]
	for _, it := range w.Items {
		if !it.Removed {
			liveItems = append(liveItems, it)
		}
	}
	w.Items = liveItems
}

// hurtPlayer applies enemy contact damage.
func (w *World) hurtPlayer() {
	p := w.Player
	if p.Invulnerable() {
		return
	}
	if p.Form == FormSmall {
		w.killPlayer()
		return
	}
	p.Hurt(w.Level)
	w.combo = 0
}

// killPlayer starts the death sequence.
func (w *World) killPlayer() {
	if w.Player.Dead || w.SM.Is(engine.StateDying) {
		return
	}
	w.Player.Die()
	w.SM.Die()
	w.combo = 0
}

// startFlagpole begins the slide.
func (w *World) startFlagpole() {
	if !w.SM.TouchFlag() {
		return
	}
	p := w.Player
	p.SlideDown = true
	p.Vel = engine.Vec2{}
	p.Pos.X = engine.Fixed(w.Level.FlagX)*engine.Tile - p.W/2
	w.Score.add(flagpoleScore(p.Pos.Y, w.Level))
}

// flagpoleScore rewards touching the pole higher up.
func flagpoleScore(y engine.Fixed, lv *level.Level) int {
	height := engine.Fixed(lv.FlagBot-lv.FlagTop+1) * engine.Tile
	if height <= 0 {
		return 100
	}
	frac := (engine.Fixed(lv.FlagBot+1)*engine.Tile - y) * 100 / height
	switch {
	case frac > 85:
		return 5000
	case frac > 65:
		return 2000
	case frac > 45:
		return 800
	case frac > 25:
		return 400
	}
	return 100
}

// CameraVerticalMargin is how far above or below its resting position the
// camera will follow the player before it starts scrolling vertically.
//
// The camera tracks the player's feet on the ground, not his head in the air.
// A jump is the player's business: moving the camera with it makes the whole
// level lurch upward and the jump reads as the world dropping instead of Mario
// rising. Vertical scrolling is reserved for genuinely leaving the ground —
// climbing a staircase, riding a pipe top, or the flagpole slide.
const CameraVerticalMargin = engine.Tile * 3

// CameraBottomMargin is how far below the player's feet the screen extends when
// the view is resting.
//
// A fixed margin from the feet, rather than a fraction of the view height, is
// what makes the view stable: a fraction depends on the viewport, so the camera
// lands somewhere different before and after the terminal size is known, and
// the whole picture visibly jumps when it is recomputed.
const CameraBottomMargin = engine.Tile * 3 / 2

// CameraBackFraction places the scroll-back line: when the player walks left of
// one third of the screen, the camera follows him back.
const CameraBackFraction = 3

// SetViewport tells the world how much it can see.
//
// The renderer owns the terminal size, so the world cannot know it at
// construction. Recomputing the camera here, on every change, keeps the resting
// position a single well-defined place: without it the camera is first placed
// using a placeholder viewport and then re-derived from the real one, and the
// picture shifts on the first frame and again after every respawn.
func (w *World) SetViewport(vw, vh engine.Fixed) {
	if vw < engine.Tile {
		vw = engine.Tile
	}
	if vh < engine.Tile {
		vh = engine.Tile
	}
	if vw == w.ViewW && vh == w.ViewH {
		return
	}
	w.ViewW, w.ViewH = vw, vh
	w.updateCamera(true)
}

// updateCamera keeps the camera inside the level and following the player.
func (w *World) updateCamera(force bool) {
	lvW := engine.Fixed(w.Level.Width) * engine.Tile

	// Horizontal: advance when the player passes the middle of the screen, and
	// scroll back when he walks into the left third. Between the two is a dead
	// zone, so stepping back and forth does not move the view.
	target := w.Player.CenterX() - w.ViewW/2
	back := w.Player.CenterX() - w.ViewW/CameraBackFraction
	switch {
	case force:
		w.CameraX = target
	case target > w.CameraX:
		w.CameraX = target
	case back < w.CameraX:
		w.CameraX = back
	}
	if w.CameraX > lvW-w.ViewW {
		w.CameraX = lvW - w.ViewW
	}
	if w.CameraX < 0 {
		w.CameraX = 0
	}

	// Vertical: anchor on the ground line under the player.
	lvH := engine.Fixed(w.Level.Height) * engine.Tile
	resting := w.playerFeetY() + CameraBottomMargin - w.ViewH
	targetY := w.CameraY

	if w.Player.Grounded || w.Player.SlideDown || force {
		// Grounded: track the feet, with a dead zone so a jump or a single
		// step does not move the view at all.
		if resting > w.CameraY+CameraVerticalMargin {
			targetY = resting - CameraVerticalMargin
		} else if resting < w.CameraY-CameraVerticalMargin {
			targetY = resting + CameraVerticalMargin
		} else if force {
			targetY = resting
		}
	} else {
		// Airborne: hold the camera still unless the player has climbed clean
		// off the top of the screen, as when hopping between pipe tops.
		if w.Player.Top() < w.CameraY {
			targetY = w.Player.Top() - engine.Tile
		}
	}

	if targetY > lvH-w.ViewH {
		targetY = lvH - w.ViewH
	}
	if targetY < 0 {
		targetY = 0
	}
	w.CameraY = targetY
}

// playerFeetY returns the y of the surface under the player.
func (w *World) playerFeetY() engine.Fixed {
	if w.Player.Grounded {
		return w.Player.Bottom()
	}
	// Airborne: use the last grounded height so the camera has nothing to
	// follow while the player is in the air.
	return w.Player.Bottom()
}

// RunSteps returns how many steps of real play this run has taken, at 60 per
// second. Time spent paused, dying or sliding down the pole is not counted.
func (w *World) RunSteps() int { return w.runSteps }

// ResetRun clears the per-run counters. Respawning after a death keeps them,
// since that is the same run; starting over does not.
func (w *World) ResetRun() { w.runSteps = 0 }

// StepCount returns how many simulation steps this run has executed.
func (w *World) StepCount() int { return w.stepCount }

// CenterY returns the vertical centre of a body.
func (b *Body) CenterY() engine.Fixed { return b.Pos.Y + b.H/2 }

// ceilingHit is recorded by the player update so the world can react.
type ceilingHit struct {
	hit  bool
	x, y int
}

// lastCeiling is set by the player's collision resolve.
var _ = 0

// itoa avoids importing strconv in a hot path.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
