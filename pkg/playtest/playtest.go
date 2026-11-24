// Package playtest runs a level the way the game does, for the editor in the
// browser (spec level-playtest).
//
// Everything here is plain Go with plain inputs and outputs, so it is tested
// natively. cmd/wasm only translates JavaScript values to these calls and
// back. That split is what lets the consistency test compare a playtest with
// the terminal game's own World step for step.
package playtest

import (
	"encoding/json"

	"super-mario/pkg/engine"
	"super-mario/pkg/game"
	"super-mario/pkg/level"
	"super-mario/pkg/level/lint"
	"super-mario/pkg/render"
	"super-mario/pkg/render/theme"
)

// Validate returns every error and warning for a level file. The editor shows
// this list; a level with any error cannot be saved or played.
func Validate(data []byte) []lint.Issue {
	return lint.Run(data, &game.Default)
}

// Format returns the level in the canonical file layout, or the first reason
// it cannot be.
func Format(data []byte) ([]byte, error) { return level.Format(data) }

// Options are the switches the editor's playtest bar offers. None of them is
// ever written into the level.
type Options struct {
	// FromColumn starts the run at this column instead of the spawn point;
	// negative means the spawn.
	FromColumn int  `json:"fromColumn"`
	StartBig   bool `json:"startBig"`
	Invincible bool `json:"invincible"`
	// Seed fixes the random blocks' draws. Zero picks one.
	Seed int64 `json:"seed"`
	// ShowDraws marks every random block with what it drew, before it is hit.
	ShowDraws bool `json:"showDraws"`
	// Zoom is art pixels per tile, as in the terminal: 4, 8, 12 or 16.
	Zoom int `json:"zoom"`
	// ViewW and ViewH are how many tiles wide and tall the picture is.
	ViewW int `json:"viewW"`
	ViewH int `json:"viewH"`
}

// Input is the keys held this frame, as a bit set. The browser has real key
// releases, so there is no release timing to infer.
type Input uint32

const (
	InLeft Input = 1 << iota
	InRight
	InDown
	InJump
	InRun
)

// End says why a playtest stopped.
type End string

const (
	// Running is not an end: the run goes on.
	Running End = ""
	Died    End = "died"
	Cleared End = "cleared"
)

// Session is one playtest.
type Session struct {
	World *game.World
	Trace game.Trace
	// Notice is a message for the editor, such as "no ground in that column".
	Notice string

	scene  render.Scene
	opts   Options
	held   Input
	ended  End
	frames int
}

// Start builds a playtest of a level file. It refuses a level with errors,
// returning them, exactly as the editor refuses to start one.
func Start(data []byte, opts Options) (*Session, []lint.Issue) {
	if probs := level.Check(data); len(probs) > 0 {
		out := make([]lint.Issue, len(probs))
		for i, p := range probs {
			out[i] = lint.Issue{Severity: lint.Error, X: p.X, Y: p.Y, Code: p.Code, Message: p.Message}
		}
		return nil, out
	}
	lv, err := level.Parse(data)
	if err != nil {
		return nil, []lint.Issue{{Severity: lint.Error, X: -1, Y: -1, Code: "internal", Message: err.Error()}}
	}
	if opts.Zoom == 0 {
		opts.Zoom = 8
	}
	if opts.ViewW <= 0 {
		opts.ViewW = 20
	}
	if opts.ViewH <= 0 {
		opts.ViewH = 15
	}

	sm := engine.NewStateMachine()
	sm.StartRun()
	w := game.NewWorld(lv, &game.Default, sm, 3)
	w.SetSeed(opts.Seed)
	w.SetPlaytest(game.PlaytestOptions{StartBig: opts.StartBig, Invincible: opts.Invincible})
	w.SetViewport(engine.Tile*engine.Fixed(opts.ViewW), engine.Tile*engine.Fixed(opts.ViewH))

	s := &Session{World: w, opts: opts}
	if opts.FromColumn >= 0 && opts.FromColumn != lv.SpawnX {
		if !w.StartAtColumn(opts.FromColumn) {
			s.Notice = "这一列没有落脚点，从出生点开始"
		}
	}
	// He arrives as in the game: pixels assembling above where he starts.
	w.StartEntrance()
	s.scene = render.ForLevel(theme.Ink, "ink", lv, theme.TrueColor)
	return s, nil
}

// Step advances one simulation step with the given keys held. It returns how
// the run stands afterwards.
func (s *Session) Step(in Input) End {
	if s.ended != Running {
		return s.ended
	}
	s.apply(in)
	s.World.Step()
	s.Trace.Record(s.World)
	s.frames++

	switch {
	case s.World.Cleared:
		s.ended = Cleared
	case s.World.SM.Is(engine.StateDying):
		// The editor goes back to editing on the first death, without the
		// death animation or a respawn: the designer wants to fix the spot,
		// not watch it again.
		s.ended = Died
	}
	return s.ended
}

// apply turns the held-key set into presses and releases. The game's input
// layer works in edges, so only changes are sent.
func (s *Session) apply(in Input) {
	keys := []struct {
		bit Input
		key engine.Key
	}{
		{InLeft, engine.KeyLeft}, {InRight, engine.KeyRight}, {InDown, engine.KeyDown},
		{InJump, engine.KeySpace}, {InRun, engine.KeyShift},
	}
	for _, k := range keys {
		was, now := s.held&k.bit != 0, in&k.bit != 0
		switch {
		case now && !was:
			s.World.Input.KeyDown(k.key)
		case was && !now:
			s.World.Input.KeyUp(k.key)
		}
	}
	s.held = in
}

// Frame is one rendered picture.
type Frame struct {
	W, H int
	// RGBA is W*H*4 bytes, row by row, ready for an ImageData.
	RGBA []byte
	// CameraX and CameraY are the view's top-left in tiles, for placing the
	// editor's overlays and for returning the editor view to where the
	// playtest ended.
	CameraX, CameraY float64
	// Draws, when ShowDraws is on, lists each untouched random block and what
	// it holds this life.
	Draws []Draw `json:"draws,omitempty"`
}

// Draw is one random block's content this life.
type Draw struct {
	X, Y int
	Kind level.BlockKind
}

// Render draws the current picture.
func (s *Session) Render() Frame {
	c := s.scene.Draw(s.World, s.opts.Zoom)
	f := Frame{
		W: c.W, H: c.H,
		RGBA:    ToRGBA(c, s.scene.T.Sky),
		CameraX: float64(s.World.CameraX) / float64(engine.Tile),
		CameraY: float64(s.World.CameraY) / float64(engine.Tile),
	}
	if s.opts.ShowDraws {
		for _, c := range s.World.Level.Contents {
			if !c.Random {
				continue
			}
			// A block that has been hit is a used block now; it shows itself.
			if t := s.World.Level.TileAt(c.X, c.Y); t.Glyph == '#' {
				continue
			}
			f.Draws = append(f.Draws, Draw{X: c.X, Y: c.Y, Kind: c.Kind})
		}
	}
	return f
}

// Summary is what the editor needs once a playtest stops.
type Summary struct {
	End End `json:"end"`
	// Column and Row are where the player was, in tiles, so the editor can
	// put its view back there.
	Column float64 `json:"column"`
	Row    float64 `json:"row"`
	// Trace is the path, in tiles, oldest first.
	Trace []TracePoint `json:"trace"`
	// Death is where the player died, if they did.
	Death *TracePoint `json:"death,omitempty"`
	Steps int         `json:"steps"`
}

// TracePoint is a step of the path in tiles.
type TracePoint struct {
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Grounded bool    `json:"g,omitempty"`
	Jump     bool    `json:"j,omitempty"`
}

// Stop ends the playtest and returns what happened.
func (s *Session) Stop() Summary {
	p := s.World.Player
	sum := Summary{
		End:    s.ended,
		Column: float64(p.CenterX()) / float64(engine.Tile),
		Row:    float64(p.Bottom()) / float64(engine.Tile),
		Steps:  s.frames,
	}
	tile := float64(engine.Tile)
	conv := func(t game.TracePoint) TracePoint {
		return TracePoint{X: float64(t.X) / tile, Y: float64(t.Y) / tile, Grounded: t.Grounded, Jump: t.Jump}
	}
	for _, t := range s.Trace.Points() {
		sum.Trace = append(sum.Trace, conv(t))
	}
	if s.Trace.Death != nil {
		d := conv(*s.Trace.Death)
		sum.Death = &d
	}
	return sum
}

// ToRGBA turns a canvas into RGBA bytes. Pixels nothing was drawn on take the
// background colour, or are fully transparent when there is none.
func ToRGBA(c *render.Canvas, bg interface{}) []byte {
	out := make([]byte, c.W*c.H*4)
	var br, bgc, bb int32
	hasBG := false
	if col, ok := bg.(interface{ RGB() (int32, int32, int32) }); ok && bg != nil {
		br, bgc, bb = col.RGB()
		hasBG = true
	}
	for i, px := range c.Px {
		o := i * 4
		if col, ok := px.(interface{ RGB() (int32, int32, int32) }); ok && px != nil {
			r, g, b := col.RGB()
			out[o], out[o+1], out[o+2], out[o+3] = byte(r), byte(g), byte(b), 255
		} else if hasBG {
			out[o], out[o+1], out[o+2], out[o+3] = byte(br), byte(bgc), byte(bb), 255
		}
	}
	return out
}

// MarshalIssues is a helper for the wasm layer.
func MarshalIssues(is []lint.Issue) string {
	if is == nil {
		is = []lint.Issue{}
	}
	b, _ := json.Marshal(is)
	return string(b)
}
