package game

import (
	"strings"

	"super-mario/pkg/engine"
	"super-mario/pkg/level"
)

// JumpSample is where a jump has got to after one step, measured from the
// take-off position. DX is forward (right), DY is up; both in game units.
type JumpSample struct{ DX, DY engine.Fixed }

// JumpTrajectory runs a real jump on a flat floor and returns its path, one
// sample per step until the landing.
//
// The player runs right for runup steps (with run held if run is set), then
// jumps, holds the jump button for hold steps and keeps steering right the whole
// way. It plays the same World and Player the game does, so the level design
// checks learn what a jump can do from the physics themselves: change the
// tuning and the answer changes with it (spec level-contribution / 设计提示项).
func JumpTrajectory(t *Tuning, run bool, runup, hold int) []JumpSample {
	const width = 140
	rows := make([]string, 15)
	for y := range rows {
		c := " "
		if y >= 13 {
			c = "X"
		}
		rows[y] = `"` + strings.Repeat(c, width) + `"`
	}
	data := `{"name":"env","world":"1-1","width":140,"height":15,"timeLimit":400,` +
		`"spawn":{"x":2,"y":12},"flag":{"x":138,"topY":2,"bottomY":12},` +
		`"rows":[` + strings.Join(rows, ",") + `],"contents":[],"entities":[]}`
	lv, err := level.Parse([]byte(data))
	if err != nil {
		return nil
	}
	sm := engine.NewStateMachine()
	sm.StartRun()
	tu := *t
	w := NewWorld(lv, &tu, sm, 3)
	in := engine.NewInput(nil)
	w.Input = in

	press := func() {
		in.KeyDown(engine.KeyRight)
		if run {
			in.KeyDown(engine.KeyShift)
		}
	}
	for i := 0; i < runup; i++ {
		press()
		w.Step()
	}
	x0, y0 := w.Player.Pos.X, w.Player.Bottom()

	var out []JumpSample
	for i := 0; i < 300; i++ {
		press()
		if i < hold {
			in.KeyDown(engine.KeySpace)
		} else {
			in.KeyUp(engine.KeySpace)
		}
		w.Step()
		out = append(out, JumpSample{DX: w.Player.Pos.X - x0, DY: y0 - w.Player.Bottom()})
		if i > 0 && w.Player.Grounded {
			break
		}
	}
	return out
}
