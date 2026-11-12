package render

import (
	"sort"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/game"
	"super-mario/pkg/render/theme"
)

// The castle at the end of the level, where the hero walks in and disappears.
//
// Drawn after a reference picture and chosen in preview/castle-image.html
// (size=A outline=B wall=C roof=A flag=B): it narrows as it rises. The wall is
// widest, with battlements along its top; three towers stand on it apart, the
// middle one wider than the side ones, and it steps in again for an upper
// storey; each tower has a ring of battlements over a light ledge, and a red
// tiled roof. The whole castle is outlined, except the windows.
//
// The windows are dark while the level is being played. When the hero grabs the
// flagpole the lamps come on, one window after another from the ground up, and
// stay lit: one solid warm colour with no frame, glowing against the night sky.
//
// Forty pixels wide and forty-four tall: ten tiles by eleven. The castle is
// symmetric about its middle; only the flags are not, since all three fly to
// the right as in the picture.
//
//	S wall          s shadow and mortar   L light stone (ledges, door frame)
//	R roof          r roof tiles          K outline
//	Y window        o door (O, a lighter board, is unused now)
//	F flag          P flagpole            . sky
var castleArt = art{
	"....................PFFF................",
	"....................PFF.................",
	"....................P...................",
	"...................KP...................",
	"..................KRRK..................",
	".................KRrrRK.................",
	"................KRRRRRRK................",
	"...............KrRrRRrRrK...............",
	"...............KRRRRRRRRK...............",
	".....PFFF.....KrRrRrrRrRrK........PFFF..",
	".....PFF.....KSSSKKSSKKSSSK.......PFF...",
	".....P.......KLLLLLLLLLLLLK.......P.....",
	".....PK......KssssssssssssK......KP.....",
	"....KRRK......KSSSSSSSSSSK......KRRK....",
	"...KrRrRK.....KSSSSYYSSSSK.....KRrRrK...",
	"..KRRRRRRK....KSSSYYYYSSSK....KRRRRRRK..",
	"..KrRrRrRK....KSSSYYYYSSSK....KRrRrRrK..",
	".KRRRRRRRRK...KSSSYYYYSSSK...KRRRRRRRRK.",
	"KSSSKKKKSSSK..KSSSSSSSSSSK..KSSSKKKKSSSK",
	"KLLLLLLLLLLKKKKSSSSSSSSSSKKKKLLLLLLLLLLK",
	"KssssssssssKSSSKKSSSSSSKKSSSKssssssssssK",
	".KSSSSSSSSKKLLLLLLLLLLLLLLLLKKSSSSSSSSK.",
	".KSSSYYSSSKKssssssssssssssssKKSSSYYSSSK.",
	".KSSYYYYSSK.KSSSSSSSSSSSSSSK.KSSYYYYSSK.",
	".KSSYYYYSSK.KSSSSSSYYSSSSSSK.KSSYYYYSSK.",
	".KSSYYYYSSK.KSSSSSYYYYSSSSSK.KSSYYYYSSK.",
	".KSSSSSSSSK.KSSSSSYYYYSSSSSK.KSSSSSSSSK.",
	".KSSSSSSSSK.KSSSSSYYYYSSSSSK.KSSSSSSSSK.",
	".KSSSSSSSSK.KSSSSSSSSSSSSSSK.KSSSSSSSSK.",
	"KKSSSSSSSSKKKSSSSSSSSSSSSSSKKKSSSSSSSSKK",
	"SSSKKSSSKKSSSKKSSSKKKKSSSKKSSSKKSSSKKSSS",
	"LLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLL",
	"ssssssssssssssssssssssssssssssssssssssss",
	"KSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSK",
	"KSSSSYYSSSSSSSSSSLLLLLLSSSSSSSSSSYYSSSSK",
	"KSSSYYYYSSSSSSSSLLooooLLSSSSSSSSYYYYSSSK",
	"KSSSYYYYSSSSSSSSLooooooLSSSSSSSSYYYYSSSK",
	"KSSSYYYYSSSSSSSSLooooooLSSSSSSSSYYYYSSSK",
	"KSSSSSSSSSSSSSSSLooooooLSSSSSSSSSSSSSSSK",
	"KSSSSSSSSSSSSSSSLooooooLSSSSSSSSSSSSSSSK",
	"KSSSSSSSSSSSSSSSLooooooLSSSSSSSSSSSSSSSK",
	"KSSSSSSSSSSSSSSSLooooooLSSSSSSSSSSSSSSSK",
	"KSSSSSSSSSSSSSSSLooooooLSSSSSSSSSSSSSSSK",
	"KSSSSSSSSSSSSSSSLooooooLSSSSSSSSSSSSSSSK",
}

// CastleDoorPx is the middle of the door within the drawing, in drawing pixels.
// The castle is placed so that this lands on the column the hero walks to.
const CastleDoorPx = 20

// CastleDoorTile is how many tiles right of the pole the castle door stands, and
// so how far he walks. It must equal game.CastleDistance; a test checks.
const CastleDoorTile = 7

// windowLightSteps is how far apart the windows come on, in simulation steps:
// a sixth of a second, so the whole castle is lit in about a second.
const windowLightSteps = 10

// castleWindow is one window of the drawing: its pixels, and the row of its
// lowest pixel, which sets when it lights.
type castleWindow struct {
	pixels [][2]int // row, column
	bottom int
}

// castleWindows are the drawing's windows, in the order they light: from the
// ground up, and left to right along a row. Each is a connected group of 'Y'
// pixels.
var castleWindows = findWindows(castleArt)

func findWindows(a art) []castleWindow {
	seen := map[[2]int]bool{}
	var out []castleWindow
	for r, row := range a {
		for c := 0; c < len(row); c++ {
			if row[c] != 'Y' || seen[[2]int{r, c}] {
				continue
			}
			var win castleWindow
			stack := [][2]int{{r, c}}
			seen[[2]int{r, c}] = true
			for len(stack) > 0 {
				p := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				win.pixels = append(win.pixels, p)
				if p[0] > win.bottom {
					win.bottom = p[0]
				}
				for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
					q := [2]int{p[0] + d[0], p[1] + d[1]}
					if q[0] < 0 || q[0] >= len(a) || q[1] < 0 || q[1] >= len(a[q[0]]) {
						continue
					}
					if a[q[0]][q[1]] == 'Y' && !seen[q] {
						seen[q] = true
						stack = append(stack, q)
					}
				}
			}
			out = append(out, win)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].bottom != out[j].bottom {
			return out[i].bottom > out[j].bottom
		}
		return out[i].pixels[0][1] < out[j].pixels[0][1]
	})
	return out
}

// windowsLit is how many windows are lit: none before the pole is grabbed,
// then one more every windowLightSteps until all of them are.
func windowsLit(w *game.World) int {
	if !w.Cleared {
		return 0
	}
	n := (w.StepCount()-w.ClearedAt)/windowLightSteps + 1
	if n > len(castleWindows) {
		n = len(castleWindows)
	}
	if n < 0 {
		n = 0
	}
	return n
}

// castlePalette colours the castle. open is set once the pole has been grabbed:
// the door stands open then, and shows as a black doorway.
func castlePalette(t theme.Tokens, open bool) palette {
	c := t.Cast
	return func(l byte) (tcell.Color, bool) {
		switch l {
		case 'S':
			return c.CastleWall, true
		case 's':
			return c.CastleWallDark, true
		case 'L':
			return c.CastleStoneLight, true
		case 'R':
			return c.CastleRoof, true
		case 'r':
			return c.CastleRoofTile, true
		case 'K':
			return c.CastleOutline, true
		case 'Y':
			return c.CastleWindowDark, true
		case 'O', 'o':
			if open {
				return c.CastleDoorOpen, true
			}
			if l == 'O' {
				return c.CastleDoor, true
			}
			return c.CastleDoorDark, true
		case 'F':
			return c.CastleFlag, true
		case 'P':
			return c.CastleFlagPole, true
		}
		return 0, false
	}
}

// castle draws the castle, standing on the ground the flagpole stands on, with
// its door open and as many windows lit as the end-of-level sequence has
// reached.
func (d drawer) castle(w *game.World) {
	lv := w.Level
	k := d.castScale()
	doorX := lv.FlagX + CastleDoorTile
	// Ground line: the row the pole's foot rests on.
	baseY := (lv.FlagBot + 1) * d.px
	x := doorX*d.px - CastleDoorPx*k - d.camX
	y := baseY - len(castleArt)*k - d.camY
	if x > d.artW || x+castleArt.width()*k < 0 {
		return
	}
	d.c.blitArt(castleArt, x, y, k, false, castlePalette(d.s.T, w.Cleared))

	lit := d.s.T.Cast.CastleWindow
	for _, win := range castleWindows[:windowsLit(w)] {
		for _, p := range win.pixels {
			d.c.FillRect(x+p[1]*k, y+p[0]*k, k, k, lit)
		}
	}
}
