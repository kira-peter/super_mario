package render

import (
	"testing"

	"super-mario/pkg/engine"
	"super-mario/pkg/game"
	"super-mario/pkg/level"
	"super-mario/pkg/render/theme"
)

func BenchmarkFrameTiny(b *testing.B) {
	lv, _ := level.LoadEmbedded("1-1")
	sm := engine.NewStateMachine()
	sm.StartRun()
	w := game.NewWorld(lv, &game.Default, sm, 3)
	w.SetViewport(engine.Tile.Mul(30), engine.Tile.Mul(19))
	w.CameraY = engine.Fixed(lv.Height)*engine.Tile - w.ViewH
	sc := Scene{T: theme.Ink}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := sc.Draw(w, 4)
		c.Convert(120, 40, true)
		w.Step()
	}
}

func BenchmarkFrameWide(b *testing.B) {
	lv, _ := level.LoadEmbedded("1-1")
	sm := engine.NewStateMachine()
	sm.StartRun()
	w := game.NewWorld(lv, &game.Default, sm, 3)
	w.SetViewport(engine.Tile.Mul(15), engine.Tile.Mul(9))
	w.CameraY = engine.Fixed(lv.Height)*engine.Tile - w.ViewH
	sc := Scene{T: theme.Ink}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := sc.Draw(w, 8)
		c.Convert(120, 40, true)
		w.Step()
	}
}
