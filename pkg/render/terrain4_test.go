package render

import (
	"strings"
	"testing"

	"super-mario/pkg/engine"
	"super-mario/pkg/game"
	"super-mario/pkg/level"
	"super-mario/pkg/render/theme"
)

// TestTerrainPatternsAreFourByFour pins the size: every hand-drawn pattern is
// four cells wide and four tall, and uses only letters the theme can resolve.
func TestTerrainPatternsAreFourByFour(t *testing.T) {
	all := map[string]pattern4{
		"ground below": terrain4.GroundBelow,
		"stair":        terrain4.Stair,
	}
	for i, p := range terrain4.GroundTop {
		all["ground top "+string(rune('A'+i))] = p
	}
	for i, p := range terrain4.Brick {
		all["brick "+string(rune('A'+i))] = p
	}
	for i, p := range terrain4.Question {
		all["question "+string(rune('A'+i))] = p
	}
	for i, p := range terrain4.Coin {
		all["coin "+string(rune('A'+i))] = p
	}
	for name, p := range all {
		for r, row := range p {
			if len(row) != 4 {
				t.Errorf("%s row %d is %d wide, want 4", name, r, len(row))
			}
			for _, ch := range []byte(row) {
				if ch == '.' {
					continue
				}
				if _, ok := terrainColor(theme.Ink, ch); !ok {
					t.Errorf("%s uses letter %q, which the theme does not know", name, string(ch))
				}
			}
		}
	}
}

// TestGroundHasTurfOnlyAtTheTop is the rule behind "lower layers do not repeat
// the edge": only the top of the ground has turf.
func TestGroundHasTurfOnlyAtTheTop(t *testing.T) {
	for i, p := range terrain4.GroundTop {
		if !strings.ContainsAny(p[0], "rR") {
			t.Errorf("ground top %d has no turf in its first row", i)
		}
	}
	for r, row := range terrain4.GroundBelow {
		if strings.ContainsAny(row, "rRG") {
			t.Errorf("a lower ground layer has turf in row %d, which would cut a stripe through the earth", r)
		}
	}
}

// TestGroundIsDarkSoilUnderTurf checks the chosen ground: one row of turf in
// two alternating patterns, over dark soil.
func TestGroundIsDarkSoilUnderTurf(t *testing.T) {
	want := [2]pattern4{
		{"rrRr", "dddd", "dddd", "dddd"},
		{"rRrr", "dddd", "dddd", "dddd"},
	}
	if terrain4.GroundTop != want {
		t.Errorf("ground top = %v, want %v", terrain4.GroundTop, want)
	}
	if terrain4.GroundTop[0][0] == terrain4.GroundTop[1][0] {
		t.Error("both turf patterns are the same, so a long floor is one repeated stamp")
	}
}

// TestBrickSeamsAreStaggered is the "错缝" requirement: neighbouring bricks must
// not have their seams in the same place, or the wall reads as a grid.
func TestBrickSeamsAreStaggered(t *testing.T) {
	a, b := terrain4.Brick[0], terrain4.Brick[1]
	if a == b {
		t.Fatal("both brick patterns are identical, so the seams line up")
	}
	// The vertical seam in the brick rows must sit in a different column.
	seam := func(p pattern4, row int) int { return strings.IndexByte(p[row], 'm') }
	if seam(a, 0) == seam(b, 0) {
		t.Errorf("the upper course has its seam in column %d in both patterns", seam(a, 0))
	}
	if seam(a, 2) == seam(b, 2) {
		t.Errorf("the lower course has its seam in column %d in both patterns", seam(a, 2))
	}
}

// TestBrickIsMostlyBrick is the reason the outline was replaced by a mortar
// seam: with the old outline, twelve of sixteen cells were dark and only three
// were brick-coloured.
func TestBrickIsMostlyBrick(t *testing.T) {
	for i, p := range terrain4.Brick {
		face := strings.Count(strings.Join(p[:], ""), "b")
		dark := strings.Count(strings.Join(p[:], ""), "o")
		if dark != 0 {
			t.Errorf("brick %d still has %d outline cells", i, dark)
		}
		if face < 4 {
			t.Errorf("brick %d has only %d brick-coloured cells, want at least 4", i, face)
		}
	}
}

// TestStairIsADifferentMaterialFromTheGround checks the stair shares no colour
// with the ground, which is what the old drawing got wrong.
func TestStairIsADifferentMaterialFromTheGround(t *testing.T) {
	stair := strings.Join(terrain4.Stair[:], "")
	ground := strings.Join(terrain4.GroundTop[0][:], "") + strings.Join(terrain4.GroundTop[1][:], "") +
		strings.Join(terrain4.GroundBelow[:], "")
	for _, ch := range stair {
		if strings.ContainsRune(ground, ch) {
			t.Errorf("the stair uses %q, which the ground also uses", string(ch))
		}
	}
	tk := theme.Ink
	if tk.Stone == tk.GroundDeep || tk.StoneLight == tk.GroundEdge {
		t.Error("the stair colours coincide with the ground's")
	}
}

// TestStairIsLitFromTheTopLeft checks the three tones are placed the way the
// preview showed: bright top and left, shaded bottom.
func TestStairIsLitFromTheTopLeft(t *testing.T) {
	p := terrain4.Stair
	if p[0] != "SSSS" {
		t.Errorf("the stair's top edge is %q, want it lit", p[0])
	}
	if p[3] != "tttt" {
		t.Errorf("the stair's bottom edge is %q, want it shaded", p[3])
	}
	if p[1][0] != 'S' || p[2][0] != 'S' {
		t.Error("the stair's left edge is not lit")
	}
}

// TestQuestionBlockPulses is the "亮点闪烁" requirement: the outline stays and
// only the middle changes between frames.
func TestQuestionBlockPulses(t *testing.T) {
	a, b := terrain4.Question[0], terrain4.Question[1]
	if a == b {
		t.Fatal("the question block has two identical frames, so it cannot pulse")
	}
	if a[0] != b[0] || a[3] != b[3] {
		t.Error("the outline changed between frames; only the middle should")
	}
	if !strings.Contains(b[1], "h") {
		t.Error("the bright frame has no glint")
	}
}

// TestFramesDoNotDependOnTheRenderRate checks the block pulses with the
// simulation, not with how often the screen is redrawn.
func TestFramesDoNotDependOnTheRenderRate(t *testing.T) {
	changes := 0
	last := questionFrame(0)
	for step := 1; step <= 120; step++ {
		if f := questionFrame(step); f != last {
			changes++
			last = f
		}
	}
	if changes != 6 {
		t.Errorf("the question block changed frame %d times in two seconds, want 6 (three pulses)", changes)
	}
}

// TestCoinsDoNotFlashInUnison checks neighbouring coins are out of phase.
func TestCoinsDoNotFlashInUnison(t *testing.T) {
	for _, step := range []int{0, 24, 48, 72} {
		if coinFrame(step, 0) == coinFrame(step, 1) {
			t.Errorf("step %d: adjacent coins show the same frame", step)
		}
	}
}

// TestCoinIsAShadedRound checks the chosen coin: a lit disc with a shaded edge,
// not a bare strip.
func TestCoinIsAShadedRound(t *testing.T) {
	for i, p := range terrain4.Coin {
		joined := strings.Join(p[:], "")
		if !strings.Contains(joined, "c") {
			t.Errorf("coin frame %d has no shaded edge", i)
		}
		if p[0][0] != '.' || p[0][3] != '.' || p[3][0] != '.' || p[3][3] != '.' {
			t.Errorf("coin frame %d has square corners, so it reads as a block and not a coin", i)
		}
	}
	if !strings.Contains(strings.Join(terrain4.Coin[0][:], ""), "*") {
		t.Error("the glint frame has no highlight")
	}
	if strings.Contains(strings.Join(terrain4.Coin[1][:], ""), "*") {
		t.Error("the plain frame still has a highlight, so nothing flashes")
	}
}

// TestEveryThemeColoursTheNewTerrain makes sure a theme cannot leave a terrain
// colour at its zero value, which would draw as the terminal's default colour.
func TestEveryThemeColoursTheNewTerrain(t *testing.T) {
	for name, tk := range map[string]theme.Tokens{"ink": theme.Ink, "paper": theme.Paper} {
		for _, ch := range []byte("GrRdbmSstoh?$*c") {
			col, ok := terrainColor(tk, ch)
			if !ok {
				t.Errorf("%s: letter %q has no colour", name, string(ch))
				continue
			}
			if col == nil {
				t.Errorf("%s: letter %q resolved to nil", name, string(ch))
			}
		}
	}
}

// TestMonochromeKeepsStoneLighterThanSoil is the colour-blind rule: with no
// colour, a stair must still be told apart from the floor, by being lighter.
func TestMonochromeKeepsStoneLighterThanSoil(t *testing.T) {
	m := theme.Degrade(theme.Ink, theme.Mono)
	if m.Stone == m.GroundDeep {
		t.Error("in monochrome the stair and the floor are the same grey")
	}
	if m.StoneLight == m.StoneDark {
		t.Error("in monochrome the stair has lost its shading")
	}
}

// renderedAt draws the real level at the default zoom, framed on a tile
// position, and returns the canvas.
func renderedAt(t *testing.T, col, row int) *Canvas {
	t.Helper()
	lv, err := level.LoadEmbedded("1-1")
	if err != nil {
		t.Fatal(err)
	}
	sm := engine.NewStateMachine()
	sm.StartRun()
	w := game.NewWorld(lv, &game.Default, sm, 3)
	w.SetViewport(engine.Tile*40, engine.Tile*19)
	w.CameraX = engine.Tile * engine.Fixed(col)
	w.CameraY = engine.Tile * engine.Fixed(row)
	w.Player.Pos.X = engine.Tile * 1000 // out of frame
	return (&Scene{T: theme.Ink}).Draw(w, 4)
}

func countColour(c *Canvas, want interface{}) int {
	n := 0
	for _, p := range c.Px {
		if p == want {
			n++
		}
	}
	return n
}

// TestRenderedGroundHasOneTurfRow checks the real level: through the whole
// pipeline, exactly one row of a column of ground is turf, because the ground
// is two tiles deep and only the top one carries it.
func TestRenderedGroundHasOneTurfRow(t *testing.T) {
	c := renderedAt(t, 3, 10)
	tk := theme.Ink
	turfRows := 0
	for y := 0; y < c.H; y++ {
		if p := c.Px[y*c.W+8]; p == tk.Cast.Grass || p == tk.Cast.GrassDark {
			turfRows++
		}
	}
	if turfRows != 1 {
		t.Errorf("a column of ground has %d turf rows, want exactly 1 however deep the ground is", turfRows)
	}
}

// TestRenderedWallUsesTheNewBrick checks the rendered brick carries mortar and
// no outline, through the real draw path.
func TestRenderedWallUsesTheNewBrick(t *testing.T) {
	c := renderedAt(t, 63, 7)
	tk := theme.Ink
	if countColour(c, tk.BrickMortar) == 0 {
		t.Error("no mortar was drawn for the bricks")
	}
	if countColour(c, tk.Brick) == 0 {
		t.Error("no brick face was drawn")
	}
}

// TestRenderedStairsAreStone checks the stair block is drawn in stone through
// the real path, and none of it is the ground's colour.
func TestRenderedStairsAreStone(t *testing.T) {
	c := renderedAt(t, 133, 8)
	tk := theme.Ink
	if countColour(c, tk.StoneLight) == 0 || countColour(c, tk.StoneDark) == 0 {
		t.Error("the stairs were not drawn with their lit and shaded tones")
	}
}

// TestBrickPatternDoesNotShimmerWhenTheCameraMoves is the regression guard for
// choosing a brick pattern by screen position: the pattern a brick gets must
// depend on its column in the level, so scrolling does not make seams jump.
func TestBrickPatternDoesNotShimmerWhenTheCameraMoves(t *testing.T) {
	// The same brick, seen from two camera positions one tile apart, must show
	// the same pattern.
	lv, _ := level.LoadEmbedded("1-1")
	var bx, by int
	found := false
	for y := 0; y < lv.Height && !found; y++ {
		for x := 0; x < lv.Width; x++ {
			if lv.TileAt(x, y).Glyph == 'B' {
				bx, by, found = x, y, true
				break
			}
		}
	}
	if !found {
		t.Skip("no brick in the level")
	}
	sample := func(camCol int) [4]string {
		c := renderedAt(t, camCol, by-2)
		x0, y0 := (bx-camCol)*4, 2*4
		var out [4]string
		for r := 0; r < 4; r++ {
			for k := 0; k < 4; k++ {
				switch c.Px[(y0+r)*c.W+x0+k] {
				case theme.Ink.Brick:
					out[r] += "b"
				case theme.Ink.BrickMortar:
					out[r] += "m"
				default:
					out[r] += "?"
				}
			}
		}
		return out
	}
	a, b := sample(bx-3), sample(bx-4)
	if a != b {
		t.Errorf("the same brick looks different from two camera positions: %v vs %v", a, b)
	}
}
