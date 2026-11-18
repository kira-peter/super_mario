package app

import (
	"super-mario/pkg/editor"
)

// The level editor, started from the title menu (spec level-editor /
// 从游戏内进入编辑器): it serves in the background on this same process, so
// the game stays usable, and is shut down when the game exits.

// openEditor starts the editor if it is not running, opens it in the browser
// and shows its address on the title screen.
func (a *App) openEditor() {
	if !editor.HasWasm() {
		a.notice = "这个程序缺少编辑器模块，请用 make build 重新编译"
		return
	}
	if a.editor == nil {
		svc, err := editor.Start(editor.Config{Catalog: a.catalog})
		if err != nil {
			a.notice = "编辑器启动失败：" + err.Error()
			return
		}
		a.editor = svc
	}
	url := a.editor.URL
	open := a.openURL
	if open == nil {
		open = editor.OpenBrowser
	}
	if err := open(url); err != nil {
		a.notice = "请在浏览器打开 " + url
		return
	}
	a.notice = "编辑器已在浏览器打开：" + url
}

// closeEditor stops the editor, if it was started.
func (a *App) closeEditor() {
	if a.editor != nil {
		a.editor.Stop()
		a.editor = nil
	}
}
