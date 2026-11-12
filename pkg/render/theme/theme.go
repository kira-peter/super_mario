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

	// GroundDeep is the dark soil under the bright crust. BrickMortar is the
	// seam between bricks, deliberately lighter than Outline: at four pixels to
	// a tile a dark outline takes more area than the brick it surrounds.
	GroundDeep  tcell.Color
	BrickMortar tcell.Color
	// StoneLight, Stone and StoneDark are the three tones of a stair block: lit
	// top-left, flat middle, shaded bottom-right. It is grey so a stair reads as
	// a different material from the earth floor beside it.
	StoneLight tcell.Color
	Stone      tcell.Color
	StoneDark  tcell.Color
	// QuestionGlint is the pale gold a question block pulses to.
	QuestionGlint tcell.Color

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
	Coin   tcell.Color
	CoinHi tcell.Color
	// CoinShade is the shaded lower-right edge of a coin.
	CoinShade  tcell.Color
	Mushroom   tcell.Color
	Star       tcell.Color
	StarAccent tcell.Color

	// Cast is the palette of the hand-drawn four-pixel characters.
	Cast Cast

	// Full is true when the terminal renders the palette as authored.
	Full bool
}

// Cast holds the colours of the mushrooms, the enemies and the flag.
//
// Every mushroom has the same shape, so its cap colour is the only thing that
// says what it does; each cap has a darker rim under it for depth.
type Cast struct {
	ShroomRed, ShroomRedRim       tcell.Color
	ShroomGreen, ShroomGreenRim   tcell.Color
	ShroomBlue, ShroomBlueRim     tcell.Color
	ShroomPurple, ShroomPurpleRim tcell.Color
	ShroomGold, ShroomGoldRim     tcell.Color
	ShroomSpot, ShroomStem        tcell.Color

	GoombaBody, GoombaEye, GoombaFeet tcell.Color

	KoopaSkin, KoopaShell, KoopaShellHi, KoopaRim tcell.Color

	FlagPole, FlagBall, FlagBallShade, FlagDark, FlagLight tcell.Color

	// CoinPlus is the green "+1" shown at the coin counter.
	CoinPlus tcell.Color

	// Cloud and CloudShade are the drifting clouds and their undersides.
	Cloud, CloudShade tcell.Color

	// The castle at the end of the level. Its windows are dark while the level
	// is played and light up, one solid warm colour, once the pole is reached.
	CastleWall, CastleWallDark, CastleStoneLight tcell.Color
	CastleRoof, CastleRoofTile, CastleOutline    tcell.Color
	CastleWindow, CastleWindowDark               tcell.Color
	// CastleDoorOpen is the doorway once the pole is grabbed: the door has
	// swung open and the inside is black.
	CastleDoor, CastleDoorDark, CastleDoorOpen tcell.Color
	CastleFlag, CastleFlagPole                 tcell.Color

	// Grass and GrassDark are the turf on top of the ground.
	Grass, GrassDark tcell.Color
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

	GroundDeep:    hex(0x7f4d22),
	BrickMortar:   hex(0x7a4a2e),
	StoneLight:    hex(0xb7bcc9),
	Stone:         hex(0x8a909e),
	StoneDark:     hex(0x5b606d),
	QuestionGlint: hex(0xfff3b0),

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
	CoinShade:  hex(0xc9951f),
	Mushroom:   hex(0xd94a3d),
	Star:       hex(0xf5c842),
	StarAccent: hex(0xfff3b0),

	Cast: Cast{
		ShroomRed: hex(0xd94a3d), ShroomRedRim: hex(0x8f2a22),
		ShroomGreen: hex(0x4caf50), ShroomGreenRim: hex(0x2f7a33),
		ShroomBlue: hex(0x3d7fe0), ShroomBlueRim: hex(0x1f4a99),
		ShroomPurple: hex(0xa056d6), ShroomPurpleRim: hex(0x5e2a85),
		ShroomGold: hex(0xf5c842), ShroomGoldRim: hex(0xc9951f),
		ShroomSpot: hex(0xf2f4f8), ShroomStem: hex(0xf0d8b0),

		GoombaBody: hex(0xa0602e), GoombaEye: hex(0xf2f4f8), GoombaFeet: hex(0xe6b07a),

		KoopaSkin: hex(0xf0c860), KoopaShell: hex(0x3f9a3f), KoopaShellHi: hex(0x8fd46e), KoopaRim: hex(0xf2f4f8),

		FlagPole: hex(0xb7bcc9), FlagBall: hex(0xf5c842), FlagBallShade: hex(0xc9951f),
		FlagDark: hex(0x1a1a1a), FlagLight: hex(0xf2f4f8),
		CoinPlus: hex(0x5fd35f),
		Cloud:    hex(0xffffff), CloudShade: hex(0xcfdcef),
		CastleWall: hex(0x9a9a9a), CastleWallDark: hex(0x5f5f5f), CastleStoneLight: hex(0xc4c4c4),
		CastleRoof: hex(0xc4472c), CastleRoofTile: hex(0x8a2b1a), CastleOutline: hex(0x262626),
		CastleWindow: hex(0xffc35a), CastleWindowDark: hex(0x1c1c1c),
		CastleDoor: hex(0x7a4a22), CastleDoorDark: hex(0x4a2a12), CastleDoorOpen: hex(0x000000),
		CastleFlag: hex(0xd43b2b), CastleFlagPole: hex(0x8a8a8a),
		Grass: hex(0x58a64b), GrassDark: hex(0x3f7f3a),
	},

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

	GroundDeep:    hex(0x8f5a28),
	BrickMortar:   hex(0x8a4a24),
	StoneLight:    hex(0xc4c8d2),
	Stone:         hex(0x9aa0ac),
	StoneDark:     hex(0x6a6f7c),
	QuestionGlint: hex(0xf2d880),

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
	CoinShade:  hex(0xa07a10),
	Mushroom:   hex(0xc0392b),
	Star:       hex(0xe0b020),
	StarAccent: hex(0xf0d060),

	// On a near-white sky the white parts need a grey to stay visible.
	Cast: Cast{
		ShroomRed: hex(0xc0392b), ShroomRedRim: hex(0x7a2018),
		ShroomGreen: hex(0x3a9a40), ShroomGreenRim: hex(0x24662a),
		ShroomBlue: hex(0x2f6fd0), ShroomBlueRim: hex(0x1a3f88),
		ShroomPurple: hex(0x8a40c0), ShroomPurpleRim: hex(0x4e2272),
		ShroomGold: hex(0xd0a020), ShroomGoldRim: hex(0x9a7410),
		ShroomSpot: hex(0xffffff), ShroomStem: hex(0xd8b888),

		GoombaBody: hex(0x8a5024), GoombaEye: hex(0xffffff), GoombaFeet: hex(0xc08850),

		KoopaSkin: hex(0xd0a030), KoopaShell: hex(0x2f8a34), KoopaShellHi: hex(0x6ab85a), KoopaRim: hex(0xb8bcc4),

		FlagPole: hex(0x8a909e), FlagBall: hex(0xd0a020), FlagBallShade: hex(0x9a7410),
		FlagDark: hex(0x1c1d20), FlagLight: hex(0xd8dbe0),
		CoinPlus: hex(0x2a9a3a),
		Cloud:    hex(0xdfe5ee), CloudShade: hex(0xc4ccd8),
		CastleWall: hex(0x9a9ea6), CastleWallDark: hex(0x6a6f78), CastleStoneLight: hex(0xc4c8d0),
		CastleRoof: hex(0xb03a24), CastleRoofTile: hex(0x7a2416), CastleOutline: hex(0x3a3c42),
		CastleWindow: hex(0xe8a032), CastleWindowDark: hex(0x3a3c42),
		CastleDoor: hex(0x6a4020), CastleDoorDark: hex(0x40240e), CastleDoorOpen: hex(0x000000),
		CastleFlag: hex(0xc0322a), CastleFlagPole: hex(0x6a6f78),
		Grass: hex(0x4a9a40), GrassDark: hex(0x2f7030),
	},

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

	// Without colour the earth floor and the stone stairs must still differ,
	// which colour alone no longer does. The soil goes to the darkest grey the
	// terminal offers and the stone keeps its three tones, so a stair is lighter
	// than the floor it stands on.
	t.GroundDeep = tcell.ColorDarkGray
	t.BrickMortar = tcell.ColorDarkGray
	t.StoneLight = tcell.ColorWhite
	t.Stone = tcell.ColorSilver
	t.StoneDark = tcell.ColorGray
	t.QuestionGlint = tcell.ColorWhite

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
	t.CoinShade = tcell.ColorSilver
	t.Mushroom = tcell.ColorWhite
	t.Star = tcell.ColorWhite
	t.StarAccent = tcell.ColorWhite

	// Without colour the mushrooms can only differ in brightness, so each cap
	// gets its own grey and its rim the next one down.
	cs := &t.Cast
	cs.ShroomRed, cs.ShroomRedRim = tcell.ColorSilver, tcell.ColorGray
	cs.ShroomGreen, cs.ShroomGreenRim = tcell.ColorGray, tcell.ColorDarkGray
	cs.ShroomBlue, cs.ShroomBlueRim = tcell.ColorWhite, tcell.ColorSilver
	cs.ShroomPurple, cs.ShroomPurpleRim = tcell.ColorDarkGray, tcell.ColorBlack
	cs.ShroomGold, cs.ShroomGoldRim = tcell.ColorWhite, tcell.ColorGray
	cs.ShroomSpot, cs.ShroomStem = tcell.ColorWhite, tcell.ColorSilver
	cs.GoombaBody, cs.GoombaEye, cs.GoombaFeet = tcell.ColorGray, tcell.ColorWhite, tcell.ColorSilver
	cs.KoopaSkin, cs.KoopaShell, cs.KoopaShellHi, cs.KoopaRim = tcell.ColorSilver, tcell.ColorGray, tcell.ColorSilver, tcell.ColorWhite
	cs.FlagPole, cs.FlagBall, cs.FlagBallShade = tcell.ColorSilver, tcell.ColorWhite, tcell.ColorGray
	cs.FlagDark, cs.FlagLight = tcell.ColorBlack, tcell.ColorWhite
	cs.CoinPlus = tcell.ColorWhite
	// Clouds are white with a silver underside, as in colour; grass is a step
	// lighter than the dark soil under it.
	cs.Cloud, cs.CloudShade = tcell.ColorWhite, tcell.ColorSilver
	// The castle keeps its shape in grey. The windows go to white, the brightest
	// grey there is, so even without colour they read as lit.
	cs.CastleWall, cs.CastleWallDark, cs.CastleStoneLight = tcell.ColorGray, tcell.ColorDarkGray, tcell.ColorSilver
	cs.CastleRoof, cs.CastleRoofTile, cs.CastleOutline = tcell.ColorSilver, tcell.ColorGray, tcell.ColorBlack
	cs.CastleWindow, cs.CastleWindowDark = tcell.ColorWhite, tcell.ColorBlack
	// Closed, the door is a mid grey so it shows against the black outline;
	// open, it is black.
	cs.CastleDoor, cs.CastleDoorDark, cs.CastleDoorOpen = tcell.ColorGray, tcell.ColorDarkGray, tcell.ColorBlack
	cs.CastleFlag, cs.CastleFlagPole = tcell.ColorWhite, tcell.ColorGray
	cs.Grass, cs.GrassDark = tcell.ColorSilver, tcell.ColorGray

	t.Full = false
	return t
}
