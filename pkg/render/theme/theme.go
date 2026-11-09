// Package theme holds the game's semantic colour tokens and the terminal
// capability model.
//
// Design decision 8: views never name a colour. They ask for a token such as
// Ground or Question, and the theme decides what that means on this terminal.
// The tokens are derived from the same ink/paper bases the sibling link_chat
// project uses, so both applications read as one family without this one
// importing that one.
package theme

import "github.com/gdamore/tcell/v2"

// Capacity describes how much colour the terminal can show.
type Capacity int

const (
	// Mono is a terminal with no colour.
	Mono Capacity = iota
	// Color256 is an eight-bit terminal.
	Color256
	// TrueColor is a 24-bit terminal.
	TrueColor
)

// Tokens is one resolved palette.
type Tokens struct {
	Name string

	// Environment.
	Sky      tcell.Color
	Backdrop tcell.Color
	HudText  tcell.Color
	HudMuted tcell.Color

	// Terrain.
	Ground     tcell.Color
	GroundEdge tcell.Color
	Brick      tcell.Color
	BrickEdge  tcell.Color
	Question   tcell.Color
	Used       tcell.Color
	Pipe       tcell.Color
	PipeHi     tcell.Color
	PipeLo     tcell.Color

	// Characters.
	PlayerBody   tcell.Color
	PlayerAccent tcell.Color
	PlayerSkin   tcell.Color
	PlayerShoe   tcell.Color
	Outline      tcell.Color

	// HeroWheel is the rim of the wheel under the hero, and HeroMark the dot on
	// it that shows which way it is turning. The mark has to stand apart from
	// the rim and from everything behind it, since it is the only thing that
	// tells the wheel is turning at all.
	HeroWheel tcell.Color
	HeroMark  tcell.Color

	EnemyBody   tcell.Color
	EnemyAccent tcell.Color

	// Pickups.
	Coin       tcell.Color
	CoinHi     tcell.Color
	Mushroom   tcell.Color
	Star       tcell.Color
	StarAccent tcell.Color

	// Full is true when the terminal renders the palette as authored.
	Full bool
}

func hex(v int32) tcell.Color { return tcell.NewHexColor(v) }

// Ink is the dark theme. It is the default.
//
// The values are deliberately desaturated: a terminal's glyphs sit directly on
// a near-black ground, so fully saturated primaries smear and vibrate. Each
// gameplay colour keeps at least a 2:1 luminance contrast against the sky so
// sprites read at a glance.
var Ink = Tokens{
	Name: "ink",

	Sky:      hex(0x12141a),
	Backdrop: hex(0x1c2028),
	HudText:  hex(0xe4e7ee),
	HudMuted: hex(0x9aa3b2),

	Ground:     hex(0xa8692f),
	GroundEdge: hex(0xc98a4b),
	Brick:      hex(0xa8643c),
	BrickEdge:  hex(0xc98a4b),
	Question:   hex(0xe0b060),
	Used:       hex(0x7a5a22),
	Pipe:       hex(0x3f9a52),
	PipeHi:     hex(0x6fcf7f),
	PipeLo:     hex(0x2d7a3e),

	PlayerBody:   hex(0xd94a3d),
	PlayerAccent: hex(0x3a5fc8),
	PlayerSkin:   hex(0xf0b088),
	PlayerShoe:   hex(0x8a5a30),
	Outline:      hex(0x4a3220),
	HeroWheel:    hex(0x6b7280),
	HeroMark:     hex(0xf2f4f8),

	EnemyBody:   hex(0xa8692f),
	EnemyAccent: hex(0x4aa848),

	Coin:       hex(0xf5c842),
	CoinHi:     hex(0xfff3b0),
	Mushroom:   hex(0xd94a3d),
	Star:       hex(0xf5c842),
	StarAccent: hex(0xfff3b0),

	Full: true,
}

// Paper is the light theme, for terminals with a light background.
var Paper = Tokens{
	Name: "paper",

	Sky:      hex(0xfbfbfa),
	Backdrop: hex(0xe9ebe8),
	HudText:  hex(0x23262b),
	HudMuted: hex(0x5d646f),

	Ground:     hex(0xb07830),
	GroundEdge: hex(0x8a5a20),
	Brick:      hex(0xb06a3a),
	BrickEdge:  hex(0x8a4a20),
	Question:   hex(0xd8a030),
	Used:       hex(0xa08850),
	Pipe:       hex(0x3a8a48),
	PipeHi:     hex(0x5aa868),
	PipeLo:     hex(0x286a34),

	PlayerBody:   hex(0xc0392b),
	PlayerAccent: hex(0x2f4fa8),
	PlayerSkin:   hex(0xe0a080),
	PlayerShoe:   hex(0x6a4020),
	Outline:      hex(0x2a1a10),
	HeroWheel:    hex(0x8b93a1),
	HeroMark:     hex(0x1c1d20),

	EnemyBody:   hex(0x906030),
	EnemyAccent: hex(0x3a8a40),

	Coin:       hex(0xd0a020),
	CoinHi:     hex(0xf0d060),
	Mushroom:   hex(0xc0392b),
	Star:       hex(0xe0b020),
	StarAccent: hex(0xf0d060),

	Full: true,
}

// Registry maps a theme name to its palette.
var Registry = map[string]Tokens{
	"ink":   Ink,
	"paper": Paper,
}

// DefaultName is used when a configured theme is missing.
const DefaultName = "ink"

// Resolve returns a palette by name, falling back rather than failing to start.
func Resolve(name string) (Tokens, bool) {
	if t, ok := Registry[name]; ok {
		return t, true
	}
	return Registry[DefaultName], false
}

// Degrade folds a terminal's capability into the palette: 256-colour terminals
// get the same hues quantised by tcell, and mono terminals get three greys so
// structure survives without colour.
//
// This is the only place that decides what "no colour" means
// (spec terminal-renderer / 色彩能力降级).
func Degrade(t Tokens, c Capacity) Tokens {
	if c == TrueColor {
		return t
	}
	if c == Color256 {
		// tcell quantises hex colours to the 256-colour cube on its own.
		return t
	}
	// Mono: three luminance levels. Anything that must stay distinguishable
	// has to differ in shape, not only here.
	t.Sky = tcell.ColorDefault
	t.Backdrop = tcell.ColorDefault
	t.HudText = tcell.ColorWhite
	t.HudMuted = tcell.ColorGray

	t.Ground = tcell.ColorGray
	t.GroundEdge = tcell.ColorSilver
	t.Brick = tcell.ColorGray
	t.BrickEdge = tcell.ColorSilver
	t.Question = tcell.ColorWhite
	t.Used = tcell.ColorGray
	t.Pipe = tcell.ColorGray
	t.PipeHi = tcell.ColorSilver
	t.PipeLo = tcell.ColorGray

	t.PlayerBody = tcell.ColorWhite
	t.PlayerAccent = tcell.ColorSilver
	t.PlayerSkin = tcell.ColorWhite
	t.PlayerShoe = tcell.ColorGray
	t.Outline = tcell.ColorBlack
	t.HeroWheel = tcell.ColorGray
	t.HeroMark = tcell.ColorWhite

	t.EnemyBody = tcell.ColorSilver
	t.EnemyAccent = tcell.ColorGray

	t.Coin = tcell.ColorWhite
	t.CoinHi = tcell.ColorWhite
	t.Mushroom = tcell.ColorWhite
	t.Star = tcell.ColorWhite
	t.StarAccent = tcell.ColorWhite

	t.Full = false
	return t
}
