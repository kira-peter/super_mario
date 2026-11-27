// Package level parses and validates declarative level data.
//
// A level file is plain JSON describing a grid of tiles plus a list of entity
// spawns. It contains no expressions and no code, so a level can be edited
// without touching or rebuilding the game (spec level-1-1 / 关卡数据格式).
package level

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"unicode/utf8"

	"super-mario/pkg/engine"
)

//go:embed all:data
var assets embed.FS

// Kind is the collision behaviour of a tile.
type Kind int

const (
	// KindEmpty is passable air.
	KindEmpty Kind = iota
	// KindSolid blocks from every side.
	KindSolid
	// KindBreakable is solid, and a big Mario smashes it from below.
	KindBreakable
	// KindQuestion is solid, and dispenses its stored content when hit.
	KindQuestion
	// KindOneWay blocks only from above.
	KindOneWay
	// KindHazard kills on contact.
	KindHazard
	// KindDecor is drawn but does not collide.
	KindDecor
)

// Solid reports whether the tile stops movement.
func (k Kind) Solid() bool {
	switch k {
	case KindSolid, KindBreakable, KindQuestion, KindOneWay:
		return true
	}
	return false
}

// Bumpable reports whether hitting the tile from below triggers something.
func (k Kind) Bumpable() bool { return k == KindBreakable || k == KindQuestion }

// Tile is one cell's static description.
type Tile struct {
	Kind  Kind
	Glyph rune // the source symbol, for rendering and error messages
}

// symbolTable is the single definition of the on-disk alphabet. The parser and
// the tests both read it, so they cannot drift (design decision 5).
var symbolTable = map[rune]Kind{
	' ': KindEmpty,
	'X': KindSolid,     // ground
	'H': KindSolid,     // hard stair block
	'B': KindBreakable, // brick
	'?': KindQuestion,  // question block
	'[': KindSolid,     // pipe lip, left half
	']': KindSolid,     // pipe lip, right half
	'{': KindSolid,     // pipe body, left half
	'}': KindSolid,     // pipe body, right half
	'F': KindDecor,     // flagpole shaft
	'T': KindDecor,     // flagpole ball
	'o': KindDecor,     // coin (collected as an entity)
}

// BlockKind is what a question block dispenses. Apart from a coin, each is a
// mushroom whose colour says what it does.
type BlockKind string

const (
	BlockCoin     BlockKind = "coin"
	BlockMushroom BlockKind = "mushroom" // red: grow
	BlockStar     BlockKind = "star"     // blue: invincible
	BlockOneUp    BlockKind = "oneup"    // green: extra life
	BlockPoison   BlockKind = "poison"   // purple: hurts
	BlockGold     BlockKind = "gold"     // gold: five coins
	// BlockRandom is not something a block dispenses: it says the block draws
	// one of its Pool at the start of each life (spec random-block-content).
	BlockRandom BlockKind = "random"
)

// Concrete reports whether k is something a block can actually dispense.
func (k BlockKind) Concrete() bool {
	switch k {
	case BlockCoin, BlockMushroom, BlockStar, BlockOneUp, BlockPoison, BlockGold:
		return true
	}
	return false
}

// ConcreteKinds lists every dispensable kind, in the order the editor shows
// them.
var ConcreteKinds = []BlockKind{BlockCoin, BlockMushroom, BlockStar, BlockOneUp, BlockPoison, BlockGold}

// Choice is one entry of a random block's pool.
type Choice struct {
	Kind   BlockKind
	Weight int
}

// Content describes a block's payload, fixed at authoring time so a given block
// always yields the same thing (spec blocks-and-items / 问号块).
type Content struct {
	X, Y   int
	Kind   BlockKind
	Count  int  // remaining uses; a coin brick may hold several
	Hidden bool // invisible until bumped
	// Random is set for a block whose Kind is drawn from Pool at the start of
	// each life. While the level is being played Kind holds this life's draw;
	// Restore puts it back to BlockRandom.
	Random bool
	Pool   []Choice
}

// Spawn is an entity placed at level load.
type Spawn struct {
	Kind string // "goomba", "koopa"
	X, Y int
}

// Meta is a level's optional description, shown in level select and kept for
// contribution records (spec level-contribution / 关卡元信息).
type Meta struct {
	Title       string
	Author      string
	Description string
	// Difficulty is 1 to 5, or 0 when the author did not say.
	Difficulty int
}

// Length limits for the meta fields, counted in runes so a Chinese title gets
// the same room as an English one.
const (
	MaxTitleLen       = 32
	MaxAuthorLen      = 24
	MaxDescriptionLen = 140
	MaxDifficulty     = 5
)

// MaxCells bounds the in-memory terrain grid and its respawn snapshot.
const MaxCells = 1 << 20

// Level is a parsed, validated level.
type Level struct {
	Name  string
	World string
	Meta  Meta
	// Style is the level's appearance; zero means the game's default look.
	Style     Style
	Width     int
	Height    int
	TimeLimit int
	SpawnX    int
	SpawnY    int
	FlagX     int
	FlagTop   int
	FlagBot   int
	tiles     []Tile
	// pristine is the terrain as loaded. Play changes tiles (a brick is
	// smashed, a block is used, a coin is taken), and a respawn puts them back
	// from here.
	pristine []Tile
	// contents0 is the block payloads as loaded, for the same reason: a coin
	// brick counts down as it is hit.
	contents0 []Content
	Contents  []Content
	Spawns    []Spawn
}

// Restore puts every tile and block payload back the way the level was
// loaded. It is what a respawn uses, so the level is the same on every life.
// A random block goes back to undrawn; the world draws it again.
func (l *Level) Restore() {
	if l.pristine == nil {
		return
	}
	copy(l.tiles, l.pristine)
	l.Contents = append(l.Contents[:0], l.contents0...)
}

// snapshot records the loaded state for Restore.
func (l *Level) snapshot() {
	l.pristine = append([]Tile(nil), l.tiles...)
	l.contents0 = append([]Content(nil), l.Contents...)
}

// TileAt returns the tile at a cell. Out-of-range columns read as solid so the
// player cannot leave the level sideways (spec world-collision / 相机边界).
func (l *Level) TileAt(x, y int) Tile {
	if y < 0 || y >= l.Height {
		return Tile{Kind: KindEmpty, Glyph: ' '}
	}
	if x < 0 || x >= l.Width {
		return Tile{Kind: KindSolid, Glyph: 'X'}
	}
	return l.tiles[y*l.Width+x]
}

// SetTile replaces a tile, used when a brick is smashed or a block is used.
func (l *Level) SetTile(x, y int, t Tile) {
	if x < 0 || x >= l.Width || y < 0 || y >= l.Height {
		return
	}
	l.tiles[y*l.Width+x] = t
}

// SolidAt reports whether a cell blocks movement.
func (l *Level) SolidAt(x, y int) bool { return l.TileAt(x, y).Kind.Solid() }

// ContentAt finds the block payload for a cell.
func (l *Level) ContentAt(x, y int) (int, bool) {
	for i := range l.Contents {
		if l.Contents[i].X == x && l.Contents[i].Y == y {
			return i, true
		}
	}
	return -1, false
}

// file is the on-disk shape.
type file struct {
	Name      string     `json:"name"`
	World     string     `json:"world"`
	Width     int        `json:"width"`
	Height    int        `json:"height"`
	TimeLimit int        `json:"timeLimit"`
	Meta      *metaSpec  `json:"meta,omitempty"`
	Style     *styleSpec `json:"style,omitempty"`
	Spawn     point      `json:"spawn"`
	Flag      flagSpec   `json:"flag"`
	Rows      []string   `json:"rows"`
	Contents  []content  `json:"contents"`
	Entities  []entity   `json:"entities"`
}

type metaSpec struct {
	Title       string `json:"title,omitempty"`
	Author      string `json:"author,omitempty"`
	Description string `json:"description,omitempty"`
	Difficulty  int    `json:"difficulty,omitempty"`
}

func (m *metaSpec) empty() bool { return m == nil || *m == metaSpec{} }

type point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type flagSpec struct {
	X       int `json:"x"`
	TopY    int `json:"topY"`
	BottomY int `json:"bottomY"`
}

type content struct {
	X      int          `json:"x"`
	Y      int          `json:"y"`
	Kind   string       `json:"kind"`
	Count  int          `json:"count"`
	Hidden bool         `json:"hidden"`
	Pool   []choiceSpec `json:"pool,omitempty"`
}

// choiceSpec is one pool entry: {"kind": "mushroom", "w": 3}.
type choiceSpec struct {
	Kind   string `json:"kind"`
	Weight int    `json:"w"`
}

type entity struct {
	Kind string `json:"kind"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
}

// LoadFile reads and parses a level from disk.
func LoadFile(path string) (*Level, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read level %s: %w", path, err)
	}
	return Parse(data)
}

// LoadEmbedded parses one of the levels compiled into the binary.
func LoadEmbedded(name string) (*Level, error) {
	data, err := assets.ReadFile("data/" + name + ".json")
	if err != nil {
		return nil, fmt.Errorf("embedded level %s: %w", name, err)
	}
	return Parse(data)
}

// Embedded is the compiled-in level directory, for tools that list or read the
// built-in levels as files (the editor's "open" list).
func Embedded() fs.FS {
	sub, err := fs.Sub(assets, "data")
	if err != nil {
		panic(err) // the directory is embedded; this cannot fail
	}
	return sub
}

// EmbeddedNames lists the compiled-in levels.
func EmbeddedNames() ([]string, error) {
	entries, err := assets.ReadDir("data")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			out = append(out, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	return out, nil
}

// Parse validates raw JSON and builds a Level.
//
// Every rejection names the row and column so a mistyped level file is a
// two-second fix rather than a mystery (spec level-1-1 / 非法数据给出明确错误).
// When a level has several problems Parse reports the first; Check lists them
// all.
func Parse(data []byte) (*Level, error) {
	f, err := decode(data)
	if err != nil {
		return nil, err
	}
	return validate(f)
}

// Problem is one reason a level cannot be loaded. X and Y are the cell it is
// about, or -1 when it is about the level as a whole.
type Problem struct {
	Code    string
	X, Y    int
	Message string
	// Args are the values in the message, by name, so the editor can say
	// the same thing in another language.
	Args map[string]string
}

func (p Problem) Error() string { return p.Message }

// Check lists every reason the level in data cannot be loaded, in the order
// the file declares things. An empty result means Parse accepts it. This is
// the single source the editor's error list, the game's loader and the
// repository test all draw on (spec level-contribution / 单一校验来源).
func Check(data []byte) []Problem {
	f, err := decode(data)
	if err != nil {
		return []Problem{{Code: "json", X: -1, Y: -1, Message: err.Error()}}
	}
	_, probs := build(f)
	return probs
}

func validate(f *file) (*Level, error) {
	l, probs := build(f)
	if len(probs) > 0 {
		return nil, probs[0]
	}
	return l, nil
}

// build checks a decoded file and, when it is sound, builds the level. It keeps
// going after a problem wherever the rest can still be judged, so one save in
// the editor reports everything wrong at once; it stops early only when the
// grid itself is unusable.
func build(f *file) (*Level, []Problem) {
	var probs []Problem
	add := func(code string, x, y int, format string, args ...interface{}) {
		probs = append(probs, Problem{Code: code, X: x, Y: y, Message: fmt.Sprintf(format, args...)})
	}
	// with names the values of the problem just added.
	with := func(kv ...interface{}) {
		p := &probs[len(probs)-1]
		p.Args = map[string]string{}
		for i := 0; i+1 < len(kv); i += 2 {
			p.Args[kv[i].(string)] = fmt.Sprint(kv[i+1])
		}
	}

	if f.Width <= 0 || f.Height <= 0 {
		add("size", -1, -1, "level width and height must be positive (got %dx%d)", f.Width, f.Height)
		with("w", f.Width, "h", f.Height)
		return nil, probs
	}
	if len(f.Rows) != f.Height {
		add("rows", -1, -1, "level declares height %d but has %d rows", f.Height, len(f.Rows))
		with("h", f.Height, "n", len(f.Rows))
		return nil, probs
	}
	if f.Spawn.X < 0 || f.Spawn.X >= f.Width || f.Spawn.Y < 0 || f.Spawn.Y >= f.Height {
		add("spawn", -1, -1, "spawn point (%d,%d) is outside the level %dx%d",
			f.Spawn.X, f.Spawn.Y, f.Width, f.Height)
		with("x", f.Spawn.X, "y", f.Spawn.Y)
	}
	if f.Flag.X < 0 || f.Flag.X >= f.Width {
		add("flag-x", -1, -1, "flagpole x=%d is outside the level width %d", f.Flag.X, f.Width)
		with("x", f.Flag.X)
	}
	if f.Flag.TopY < 0 || f.Flag.BottomY >= f.Height || f.Flag.TopY > f.Flag.BottomY {
		add("flag", f.Flag.X, -1, "flagpole rows %d..%d are invalid for height %d",
			f.Flag.TopY, f.Flag.BottomY, f.Height)
		with("top", f.Flag.TopY, "bottom", f.Flag.BottomY)
	}

	meta, err := validateMeta(f.Meta)
	if err != nil {
		add("meta", -1, -1, "%s", err.Error())
		with("detail", err.Error())
	}
	style, err := validateStyle(f.Style)
	if err != nil {
		add("style", -1, -1, "%s", err.Error())
		with("detail", err.Error())
	}

	gridOK := true
	for y, row := range f.Rows {
		if n := utf8.RuneCountInString(row); n != f.Width {
			add("row-width", -1, y, "row %d has %d columns, want %d", y, n, f.Width)
			with("row", y, "n", n, "want", f.Width)
			gridOK = false
		}
	}
	if f.Width > MaxCells/f.Height {
		add("grid-limit", -1, -1, "level grid exceeds %d cells", MaxCells)
		with("limit", MaxCells)
		return nil, probs
	}

	l := &Level{
		Name:      f.Name,
		World:     f.World,
		Meta:      meta,
		Style:     style,
		Width:     f.Width,
		Height:    f.Height,
		TimeLimit: f.TimeLimit,
		SpawnX:    f.Spawn.X,
		SpawnY:    f.Spawn.Y,
		FlagX:     f.Flag.X,
		FlagTop:   f.Flag.TopY,
		FlagBot:   f.Flag.BottomY,
	}
	if gridOK {
		l.tiles = make([]Tile, f.Width*f.Height)
	}
	if l.TimeLimit == 0 {
		l.TimeLimit = 400
	}

	// declared marks every cell a content entry names, valid or not, so a
	// block whose entry was rejected above is not also reported as having no
	// entry at all.
	declared := map[[2]int]bool{}
	for _, c := range f.Contents {
		declared[[2]int{c.X, c.Y}] = true
	}
	for y, row := range f.Rows {
		if !gridOK && utf8.RuneCountInString(row) != f.Width {
			continue
		}
		x := 0
		for _, ch := range row {
			kind, ok := symbolTable[ch]
			if !ok {
				add("symbol", x, y, "row %d col %d: unknown tile symbol %q", y, x, string(ch))
				with("symbol", string(ch))
				kind = KindEmpty
			}
			if gridOK {
				l.tiles[y*f.Width+x] = Tile{Kind: kind, Glyph: ch}
			}
			x++
		}
	}

	for i, c := range f.Contents {
		if c.X < 0 || c.X >= f.Width || c.Y < 0 || c.Y >= f.Height {
			add("block-outside", -1, -1, "contents[%d]: block (%d,%d) is outside the level", i, c.X, c.Y)
			with("x", c.X, "y", c.Y)
			continue
		}
		kind := BlockKind(c.Kind)
		if !kind.Concrete() && kind != BlockRandom {
			add("block-kind", c.X, c.Y, "contents[%d]: unknown block kind %q", i, c.Kind)
			with("kind", c.Kind)
			continue
		}
		pool, err := validatePool(i, c)
		if err != nil {
			add("block-pool", c.X, c.Y, "%s", err.Error())
			with("detail", err.Error())
			continue
		}
		count := c.Count
		if count <= 0 {
			count = 1
		}
		l.Contents = append(l.Contents, Content{
			X: c.X, Y: c.Y, Kind: kind, Count: count, Hidden: c.Hidden,
			Random: kind == BlockRandom, Pool: pool,
		})
		if !gridOK {
			continue
		}
		// A visible block must actually be a question tile; a hidden one sits in air.
		t := l.TileAt(c.X, c.Y)
		if c.Hidden {
			// hidden blocks are solid only once revealed; keep the air there
			if t.Kind != KindEmpty {
				add("hidden-solid", c.X, c.Y, "contents[%d]: hidden block at (%d,%d) must be placed in empty air, found %q",
					i, c.X, c.Y, string(t.Glyph))
				with("symbol", string(t.Glyph))
			}
		} else if t.Kind != KindQuestion && t.Kind != KindBreakable {
			add("block-tile", c.X, c.Y, "contents[%d]: block at (%d,%d) is %q, not a question or brick tile",
				i, c.X, c.Y, string(t.Glyph))
			with("symbol", string(t.Glyph))
		}
	}

	for i, e := range f.Entities {
		if e.X < 0 || e.X >= f.Width || e.Y < 0 || e.Y >= f.Height {
			add("entity-outside", -1, -1, "entities[%d]: (%d,%d) is outside the level", i, e.X, e.Y)
			with("x", e.X, "y", e.Y)
			continue
		}
		switch e.Kind {
		case "goomba", "koopa":
		default:
			add("entity-kind", e.X, e.Y, "entities[%d]: unknown entity kind %q", i, e.Kind)
			with("kind", e.Kind)
			continue
		}
		l.Spawns = append(l.Spawns, Spawn{Kind: e.Kind, X: e.X, Y: e.Y})
	}

	// Every visible question block must declare what it dispenses. Without
	// this, a missing entry turns into a block that silently does nothing,
	// which is the hardest kind of authoring bug to notice.
	if gridOK {
		for y := 0; y < l.Height; y++ {
			for x := 0; x < l.Width; x++ {
				if l.TileAt(x, y).Kind != KindQuestion || declared[[2]int{x, y}] {
					continue
				}
				if _, ok := l.ContentAt(x, y); !ok {
					add("block-empty", x, y, "row %d col %d: question block has no content entry", y, x)
				}
			}
		}
	}

	if len(probs) > 0 {
		return nil, probs
	}
	l.snapshot()
	return l, nil
}

// validatePool checks a block's random pool. A random block must have one with
// a positive total weight; any other block must not have one, since a pool on a
// fixed block would be silently ignored.
func validatePool(i int, c content) ([]Choice, error) {
	if BlockKind(c.Kind) != BlockRandom {
		if len(c.Pool) > 0 {
			return nil, fmt.Errorf("contents[%d]: block (%d,%d) has a pool but its kind is %q, not \"random\"", i, c.X, c.Y, c.Kind)
		}
		return nil, nil
	}
	if len(c.Pool) == 0 {
		return nil, fmt.Errorf("contents[%d]: random block (%d,%d) has an empty pool", i, c.X, c.Y)
	}
	total := 0
	seen := map[BlockKind]bool{}
	out := make([]Choice, 0, len(c.Pool))
	for j, ch := range c.Pool {
		k := BlockKind(ch.Kind)
		if !k.Concrete() {
			return nil, fmt.Errorf("contents[%d]: random block (%d,%d) pool[%d] has unknown kind %q", i, c.X, c.Y, j, ch.Kind)
		}
		if ch.Weight < 0 {
			return nil, fmt.Errorf("contents[%d]: random block (%d,%d) pool[%d] has negative weight %d", i, c.X, c.Y, j, ch.Weight)
		}
		if seen[k] {
			return nil, fmt.Errorf("contents[%d]: random block (%d,%d) lists %q twice", i, c.X, c.Y, ch.Kind)
		}
		seen[k] = true
		total += ch.Weight
		out = append(out, Choice{Kind: k, Weight: ch.Weight})
	}
	if total == 0 {
		return nil, fmt.Errorf("contents[%d]: random block (%d,%d) has weights that add up to 0", i, c.X, c.Y)
	}
	return out, nil
}

// validateMeta checks the optional description block. A level without one is
// fine; a level with one must keep each field within its limit.
func validateMeta(m *metaSpec) (Meta, error) {
	if m == nil {
		return Meta{}, nil
	}
	limits := []struct {
		field string
		value string
		max   int
	}{
		{"title", m.Title, MaxTitleLen},
		{"author", m.Author, MaxAuthorLen},
		{"description", m.Description, MaxDescriptionLen},
	}
	for _, f := range limits {
		if n := len([]rune(f.value)); n > f.max {
			return Meta{}, fmt.Errorf("meta.%s is %d characters long, the limit is %d", f.field, n, f.max)
		}
	}
	if m.Difficulty < 0 || m.Difficulty > MaxDifficulty {
		return Meta{}, fmt.Errorf("meta.difficulty is %d, it must be between 1 and %d", m.Difficulty, MaxDifficulty)
	}
	return Meta{Title: m.Title, Author: m.Author, Description: m.Description, Difficulty: m.Difficulty}, nil
}

// TileRect returns the world-space rectangle of a cell.
func TileRect(x, y int) engine.Rect {
	return engine.Rect{
		X: engine.Fixed(x) * engine.Tile,
		Y: engine.Fixed(y) * engine.Tile,
		W: engine.Tile,
		H: engine.Tile,
	}
}
