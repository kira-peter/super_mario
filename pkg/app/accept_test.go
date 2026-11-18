package app

import (
	"os"
	"testing"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/catalog"
	"super-mario/pkg/engine"
	"super-mario/pkg/game"
	"super-mario/pkg/save"
)

// TestAcceptanceLevelPlaysThroughSelect is the terminal half of acceptance
// check 9.1: a level saved by the editor and compiled in appears in the level
// list, starts from it, and can be played to the flag in the real app.
//
// It runs only when ACCEPT_LEVEL names the level, since the level exists only
// in the scratch worktree the acceptance script builds.
func TestAcceptanceLevelPlaysThroughSelect(t *testing.T) {
	id := os.Getenv("ACCEPT_LEVEL")
	if id == "" {
		t.Skip("set ACCEPT_LEVEL to run the editor acceptance check")
	}
	a := headlessTitleApp(t)
	a.store = save.NewAt(t.TempDir())
	// The acceptance level is read where the editor saved it.
	a.catalog = catalog.Default()

	// Open the list and move to the level by its row.
	for a.title.menu.items[a.title.menu.cursor].label != "选择关卡" {
		a.handleEvent(keyEvent(tcell.KeyDown, 0))
	}
	a.handleEvent(keyEvent(tcell.KeyEnter, 0))
	found := false
	for i := 0; i < len(a.title.levels.list); i++ {
		if a.title.levels.list[a.title.levels.cursor].ID == id {
			found = true
			break
		}
		a.handleEvent(keyEvent(tcell.KeyDown, 0))
	}
	if !found {
		t.Fatalf("%s is not in the level list", id)
	}
	text := ""
	for _, r := range renderTitleText(t, a, 110, 30) {
		text += r + "\n"
	}
	t.Logf("level list:\n%s", text)

	a.handleEvent(keyEvent(tcell.KeyEnter, 0))
	if a.lv.Name != id || !a.sm.Is(engine.StatePlaying) {
		t.Fatalf("started %s in state %v", a.levelKey, a.sm.Current())
	}
	if a.lv.Style.Theme == "" {
		t.Error("the level's style did not come through")
	}

	// Play: run right, jumping in rhythm, invincible so enemies and the
	// random block's draw cannot end the run before the flag. This checks the
	// level is completable in the real app, not a player's skill.
	a.world.SetPlaytest(game.PlaytestOptions{Invincible: true})
	for i := 0; i < 6000 && !a.world.Cleared; i++ {
		a.input.KeyDown(engine.KeyRight)
		a.input.KeyDown(engine.KeyShift)
		if i%40 < 18 {
			a.input.KeyDown(engine.KeySpace)
		} else {
			a.input.KeyUp(engine.KeySpace)
		}
		a.stepOnce()
	}
	if !a.world.Cleared {
		t.Fatalf("did not reach the flag; stopped at column %d", int(a.world.Player.CenterX()/engine.Tile))
	}
	t.Logf("cleared %s with %d points", id, a.world.Score.Points)
}
