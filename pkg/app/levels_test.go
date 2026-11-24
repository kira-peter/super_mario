package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/catalog"
	"super-mario/pkg/engine"
	"super-mario/pkg/level"
	"super-mario/pkg/save"
)

// testCatalog is a catalog of the compiled-in levels plus an empty local
// folder, so tests never read the developer's own levels.
func testCatalog(t *testing.T) *catalog.Catalog {
	t.Helper()
	return &catalog.Catalog{Embedded: level.Embedded(), LocalDir: filepath.Join(t.TempDir(), "levels")}
}

// localLevel writes a copy of 1-1 into the app's local folder under id, with
// a title.
func localLevel(t *testing.T, a *App, id, title string) {
	t.Helper()
	data, _ := os.ReadFile(filepath.Join("..", "level", "data", "1-1.json"))
	// 1-1 has a meta line of its own; replace it with one naming this copy.
	lines := strings.Split(string(data), "\n")
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), `"meta":`) {
			lines[i] = `  "meta": {"title": "` + title + `"},`
		}
	}
	data = []byte(strings.Join(lines, "\n"))
	if _, err := a.catalog.Write(catalog.Local, id, data); err != nil {
		t.Fatal(err)
	}
}

func TestLevelsSortOfficialFirstThenCommunityByTitle(t *testing.T) {
	l := []levelEntry{
		{Source: catalog.Local, ID: "c-loc-aa", Title: "aaa"},
		{ID: "c-amy-zz", Title: "Zebra"},
		{ID: "1-10", Official: true},
		{ID: "c-tom-aa", Title: "apple"},
		{ID: "1-2", Official: true},
		{ID: "2-1", Official: true},
		{ID: "1-1", Official: true},
		{ID: "c-bob-mm"}, // no title: sorts by its id
	}
	for i := range l {
		if l[i].Source == "" {
			l[i].Source = catalog.Builtin
		}
	}
	sortLevels(l)
	var got []string
	for _, e := range l {
		got = append(got, e.ID)
	}
	want := "1-1 1-2 1-10 2-1 c-tom-aa c-bob-mm c-amy-zz c-loc-aa"
	if strings.Join(got, " ") != want {
		t.Errorf("order = %v, want %s", got, want)
	}
}

func TestMissingMetaShowsNoPlaceholder(t *testing.T) {
	a := headlessTitleApp(t)
	rows := levelRows([]levelEntry{{Source: catalog.Builtin, ID: "c-bob-plain"}, {Source: catalog.Builtin, ID: "c-amy-x", Title: "Hills", Author: "amy", Difficulty: 2}}, a)
	if strings.TrimSpace(rows[0].text) != "c-bob-plain" {
		t.Errorf("a level with no meta reads %q; want just its id", rows[0].text)
	}
	if !strings.Contains(rows[1].text, "amy") || !strings.Contains(rows[1].text, "★★☆☆☆") {
		t.Errorf("author and difficulty missing: %q", rows[1].text)
	}
	for _, bad := range []string{"未知", "无", "--", "N/A"} {
		if strings.Contains(rows[0].text, bad) {
			t.Errorf("placeholder %q shown for a missing field", bad)
		}
	}
}

func TestLevelListShowsTheBuiltInLevels(t *testing.T) {
	a := headlessTitleApp(t)
	list := a.levelList()
	if len(list) == 0 || list[0].ID != "1-1" || !list[0].Official {
		t.Fatalf("list = %+v", list)
	}
}

// openSelect drives the title menu to 选择关卡 and opens it.
func openSelect(t *testing.T, a *App) {
	t.Helper()
	for a.title.menu.items[a.title.menu.cursor].label != "选择关卡" {
		a.handleEvent(keyEvent(tcell.KeyDown, 0))
	}
	a.handleEvent(keyEvent(tcell.KeyEnter, 0))
	if a.title.page != pageLevels {
		t.Fatalf("page = %v after 选择关卡", a.title.page)
	}
}

func TestSelectEntersAndEscReturns(t *testing.T) {
	a := headlessTitleApp(t)
	openSelect(t, a)
	rows := renderTitleText(t, a, 100, 30)
	// An official level with a title is listed by its title.
	if text := strings.Join(rows, "\n"); !strings.Contains(text, "1-1") && !strings.Contains(strings.ReplaceAll(text, " ", ""), "草原初探") {
		t.Error("the list does not show 1-1")
	}
	a.handleEvent(keyEvent(tcell.KeyEscape, 0))
	if a.title.page != pageMenu || a.quit {
		t.Errorf("Esc: page %v, quit %v; want back on the menu", a.title.page, a.quit)
	}
}

func TestEnterStartsTheHighlightedLevel(t *testing.T) {
	a := headlessTitleApp(t)
	openSelect(t, a)
	a.handleEvent(keyEvent(tcell.KeyEnter, 0))
	if !a.sm.Is(engine.StatePlaying) {
		t.Fatalf("state = %v", a.sm.Current())
	}
	if a.lv.Name != "1-1" || !a.fromSelect {
		t.Errorf("level %s, fromSelect %v", a.lv.Name, a.fromSelect)
	}
}

// TestRunFromSelectReturnsToTheList covers the three ways a run ends: the
// results screen, and quitting to the title from the pause menu.
func TestRunFromSelectReturnsToTheList(t *testing.T) {
	a := headlessTitleApp(t)
	openSelect(t, a)
	a.handleEvent(keyEvent(tcell.KeyEnter, 0))
	a.leaveRun(0)
	if a.title.page != pageLevels {
		t.Fatalf("after the run: page %v, want the level list", a.title.page)
	}
	if got := a.title.levels.list[a.title.levels.cursor].ID; got != "1-1" {
		t.Errorf("highlighted %s, want the level just played", got)
	}

	// From the pause menu's 回到标题 as well.
	a.handleEvent(keyEvent(tcell.KeyEnter, 0))
	a.world.Score.Points = 100
	a.toTitle()
	if a.title.page != pageLevels {
		t.Errorf("after 回到标题: page %v, want the level list", a.title.page)
	}
}

func TestStartGameStillStartsOneOne(t *testing.T) {
	a := headlessTitleApp(t)
	a.handleEvent(keyEvent(tcell.KeyEnter, 0))
	if a.lv.Name != "1-1" || a.fromSelect {
		t.Errorf("开始游戏 started %s (fromSelect %v)", a.lv.Name, a.fromSelect)
	}
	// And a run started this way does not end on the list.
	a.leaveRun(0)
	if a.title.page == pageLevels {
		t.Error("a run from 开始游戏 ended on the level list")
	}
}

func TestListShowsEachLevelsBest(t *testing.T) {
	a := headlessTitleApp(t)
	a.board.Add(save.Entry{Score: 12000, Level: "1-1", Steps: 600})
	a.board.Add(save.Entry{Score: 99000, Level: "c-other-level", Steps: 600})
	openSelect(t, a)
	text := strings.Join(renderTitleText(t, a, 110, 30), "\n")
	if !strings.Contains(text, "012000") {
		t.Errorf("1-1's best is not shown:\n%s", text)
	}
	if strings.Contains(text, "099000") {
		t.Error("another level's score shows against 1-1")
	}
}

// TestRunsAreRecordedAgainstTheirLevel: the board entry names the level by
// its id, which is what the list looks bests up by.
func TestRunsAreRecordedAgainstTheirLevel(t *testing.T) {
	a := headlessTitleApp(t)
	a.store = save.NewAt(t.TempDir())
	a.handleEvent(keyEvent(tcell.KeyEnter, 0))
	a.world.Score.Points = 3400
	a.recordRun(true)
	if e, ok := a.board.BestFor("1-1"); !ok || e.Score != 3400 {
		t.Errorf("best for 1-1 = %+v %v", e, ok)
	}
}

// TestLocalLevelsAppearWithoutRebuilding: a file dropped into levels/ shows in
// the list, marked local, and plays.
func TestLocalLevelsAppearWithoutRebuilding(t *testing.T) {
	a := headlessTitleApp(t)
	localLevel(t, a, "c-amy-hills", "Hills")
	openSelect(t, a)
	var found bool
	for i, e := range a.title.levels.list {
		if e.ID == "c-amy-hills" && e.Source == catalog.Local {
			found = true
			a.title.levels.cursor = i
		}
	}
	if !found {
		t.Fatalf("the local level is not listed: %+v", a.title.levels.list)
	}
	text := strings.Join(renderTitleText(t, a, 110, 30), "\n")
	// Wide characters take two cells, so the screen text has a space after
	// each; compare without them.
	if !strings.Contains(strings.ReplaceAll(text, " ", ""), "[本地]") {
		t.Errorf("the local level is not marked:\n%s", text)
	}
	a.handleEvent(keyEvent(tcell.KeyEnter, 0))
	if a.levelKey != "local:c-amy-hills" || !a.sm.Is(engine.StatePlaying) {
		t.Errorf("started %s, state %v", a.levelKey, a.sm.Current())
	}
}

// TestSameNameBothListedAndScoredApart: a local 1-1 sits beside the built-in
// one, and their scores do not mix.
func TestSameNameBothListedAndScoredApart(t *testing.T) {
	a := headlessTitleApp(t)
	localLevel(t, a, "1-1", "My 1-1")
	list := a.levelList()
	n := 0
	for _, e := range list {
		if e.ID == "1-1" {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("1-1 listed %d times: %+v", n, list)
	}
	if err := a.switchLevel(catalog.Local, "1-1"); err != nil {
		t.Fatal(err)
	}
	a.startRun()
	a.world.Score.Points = 777
	a.recordRun(true)
	if _, ok := a.board.BestFor("1-1"); ok {
		t.Error("the local copy's score went to the built-in 1-1")
	}
	if e, ok := a.board.BestFor("local:1-1"); !ok || e.Score != 777 {
		t.Errorf("local best = %+v %v", e, ok)
	}
}

func TestBrokenLocalFileIsSkipped(t *testing.T) {
	a := headlessTitleApp(t)
	builtin := len(a.levelList())
	os.MkdirAll(a.catalog.LocalDir, 0o755)
	os.WriteFile(filepath.Join(a.catalog.LocalDir, "c-bad-file.json"), []byte("{oops"), 0o644)
	localLevel(t, a, "c-good-one", "Good")
	for _, e := range a.levelList() {
		if e.ID == "c-bad-file" {
			t.Error("a broken file was listed")
		}
	}
	if got := len(a.levelList()); got != builtin+1 {
		t.Errorf("list has %d levels, want the %d built-in plus the good local one", got, builtin)
	}
}

// TestFreshlySavedLevelPlaysAsSaved: the level is read from disk each time it
// is chosen, so an edit saved in the editor plays without a restart.
func TestFreshlySavedLevelPlaysAsSaved(t *testing.T) {
	a := headlessTitleApp(t)
	localLevel(t, a, "c-amy-hills", "First")
	a.switchLevel(catalog.Local, "c-amy-hills")
	localLevel(t, a, "c-amy-hills", "Second")
	a.switchLevel(catalog.Local, "c-amy-hills")
	if a.lv.Meta.Title != "Second" {
		t.Errorf("title = %q; the old copy was kept", a.lv.Meta.Title)
	}
}
