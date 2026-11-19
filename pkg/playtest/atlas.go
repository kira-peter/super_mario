package playtest

import (
	"strings"

	"super-mario/pkg/engine"
	"super-mario/pkg/game"
	"super-mario/pkg/level"
	"super-mario/pkg/render"
	"super-mario/pkg/render/theme"
)

// Sprite is one element as the game draws it, for the editor's canvas and
// palette. The picture is cut to what is actually drawn; OX and OY say where
// its top-left sits relative to the top-left of the element's tile, so a
// sprite taller than a tile (a Koopa, a flag) is placed exactly as in the game.
type Sprite struct {
	ID     string `json:"id"`
	W      int    `json:"w"`
	H      int    `json:"h"`
	OX     int    `json:"ox"`
	OY     int    `json:"oy"`
	RGBA   []byte `json:"-"`
	Zoom   int    `json:"zoom"`
	Glyph  string `json:"glyph,omitempty"`
	Entity string `json:"entity,omitempty"`
}

// atlasCase is one element: how to put it in a scratch level.
type atlasCase struct {
	id     string
	glyph  byte   // a tile glyph, or 0
	entity string // "goomba", "koopa", or ""
	// variant picks the turf ground ("top") or the soil under it ("below").
	below bool
	// item draws a mushroom of this kind standing on the tile.
	item   game.ItemKind
	isItem bool
}

var atlasCases = []atlasCase{
	{id: "ground", glyph: 'X'},
	{id: "ground-below", glyph: 'X', below: true},
	{id: "stair", glyph: 'H'},
	{id: "brick", glyph: 'B'},
	{id: "question", glyph: '?'},
	{id: "used", glyph: '#'},
	{id: "pipe-lip-l", glyph: '['},
	{id: "pipe-lip-r", glyph: ']'},
	{id: "pipe-body-l", glyph: '{'},
	{id: "pipe-body-r", glyph: '}'},
	{id: "coin", glyph: 'o'},
	{id: "goomba", entity: "goomba"},
	{id: "koopa", entity: "koopa"},
	// What a block can hold, drawn as the game draws it popping out. The
	// editor marks blocks with these, so a designer sees the real colours.
	{id: "item-mushroom", item: game.ItemMushroom, isItem: true},
	{id: "item-star", item: game.ItemStar, isItem: true},
	{id: "item-oneup", item: game.ItemOneUp, isItem: true},
	{id: "item-poison", item: game.ItemPoison, isItem: true},
	{id: "item-gold", item: game.ItemGold, isItem: true},
}

// The scratch level the atlas draws in: the element at column 3, row 8, with
// open air around it so nothing else is drawn near it.
const (
	atlasW, atlasH = 8, 15
	atlasCol       = 3
	atlasRow       = 8
)

// Sprites draws every editor element at a zoom, in a level's style. Each one is
// drawn by the game's own renderer in a level holding only that element, so the
// editor cannot show anything the game would draw differently.
func Sprites(style level.Style, zoom int) []Sprite {
	if zoom <= 0 {
		zoom = 8
	}
	var out []Sprite
	for _, c := range atlasCases {
		sp, ok := drawAlone(c, style, zoom)
		if ok {
			out = append(out, sp)
		}
	}
	return out
}

func drawAlone(c atlasCase, style level.Style, zoom int) (Sprite, bool) {
	rows := make([][]byte, atlasH)
	for y := range rows {
		rows[y] = []byte(strings.Repeat(" ", atlasW))
	}
	var contents, entities string
	switch {
	case c.glyph == '#':
		// A used block is what a question block turns into; there is no
		// symbol for it in level files, so make one and use it up.
		rows[atlasRow][atlasCol] = '?'
		contents = `{"x":3,"y":8,"kind":"coin"}`
	case c.glyph != 0:
		rows[atlasRow][atlasCol] = c.glyph
		if c.glyph == '?' {
			contents = `{"x":3,"y":8,"kind":"coin"}`
		}
		if c.below {
			// Ground with ground above it is drawn as plain soil.
			rows[atlasRow-1][atlasCol] = 'X'
		}
	case c.entity != "" || c.isItem:
		// An enemy or a mushroom stands on a floor; the floor is cut away
		// below.
		rows[atlasRow+1][atlasCol] = 'H'
		if c.entity != "" {
			entities = `{"kind":"` + c.entity + `","x":3,"y":8}`
		}
	}
	var b strings.Builder
	b.WriteString(`{"name":"atlas","world":"1-1","width":8,"height":15,"timeLimit":400,`)
	b.WriteString(`"spawn":{"x":0,"y":0},"flag":{"x":7,"topY":0,"bottomY":0},"rows":[`)
	for y, r := range rows {
		if y > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`"` + string(r) + `"`)
	}
	b.WriteString(`],"contents":[` + contents + `],"entities":[` + entities + `]}`)
	lv, err := level.Parse([]byte(b.String()))
	if err != nil {
		return Sprite{}, false
	}
	lv.Style = style
	if c.glyph == '#' {
		lv.SetTile(atlasCol, atlasRow, level.Tile{Kind: level.KindSolid, Glyph: '#'})
	}

	sm := engine.NewStateMachine()
	sm.StartRun()
	w := game.NewWorld(lv, &game.Default, sm, 3)
	w.Player.Hidden = true
	w.SetViewport(engine.Tile*atlasW, engine.Tile*atlasH)
	w.CameraX, w.CameraY = 0, 0
	if c.isItem {
		it := game.NewItem(c.item, atlasCol, atlasRow)
		it.Emerging = 0
		w.Items = append(w.Items, it)
	}

	sc := render.ForLevel(theme.Ink, "ink", lv, theme.TrueColor)
	sc.Transparent = true
	canvas := sc.Draw(w, zoom)

	// Keep only what belongs to the element: the cells around its tile, and
	// for an enemy the cells above it too, never the floor it stands on.
	x0, y0 := (atlasCol-1)*zoom, (atlasRow-2)*zoom
	x1, y1 := (atlasCol+2)*zoom, (atlasRow+1)*zoom
	if c.below {
		y0 = atlasRow * zoom
	}
	minX, minY, maxX, maxY := x1, y1, x0-1, y0-1
	for y := y0; y < y1 && y < canvas.H; y++ {
		for x := x0; x < x1 && x < canvas.W; x++ {
			if canvas.Px[y*canvas.W+x] != nil {
				minX, minY = min(minX, x), min(minY, y)
				maxX, maxY = max(maxX, x), max(maxY, y)
			}
		}
	}
	if maxX < minX {
		return Sprite{}, false
	}
	sw, sh := maxX-minX+1, maxY-minY+1
	crop := render.NewCanvas(sw, sh)
	for y := 0; y < sh; y++ {
		for x := 0; x < sw; x++ {
			crop.Px[y*sw+x] = canvas.Px[(minY+y)*canvas.W+minX+x]
		}
	}
	sp := Sprite{
		ID: c.id, W: sw, H: sh, Zoom: zoom,
		OX: minX - atlasCol*zoom, OY: minY - atlasRow*zoom,
		RGBA: ToRGBA(crop, nil), Entity: c.entity,
	}
	if c.glyph != 0 && c.glyph != '#' {
		sp.Glyph = string(c.glyph)
	}
	return sp, true
}

// SkyColours returns the sky as the editor should paint it behind its canvas:
// one colour per art row of a level, top to bottom, as #rrggbb.
func SkyColours(style level.Style, zoom int) []string {
	_, fill := theme.Apply(theme.Ink, style, theme.TrueColor)
	if !fill.Gradient() && fill.Top == 0 {
		fill = theme.SkyFill{Top: theme.Ink.Sky, Bottom: theme.Ink.Sky}
	}
	n := 15 * zoom
	out := make([]string, n)
	for y := 0; y < n; y++ {
		r, g, b := fill.At(float64(y) / float64(n-1)).RGB()
		out[y] = hex(r, g, b)
	}
	return out
}

// SkyFrame draws a whole level's sky, with its stars, and nothing else: every
// tile emptied and the player hidden. Tests compare it with the editor's star
// list.
func SkyFrame(lv *level.Level, zoom int) Frame {
	for y := 0; y < lv.Height; y++ {
		for x := 0; x < lv.Width; x++ {
			lv.SetTile(x, y, level.Tile{Kind: level.KindEmpty, Glyph: ' '})
		}
	}
	lv.Contents = nil
	lv.Spawns = nil
	sm := engine.NewStateMachine()
	sm.StartRun()
	w := game.NewWorld(lv, &game.Default, sm, 3)
	w.Player.Hidden = true
	w.SetViewport(engine.Tile*engine.Fixed(lv.Width), engine.Tile*engine.Fixed(lv.Height))
	w.CameraX, w.CameraY = 0, 0
	sc := render.ForLevel(theme.Ink, "ink", lv, theme.TrueColor)
	c := sc.Draw(w, zoom)
	return Frame{W: c.W, H: c.H, RGBA: ToRGBA(c, sc.T.Sky)}
}

// EditorStar is one star for the editor's canvas: its position in pixels of the
// four-pixel-tile drawing, and its colour at the step asked for ("" while the
// star is out for a moment of its twinkle).
type EditorStar struct {
	X, Y  int
	Color string
	Cross bool
}

// Stars lists the stars a level at night shows, coloured as the game draws them
// at a simulation step, so the editor's canvas twinkles exactly like the
// playtest. Empty for any other sky.
func Stars(style level.Style, width, height, step int) []EditorStar {
	sc := render.ForLevel(theme.Ink, "ink", &level.Level{Style: style}, theme.TrueColor)
	fill := sc.Fill
	if !fill.Gradient() && fill.Top == 0 {
		fill = theme.SkyFill{Top: theme.Ink.Sky, Bottom: theme.Ink.Sky}
	}
	out := []EditorStar{}
	for _, s := range render.StarField(style, width, height, fill.At) {
		e := EditorStar{X: s.X, Y: s.Y, Cross: s.Cross}
		if c, lit := sc.Twinkle(s, step); lit {
			r, g, b := c.RGB()
			e.Color = hex(r, g, b)
		}
		out = append(out, e)
	}
	return out
}

func hex(r, g, b int32) string {
	const d = "0123456789abcdef"
	return string([]byte{'#', d[r>>4], d[r&15], d[g>>4], d[g&15], d[b>>4], d[b&15]})
}

// ThemePreview draws a short stretch of sample terrain in a theme, for the
// editor's theme cards: ground with a gap, a brick row with a question block,
// a pipe and a step of stairs.
func ThemePreview(themeID string, zoom int) Frame {
	if zoom <= 0 {
		zoom = 4
	}
	rows := []string{
		"                ",
		"                ",
		"                ",
		"                ",
		"                ",
		"                ",
		"                ",
		"                ",
		"                ",
		"   B?B          ",
		"                ",
		"         []   H ",
		"         {}  HH ",
		"XXXXXX  XXXXXXXX",
		"XXXXXX  XXXXXXXX",
	}
	var b strings.Builder
	b.WriteString(`{"name":"preview","world":"1-1","width":16,"height":15,"timeLimit":400,`)
	b.WriteString(`"style":{"theme":"` + themeID + `"},`)
	b.WriteString(`"spawn":{"x":1,"y":12},"flag":{"x":15,"topY":0,"bottomY":0},"rows":[`)
	for i, r := range rows {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`"` + r + `"`)
	}
	b.WriteString(`],"contents":[{"x":4,"y":9,"kind":"coin"}],"entities":[{"kind":"goomba","x":11,"y":10}]}`)
	lv, err := level.Parse([]byte(b.String()))
	if err != nil {
		return Frame{}
	}
	sm := engine.NewStateMachine()
	sm.StartRun()
	w := game.NewWorld(lv, &game.Default, sm, 3)
	w.Player.Hidden = true
	w.SetViewport(engine.Tile*16, engine.Tile*8)
	w.CameraX, w.CameraY = 0, engine.Tile*7
	sc := render.ForLevel(theme.Ink, "ink", lv, theme.TrueColor)
	c := sc.Draw(w, zoom)
	return Frame{W: c.W, H: c.H, RGBA: ToRGBA(c, sc.T.Sky)}
}
