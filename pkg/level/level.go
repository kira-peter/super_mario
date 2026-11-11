// Package level parses and validates declarative level data.
//
// A level file is plain JSON describing a grid of tiles plus a list of entity
// spawns. It contains no expressions and no code, so a level can be edited
// without touching or rebuilding the game (spec level-1-1 / 关卡数据格式).
package level

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
)

// Content describes a block's payload, fixed at authoring time so a given block
// always yields the same thing (spec blocks-and-items / 问号块).
type Content struct {
	X, Y   int
	Kind   BlockKind
	Count  int  // remaining uses; a coin brick may hold several
	Hidden bool // invisible until bumped
}

// Spawn is an entity placed at level load.
type Spawn struct {
	Kind string // "goomba", "koopa"
	X, Y int
}

// Level is a parsed, validated level.
type Level struct {
	Name      string
	World     string
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
	Name      string    `json:"name"`
	World     string    `json:"world"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	TimeLimit int       `json:"timeLimit"`
	Spawn     point     `json:"spawn"`
	Flag      flagSpec  `json:"flag"`
	Rows      []string  `json:"rows"`
	Contents  []content `json:"contents"`
	Entities  []entity  `json:"entities"`
}

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
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Kind   string `json:"kind"`
	Count  int    `json:"count"`
	Hidden bool   `json:"hidden"`
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
func Parse(data []byte) (*Level, error) {
	var f file
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("parse level json: %w", err)
	}
	return validate(&f)
}

func validate(f *file) (*Level, error) {
	if f.Width <= 0 || f.Height <= 0 {
		return nil, fmt.Errorf("level width and height must be positive (got %dx%d)", f.Width, f.Height)
	}
	if len(f.Rows) != f.Height {
		return nil, fmt.Errorf("level declares height %d but has %d rows", f.Height, len(f.Rows))
	}
	if f.Spawn.X < 0 || f.Spawn.X >= f.Width || f.Spawn.Y < 0 || f.Spawn.Y >= f.Height {
		return nil, fmt.Errorf("spawn point (%d,%d) is outside the level %dx%d",
			f.Spawn.X, f.Spawn.Y, f.Width, f.Height)
	}
	if f.Flag.X < 0 || f.Flag.X >= f.Width {
		return nil, fmt.Errorf("flagpole x=%d is outside the level width %d", f.Flag.X, f.Width)
	}
	if f.Flag.TopY < 0 || f.Flag.BottomY >= f.Height || f.Flag.TopY > f.Flag.BottomY {
		return nil, fmt.Errorf("flagpole rows %d..%d are invalid for height %d",
			f.Flag.TopY, f.Flag.BottomY, f.Height)
	}

	l := &Level{
		Name:      f.Name,
		World:     f.World,
		Width:     f.Width,
		Height:    f.Height,
		TimeLimit: f.TimeLimit,
		SpawnX:    f.Spawn.X,
		SpawnY:    f.Spawn.Y,
		FlagX:     f.Flag.X,
		FlagTop:   f.Flag.TopY,
		FlagBot:   f.Flag.BottomY,
		tiles:     make([]Tile, f.Width*f.Height),
	}
	if l.TimeLimit == 0 {
		l.TimeLimit = 400
	}

	for y, row := range f.Rows {
		runes := []rune(row)
		if len(runes) != f.Width {
			return nil, fmt.Errorf("row %d has %d columns, want %d", y, len(runes), f.Width)
		}
		for x, ch := range runes {
			kind, ok := symbolTable[ch]
			if !ok {
				return nil, fmt.Errorf("row %d col %d: unknown tile symbol %q", y, x, string(ch))
			}
			l.tiles[y*f.Width+x] = Tile{Kind: kind, Glyph: ch}
		}
	}

	for i, c := range f.Contents {
		if c.X < 0 || c.X >= f.Width || c.Y < 0 || c.Y >= f.Height {
			return nil, fmt.Errorf("contents[%d]: block (%d,%d) is outside the level", i, c.X, c.Y)
		}
		switch BlockKind(c.Kind) {
		case BlockCoin, BlockMushroom, BlockStar, BlockOneUp, BlockPoison, BlockGold:
		default:
			return nil, fmt.Errorf("contents[%d]: unknown block kind %q", i, c.Kind)
		}
		count := c.Count
		if count <= 0 {
			count = 1
		}
		l.Contents = append(l.Contents, Content{
			X: c.X, Y: c.Y, Kind: BlockKind(c.Kind), Count: count, Hidden: c.Hidden,
		})
		// A visible block must actually be a question tile; a hidden one sits in air.
		t := l.TileAt(c.X, c.Y)
		if c.Hidden {
			// hidden blocks are solid only once revealed; keep the air there
			if t.Kind != KindEmpty {
				return nil, fmt.Errorf("contents[%d]: hidden block at (%d,%d) must be placed in empty air, found %q",
					i, c.X, c.Y, string(t.Glyph))
			}
		} else if t.Kind != KindQuestion && t.Kind != KindBreakable {
			return nil, fmt.Errorf("contents[%d]: block at (%d,%d) is %q, not a question or brick tile",
				i, c.X, c.Y, string(t.Glyph))
		}
	}

	for i, e := range f.Entities {
		if e.X < 0 || e.X >= f.Width || e.Y < 0 || e.Y >= f.Height {
			return nil, fmt.Errorf("entities[%d]: (%d,%d) is outside the level", i, e.X, e.Y)
		}
		switch e.Kind {
		case "goomba", "koopa":
		default:
			return nil, fmt.Errorf("entities[%d]: unknown entity kind %q", i, e.Kind)
		}
		l.Spawns = append(l.Spawns, Spawn{Kind: e.Kind, X: e.X, Y: e.Y})
	}

	// Every visible question block must declare what it dispenses. Without
	// this, a missing entry turns into a block that silently does nothing,
	// which is the hardest kind of authoring bug to notice.
	for y := 0; y < l.Height; y++ {
		for x := 0; x < l.Width; x++ {
			if l.TileAt(x, y).Kind != KindQuestion {
				continue
			}
			if _, ok := l.ContentAt(x, y); !ok {
				return nil, fmt.Errorf("row %d col %d: question block has no content entry", y, x)
			}
		}
	}

	l.snapshot()
	return l, nil
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
