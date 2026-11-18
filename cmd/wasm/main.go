//go:build js && wasm

// Command wasm is the level editor's link to the game. It runs in the browser
// and exposes, on globalThis.mario, the game's own level checking, formatting,
// drawing and playtest (design decision 2). It holds no logic of its own: every
// call goes straight to pkg/playtest, which is tested natively.
//
// Values cross as JSON strings and Uint8Arrays. That is a little slower than
// building JS objects field by field, but it keeps this layer trivial, and the
// only call that runs every frame (step) passes a single integer.
package main

import (
	"encoding/json"
	"syscall/js"

	"super-mario/pkg/game"
	"super-mario/pkg/level"
	"super-mario/pkg/playtest"
)

var session *playtest.Session

func main() {
	api := js.Global().Get("Object").New()
	api.Set("validate", js.FuncOf(validate))
	api.Set("format", js.FuncOf(format))
	api.Set("sprites", js.FuncOf(sprites))
	api.Set("sky", js.FuncOf(sky))
	api.Set("themePreview", js.FuncOf(themePreview))
	api.Set("palettes", js.FuncOf(palettes))
	api.Set("drawRandom", js.FuncOf(drawRandom))
	api.Set("start", js.FuncOf(start))
	api.Set("step", js.FuncOf(step))
	api.Set("frame", js.FuncOf(frame))
	api.Set("stop", js.FuncOf(stop))
	js.Global().Set("mario", api)
	if ready := js.Global().Get("marioReady"); ready.Type() == js.TypeFunction {
		ready.Invoke()
	}
	select {} // stay alive for callbacks
}

func jsonOut(v interface{}) js.Value {
	b, err := json.Marshal(v)
	if err != nil {
		return js.ValueOf(`{"error":` + quote(err.Error()) + `}`)
	}
	return js.ValueOf(string(b))
}

func quote(s string) string { b, _ := json.Marshal(s); return string(b) }

func bytesOut(b []byte) js.Value {
	a := js.Global().Get("Uint8ClampedArray").New(len(b))
	js.CopyBytesToJS(a, b)
	return a
}

// validate(levelJSON) -> issues JSON
func validate(_ js.Value, args []js.Value) interface{} {
	return js.ValueOf(playtest.MarshalIssues(playtest.Validate([]byte(args[0].String()))))
}

// format(levelJSON) -> {"text": ...} or {"error": ...}
func format(_ js.Value, args []js.Value) interface{} {
	out, err := playtest.Format([]byte(args[0].String()))
	if err != nil {
		return jsonOut(map[string]string{"error": err.Error()})
	}
	return jsonOut(map[string]string{"text": string(out)})
}

// styleOf reads a level file's style, or none when it does not load.
func styleOf(levelJSON string) level.Style {
	lv, err := level.Parse([]byte(levelJSON))
	if err != nil {
		return level.Style{}
	}
	return lv.Style
}

// sprites(levelJSON, zoom) -> [{id, w, h, ox, oy, glyph, entity, rgba}]
func sprites(_ js.Value, args []js.Value) interface{} {
	sp := playtest.Sprites(styleOf(args[0].String()), args[1].Int())
	out := js.Global().Get("Array").New(len(sp))
	for i, s := range sp {
		o := js.Global().Get("Object").New()
		o.Set("id", s.ID)
		o.Set("w", s.W)
		o.Set("h", s.H)
		o.Set("ox", s.OX)
		o.Set("oy", s.OY)
		o.Set("glyph", s.Glyph)
		o.Set("entity", s.Entity)
		o.Set("rgba", bytesOut(s.RGBA))
		out.SetIndex(i, o)
	}
	return out
}

// sky(levelJSON, zoom) -> JSON array of #rrggbb, one per art row
func sky(_ js.Value, args []js.Value) interface{} {
	return jsonOut(playtest.SkyColours(styleOf(args[0].String()), args[1].Int()))
}

// themePreview(themeID, zoom) -> {w, h, rgba}
func themePreview(_ js.Value, args []js.Value) interface{} {
	return frameOut(playtest.ThemePreview(args[0].String(), args[1].Int()))
}

// palettes() -> every theme, sky preset and colour list, so the editor offers
// exactly what level files accept.
func palettes(_ js.Value, _ []js.Value) interface{} {
	type sw struct {
		ID, Label, LabelEn, Hex string
	}
	list := func(name string, l []level.Swatch) []sw {
		out := make([]sw, len(l))
		for i, s := range l {
			out[i] = sw{s.ID, s.Label, level.EnglishName(name, s.ID), s.RGB.Hex()}
		}
		return out
	}
	type sky struct{ ID, Label, LabelEn, Top, Bottom string }
	type theme struct {
		ID, Label, LabelEn, SkyTop, SkyBottom, Ground, Turf, Brick, Stair, Pipe string
	}
	var skies []sky
	for _, s := range level.SkyPresets {
		skies = append(skies, sky{s.ID, s.Label, level.EnglishName("sky", s.ID), s.Sky.Top.Hex(), s.Sky.Bottom.Hex()})
	}
	var themes []theme
	for _, t := range level.Themes {
		themes = append(themes, theme{t.ID, t.Label, level.EnglishName("theme", t.ID), t.Sky.Top.Hex(), t.Sky.Bottom.Hex(),
			t.Ground.Hex(), t.Turf.Hex(), t.Brick.Hex(), t.Stair.Hex(), t.Pipe.Hex()})
	}
	var kinds []string
	for _, k := range level.ConcreteKinds {
		kinds = append(kinds, string(k))
	}
	return jsonOut(map[string]interface{}{
		"themes": themes, "skies": skies,
		"ground": list("ground", level.GroundColors), "turf": list("turf", level.TurfColors),
		"brick": list("brick", level.BrickColors), "stair": list("stair", level.StairColors), "pipe": list("pipe", level.PipeColors),
		"blockKinds": kinds,
	})
}

// drawRandom(seed, life, x, y, poolJSON) -> kind, the same rule the game uses
func drawRandom(_ js.Value, args []js.Value) interface{} {
	var pool []struct {
		Kind string `json:"kind"`
		W    int    `json:"w"`
	}
	if err := json.Unmarshal([]byte(args[4].String()), &pool); err != nil {
		return js.ValueOf("")
	}
	ch := make([]level.Choice, len(pool))
	for i, p := range pool {
		ch[i] = level.Choice{Kind: level.BlockKind(p.Kind), Weight: p.W}
	}
	return js.ValueOf(string(game.DrawRandom(int64(args[0].Float()), args[1].Int(), args[2].Int(), args[3].Int(), ch)))
}

// start(levelJSON, optionsJSON) -> {"ok": true, "notice": ...} or {"issues": [...]}
func start(_ js.Value, args []js.Value) interface{} {
	var opts playtest.Options
	opts.FromColumn = -1
	_ = json.Unmarshal([]byte(args[1].String()), &opts)
	s, issues := playtest.Start([]byte(args[0].String()), opts)
	if s == nil {
		return js.ValueOf(`{"issues":` + playtest.MarshalIssues(issues) + `}`)
	}
	session = s
	return jsonOut(map[string]interface{}{"ok": true, "notice": s.Notice})
}

// step(keys, n) -> "" while running, "died" or "cleared"
func step(_ js.Value, args []js.Value) interface{} {
	if session == nil {
		return js.ValueOf("stopped")
	}
	n := 1
	if len(args) > 1 {
		n = args[1].Int()
	}
	var end playtest.End
	for i := 0; i < n; i++ {
		end = session.Step(playtest.Input(args[0].Int()))
		if end != playtest.Running {
			break
		}
	}
	return js.ValueOf(string(end))
}

// frame() -> {w, h, rgba, cameraX, cameraY, draws}
func frame(_ js.Value, _ []js.Value) interface{} {
	if session == nil {
		return js.Null()
	}
	return frameOut(session.Render())
}

func frameOut(f playtest.Frame) js.Value {
	o := js.Global().Get("Object").New()
	o.Set("w", f.W)
	o.Set("h", f.H)
	o.Set("rgba", bytesOut(f.RGBA))
	o.Set("cameraX", f.CameraX)
	o.Set("cameraY", f.CameraY)
	b, _ := json.Marshal(f.Draws)
	o.Set("draws", string(b))
	return o
}

// stop() -> summary JSON
func stop(_ js.Value, _ []js.Value) interface{} {
	if session == nil {
		return js.Null()
	}
	sum := session.Stop()
	session = nil
	return jsonOut(sum)
}
