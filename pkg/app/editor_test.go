package app

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/editor"
)

// pickMenu moves the title cursor to an entry and activates it.
func pickMenu(t *testing.T, a *App, label string) {
	t.Helper()
	for i := 0; a.title.menu.items[a.title.menu.cursor].label != label; i++ {
		if i > len(a.title.menu.items) {
			t.Fatalf("no menu entry %q", label)
		}
		a.handleEvent(keyEvent(tcell.KeyDown, 0))
	}
	a.handleEvent(keyEvent(tcell.KeyEnter, 0))
}

func TestDesignEntryStartsTheEditorInTheBackground(t *testing.T) {
	if !editor.HasWasm() {
		t.Skip("built without the editor module (run make build)")
	}
	a := headlessTitleApp(t)
	var opened []string
	a.openURL = func(u string) error { opened = append(opened, u); return nil }
	defer a.closeEditor()

	pickMenu(t, a, "设计关卡")
	if len(opened) != 1 || !strings.HasPrefix(opened[0], "http://127.0.0.1:") {
		t.Fatalf("browser opened %v", opened)
	}
	// The game is still on the title screen and the address is shown.
	if a.title.page != pageMenu || a.quit {
		t.Error("choosing 设计关卡 left the title screen")
	}
	if !strings.Contains(a.notice, opened[0]) {
		t.Errorf("notice %q does not give the address", a.notice)
	}
	resp, err := http.Get(opened[0] + "api/info")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("editor answered %d", resp.StatusCode)
	}

	// Choosing it again reuses the running editor.
	a.handleEvent(keyEvent(tcell.KeyEnter, 0))
	if len(opened) != 2 || opened[1] != opened[0] {
		t.Errorf("second open %v, want the same address", opened)
	}
}

func TestDesignEntryShowsTheAddressWhenNoBrowser(t *testing.T) {
	if !editor.HasWasm() {
		t.Skip("built without the editor module (run make build)")
	}
	a := headlessTitleApp(t)
	a.openURL = func(string) error { return errors.New("no browser") }
	defer a.closeEditor()
	pickMenu(t, a, "设计关卡")
	if !strings.Contains(a.notice, "http://127.0.0.1:") {
		t.Errorf("notice %q should tell the address to open", a.notice)
	}
}

func TestDesignEntryWithoutTheModuleSaysWhy(t *testing.T) {
	if editor.HasWasm() {
		t.Skip("this binary has the editor module")
	}
	a := headlessTitleApp(t)
	pickMenu(t, a, "设计关卡")
	if !strings.Contains(a.notice, "make build") {
		t.Errorf("notice %q", a.notice)
	}
}
