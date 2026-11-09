//go:build windows

package app

import (
	"syscall"

	"super-mario/pkg/engine"
)

var procGetAsyncKeyState = syscall.NewLazyDLL("user32.dll").NewProc("GetAsyncKeyState")

// virtualKeys maps the game's keys to Windows virtual-key codes.
var virtualKeys = map[engine.Key]uintptr{
	engine.KeyLeft:   0x25,
	engine.KeyUp:     0x26,
	engine.KeyRight:  0x27,
	engine.KeyDown:   0x28,
	engine.KeySpace:  0x20,
	engine.KeyEnter:  0x0D,
	engine.KeyEscape: 0x1B,
	engine.KeyTab:    0x09,
	engine.KeyP:      0x50,
	engine.KeyQ:      0x51,
	engine.KeyR:      0x52,
	engine.KeyX:      0x58,
	engine.KeyZ:      0x5A,
}

// platformProbe asks Windows which keys are really down.
//
// A console program only receives key-press events, never key-up, so without
// this the game has to guess when a key is let go. GetAsyncKeyState reads the
// true state. It is system-wide, so the game only uses it to confirm keys it
// has already seen pressed in its own window; it never starts an action from a
// key pressed elsewhere.
func platformProbe() keyProbe {
	if procGetAsyncKeyState.Find() != nil {
		return nil
	}
	return func(k engine.Key) (bool, bool) {
		code, ok := virtualKeys[k]
		if !ok {
			return false, false
		}
		r, _, _ := procGetAsyncKeyState.Call(code)
		// The high bit of the returned SHORT is "currently down".
		return uint16(r)&0x8000 != 0, true
	}
}
