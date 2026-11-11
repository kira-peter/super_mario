package render

import (
	"testing"

	"super-mario/pkg/engine"
	"super-mario/pkg/game"
	"super-mario/pkg/level"
	"super-mario/pkg/render/theme"
)

// TestCastDrawingsAreRectangularAndKnown checks every drawing is rectangular
// and uses only letters its palette resolves, so no pixel silently vanishes.
func TestCastDrawingsAreRectangularAndKnown(t *testing.T) {
	tk := theme.Ink
	check := func(name string, a art, pal palette) {
		w := a.width()
		for r, row := range a {
			if len(row) != w {
				t.Errorf("%s row %d is %d wide, want %d", name, r, len(row), w)
			}
			for _, ch := range []byte(row) {
				if ch == '.' {
					continue
				}
				if _, ok := pal(ch); !ok {
					t.Errorf("%s uses letter %q, which its palette does not know", name, string(ch))
				}
			}
		}
	}
	check("mushroom", cast4.Mushroom, mushroomPalette(tk, game.ItemMushroom))
	for i, f := range cast4.Goomba {
		check("goomba frame "+string(rune('1'+i)), f, goombaPalette(tk))
	}
	check("goomba flat", cast4.GoombaFlat, goombaPalette(tk))
	for i, f := range cast4.Koopa {
		check("koopa frame "+string(rune('1'+i)), f, koopaPalette(tk))
	}
	check("shell", cast4.Shell, koopaPalette(tk))
	for name, a := range map[string]art{"pole": cast4.Pole, "ball": cast4.Ball, "cloth": cast4.Cloth} {
		check(name, a, flagPalette(tk))
	}
}

// TestWalkFramesDiffer checks the walking enemies actually move their feet.
func TestWalkFramesDiffer(t *testing.T) {
	for name, frames := range map[string][2]art{"goomba": cast4.Goomba, "koopa": cast4.Koopa} {
		if len(frames[0]) != len(frames[1]) || frames[0].width() != frames[1].width() {
			t.Errorf("%s frames differ in size; it would jump as it walks", name)
		}
		if frames[0][len(frames[0])-1] == frames[1][len(frames[1])-1] {
			t.Errorf("%s feet are the same in both frames", name)
		}
	}
}

// TestEveryMushroomHasItsOwnColour is the point of the five mushrooms: they
// share one shape, so the cap colour is all that tells them apart, in colour
// and in monochrome alike.
func TestEveryMushroomHasItsOwnColour(t *testing.T) {
	kinds := []game.ItemKind{game.ItemMushroom, game.ItemStar, game.ItemOneUp, game.ItemPoison, game.ItemGold}
	for _, tk := range []theme.Tokens{theme.Ink, theme.Paper, theme.Degrade(theme.Ink, theme.Mono)} {
		seen := map[[2]interface{}]game.ItemKind{}
		for _, k := range kinds {
			pal := mushroomPalette(tk, k)
			cap, _ := pal('C')
			rim, _ := pal('c')
			key := [2]interface{}{cap, rim}
			if other, dup := seen[key]; dup {
				t.Errorf("%s theme: mushrooms %d and %d have the same colours", tk.Name, other, k)
			}
			seen[key] = k
		}
	}
}

// castWorld is the real level with nothing but what a test places in view.
func castWorld(t *testing.T) *game.World {
	t.Helper()
	lv, err := level.LoadEmbedded("1-1")
	if err != nil {
		t.Fatal(err)
	}
	sm := engine.NewStateMachine()
	sm.StartRun()
	w := game.NewWorld(lv, &game.Default, sm, 3)
	w.SetViewport(engine.Tile*40, engine.Tile*15)
	w.Player.Pos.X = engine.Tile * 1000 // out of frame
	w.Enemies = nil
	return w
}

// TestRenderedMushroomUsesItsColour draws a gold mushroom and checks the gold
// cap is on screen and the red one is not.
func TestRenderedMushroomUsesItsColour(t *testing.T) {
	w := castWorld(t)
	it := game.NewItem(game.ItemGold, 5, 11)
	it.Emerging = 0
	w.Items = append(w.Items, it)
	c := (&Scene{T: theme.Ink}).Draw(w, 4)
	if countColour(c, theme.Ink.Cast.ShroomGold) == 0 {
		t.Error("the gold mushroom drew no gold cap")
	}
	if countColour(c, theme.Ink.Cast.ShroomRed) != 0 {
		t.Error("a red cap is on screen, but only a gold mushroom was placed")
	}
}

// TestKoopaFacesTheWayItWalks checks a Koopa walking right is drawn mirrored:
// its head, the only skin-coloured part above the shell, is on the right.
func TestKoopaFacesTheWayItWalks(t *testing.T) {
	headSide := func(facing int) int {
		w := castWorld(t)
		e := game.NewKoopa(5, 12)
		e.Facing = facing
		w.Enemies = []*game.Enemy{e}
		c := (&Scene{T: theme.Ink}).Draw(w, 4)
		// The head row is the top row of the drawing: find skin there.
		skin := theme.Ink.Cast.KoopaSkin
		for y := 0; y < c.H; y++ {
			minX, maxX := -1, -1
			shell := false
			for x := 0; x < c.W; x++ {
				switch c.Px[y*c.W+x] {
				case skin:
					if minX < 0 {
						minX = x
					}
					maxX = x
				case theme.Ink.Cast.KoopaShell:
					shell = true
				}
			}
			if minX >= 0 && shell {
				return minX + maxX
			}
		}
		t.Fatal("no Koopa head found on screen")
		return 0
	}
	left, right := headSide(-1), headSide(1)
	if right <= left {
		t.Errorf("walking right the head is at %d, walking left at %d; it should move right", right, left)
	}
}

// TestFlagClothHangsLeftOfThePole checks the chequered cloth is drawn and sits
// to the left of the shaft.
func TestFlagClothHangsLeftOfThePole(t *testing.T) {
	w := castWorld(t)
	w.CameraX = engine.Fixed(w.Level.FlagX-20) * engine.Tile
	w.CameraY = 0
	c := (&Scene{T: theme.Ink}).Draw(w, 4)
	poleX := (w.Level.FlagX-(w.Level.FlagX-20))*4 + poleColumn
	dark, light := 0, 0
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			switch c.Px[y*c.W+x] {
			case theme.Ink.Cast.FlagDark:
				dark++
				if x >= poleX {
					t.Fatalf("flag cloth at x=%d, right of the pole at %d", x, poleX)
				}
			case theme.Ink.Cast.FlagLight:
				light++
			}
		}
	}
	if dark == 0 || light == 0 {
		t.Errorf("flag cloth has %d dark and %d light pixels; want both", dark, light)
	}
}

// TestCastScalesByWholePixels checks the larger zooms show the same drawing
// blown up, not a different one: at 8 columns a tile, a mushroom is 8 by 8.
func TestCastScalesByWholePixels(t *testing.T) {
	for _, zoom := range []int{4, 8, 12, 16} {
		w := castWorld(t)
		w.SetViewport(engine.Tile*20, engine.Tile*15)
		w.CameraX, w.CameraY = 0, 0 // the player is parked off-screen; do not follow him
		it := game.NewItem(game.ItemOneUp, 3, 11)
		it.Emerging = 0
		w.Items = append(w.Items, it)
		c := (&Scene{T: theme.Ink}).Draw(w, zoom)
		k := zoom / 4
		want := 0
		for _, row := range cast4.Mushroom {
			for _, ch := range []byte(row) {
				if ch == 'C' {
					want++
				}
			}
		}
		if got := countColour(c, theme.Ink.Cast.ShroomGreen); got != want*k*k {
			t.Errorf("zoom %d: %d green cap pixels, want %d", zoom, got, want*k*k)
		}
	}
}
