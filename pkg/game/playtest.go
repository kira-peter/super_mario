package game

import "super-mario/pkg/engine"

// PlaytestOptions are switches only the level editor's playtest uses. The
// terminal game always runs with the zero value, which changes nothing, and
// never offers them to a player (spec level-playtest / 试玩选项).
type PlaytestOptions struct {
	// StartBig starts every life as big Mario, to try a level's second half
	// without hunting for a mushroom first.
	StartBig bool
	// Invincible stops every way of dying. Enemies, poison and the clock do
	// nothing; a fall into a pit puts the player back on the last ground they
	// stood on, so the rest of the level can still be reached.
	Invincible bool
}

// SetPlaytest switches the playtest options on for this world and applies them
// to the current life.
func (w *World) SetPlaytest(o PlaytestOptions) {
	w.playtest = o
	w.applyPlaytestStart()
}

// applyPlaytestStart is run at the start of every life.
func (w *World) applyPlaytestStart() {
	if w.playtest.StartBig && w.Player.Form == FormSmall {
		w.Player.applyForm(w.Level, FormBig)
	}
}

// rememberGround notes where the player last stood, for the invincible
// playtest's rescue from a pit.
func (w *World) rememberGround() {
	if w.Player.Grounded && !w.Player.Dead {
		w.lastGround = w.Player.Pos
	}
}

// rescueFromPit puts an invincible player back on solid ground instead of
// letting a fall kill them.
func (w *World) rescueFromPit() {
	p := w.Player
	p.Pos = w.lastGround
	p.Vel = engine.Vec2{}
	p.Grounded = true
	p.Jumping = false
}

// StartAtColumn moves the start of the current life to a column, for "play
// from here". The player stands on the highest spot in that column they can
// stand on, and the camera jumps there. The level itself is not changed.
//
// It returns false, and leaves the player at the level's spawn, when the column
// has nowhere to stand (a pit, or solid from top to bottom).
func (w *World) StartAtColumn(col int) bool {
	if col < 0 || col >= w.Level.Width {
		return false
	}
	for row := 0; row < w.Level.Height-1; row++ {
		if !w.Level.SolidAt(col, row) && w.Level.SolidAt(col, row+1) {
			form := w.Player.Form
			w.Player = NewPlayer(w.T, col, row)
			if form != FormSmall {
				w.Player.applyForm(w.Level, form)
			}
			// A big body needs two open rows; if this spot only has one,
			// keep looking lower down.
			if overlapsSolid(w.Level, w.Player.Rect()) {
				continue
			}
			w.lastGround = w.Player.Pos
			w.updateCamera(true)
			return true
		}
	}
	return false
}

// TracePoint is one step of a playtest run, kept for drawing the path the
// player took over the editor canvas (spec level-playtest / 轨迹回放).
type TracePoint struct {
	// X and Y are the middle of the player's feet, in game units.
	X, Y engine.Fixed
	// Grounded is whether they were standing.
	Grounded bool
	// Jump is set on the step a jump started.
	Jump bool
}

// MaxTrace is how many steps a trace keeps: one minute at sixty steps a second.
// A longer run keeps its last minute.
const MaxTrace = 3600

// Trace records a playtest's path. Once full it keeps the most recent
// MaxTrace steps, in a ring so recording stays cheap every step.
type Trace struct {
	ring  []TracePoint
	start int
	// Dropped counts the oldest points thrown away once the trace was full.
	Dropped int
	// Death is where the player died, if they did.
	Death *TracePoint
	last  TracePoint
	any   bool
}

// Record appends the current step.
func (tr *Trace) Record(w *World) {
	p := w.Player
	pt := TracePoint{X: p.CenterX(), Y: p.Bottom(), Grounded: p.Grounded}
	if tr.any {
		pt.Jump = tr.last.Grounded && !p.Grounded && p.Vel.Y < 0
	}
	if p.Dead && tr.Death == nil {
		d := pt
		tr.Death = &d
	}
	if len(tr.ring) < MaxTrace {
		tr.ring = append(tr.ring, pt)
	} else {
		tr.ring[tr.start] = pt
		tr.start = (tr.start + 1) % MaxTrace
		tr.Dropped++
	}
	tr.last, tr.any = pt, true
}

// Points returns the recorded path, oldest first.
func (tr *Trace) Points() []TracePoint {
	out := make([]TracePoint, 0, len(tr.ring))
	out = append(out, tr.ring[tr.start:]...)
	return append(out, tr.ring[:tr.start]...)
}
