package render

import (
	"strings"
	"testing"

	"super-mario/pkg/engine"
	"super-mario/pkg/game"
	"super-mario/pkg/render/theme"
)

// TestCastleArtIsRectangularAndKnown checks every row of the castle drawing is
// the same width and only uses letters its palette resolves.
func TestCastleArtIsRectangularAndKnown(t *testing.T) {
	pal := castlePalette(theme.Ink, false)
	w := castleArt.width()
	if w != 40 {
		t.Errorf("the castle is %d pixels wide, want 40 (ten tiles)", w)
	}
	for r, row := range castleArt {
		if len(row) != w {
			t.Errorf("castle row %d is %d wide, want %d", r, len(row), w)
		}
		for _, ch := range []byte(row) {
			if ch == '.' {
				continue
			}
			if _, ok := pal(ch); !ok {
				t.Errorf("castle row %d uses %q, which has no colour", r, string(ch))
			}
		}
	}
}

// TestCastleIsSymmetricBelowTheFlags is the rule the castle was built to: left
// and right mirror each other. Only the flags may differ, since all three fly
// the same way, as in the reference picture.
func TestCastleIsSymmetricBelowTheFlags(t *testing.T) {
	for r, row := range castleArt {
		masked := []byte(row)
		for x, ch := range masked {
			if ch == 'F' || ch == 'P' {
				masked[x] = '.'
			}
		}
		// The outline beside a flagpole belongs to the flag too.
		for x := range masked {
			if row[x] == 'K' && ((x > 0 && row[x-1] == 'P') || (x+1 < len(row) && row[x+1] == 'P')) {
				masked[x] = '.'
			}
		}
		s := string(masked)
		rev := []byte(s)
		for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
			rev[i], rev[j] = rev[j], rev[i]
		}
		if s != string(rev) {
			t.Errorf("castle row %d is not symmetric:\n  %s", r, row)
		}
	}
}

// TestCastleNarrowsAsItRises checks the shape asked for: the base is the
// widest part (a ledge may overhang it by a pixel each side, as a cornice
// does), and the towers stand apart above it with sky between them.
func TestCastleNarrowsAsItRises(t *testing.T) {
	solid := func(row string) int {
		n := 0
		for _, ch := range row {
			if !strings.ContainsRune(".KFP", ch) {
				n++
			}
		}
		return n
	}
	base := solid(castleArt[len(castleArt)-1])
	for r, row := range castleArt {
		if solid(row) > base+2 {
			t.Errorf("row %d is %d wide, wider than the base (%d)", r, solid(row), base)
		}
	}
	// Somewhere above the wall, a row has wall, then sky, then wall: the towers
	// are apart.
	apart := false
	for _, row := range castleArt[:len(castleArt)/2] {
		trimmed := strings.Trim(row, ".")
		if strings.Contains(trimmed, "..") {
			apart = true
			break
		}
	}
	if !apart {
		t.Error("the towers are not separated by sky anywhere")
	}
}

// TestCastleWindowsAreLit checks the windows glow: one solid warm colour, with
// no frame, glow or dark pane left in them.
func TestCastleWindowsAreLit(t *testing.T) {
	all := strings.Join(castleArt, "")
	if strings.Count(all, "Y") == 0 {
		t.Fatal("the castle has no lit windows")
	}
	if strings.ContainsAny(all, "yD") {
		t.Error("a window has a second colour in it; they are meant to be one solid colour")
	}
	// No outline around a window: a lit pixel never touches the outline.
	for r, row := range castleArt {
		for x := 0; x < len(row); x++ {
			if row[x] != 'Y' {
				continue
			}
			for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				rr, xx := r+d[1], x+d[0]
				if rr >= 0 && rr < len(castleArt) && xx >= 0 && xx < len(row) && castleArt[rr][xx] == 'K' {
					t.Errorf("a window at row %d col %d has an outline next to it", r, x)
				}
			}
		}
	}
	// The light has to be warm: more red than blue, in both colour themes.
	for _, tk := range []theme.Tokens{theme.Ink, theme.Paper} {
		r, _, b := tk.Cast.CastleWindow.RGB()
		if r <= b {
			t.Errorf("%s: the window colour is not warm (red %d, blue %d)", tk.Name, r, b)
		}
	}
}

// TestCastleDoorReachesTheGround checks the door is a way in: it touches the
// bottom row, where the hero walks, centred on CastleDoorPx.
func TestCastleDoorReachesTheGround(t *testing.T) {
	bottom := castleArt[len(castleArt)-1]
	first, last := strings.IndexAny(bottom, "Oo"), strings.LastIndexAny(bottom, "Oo")
	if first < 0 {
		t.Fatal("the door does not reach the bottom row")
	}
	if mid := (first + last + 1) / 2; mid != CastleDoorPx {
		t.Errorf("the door is centred on pixel %d, want CastleDoorPx (%d)", mid, CastleDoorPx)
	}
}

// TestCastleMiddleTowerIsHighest checks the silhouette: the middle roof is the
// highest.
func TestCastleMiddleTowerIsHighest(t *testing.T) {
	top := [3]int{-1, -1, -1}
	for r, row := range castleArt {
		for x := 0; x < len(row); x++ {
			if row[x] != 'R' && row[x] != 'r' {
				continue
			}
			i := x * 3 / len(row)
			if top[i] < 0 || r < top[i] {
				top[i] = r
			}
		}
	}
	if !(top[1] < top[0] && top[1] < top[2]) {
		t.Errorf("roof tops at rows %v: the middle one should be highest", top)
	}
}

// TestCastleFitsInTheLevel checks the castle stays inside the level: its right
// edge is not past the last column, and its top is not above the first row.
func TestCastleFitsInTheLevel(t *testing.T) {
	lv := castWorld(t).Level
	doorPx := (lv.FlagX + CastleDoorTile) * heroDesignPx
	left := doorPx - CastleDoorPx
	right := left + castleArt.width()
	if left < 0 || right > lv.Width*heroDesignPx {
		t.Errorf("the castle spans pixels %d..%d, the level is %d wide", left, right, lv.Width*heroDesignPx)
	}
	top := (lv.FlagBot+1)*heroDesignPx - len(castleArt)
	if top < 0 {
		t.Errorf("the castle's top is at pixel row %d, above the level", top)
	}
}

// TestRenderedCastleDoorIsWhereHeWalks is the alignment check: the hero walks
// to CastleDoorTile past the pole, and that has to be the drawn door, or he
// vanishes in mid air beside the castle.
func TestRenderedCastleDoorIsWhereHeWalks(t *testing.T) {
	w := castWorld(t)
	doorTile := w.Level.FlagX + CastleDoorTile
	w.CameraX = engine.Fixed(doorTile-30) * engine.Tile
	w.CameraY = 0
	c := (&Scene{T: theme.Ink}).Draw(w, 4)

	x := doorTile*4 - WorldToArtScaled(w.CameraX, 4)
	if x < 0 || x >= c.W {
		t.Fatalf("the door column %d is off screen", x)
	}
	found := false
	for y := 0; y < c.H; y++ {
		if p := c.Px[y*c.W+x]; p == theme.Ink.Cast.CastleDoor || p == theme.Ink.Cast.CastleDoorDark {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no door is drawn at column %d, where he walks to disappear", x)
	}
}

// TestGameWalksToTheDrawnDoor ties the two halves together: the distance the
// game walks him equals where the renderer puts the door.
func TestGameWalksToTheDrawnDoor(t *testing.T) {
	if game.CastleDistance != CastleDoorTile {
		t.Errorf("the game walks him %d tiles past the pole, the door is drawn %d past it",
			game.CastleDistance, CastleDoorTile)
	}
}

// litPixels counts lit window pixels in a frame of the castle.
func litPixels(t *testing.T, w *game.World) int {
	t.Helper()
	w.CameraX = engine.Fixed(w.Level.FlagX+CastleDoorTile-20) * engine.Tile
	w.CameraY = 0
	c := (&Scene{T: theme.Ink}).Draw(w, 4)
	return countColour(c, theme.Ink.Cast.CastleWindow)
}

// TestCastleWindowsAreDarkUntilThePole checks the lamps are out while the
// level is played.
func TestCastleWindowsAreDarkUntilThePole(t *testing.T) {
	w := castWorld(t)
	if n := litPixels(t, w); n != 0 {
		t.Errorf("%d window pixels are lit before the pole was reached", n)
	}
}

// TestCastleWindowsLightUpFromTheGround checks the lamps come on after the
// pole is grabbed, one window at a time from the ground up, until all are lit.
func TestCastleWindowsLightUpFromTheGround(t *testing.T) {
	if len(castleWindows) < 3 {
		t.Fatalf("the castle has %d windows, want several", len(castleWindows))
	}
	for i := 1; i < len(castleWindows); i++ {
		if castleWindows[i].bottom > castleWindows[i-1].bottom {
			t.Errorf("window %d lights before a lower one", i-1)
		}
	}

	w := castWorld(t)
	w.Cleared = true
	w.ClearedAt = w.StepCount()
	first := litPixels(t, w)
	if first == 0 {
		t.Fatal("no window lit the moment the pole was grabbed")
	}
	total := 0
	for _, win := range castleWindows {
		total += len(win.pixels)
	}
	if first >= total {
		t.Error("every window came on at once; they should light one after another")
	}

	w.ClearedAt = w.StepCount() - windowLightSteps*len(castleWindows)
	if n := litPixels(t, w); n != total {
		t.Errorf("%d of %d window pixels lit once the sequence is over", n, total)
	}
}

// TestRespawnPutsTheLampsOut checks a new life starts with a dark castle.
func TestRespawnPutsTheLampsOut(t *testing.T) {
	w := castWorld(t)
	w.Cleared = true
	w.Reset()
	w.Enemies = nil
	w.Player.Pos.X = engine.Tile * 1000
	if n := litPixels(t, w); n != 0 {
		t.Errorf("%d window pixels still lit after a respawn", n)
	}
}

// TestCastleDoorOpensAtThePole checks the door is shut, in wood, while the
// level is played, and turns black, standing open, the moment the pole is
// grabbed. Every door pixel changes; none of the wood is left.
func TestCastleDoorOpensAtThePole(t *testing.T) {
	doorPx := 0
	for _, row := range castleArt {
		doorPx += strings.Count(row, "o") + strings.Count(row, "O")
	}
	if doorPx == 0 {
		t.Fatal("the castle has no door")
	}
	frame := func(w *game.World) *Canvas {
		w.CameraX = engine.Fixed(w.Level.FlagX+CastleDoorTile-20) * engine.Tile
		w.CameraY = 0
		return (&Scene{T: theme.Ink}).Draw(w, 4)
	}
	tk := theme.Ink.Cast

	w := castWorld(t)
	shut := frame(w)
	if n := countColour(shut, tk.CastleDoorOpen); n != 0 {
		t.Errorf("%d black doorway pixels before the pole; the door should be shut", n)
	}
	if countColour(shut, tk.CastleDoorDark)+countColour(shut, tk.CastleDoor) == 0 {
		t.Error("the shut door is not drawn in wood")
	}

	w.Cleared = true
	w.ClearedAt = w.StepCount()
	open := frame(w)
	if n := countColour(open, tk.CastleDoorOpen); n < doorPx {
		t.Errorf("%d of %d door pixels are black once the pole is grabbed", n, doorPx)
	}
	if n := countColour(open, tk.CastleDoorDark) + countColour(open, tk.CastleDoor); n != 0 {
		t.Errorf("%d pixels of wood are left in the open doorway", n)
	}
	if r, g, b := tk.CastleDoorOpen.RGB(); r != 0 || g != 0 || b != 0 {
		t.Errorf("the open doorway is #%02x%02x%02x, want black", r, g, b)
	}
}
