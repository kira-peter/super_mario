package render

// SpriteSet is one complete cast at a given art resolution.
//
// Two sets exist because the terminal has two very different jobs to do. The
// detailed set is 16 art pixels per tile, which is enough to draw Mario's hat,
// nose, moustache and overalls separately. The compact set is 8, which is the
// smallest size at which a hat and a pair of legs still read as a person.
//
// The choice is not cosmetic: a square tile needs art pixels per tile equal to
// terminal columns per tile, so the set fixes how much of the level fits on
// screen. Sixteen shows a close view with plenty of detail; eight shows nearly
// three times the area, and the player can finally see a gap before jumping it.
type SpriteSet struct {
	Name string
	// Px is the art resolution: art pixels per tile.
	Px int

	Goomba     Sprite
	GoombaFlat Sprite
	Koopa      Sprite
	Shell      Sprite

	Mushroom Sprite
	Star     Sprite
	Coin     []Sprite
}

// Detailed is the 16-pixel set used for the close-up view.
var Detailed = SpriteSet{
	Name:       "detailed",
	Px:         16,
	Goomba:     Goomba,
	GoombaFlat: GoombaFlat,
	Koopa:      Koopa,
	Shell:      Shell,
	Mushroom:   Mushroom,
	Star:       Star,
	Coin:       CoinFrames,
}

// Compact is the 8-pixel set used for the wide view.
var Compact = SpriteSet{
	Name:       "compact",
	Px:         8,
	Goomba:     Goomba8,
	GoombaFlat: Goomba8Flat,
	Koopa:      Koopa8,
	Shell:      Shell8,
	Mushroom:   Mushroom8,
	Star:       Star8,
	Coin:       Coin8Frames,
}

// SetFor returns the sprite set matching an art resolution.
//
// The compact set is used for anything at or below its own resolution; above
// that the detailed set is used and the canvas downsamples it, which is how a
// mid-size view stays sharp without a third hand-drawn cast.
func SetFor(artPerTile int) SpriteSet {
	if artPerTile <= Compact.Px {
		return Compact
	}
	return Detailed
}

// --- compact cast, 8 art pixels per tile ---
//
// At this size there is one pixel per feature: a hat, an eye, a strap, a pair
// of shoes. Anything finer turns to noise, so the shapes are simplified down to
// exactly those four signals rather than being a shrunken copy of the detailed
// art.

// Goomba8 keeps the wide cap and the short feet, which is the silhouette that
// still tells it apart from a Koopa when the terminal has no colour.
var Goomba8 = sprite(
	".KKKKK..",
	"KTTTTTK.",
	"KTWKTWK.",
	"KTTTTTK.",
	"KTTTTTK.",
	".KTTTK..",
	".HHHHH..",
	".H...H..",
)

// Goomba8Flat is the squashed frame.
var Goomba8Flat = sprite(
	"........",
	"........",
	"........",
	"........",
	"........",
	".KKKKKK.",
	"KTTTWTWK",
	".HHHHHH.",
)

// Koopa8 stands two art pixels taller than a Goomba, so the height difference
// survives with no colour at all.
var Koopa8 = sprite(
	"..KKK...",
	".KGGGK..",
	"KSSKGGK.",
	"KSSKGGK.",
	".KKGGGK.",
	"KGGGGGK.",
	"KGKGKGK.",
	"KGGGGGK.",
	".KKKKK..",
	".HHHHH..",
)

// Shell8 is the stomped Koopa.
var Shell8 = sprite(
	".KKKKK..",
	"KGGGGGK.",
	"KGKGGKGK",
	"KGGGGGK.",
	"KGKGGKGK",
	".KKKKK..",
)

// Mushroom8 is the grow pickup.
var Mushroom8 = sprite(
	".KKKKK..",
	"KRWWRRK.",
	"KRRWRRK.",
	"KRRRRRK.",
	".KSSSK..",
	".KSSSK..",
	".KSSSK..",
	"..KKK...",
)

// Star8 is the invincibility pickup.
var Star8 = sprite(
	"...K....",
	"..KCK...",
	"KKKCCKKK",
	"KCCCCCCK",
	".KCCCCK.",
	".KCCKCCK",
	"KCK..KCK",
	"........",
)

// Coin8Frames is the four-frame spin at the compact resolution.
var Coin8Frames = []Sprite{
	sprite(
		"..CCCC..",
		".CCMMCC.",
		".CMCCMC.",
		".CMCCMC.",
		".CMCCMC.",
		".CMCCMC.",
		".CCMMCC.",
		"..CCCC..",
	),
	sprite(
		"...CC...",
		"..CMCC..",
		"..CMCC..",
		"..CMCC..",
		"..CMCC..",
		"..CMCC..",
		"..CMCC..",
		"...CC...",
	),
	sprite(
		"...CC...",
		"...CC...",
		"...CC...",
		"...CC...",
		"...CC...",
		"...CC...",
		"...CC...",
		"...CC...",
	),
	sprite(
		"...CC...",
		"..CCMC..",
		"..CCMC..",
		"..CCMC..",
		"..CCMC..",
		"..CCMC..",
		"..CCMC..",
		"...CC...",
	),
}
