package render

// The hero is drawn from three independent parts: a body, a pair of hands that
// are already part of the body drawing, and a wheel underneath.
//
// They are kept apart because they change for different reasons. The body
// depends on the pose (standing, walking, in the air). The wheel depends only
// on how far the hero has travelled, and stops while he is airborne. Drawing
// the wheel as part of a body frame would need one frame per pose per wheel
// phase, and would tie the roll speed to the animation clock instead of to the
// ground that is actually passing beneath him.
//
// Everything below is drawn facing right and mirrored for facing left. A
// mirrored wheel turns the other way, which is exactly what rolling left looks
// like, so no second set of frames is needed.

// Hero palette slots.
const (
	// SlotHeroWheel is the rim of the wheel. It is not 'W': that is the HUD
	// text colour, and reusing it would turn the wheel into text-coloured grey
	// on one theme and something else on the next.
	SlotHeroWheel = 'O'
	// SlotHeroMark is the dot that shows which way the wheel is turning. It is
	// not 'M' for the same reason: that is the coin highlight.
	SlotHeroMark = 'Q'
)

// HeroPose selects the body drawing.
type HeroPose int

const (
	// HeroStand is standing or walking: the hands are by the body.
	HeroStand HeroPose = iota
	// HeroJump raises the front hand one row and drops the back hand out to
	// the waist. The red row runs the full width, so each hand is a single
	// skin cell touching the end of the sleeve.
	HeroJump
)

// heroBody is one body drawing and the facts the renderer needs about it.
type heroBody struct {
	Rows []string
	// Width and Height are in art pixels at the designed (4 per tile) scale.
	Width, Height int
}

// The small hero is 6 wide. Rows: hat, face, shoulders, waist, then an empty
// row, so he floats above the wheel.
var heroSmall = map[HeroPose]heroBody{
	HeroStand: body(
		"..RR..",
		"..SS..",
		".RRRRS",
		".SBBB.",
		"......",
	),
	HeroJump: body(
		"..RR..",
		"..SS.S",
		"RRRRRR",
		"SBBBB.",
		"......",
	),
}

// The big hero is 7 wide. One hat row, two face rows, a single skin-coloured
// neck cell in the middle, one red row, two blue rows, an empty row, and the
// wheel. On the jump the red row is drawn across the whole width and the hands
// sit at its two ends: the front hand one row above it, the back hand one row
// below it and out at the edge.
var heroBig = map[HeroPose]heroBody{
	HeroStand: body(
		"..RRR..",
		"..SSS..",
		"..SSS..",
		"...S...",
		".RRRRRS",
		".SBBBB.",
		".BBBBB.",
		".......",
	),
	HeroJump: body(
		"..RRR..",
		"..SSS..",
		"..SSS..",
		"...S..S",
		"RRRRRRR",
		"SBBBBB.",
		".BBBBB.",
		".......",
	),
}

func body(rows ...string) heroBody {
	w := 0
	for _, r := range rows {
		if len(r) > w {
			w = len(r)
		}
	}
	return heroBody{Rows: rows, Width: w, Height: len(rows)}
}

// The small wheel is 2x2. Every cell is on the rim, so the marker visits all
// four: top-left, top-right, bottom-right, bottom-left. That order is clockwise
// on screen, so a hero rolling right turns clockwise.
var smallMarkPath = [][2]int{{0, 0}, {0, 1}, {1, 1}, {1, 0}}

// The big wheel is 3x3 with a hollow hub. The marker walks the eight rim cells
// clockwise from the top-left. The hub is transparent, not a fixed colour, so
// whatever is behind the wheel shows through it.
var bigMarkPath = [][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 2}, {2, 2}, {2, 1}, {2, 0}, {1, 0}}

// WheelPhases is how many distinct frames the wheel has.
func WheelPhases(big bool) int {
	if big {
		return len(bigMarkPath)
	}
	return len(smallMarkPath)
}

// wheelRows builds the wheel for a phase. The hub of the big wheel is the
// empty slot, which the renderer skips.
func wheelRows(big bool, phase int) []string {
	path := smallMarkPath
	grid := [][]byte{[]byte("OO"), []byte("OO")}
	if big {
		path = bigMarkPath
		grid = [][]byte{[]byte("OOO"), []byte("O.O"), []byte("OOO")}
	}
	n := len(path)
	cell := path[((phase%n)+n)%n]
	grid[cell[0]][cell[1]] = SlotHeroMark
	out := make([]string, len(grid))
	for i, r := range grid {
		out[i] = string(r)
	}
	return out
}

// HeroDrawing is a hero ready to draw: body and wheel stacked into one sprite,
// plus the anchor the renderer needs to place it.
type HeroDrawing struct {
	Sprite Sprite
	// Width is the sprite's width in art pixels. It is wider than the collision
	// box, and the surplus is shared equally between the two sides.
	Width int
	// BodyRows is how many rows are body; the rest is wheel.
	BodyRows int
}

// Hero assembles the hero for a pose and a wheel phase.
//
// The wheel is centred under the body. The body has an empty row at its bottom
// edge, which is the gap that makes him look like he is floating.
func Hero(big bool, pose HeroPose, phase int) HeroDrawing {
	bodies := heroSmall
	if big {
		bodies = heroBig
	}
	b := bodies[pose]
	wheel := wheelRows(big, phase)

	ww := len(wheel[0])
	pad := (b.Width - ww) / 2

	rows := make([]string, 0, b.Height+len(wheel))
	rows = append(rows, b.Rows...)
	for _, wr := range wheel {
		line := make([]byte, b.Width)
		for i := range line {
			line[i] = SlotNone
		}
		copy(line[pad:], wr)
		rows = append(rows, string(line))
	}
	return HeroDrawing{Sprite: NewSprite(rows...), Width: b.Width, BodyRows: b.Height}
}

// Mirrored returns the drawing flipped left to right.
func (h HeroDrawing) Mirrored() HeroDrawing {
	h.Sprite = mirror(h.Sprite)
	return h
}
