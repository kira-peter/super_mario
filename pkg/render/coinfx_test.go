package render

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/game"
	"super-mario/pkg/render/theme"
)

const tcellBold = tcell.AttrBold

// TestFlyingCoinIsDrawn checks a coin in flight shows up on screen.
func TestFlyingCoinIsDrawn(t *testing.T) {
	w := castWorld(t)
	w.CameraX, w.CameraY = 0, 0
	w.CoinFX = []*game.CoinFX{{Fly: game.CoinFlySteps / 2}}
	w.CoinFX[0].Pos.X = w.ViewW / 2
	w.CoinFX[0].Pos.Y = w.ViewH / 2
	c := (&Scene{T: theme.Ink}).Draw(w, 4)
	if countColour(c, theme.Ink.Coin) == 0 {
		t.Error("a coin halfway to the counter drew nothing")
	}
}

// TestCoinPlusPopsHoldsAndFades checks the "+1" shows, is boldest in the
// middle, and is gone at the end of its time.
func TestCoinPlusPopsHoldsAndFades(t *testing.T) {
	tk := theme.Ink
	if _, ok := CoinPlusStyle(tk, 0); !ok {
		t.Error("the \"+1\" does not show when the coin lands")
	}
	mid, ok := CoinPlusStyle(tk, game.CoinPlusSteps/3)
	if !ok {
		t.Fatal("the \"+1\" is gone too early")
	}
	if fg, _, attr := mid.Decompose(); fg != tk.Cast.CoinPlus || attr&tcellBold == 0 {
		t.Error("in the middle the \"+1\" should be full green and bold")
	}
	late, _ := CoinPlusStyle(tk, game.CoinPlusSteps-2)
	if fg, _, _ := late.Decompose(); fg == tk.Cast.CoinPlus {
		t.Error("near the end the \"+1\" has not faded")
	}
	if _, ok := CoinPlusStyle(tk, game.CoinPlusSteps); ok {
		t.Error("the \"+1\" is still showing after its time")
	}
}
