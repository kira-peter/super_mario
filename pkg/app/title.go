package app

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/engine"
	"super-mario/pkg/render"
	"super-mario/pkg/save"
)

// titlePage is which screen the title area is showing.
type titlePage int

const (
	// pageMenu is the main menu under the logo.
	pageMenu titlePage = iota
	// pageBoard is the leaderboard.
	pageBoard
	// pageHelp is the controls reference.
	pageHelp
)

// titleState is everything the title screen remembers.
type titleState struct {
	page   titlePage
	menu   *pauseMenu
	justAt int // rank of the run that was just recorded, 0 for none
}

// newTitleState builds the main menu.
func newTitleState(a *App) *titleState {
	return &titleState{
		page: pageMenu,
		menu: &pauseMenu{items: []menuItem{
			{"开始游戏", func(a *App) { a.startSelectedLevel() }},
			{"排行榜", func(a *App) { a.openTitlePage(pageBoard) }},
			{"操作说明", func(a *App) { a.openTitlePage(pageHelp) }},
			{"退出", func(a *App) { a.quit = true }},
		}},
	}
}

// startSelectedLevel begins a run on the chosen level.
//
// There is one level today, so this just starts it. It is a separate function
// so that a level-select screen only has to change what calls it: the menu
// entry, the key binding and the restart path all go through here.
func (a *App) startSelectedLevel() { a.startRun() }

// openTitlePage switches the title area to another page.
func (a *App) openTitlePage(p titlePage) {
	a.title.page = p
	a.invalidateFrame()
}

// backToMenu returns to the main menu page.
func (a *App) backToMenu() { a.openTitlePage(pageMenu) }

// handleTitleKey routes a key press on the title screen.
//
// Like the pause menu this reads raw terminal keys: the game's action map is
// tuned for play, and a menu wants arrows, Enter and Escape however play is
// bound.
func (a *App) handleTitleKey(e *tcell.EventKey) {
	if a.title == nil {
		a.title = newTitleState(a)
	}
	// Sub-pages only have one way out.
	if a.title.page != pageMenu {
		switch e.Key() {
		case tcell.KeyEscape, tcell.KeyEnter, tcell.KeyBackspace, tcell.KeyBackspace2:
			a.backToMenu()
		case tcell.KeyRune:
			if r := e.Rune(); r == ' ' || r == 'q' || r == 'Q' {
				a.backToMenu()
			}
		}
		return
	}

	m := a.title.menu
	switch e.Key() {
	case tcell.KeyUp:
		m.move(-1)
	case tcell.KeyDown:
		m.move(1)
	case tcell.KeyEnter:
		m.activate(a)
	case tcell.KeyEscape:
		a.quit = true
	case tcell.KeyRune:
		switch e.Rune() {
		case 'k', 'K':
			m.move(-1)
		case 'j', 'J':
			m.move(1)
		case ' ':
			m.activate(a)
		case 'q', 'Q':
			a.quit = true
		}
	}
}

// --- drawing ---------------------------------------------------------------

// minLogoCols is the narrowest terminal that fits the pixel logo. Below it the
// title falls back to plain spaced letters instead of overflowing.
const minLogoCols = 38

// drawTitleScreen paints whichever title page is active.
func (a *App) drawTitleScreen(w, h int) {
	if a.title == nil {
		a.title = newTitleState(a)
	}
	a.fillBackground(w, h)
	switch a.title.page {
	case pageBoard:
		a.drawBoard(w, h)
	case pageHelp:
		a.drawHelp(w, h)
	default:
		a.drawMainMenu(w, h)
	}
}

// fillBackground paints the sky colour so the title is not left on whatever
// the terminal's own background happens to be.
func (a *App) fillBackground(w, h int) {
	st := tcell.StyleDefault.Foreground(a.theme.HudText).Background(a.theme.Sky)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a.screen.SetContent(x, y, ' ', nil, st)
		}
	}
}

// drawMainMenu paints the logo and the menu beneath it.
func (a *App) drawMainMenu(w, h int) {
	sky := tcell.StyleDefault.Foreground(a.theme.HudText).Background(a.theme.Sky)
	muted := tcell.StyleDefault.Foreground(a.theme.HudMuted).Background(a.theme.Sky)
	sel := tcell.StyleDefault.Foreground(a.theme.Sky).Background(a.theme.HudText)

	logoRows := 0
	if w >= minLogoCols {
		top, bottom := LogoSprites()
		// Each terminal row holds two logo dots, so a 6-dot word is 3 rows.
		logoRows = (top.H+1)/2 + 1 + (bottom.H+1)/2
	} else {
		logoRows = 2
	}

	menuRows := len(a.title.menu.items)
	// logo, gap, subtitle, gap, menu
	total := logoRows + 1 + 1 + 2 + menuRows
	y := (h - total) / 2
	if y < 1 {
		y = 1
	}

	if w >= minLogoCols {
		top, bottom := LogoSprites()
		y = a.drawLogoWord(top, y, w) + 1
		y = a.drawLogoWord(bottom, y, w)
	} else {
		accent := tcell.StyleDefault.Foreground(a.theme.PlayerBody).Background(a.theme.Sky)
		writeCentered(a.screen, y, w, "S U P E R", accent)
		writeCentered(a.screen, y+1, w, "M A R I O", accent)
		y += 2
	}

	y += 1
	writeCentered(a.screen, y, w, "WORLD  1 - 1", muted)
	y += 3

	// The menu block: every entry shares one left edge, with a fixed marker
	// column in front, for the same reason the pause menu does (a selected
	// row with a different prefix drifts sideways).
	labelW := 0
	for _, it := range a.title.menu.items {
		if n := displayWidth([]rune(it.label)); n > labelW {
			labelW = n
		}
	}
	barW := labelW + menuMarkerW + 6
	left := (w - barW) / 2
	for i, it := range a.title.menu.items {
		row := y + i
		textX := left + 3 + menuMarkerW
		if i == a.title.menu.cursor {
			fillRow(a.screen, left, row, barW, sel)
			writeRunes(a.screen, left+3, row, ">", sel)
			writeRunes(a.screen, textX, row, it.label, sel)
		} else {
			writeRunes(a.screen, textX, row, it.label, sky)
		}
	}

	if h >= 6 {
		hint := "↑↓ 选择 · Enter 确认 · Esc 退出"
		writeCentered(a.screen, h-2, w, hint, muted)
	}
	if a.notice != "" && h >= 8 {
		writeCentered(a.screen, h-4, w, a.notice, muted)
	}
}

// drawLogoWord paints one logo word centred, two dots per terminal row, and
// returns the row after it.
func (a *App) drawLogoWord(sp render.Sprite, y, w int) int {
	rows := (sp.H + 1) / 2
	x0 := (w - sp.W) / 2
	if x0 < 0 {
		x0 = 0
	}
	for r := 0; r < rows; r++ {
		for x := 0; x < sp.W; x++ {
			topSlot := sp.Rows[r*2][x]
			var botSlot byte = '.'
			if r*2+1 < sp.H {
				botSlot = sp.Rows[r*2+1][x]
			}
			a.putHalfBlock(x0+x, y+r, topSlot, botSlot)
		}
	}
	return y + rows
}

// putHalfBlock draws one terminal cell holding two stacked dots.
func (a *App) putHalfBlock(x, y int, top, bot byte) {
	tc, topOK := render.SlotColor(a.theme, top)
	bc, botOK := render.SlotColor(a.theme, bot)
	base := a.theme.Sky
	switch {
	case !topOK && !botOK:
		return
	case topOK && botOK && tc == bc:
		a.screen.SetContent(x, y, ' ', nil, tcell.StyleDefault.Background(tc.(tcell.Color)))
	case topOK && botOK:
		a.screen.SetContent(x, y, '▀', nil,
			tcell.StyleDefault.Foreground(tc.(tcell.Color)).Background(bc.(tcell.Color)))
	case topOK:
		a.screen.SetContent(x, y, '▀', nil,
			tcell.StyleDefault.Foreground(tc.(tcell.Color)).Background(base))
	default:
		a.screen.SetContent(x, y, '▄', nil,
			tcell.StyleDefault.Foreground(bc.(tcell.Color)).Background(base))
	}
}

// drawBoard paints the leaderboard.
func (a *App) drawBoard(w, h int) {
	sky := tcell.StyleDefault.Foreground(a.theme.HudText).Background(a.theme.Sky)
	muted := tcell.StyleDefault.Foreground(a.theme.HudMuted).Background(a.theme.Sky)
	accent := tcell.StyleDefault.Foreground(a.theme.PlayerBody).Background(a.theme.Sky)
	gold := tcell.StyleDefault.Foreground(a.theme.Question).Background(a.theme.Sky)
	hot := tcell.StyleDefault.Foreground(a.theme.Sky).Background(a.theme.HudText)

	entries := a.board.Entries
	rowsNeeded := len(entries)
	if rowsNeeded == 0 {
		rowsNeeded = 1
	}
	total := 4 + rowsNeeded + 2
	y := (h - total) / 2
	if y < 1 {
		y = 1
	}

	writeCentered(a.screen, y, w, "排 行 榜", accent)
	y += 2

	// Columns: rank, score, time, date. Widths are fixed so every row lines
	// up regardless of the digits in it.
	const (
		cRank  = 4
		cScore = 9
		cTime  = 10
		cDate  = 12
	)
	tableW := cRank + cScore + cTime + cDate + 3 + 7
	left := (w - tableW) / 2
	if left < 0 {
		left = 0
	}
	cols := []struct {
		x     int
		title string
	}{
		{left, "名次"},
		{left + cRank + 1, "分数"},
		{left + cRank + cScore + 2, "用时"},
		{left + cRank + cScore + cTime + 3, "日期"},
	}
	for _, c := range cols {
		writeRunes(a.screen, c.x, y, c.title, muted)
	}
	y++

	if len(entries) == 0 {
		writeCentered(a.screen, y+1, w, "还没有成绩，去通关一次吧", muted)
		y += 3
	}
	for i, e := range entries {
		st := sky
		switch i {
		case 0:
			st = gold
		}
		if a.title != nil && a.title.justAt == i+1 {
			st = hot
			fillRow(a.screen, left-1, y+i, tableW+2, hot)
		}
		writeRunes(a.screen, cols[0].x, y+i, fmt.Sprintf("%2d", i+1), st)
		writeRunes(a.screen, cols[1].x, y+i, fmt.Sprintf("%06d", e.Score), st)
		writeRunes(a.screen, cols[2].x, y+i, boardTime(e), st)
		writeRunes(a.screen, cols[3].x, y+i, boardDate(e), st)
		if !e.Cleared && e.Steps > 0 {
			writeRunes(a.screen, cols[3].x+cDate+1, y+i, "未通关", muted)
		}
	}

	writeCentered(a.screen, h-2, w, "Esc 返回", muted)
}

// boardTime renders an entry's run time, or a dash for an entry migrated from
// an old save that never recorded one.
func boardTime(e save.Entry) string {
	if e.Steps <= 0 {
		return "  --"
	}
	return save.FormatDuration(e.Steps)
}

// boardDate renders when an entry was set, in the player's own time zone.
func boardDate(e save.Entry) string {
	if e.At.IsZero() {
		return "--"
	}
	return e.At.In(time.Local).Format("01-02 15:04")
}

// drawHelp paints the controls reference.
func (a *App) drawHelp(w, h int) {
	sky := tcell.StyleDefault.Foreground(a.theme.HudText).Background(a.theme.Sky)
	muted := tcell.StyleDefault.Foreground(a.theme.HudMuted).Background(a.theme.Sky)
	accent := tcell.StyleDefault.Foreground(a.theme.PlayerBody).Background(a.theme.Sky)

	rows := [][2]string{
		{"← →", "移动"},
		{"空格 / Z / ↑", "跳跃（长按跳得更高）"},
		{"空格 再按一次", "空中二次跳跃"},
		{"X / Shift", "加速跑"},
		{"Esc / P", "暂停菜单"},
		{"Ctrl-C", "任何时候退出"},
	}
	total := 3 + len(rows) + 3
	y := (h - total) / 2
	if y < 1 {
		y = 1
	}
	writeCentered(a.screen, y, w, "操 作 说 明", accent)
	y += 2

	keyW := 0
	for _, r := range rows {
		if n := displayWidth([]rune(r[0])); n > keyW {
			keyW = n
		}
	}
	left := (w - (keyW + 3 + 20)) / 2
	if left < 0 {
		left = 0
	}
	for i, r := range rows {
		writeRunes(a.screen, left, y+i, r[0], sky)
		writeRunes(a.screen, left+keyW+3, y+i, r[1], muted)
	}
	y += len(rows) + 1
	writeCentered(a.screen, y, w, "部分终端会吞掉 Shift 与方向键的组合，用 X 代替即可", muted)

	writeCentered(a.screen, h-2, w, "Esc 返回", muted)
}

// recordRun adds a finished run to the leaderboard and saves it.
//
// Every run is recorded, cleared or not: a score reached before running out of
// lives is still a score. It returns the rank, or 0 if the run did not place.
//
// The entry the run displaced is remembered before inserting, because afterwards
// the run is itself on the board and "who was ahead of me" can no longer be
// answered from it.
func (a *App) recordRun(cleared bool) int {
	score := a.world.Score.Points
	if score <= 0 {
		return 0
	}
	prevBest, hadBest := a.board.Best()

	e := save.NewEntry(score, a.world.RunSteps(), a.lv.World, cleared)
	rank := a.board.Add(e)
	a.store.Save(a.board)

	a.resultSteps = e.Steps
	a.resultPrevBest, a.resultHadPrev = prevBest, hadBest
	return rank
}

// leaveRun returns to the title screen after a run, landing on the board when
// the run placed so the player sees where they stand.
func (a *App) leaveRun(rank int) {
	a.sm.ToTitle()
	a.menu = nil
	a.input.ReleaseAll()
	a.heldAt = map[engine.Key]timeStamp{}
	a.title = newTitleState(a)
	if rank > 0 {
		a.title.page = pageBoard
		a.title.justAt = rank
	}
	a.invalidateFrame()
}
