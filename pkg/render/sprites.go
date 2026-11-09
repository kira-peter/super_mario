package render

import "super-mario/pkg/render/theme"

// Sprite is a small bitmap authored with one character per art pixel.
//
// The characters are palette slots, not colours: 'R' for the body red, 'K' for
// the outline, and so on. Rendering maps a slot to a theme token, which is what
// keeps colour decisions out of the art (spec terminal-renderer / 令牌取色).
type Sprite struct {
	W, H int
	Rows []string
}

// Paints are the slots a sprite may use.
const (
	SlotNone     = '.'
	SlotOutline  = 'K'
	SlotSkin     = 'S'
	SlotBody     = 'R'
	SlotAccent   = 'B'
	SlotWhite    = 'W'
	SlotCoin     = 'C'
	SlotCoinHi   = 'M'
	SlotEnemy    = 'T'
	SlotEnemyAcc = 'G'
	SlotShoe     = 'H'
	SlotYellow   = 'Y'
)

func sprite(rows ...string) Sprite {
	w := 0
	for _, r := range rows {
		if len(r) > w {
			w = len(r)
		}
	}
	padded := make([]string, len(rows))
	for i, r := range rows {
		for len(r) < w {
			r += "."
		}
		padded[i] = r
	}
	return Sprite{W: w, H: len(rows), Rows: padded}
}

// Goomba walks left by default.
var Goomba = sprite(
	"...KKKKKKKKK....",
	"..KTTTTTTTTTK...",
	".KTTTTTTTTTTTK..",
	"KTTWWTTTTTTWWTTK",
	"KTTWKTTTTTTKWTTK",
	"KTTTTTTTTTTTTTTK",
	"KTTTTKKKKKKTTTTK",
	".KTTTTTTTTTTTTK.",
	"..KTTTTTTTTTTK..",
	"...KKTTTTTTKK...",
	"....KKKKKKKK....",
	"...TTTTTTTTTT...",
	"..TTTTTTTTTTTT..",
	"..HHHH....HHHH..",
	".HHHHH....HHHHH.",
)

// GoombaFlat is the squashed frame shown for a moment after a stomp.
var GoombaFlat = sprite(
	"................",
	"................",
	"................",
	"................",
	"................",
	"................",
	"................",
	"................",
	"................",
	"................",
	"..KKKKKKKKKKKK..",
	".KTTWWTTTTWWTTK.",
	"KTTTTTTTTTTTTTTK",
	".KTTKKKKKKKKTTK.",
	"..HHHHHHHHHHHH..",
)

// Koopa stands taller than a Goomba, which is what keeps the two readable in a
// monochrome terminal where colour carries no information.
var Koopa = sprite(
	"......GGGG......",
	".....GGGGGG.....",
	"....KKGGGGGK....",
	"...KSSKGGGGK....",
	"...KSSSSGGGK....",
	"...KSSKGGGGK....",
	"....KKGGGGGK....",
	"....GGGGGGGG....",
	"...GGKGGGGKGG...",
	"..GGGGGGGGGGGG..",
	"..GGKGGGGGGKGG..",
	"..GGGGGGGGGGGG..",
	"..GGKGGGGGGKGG..",
	"..GGGGGGGGGGGG..",
	"...GGKGGGGKGG...",
	"....GGGGGGGG....",
	"....HHHHHHHH....",
	"...HHH....HHH...",
	"..HHHH....HHHH..",
)

// Shell is the 16x12 shell a stomped Koopa turns into.
var Shell = sprite(
	"....KKKKKKKK....",
	"..KKGGGGGGGGKK..",
	".KGGKGGGGGGKGGK.",
	"KGGGKGGGGGGKGGGK",
	"KGGGGGGGGGGGGGGK",
	"KGGKGGGGGGGGKGGK",
	"KGGGGGGGGGGGGGGK",
	"KGGKGGGGGGGGKGGK",
	"KGGGGGGGGGGGGGGK",
	".KGGKGGGGGGKGGK.",
	"..KKGGGGGGGGKK..",
	"....KKKKKKKK....",
)

// Mushroom is the grow pickup.
var Mushroom = sprite(
	".....KKKKKK.....",
	"...KKRRRRRRKK...",
	"..KRRWWRRWWRRK..",
	".KRRWWWRRWWWRRK.",
	".KRWWWWWWWWWWRK.",
	".KRRWWWWWWWWRRK.",
	".KRRRWWWWWWRRRK.",
	"..KRRRRRRRRRRK..",
	"...KKSSSSSSKK...",
	"....KSSSSSSK....",
	"....KSSSSSSK....",
	"....KSSSSSSK....",
	"....KSSSSSSK....",
	".....KKKKKK.....",
)

// Star is the invincibility pickup.
var Star = sprite(
	".......KK.......",
	"......KCCK......",
	"......KCCK......",
	".....KCCCK......",
	"KKKKKKCCCKKKKKKK",
	"KCCCCCCCCCCCCCCK",
	".KCCCCCCCCCCCCK.",
	"..KCCCCCCCCCCK..",
	"...KCCCCCCCCK...",
	"...KCCCKKCCCK...",
	"..KCCCK..KCCCK..",
	".KCCCK....KCCCK.",
	".KCCK......KCCK.",
	".KKK........KKK.",
)

// CoinFrames is the four-frame spin. The width changes to sell the rotation.
var CoinFrames = []Sprite{
	sprite(
		"....CCCC....",
		"..CCCCCCCC..",
		".CCCMMMMCC..",
		".CCMMCCMMCC.",
		".CCMMCCMMCC.",
		".CCMMCCMMCC.",
		".CCMMCCMMCC.",
		".CCMMCCMMCC.",
		".CCMMCCMMCC.",
		".CCCMMMMCC..",
		"..CCCCCCCC..",
		"....CCCC....",
	),
	sprite(
		"...CCCC...",
		".CCCCCCCC.",
		".CCMMMMCC.",
		".CCMCCMCC.",
		".CCMCCMCC.",
		".CCMCCMCC.",
		".CCMCCMCC.",
		".CCMCCMCC.",
		".CCMCCMCC.",
		".CCMMMMCC.",
		".CCCCCCCC.",
		"...CCCC...",
	),
	sprite(
		"....CC....",
		"...CCCC...",
		"...CCCC...",
		"...CCCC...",
		"...CMCC...",
		"...CMCC...",
		"...CMCC...",
		"...CMCC...",
		"...CCCC...",
		"...CCCC...",
		"...CCCC...",
		"....CC....",
	),
	sprite(
		"...CCCC...",
		".CCCCCCCC.",
		".CCCCMMCC.",
		".CCMCCMCC.",
		".CCMCCMCC.",
		".CCMCCMCC.",
		".CCMCCMCC.",
		".CCMCCMCC.",
		".CCMCCMCC.",
		".CCCCMMCC.",
		".CCCCCCCC.",
		"...CCCC...",
	),
}

// SlotColor maps a sprite slot onto a theme token.
func SlotColor(t theme.Tokens, slot byte) (interface{}, bool) {
	switch slot {
	case SlotNone:
		return nil, false
	case SlotOutline:
		return t.Outline, true
	case SlotSkin:
		return t.PlayerSkin, true
	case SlotBody:
		return t.PlayerBody, true
	case SlotAccent:
		return t.PlayerAccent, true
	case SlotWhite:
		return t.HudText, true
	case SlotCoin:
		return t.Coin, true
	case SlotCoinHi:
		return t.CoinHi, true
	case SlotEnemy:
		return t.EnemyBody, true
	case SlotEnemyAcc:
		return t.EnemyAccent, true
	case SlotShoe:
		return t.PlayerShoe, true
	case SlotYellow:
		return t.Star, true
	case SlotHeroWheel:
		return t.HeroWheel, true
	case SlotHeroMark:
		return t.HeroMark, true
	}
	return nil, false
}

// NewSprite builds a sprite from rows of slot letters, padding short rows so
// the sprite is rectangular. It is the exported form of the package's own
// constructor, for art that lives outside this package, such as the title logo.
func NewSprite(rows ...string) Sprite { return sprite(rows...) }
