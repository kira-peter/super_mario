// Package app wires the terminal, the fixed-step loop and the game together.
package app

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/engine"
	"super-mario/pkg/game"
	"super-mario/pkg/level"
	"super-mario/pkg/render"
	"super-mario/pkg/render/theme"
	"super-mario/pkg/save"
)

// Options configures a run.
type Options struct {
	LevelPath string
	ThemeName string
	Scale     string
	FPS       int
	// AirJump enables the mid-air second jump. On by default.
	AirJump bool
}

// MinCols and MinRows are the smallest terminal that still shows a playable
// view. Below this the renderer would have to drop so much that the player
// could not see a jump ahead, so the game says so plainly instead
// (spec terminal-renderer / 最小尺寸).
const (
	MinCols = 40
	MinRows = 12
)

// App is the running game.
type App struct {
	screen tcell.Screen
	opts   Options
	store  *save.Store

	input *engine.Input
	clock *engine.Clock
	sm    *engine.StateMachine
	lv    *level.Level
	world *game.World

	theme  theme.Tokens
	scene  render.Scene
	notice string

	colsPerTile int
	unicode     bool

	// frameEvery is how many simulation steps pass between renders.
	frameEvery int
	frameCount int
	// frameID counts rendered frames, used for the periodic full repaint.
	frameID int

	// prevCells is the last rendered character layer, so only changed cells
	// are written to the terminal.
	prevCells []render.Cell
	prevW     int
	prevH     int

	// heldAt records when each key was last pressed. Terminals do not send
	// key-up events, so a key is released once its presses stop arriving.
	heldAt map[engine.Key]timeStamp

	// timeNow is the clock for key timing; tests replace it.
	timeNow func() time.Time
	// probe reports whether a key is physically down, on platforms that can
	// say. It is nil where they cannot, and key timing falls back to inference.
	probe keyProbe

	// menu is the pause menu while the game is paused, nil otherwise.
	menu *pauseMenu

	// title is the title screen's own state: which page and which entry.
	title *titleState
	// board is the leaderboard, loaded at start and rewritten after each run.
	board save.Board
	// resultRank is the rank the finished run earned, shown on the results
	// screen. Zero means it did not place.
	resultRank int
	// resultCleared reports whether the finished run reached the flagpole.
	resultCleared bool
	// recorded guards against writing the same finished run twice.
	recorded bool
	// resultSteps is the finished run's duration, kept so the results screen
	// shows exactly what was written to the board.
	resultSteps int
	// resultPrevBest is the board's top entry before this run was added, used
	// to show how far ahead of or behind it the run finished.
	resultPrevBest save.Entry
	resultHadPrev  bool

	quit bool
}

// New builds an app with an initialised terminal.
func New(s tcell.Screen, opts Options) (*App, error) {
	lv, err := loadLevel(opts.LevelPath)
	if err != nil {
		return nil, err
	}
	a, err := newApp(s, opts, lv)
	if err != nil {
		return nil, err
	}
	// Only the real entry point asks the keyboard what is physically down.
	// Tests build apps through newApp and must not depend on the state of the
	// machine they happen to run on.
	a.probe = platformProbe()
	return a, nil
}

// newApp assembles the app around an already-loaded level.
func newApp(s tcell.Screen, opts Options, lv *level.Level) (*App, error) {
	tok, _ := theme.Resolve(opts.ThemeName)
	cap := detectCapacity(s)
	tok = theme.Degrade(tok, cap)

	a := &App{
		screen:  s,
		opts:    opts,
		store:   save.New(),
		input:   engine.NewInput(nil),
		clock:   engine.NewClock(),
		sm:      engine.NewStateMachine(),
		lv:      lv,
		theme:   tok,
		unicode: true,
		heldAt:  map[engine.Key]timeStamp{},
		timeNow: time.Now,
	}
	a.store.Warn = func(msg string) { a.notice = msg }

	a.board = a.store.Load()
	tuning := game.Default
	tuning.AirJump = opts.AirJump
	a.world = game.NewWorld(lv, &tuning, a.sm, 3)
	// The world must sample the same input the terminal writes into. Giving
	// them separate instances makes the game tick along with no controls.
	a.world.Input = a.input
	a.title = newTitleState(a)

	a.scene = render.Scene{T: tok, Theme: opts.ThemeName, Sky: render.NewSky(time.Now().UnixNano())}
	a.frameEvery = 60 / max(1, opts.FPS)
	tw, th := 120, 40
	if s != nil {
		tw, th = s.Size()
	}
	a.colsPerTile = pickColsPerTile(opts.Scale, tw, th)
	return a, nil
}

// repeatGap is the longest pause between two events that still counts as the
// keyboard auto-repeating one held key.
//
// Auto-repeat fires every 30 to 50 ms once it has started. A person pressing a
// key twice on purpose takes noticeably longer than that between the two
// presses, so anything slower than this is a fresh press.
const repeatGap = 70 * time.Millisecond

// handleKeyDown records a key press and its timestamp.
//
// A key may map to more than one action (Space is both jump and confirm), so
// every candidate is recorded and the state decides which one matters.
//
// Terminals send no key-up. Without one, a second press of a key that is still
// "held" would produce no press edge at all, and a double jump could never be
// triggered: the input layer would see a key that never went up and ignore the
// repeat. So a press that arrives after more than repeatGap since the last one
// is treated as the key having been released and pressed again.
func (a *App) handleKeyDown(k engine.Key) {
	b := a.input.Binding()
	acts := b.ActionsFor(k)
	if len(acts) == 0 {
		return
	}
	now := a.timeNow()
	if last, held := a.heldAt[k]; held && now.Sub(last) > repeatGap {
		for _, act := range acts {
			a.input.ReleaseAction(act)
		}
	}
	a.heldAt[k] = now
	for _, act := range acts {
		a.input.PressAction(act)
	}
}

func loadLevel(path string) (*level.Level, error) {
	if path != "" {
		return level.LoadFile(path)
	}
	return level.LoadEmbedded("1-1")
}

// detectCapacity asks the terminal what it can do.
func detectCapacity(s tcell.Screen) theme.Capacity {
	if s == nil {
		return theme.Color256
	}
	if s.Colors() >= 1<<24 {
		return theme.TrueColor
	}
	return theme.Color256
}

// pickColsPerTile chooses the zoom.
//
// A tile is drawn with the same number of art pixels as it has terminal
// columns, which is what keeps it square: the terminal cell is about twice as
// tall as it is wide, and two art pixels stack into one cell.
//
//	8   wide view   eight art pixels per tile; the whole screen shows about
//	                twenty tiles across and nine down, so a gap is visible
//	                before it has to be jumped
//	4   tiny view   four art pixels per tile; about forty tiles across. Mario
//	                becomes a coloured blob, but the level reads at a glance.
//	                The picture is downsampled from the compact art rather
//	                than hand-drawn, because at this size there is nothing
//	                left to draw intentionally.
//	12  mid view    downsampled from the detailed art, more detail, less area
//	16  close view  the detailed art at one pixel per column; needs 160
//	                columns for ten tiles
//
// Auto picks the tiny view: the level has to be readable at a glance, and at
// this size the whole screen shows about thirty tiles across and nineteen down,
// so a gap, a pipe and the enemy beyond it are all visible at once. Detail on
// the sprite is what has been traded away for that, deliberately.
func pickColsPerTile(scale string, w, h int) int {
	switch scale {
	case "native":
		return 16
	case "mid", "small":
		return 12
	case "wide":
		return 8
	case "tiny":
		return 4
	}
	return 4
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Run drives the game until the player quits.
func (a *App) Run() error {
	defer a.screen.Fini()

	events := make(chan tcell.Event, 64)
	go func() {
		for {
			ev := a.screen.PollEvent()
			if ev == nil {
				close(events)
				return
			}
			events <- ev
		}
	}()

	ticker := time.NewTicker(time.Second / 60)
	defer ticker.Stop()

	last := time.Now()
	for !a.quit {
		select {
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			a.handleEvent(ev)
		case now := <-ticker.C:
			delta := now.Sub(last)
			last = now
			steps := a.clock.Advance(delta)
			for i := 0; i < steps; i++ {
				// Keys with no key-up event are released here, on the same
				// goroutine that reads them, so there is no race with the
				// event handler.
				a.sweepReleases()
				a.stepOnce()
			}
			if steps > 0 {
				a.frameCount++
			}
			if a.frameCount >= a.frameEvery {
				a.frameCount = 0
				if err := a.draw(); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// stepOnce advances the simulation by exactly one fixed step.
func (a *App) stepOnce() {
	switch a.sm.Current() {
	case engine.StateTitle:
		// The title is driven by key events (see handleTitleKey), not by the
		// game's action sampler, so there is nothing to simulate here. The
		// pending edges are still cleared so a press made on the title does
		// not fire as a jump the instant the run starts.
		a.input.ConsumeEdges()

	case engine.StateResults:
		// The end-of-level tally lives in the world and has to be stepped,
		// or the remaining time never converts to score: it was only ever
		// exercised by a unit test that called it directly.
		a.world.Step()
		in := a.input.Sample()
		a.input.ConsumeEdges()
		if a.world.Tally.Done || a.resultCleared == false {
			a.finishRunOnce()
			if in.HasEdge(engine.ActionConfirm) {
				a.leaveRun(a.resultRank)
			}
		}

	default:
		a.world.Step()
		// A run can end without the results screen being entered through the
		// flagpole: running out of lives goes straight to it.
		if a.sm.Is(engine.StateResults) && !a.recorded && !a.world.Tally.Done {
			a.resultCleared = a.world.Cleared
		}
	}
}

// finishRunOnce records the finished run on the leaderboard a single time.
//
// The results state is stepped every frame, so without the guard a run would be
// written once per step.
func (a *App) finishRunOnce() {
	if a.recorded {
		return
	}
	a.recorded = true
	a.resultRank = a.recordRun(a.resultCleared)
}

// startRun resets the world and enters gameplay.
func (a *App) startRun() {
	a.recorded = false
	a.resultRank = 0
	a.resultSteps = 0
	a.resultHadPrev = false
	a.resultCleared = false
	a.world.ResetRun()
	a.world.Reset()
	a.world.Score.Points = 0
	a.world.Score.Coins = 0
	a.world.Score.Lives = 3
	a.sm.ToTitle()
	a.sm.StartRun()
	a.clock.Reset()
	// Every run gets its own clouds.
	a.scene.Sky = render.NewSky(time.Now().UnixNano())
}

// handleEvent translates a terminal event into an input action.
func (a *App) handleEvent(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventKey:
		// Ctrl-C always quits, in every state.
		if e.Key() == tcell.KeyCtrlC {
			a.quit = true
			return
		}

		// The title screen has its own menus and keys.
		if a.sm.Is(engine.StateTitle) {
			a.handleTitleKey(e)
			if a.screen != nil && !a.quit {
				_ = a.draw()
			}
			return
		}

		// While paused the menu owns the keyboard. Routing here, before the
		// game's key map, is what stops Space from jumping behind the menu
		// and Escape from being swallowed by a state that never reads it.
		if a.sm.Is(engine.StatePaused) {
			a.handlePauseKey(e)
			// A menu action should show at once. Waiting for the next render
			// tick makes closing it feel like it lingers for a moment.
			if a.screen != nil && !a.quit {
				_ = a.draw()
			}
			return
		}

		// Escape opens the menu during play.
		if e.Key() == tcell.KeyEscape && a.sm.Is(engine.StatePlaying) {
			a.openPause()
			if a.screen != nil {
				_ = a.draw()
			}
			return
		}

		if k, ok := mapKey(e); ok {
			// Pause has its own key as well as Escape.
			if k == engine.KeyP && a.sm.Is(engine.StatePlaying) {
				a.openPause()
				return
			}
			a.handleKeyDown(k)
		}
	case *tcell.EventResize:
		a.screen.Sync()
		_ = a.draw()
	}
}

// releaseWindow is how long a key stays "held" after the last press event.
//
// Terminal key repeat typically starts around 500 ms and repeats every 30-50 ms
// once running, so a window well above one repeat interval keeps a held key
// down while letting go of it stop the player quickly. Too short and movement
// stutters between repeats; too long and the player keeps sliding after the key
// is released. 100 ms is two repeat intervals at the slow end: it rides over a
// missed repeat, and lets go about twice as fast as the 180 ms it replaced.
const releaseWindow = 100 * time.Millisecond

// sweepReleases drops keys that have not been seen recently.
//
// This runs on the simulation goroutine, not from a timer, so the input state
// has a single writer and there is no data race with the event handler.
func (a *App) sweepReleases() {
	if len(a.heldAt) == 0 {
		return
	}
	now := a.timeNow()
	for k, last := range a.heldAt {
		// Where the platform can say whether a key is really down, believe it.
		// Inference from repeat events is wrong in exactly the case that
		// matters: the keyboard repeats only the most recently pressed key, so
		// holding Right and then tapping Space to jump silences Right's
		// repeats for good. The game would think Right was let go the moment
		// Mario left the ground.
		if a.probe != nil {
			if down, known := a.probe(k); known {
				if down {
					a.heldAt[k] = now
					continue
				}
				a.releaseKey(k)
				delete(a.heldAt, k)
				continue
			}
		}
		if now.Sub(last) > releaseWindow {
			a.releaseKey(k)
			delete(a.heldAt, k)
		}
	}
}

// releaseKey releases every action a key is bound to.
//
// A key can serve several actions (Space is jump and confirm), and handleKeyDown
// presses all of them. Releasing only the first left the rest held for good,
// and a held action never produces a press edge: after one jump, Space no
// longer counted as "confirm" on the results screen.
func (a *App) releaseKey(k engine.Key) {
	b := a.input.Binding()
	for _, act := range b.ActionsFor(k) {
		a.input.ReleaseAction(act)
	}
}

// mapKey converts a tcell key onto the game's key identifiers.
func mapKey(e *tcell.EventKey) (engine.Key, bool) {
	switch e.Key() {
	case tcell.KeyLeft:
		return engine.KeyLeft, true
	case tcell.KeyRight:
		return engine.KeyRight, true
	case tcell.KeyUp:
		return engine.KeyUp, true
	case tcell.KeyDown:
		return engine.KeyDown, true
	case tcell.KeyEnter:
		return engine.KeyEnter, true
	case tcell.KeyEscape:
		return engine.KeyEscape, true
	case tcell.KeyTab:
		return engine.KeyTab, true
	case tcell.KeyRune:
		switch e.Rune() {
		case ' ':
			return engine.KeySpace, true
		case 'p', 'P':
			return engine.KeyP, true
		case 'q', 'Q':
			return engine.KeyQ, true
		case 'x', 'X':
			return engine.KeyX, true
		case 'z', 'Z':
			return engine.KeyZ, true
		case 'r', 'R':
			return engine.KeyR, true
		case '?':
			return engine.KeyQuestion, true
		}
	}
	return engine.KeyUnknown, false
}

// draw renders the current state.
func (a *App) draw() error {
	w, h := a.screen.Size()

	// A terminal smaller than this cannot show a usable view; say so rather
	// than render something broken.
	if w < MinCols || h < MinRows {
		a.drawTooSmall(w, h)
		return nil
	}

	switch a.sm.Current() {
	case engine.StateTitle:
		a.screen.Clear()
		a.drawTitleScreen(w, h)
	case engine.StateResults:
		a.screen.Clear()
		a.drawResults(w, h)
	default:
		a.drawGame(w, h)
		if a.sm.Is(engine.StatePaused) {
			a.drawPauseMenu(w, h)
		}
	}

	a.screen.Show()
	a.frameID++
	return nil
}

// drawTooSmall replaces the view with a single explanatory line.
func (a *App) drawTooSmall(w, h int) {
	a.screen.Clear()
	style := tcell.StyleDefault.Foreground(a.theme.HudText).Background(a.theme.Sky)
	msg := fmt.Sprintf("窗口太小：当前 %d×%d，至少需要 %d×%d", w, h, MinCols, MinRows)
	writeCentered(a.screen, h/2, w, msg, style)
	a.screen.Show()
	a.prevW = 0 // force a full repaint once the window grows again
}

// fullRepaint is true every sixtieth frame and whenever the geometry changed.
//
// A periodic full paint defends against the terminal's own idea of its screen
// drifting away from ours — after a suspend, a multiplexer redraw, or a font
// change that clears cells behind the game's back
// (spec terminal-renderer / 差量刷新).

// drawGame paints the level plus the status line.
// applyViewport sizes the world's camera window for a terminal of cols by rows
// cells.
//
// Both axes divide by the same zoom. Terminal cells are about twice as tall as
// they are wide and two art pixels stack into one cell, so a square art grid
// needs the same art pixels per tile on both axes. Dividing rows by a constant
// while dividing columns by the zoom stretches every pixel vertically: the
// picture gets thinner but keeps its full height.
//
// The result is deliberately not rounded to whole tiles. Rounding down leaves a
// strip of unused terminal rows, and the renderer then has to spread the art
// over them, which stretches the picture the same way a wrong divisor does. A
// viewport of 7.5 tiles is perfectly fine: the camera only needs a size in game
// units, and nothing about it has to line up with the tile grid.
func (a *App) applyViewport(cols, rows int) {
	vrows := rows * 2
	zoom := a.colsPerTile
	if zoom < 1 {
		zoom = 1
	}
	if cols < 1 {
		cols = 1
	}
	if vrows < 1 {
		vrows = 1
	}
	w := engine.Tile * engine.Fixed(cols) / engine.Fixed(zoom)
	h := engine.Tile * engine.Fixed(vrows) / engine.Fixed(zoom)
	a.world.SetViewport(w, h)
}

// hudRows is how many terminal rows the status line takes, at the top of the
// screen as in the original: the picture is drawn below it.
const hudRows = 1

func (a *App) drawGame(w, h int) {
	rows := h - hudRows
	if rows < 1 {
		rows = 1
	}
	cols := w

	a.applyViewport(cols, rows)

	c := a.scene.Draw(a.world, a.colsPerTile)
	cells := c.Convert(cols, rows, a.unicode)
	render.OverlayFloatTexts(cells, cols, rows, a.world, a.colsPerTile, a.theme)

	// Write only cells that changed since the last frame, except on a periodic
	// full repaint where every cell is written to resynchronise with the
	// terminal.
	full := a.frameID%60 == 0 || a.prevW != cols || a.prevH != rows || a.prevCells == nil
	for cy := 0; cy < rows; cy++ {
		for cx := 0; cx < cols; cx++ {
			i := cy*cols + cx
			cell := cells[i]
			if !full && sameCell(cell, a.prevCells[i]) {
				continue
			}
			style := tcell.StyleDefault
			if cell.FG != nil {
				style = style.Foreground(cell.FG.(tcell.Color))
			}
			if cell.BG != nil {
				style = style.Background(cell.BG.(tcell.Color))
			}
			a.screen.SetContent(cx, cy+hudRows, cell.Rune, nil, style)
		}
	}
	a.prevCells = cells
	a.prevW, a.prevH = cols, rows

	a.drawHUD(w, h)
}

// sameCell reports whether two rendered cells are identical, so a redraw can
// skip writing them.
func sameCell(a, b render.Cell) bool {
	return a.Rune == b.Rune && a.FG == b.FG && a.BG == b.BG
}

// drawHUD writes the status line on the top row, and a notice, if any, on the
// row under it.
func (a *App) drawHUD(w, h int) {
	style := tcell.StyleDefault.
		Foreground(a.theme.HudText).
		Background(a.theme.Sky)
	row := 0
	text := render.HUDRow(a.world, w)
	x := 0
	for _, r := range text {
		if x >= w {
			break
		}
		a.screen.SetContent(x, row, r, nil, style)
		x++
	}
	a.drawCoinPlus(w, row)
	if a.notice != "" && h > 2 {
		nstyle := tcell.StyleDefault.Foreground(a.theme.HudMuted).Background(a.theme.Sky)
		x = 0
		for _, r := range a.notice {
			if x >= w {
				break
			}
			a.screen.SetContent(x, row+1, r, nil, nstyle)
			x++
		}
	}
}

// drawCoinPlus shows the green "+1" just after the coin count, while coins are
// landing on it. It sits on the status line, over the spaces after the count.
func (a *App) drawCoinPlus(w, row int) {
	for _, p := range a.world.CoinPlus {
		st, ok := render.CoinPlusStyle(a.theme, p.Age)
		if !ok {
			continue
		}
		x := render.CoinCounterCol + 4 // after "ⓒ×NN"
		for _, r := range fmt.Sprintf("+%d", p.Amount) {
			if x >= w {
				break
			}
			a.screen.SetContent(x, row, r, nil, st)
			x++
		}
	}
}

// drawSprite paints a sprite directly to the screen, one art pixel per cell.
func drawSprite(s tcell.Screen, sp render.Sprite, x0, y0 int, t theme.Tokens) {
	for y := 0; y < sp.H; y++ {
		for x := 0; x < sp.W; x++ {
			slot := sp.Rows[y][x]
			if slot == render.SlotNone {
				continue
			}
			col, ok := render.SlotColor(t, slot)
			if !ok {
				continue
			}
			style := tcell.StyleDefault.Foreground(col.(tcell.Color)).Background(t.Sky)
			s.SetContent(x0+x, y0+y, '█', nil, style)
		}
	}
}
