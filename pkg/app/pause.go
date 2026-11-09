package app

import (
	"time"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/engine"
)

// timeStamp is the type of a key's last-seen time.
type timeStamp = time.Time

// menuItem is one row of the pause menu.
type menuItem struct {
	label  string
	action func(a *App)
}

// pauseMenu is the list shown while the game is paused.
//
// It lives in the app rather than the engine: what the entries do (restart the
// run, quit the process) is the application's business, while the engine only
// knows that the world is frozen.
type pauseMenu struct {
	items  []menuItem
	cursor int
}

// newPauseMenu builds the menu with its default entries.
func newPauseMenu() *pauseMenu {
	return &pauseMenu{
		items: []menuItem{
			{"继续游戏", func(a *App) { a.resume() }},
			{"重新开始", func(a *App) { a.restartRun() }},
			{"回到标题", func(a *App) { a.toTitle() }},
			{"退出游戏", func(a *App) { a.quit = true }},
		},
	}
}

// move shifts the cursor, wrapping at both ends.
func (m *pauseMenu) move(delta int) {
	n := len(m.items)
	if n == 0 {
		return
	}
	m.cursor = ((m.cursor+delta)%n + n) % n
}

// activate runs the highlighted entry.
func (m *pauseMenu) activate(a *App) {
	if m.cursor >= 0 && m.cursor < len(m.items) {
		m.items[m.cursor].action(a)
	}
}

// invalidateFrame forgets what the game layer last wrote, so the next frame
// rewrites every cell.
//
// The game layer only writes cells that differ from the previous frame, but
// anything drawn straight onto the screen (the pause menu) is outside that
// comparison. Cells under it look unchanged to the game layer, so without this
// they are never repainted and the overlay lingers after it has closed, until
// the periodic full repaint happens to clear it.
func (a *App) invalidateFrame() {
	a.prevCells = nil
}

// openPause freezes the game and shows the menu.
func (a *App) openPause() {
	if !a.sm.Is(engine.StatePlaying) {
		return
	}
	_ = a.sm.Pause()
	a.menu = newPauseMenu()
	a.invalidateFrame()
	// Letting go of everything stops a held direction from carrying straight
	// through the menu and into the game on resume.
	a.input.ReleaseAll()
	a.heldAt = map[engine.Key]timeStamp{}
}

// resume closes the menu and unfreezes the game.
func (a *App) resume() {
	if a.sm.Is(engine.StatePaused) {
		_ = a.sm.Pause()
	}
	a.menu = nil
	a.invalidateFrame()
	a.input.ReleaseAll()
	a.heldAt = map[engine.Key]timeStamp{}
	// The clock kept no time while paused, but a stale remainder would still
	// grant a burst of steps on the first frame back.
	a.clock.Reset()
}

// restartRun starts a fresh run from the beginning.
func (a *App) restartRun() {
	a.menu = nil
	a.invalidateFrame()
	a.sm.ToTitle()
	a.startRun()
}

// toTitle abandons the run and returns to the title screen.
//
// A run abandoned with points on it still goes on the board, marked as not
// cleared: the score was earned, and quitting to the menu should not be a way
// of losing it.
func (a *App) toTitle() {
	rank := 0
	if !a.recorded {
		a.recorded = true
		rank = a.recordRun(false)
	}
	a.leaveRun(rank)
}

// handlePauseKey routes a key press while the menu is open.
//
// The pause menu reads raw terminal keys rather than the game's action map on
// purpose: the actions are tuned for play (Space is jump, Shift is run), and a
// menu wants arrows, Enter and Escape regardless of how play is bound.
func (a *App) handlePauseKey(e *tcell.EventKey) {
	if a.menu == nil {
		a.menu = newPauseMenu()
	}
	switch e.Key() {
	case tcell.KeyEscape:
		a.resume()
	case tcell.KeyUp:
		a.menu.move(-1)
	case tcell.KeyDown:
		a.menu.move(1)
	case tcell.KeyEnter:
		a.menu.activate(a)
	case tcell.KeyRune:
		switch e.Rune() {
		case 'p', 'P':
			a.resume()
		case 'q', 'Q':
			a.quit = true
		case 'r', 'R':
			a.restartRun()
		case 'k', 'K':
			a.menu.move(-1)
		case 'j', 'J':
			a.menu.move(1)
		case ' ':
			a.menu.activate(a)
		}
	}
}

// drawPauseMenu paints the menu over the frozen game.
//
// The last game frame is left on screen underneath, so the player can see
// exactly what they paused; the menu is a bordered panel on top of it.
func (a *App) drawPauseMenu(w, h int) {
	if a.menu == nil {
		return
	}
	title := "已暂停"
	hint := "↑↓ 选择 · Enter 确认 · Esc 继续"

	// Size the panel to its widest line, in display cells.
	inner := displayWidth([]rune(hint))
	for _, it := range a.menu.items {
		if n := displayWidth([]rune(it.label)) + menuMarkerW; n > inner {
			inner = n
		}
	}
	if n := displayWidth([]rune(title)); n > inner {
		inner = n
	}
	panelW := inner + 4
	panelH := len(a.menu.items) + 6
	if panelW > w {
		panelW = w
	}
	if panelH > h {
		panelH = h
	}
	x0 := (w - panelW) / 2
	y0 := (h - panelH) / 2

	panel := tcell.StyleDefault.Foreground(a.theme.HudText).Background(a.theme.Backdrop)
	border := tcell.StyleDefault.Foreground(a.theme.HudMuted).Background(a.theme.Backdrop)
	muted := tcell.StyleDefault.Foreground(a.theme.HudMuted).Background(a.theme.Backdrop)
	sel := tcell.StyleDefault.Foreground(a.theme.Sky).Background(a.theme.HudText)

	// Fill and frame.
	for y := 0; y < panelH; y++ {
		for x := 0; x < panelW; x++ {
			ch := ' '
			st := panel
			switch {
			case y == 0 && x == 0:
				ch, st = '┌', border
			case y == 0 && x == panelW-1:
				ch, st = '┐', border
			case y == panelH-1 && x == 0:
				ch, st = '└', border
			case y == panelH-1 && x == panelW-1:
				ch, st = '┘', border
			case y == 0 || y == panelH-1:
				ch, st = '─', border
			case x == 0 || x == panelW-1:
				ch, st = '│', border
			}
			a.screen.SetContent(x0+x, y0+y, ch, nil, st)
		}
	}

	writeAt(a.screen, x0, y0+1, panelW, title, panel, true)
	// Every entry shares one left edge. Centring each row on its own, with a
	// different prefix on the selected one, shifted that row sideways: the
	// marker glyph is ambiguous-width and some terminals draw it two cells
	// wide. A fixed marker column of plain ASCII removes the dependency on how
	// any single glyph is measured.
	labelW := 0
	for _, it := range a.menu.items {
		if n := displayWidth([]rune(it.label)); n > labelW {
			labelW = n
		}
	}
	blockW := menuMarkerW + labelW
	left := x0 + (panelW-blockW)/2
	for i, it := range a.menu.items {
		row := y0 + 3 + i
		if i == a.menu.cursor {
			fillRow(a.screen, x0+1, row, panelW-2, sel)
			writeRunes(a.screen, left, row, ">", sel)
			writeRunes(a.screen, left+menuMarkerW, row, it.label, sel)
		} else {
			writeRunes(a.screen, left+menuMarkerW, row, it.label, panel)
		}
	}
	writeAt(a.screen, x0, y0+panelH-2, panelW, hint, muted, true)
}

// menuMarkerW is the width of the selection marker column: the marker and one
// space. It is reserved on every row so selected and unselected labels start at
// the same cell.
const menuMarkerW = 2
