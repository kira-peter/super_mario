package app

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/catalog"
	"super-mario/pkg/level"
	"super-mario/pkg/render"
)

// Level select (spec level-select): built-in levels, then the local ones
// beside the program. Within each, official levels come first in world order,
// then community ones by title. A built-in and a local level with the same id
// are both listed.

// levelEntry is one row of the list.
type levelEntry struct {
	Source     catalog.Source
	ID         string
	Title      string
	Author     string
	Difficulty int
	Desc       string
	Official   bool
}

// key names the level across both sources, as the leaderboard records it.
func (e levelEntry) key() string { return catalog.Key(e.Source, e.ID) }

// levelList reads both level folders for the select page. A file that fails
// to load is left out, so one bad file someone dropped in levels/ never stops
// the game (spec level-select / 坏文件不影响).
func (a *App) levelList() []levelEntry {
	var out []levelEntry
	for _, e := range a.catalog.List() {
		if e.Err != "" {
			continue
		}
		out = append(out, levelEntry{Source: e.Source, ID: e.ID, Title: e.Title, Author: e.Author,
			Difficulty: e.Difficulty, Desc: e.Description, Official: e.Official})
	}
	return out
}

// sortLevels orders a list the way the select page shows it: built-in before
// local, then official levels by world and stage as numbers (so 1-10 comes
// after 1-9), then community levels by title.
func sortLevels(l []levelEntry) {
	sort.SliceStable(l, func(i, j int) bool {
		a, b := l[i], l[j]
		if a.Source != b.Source {
			return a.Source == catalog.Builtin
		}
		if a.Official != b.Official {
			return a.Official
		}
		if a.Official {
			aw, as := worldStage(a.ID)
			bw, bs := worldStage(b.ID)
			if aw != bw {
				return aw < bw
			}
			return as < bs
		}
		return strings.ToLower(a.label()) < strings.ToLower(b.label())
	})
}

var officialID = regexp.MustCompile(`^(\d+)-(\d+)$`)

func worldStage(id string) (int, int) {
	m := officialID.FindStringSubmatch(id)
	if m == nil {
		return 0, 0
	}
	w, _ := strconv.Atoi(m[1])
	s, _ := strconv.Atoi(m[2])
	return w, s
}

// label is what the list shows as the name: the title, or the id when the
// level has none.
func (e levelEntry) label() string {
	if e.Title != "" {
		return e.Title
	}
	return e.ID
}

// levelsState is the select page's own state.
type levelsState struct {
	list   []levelEntry
	cursor int
}

// openLevelSelect shows the list, with the cursor on the current level.
func (a *App) openLevelSelect() {
	ls := &levelsState{list: a.levelList()}
	for i, e := range ls.list {
		if e.key() == a.levelKey {
			ls.cursor = i
		}
	}
	a.title.levels = ls
	a.openTitlePage(pageLevels)
}

// handleLevelsKey moves through the list and starts the chosen level.
func (a *App) handleLevelsKey(e *tcell.EventKey) {
	ls := a.title.levels
	n := len(ls.list)
	move := func(d int) {
		if n > 0 {
			ls.cursor = ((ls.cursor+d)%n + n) % n
		}
	}
	switch e.Key() {
	case tcell.KeyUp:
		move(-1)
	case tcell.KeyDown:
		move(1)
	case tcell.KeyEnter:
		a.startFromSelect()
	case tcell.KeyEscape, tcell.KeyBackspace, tcell.KeyBackspace2:
		a.backToMenu()
	case tcell.KeyRune:
		switch e.Rune() {
		case 'k', 'K':
			move(-1)
		case 'j', 'J':
			move(1)
		case ' ':
			a.startFromSelect()
		case 'q', 'Q':
			a.backToMenu()
		}
	}
}

// startFromSelect starts the highlighted level and remembers that the run
// came from the list, so it ends back there.
func (a *App) startFromSelect() {
	ls := a.title.levels
	if ls == nil || len(ls.list) == 0 {
		return
	}
	e := ls.list[ls.cursor]
	if err := a.switchLevel(e.Source, e.ID); err != nil {
		a.notice = "无法载入关卡 " + e.ID + "：" + err.Error()
		return
	}
	a.fromSelect = true
	a.startRun()
}

// switchLevel makes a level the current one: a new world on it, and a scene
// in its colours. It is read afresh each time, so a level the editor has just
// saved plays as saved.
func (a *App) switchLevel(src catalog.Source, id string) error {
	lv, err := a.catalog.Load(src, id)
	if err != nil {
		return err
	}
	a.levelKey = catalog.Key(src, id)
	a.useLevel(lv)
	return nil
}

// useLevel rebuilds everything that depends on the level.
func (a *App) useLevel(lv *level.Level) {
	a.lv = lv
	tuning := *a.world.T
	vw, vh := a.world.ViewW, a.world.ViewH
	a.world = newWorldFor(lv, &tuning, a)
	a.world.SetViewport(vw, vh)
	sky := a.scene.Sky
	a.scene = render.ForLevel(a.theme, a.opts.ThemeName, lv, a.capacity)
	a.scene.Sky = sky
	a.prevCells = nil
}

// drawLevels paints the select page.
func (a *App) drawLevels(w, h int) {
	sky := tcell.StyleDefault.Foreground(a.theme.HudText).Background(a.theme.Sky)
	muted := tcell.StyleDefault.Foreground(a.theme.HudMuted).Background(a.theme.Sky)
	accent := tcell.StyleDefault.Foreground(a.theme.PlayerBody).Background(a.theme.Sky)
	sel := tcell.StyleDefault.Foreground(a.theme.Sky).Background(a.theme.HudText)
	gold := tcell.StyleDefault.Foreground(a.theme.Question).Background(a.theme.Sky)

	ls := a.title.levels
	rows := levelRows(ls.list, a)

	// Title, gap, list (scrolling if it does not fit), gap, description, hint.
	listH := h - 9
	if listH < 3 {
		listH = 3
	}
	top := 0
	if ls.cursor >= listH {
		top = ls.cursor - listH + 1
	}
	shown := len(rows) - top
	if shown > listH {
		shown = listH
	}
	y := (h - (shown + 7)) / 2
	if y < 1 {
		y = 1
	}
	writeCentered(a.screen, y, w, "选 择 关 卡", accent)
	y += 2

	width := 0
	for _, r := range rows {
		if n := displayWidth([]rune(r.text)); n > width {
			width = n
		}
	}
	barW := width + menuMarkerW + 6
	left := (w - barW) / 2
	if left < 0 {
		left = 0
	}
	if len(rows) == 0 {
		writeCentered(a.screen, y, w, "没有可用的关卡", muted)
	}
	for i := 0; i < shown; i++ {
		idx := top + i
		r := rows[idx]
		row := y + i
		if idx == ls.cursor {
			fillRow(a.screen, left, row, barW, sel)
			writeRunes(a.screen, left+3, row, ">", sel)
			writeRunes(a.screen, left+3+menuMarkerW, row, r.text, sel)
			continue
		}
		writeRunes(a.screen, left+3+menuMarkerW, row, r.name, sky)
		writeRunes(a.screen, left+3+menuMarkerW+displayWidth([]rune(r.name)), row, r.rest, muted)
		if r.best != "" {
			bx := left + 3 + menuMarkerW + width - displayWidth([]rune(r.best))
			writeRunes(a.screen, bx, row, r.best, gold)
		}
	}
	y += shown + 1

	// The highlighted level's description, under the list.
	if ls.cursor < len(ls.list) {
		if d := ls.list[ls.cursor].Desc; d != "" {
			writeCentered(a.screen, y, w, d, muted)
		}
	}
	writeCentered(a.screen, h-2, w, "↑↓ 选择 · Enter 开始 · Esc 返回", muted)
}

// levelRow is one list row laid out: the name, the rest (author, difficulty)
// and the best score, plus the three run together for the highlighted row.
type levelRow struct {
	name, rest, best, text string
}

// levelRows lays the list out with aligned columns. Missing meta shows as
// nothing, never as a placeholder (spec level-select / 元信息缺省).
func levelRows(list []levelEntry, a *App) []levelRow {
	nameW := 0
	for _, e := range list {
		if n := displayWidth([]rune(e.label())); n > nameW {
			nameW = n
		}
	}
	out := make([]levelRow, len(list))
	for i, e := range list {
		name := e.label() + strings.Repeat(" ", nameW-displayWidth([]rune(e.label())))
		var rest []string
		if e.Source == catalog.Local {
			rest = append(rest, "[本地]")
		}
		if e.Author != "" {
			rest = append(rest, e.Author)
		}
		if e.Difficulty > 0 {
			rest = append(rest, strings.Repeat("★", e.Difficulty)+strings.Repeat("☆", level.MaxDifficulty-e.Difficulty))
		}
		r := levelRow{name: name}
		if len(rest) > 0 {
			r.rest = "  " + strings.Join(rest, "  ")
		}
		if best, ok := a.board.BestFor(e.key()); ok {
			r.best = fmt.Sprintf("  最高 %06d", best.Score)
		}
		r.text = r.name + r.rest + r.best
		out[i] = r
	}
	// Pad every row's text to the widest, so the best scores line up.
	width := 0
	for _, r := range out {
		if n := displayWidth([]rune(r.name + r.rest)); n > width {
			width = n
		}
	}
	for i := range out {
		pad := width - displayWidth([]rune(out[i].name+out[i].rest))
		out[i].text = out[i].name + out[i].rest + strings.Repeat(" ", pad) + out[i].best
	}
	return out
}
