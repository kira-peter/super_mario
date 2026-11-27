package render

import (
	"slices"
	"testing"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/engine"
	"super-mario/pkg/game"
	"super-mario/pkg/level"
	"super-mario/pkg/render/theme"
)

// styledWorld is the 1-1 terrain with a given style, viewed from the start.
func styledWorld(t *testing.T, st level.Style) *game.World {
	t.Helper()
	w := castWorld(t)
	w.Level.Style = st
	return w
}

func TestIndoorScenesKeepCloudsOut(t *testing.T) {
	for _, id := range []string{"lava", "underground", "grassland"} {
		t.Run(id, func(t *testing.T) {
			w := styledWorld(t, level.Style{Theme: id})
			w.SetViewport(engine.Tile*35, engine.Tile*15)
			scene := ForLevel(theme.Ink, "ink", w.Level, theme.TrueColor)
			clear := scene.Draw(w, 4)
			scene.Sky = NewSky(1)
			cloudy := scene.Draw(w, 4)
			if same := slices.Equal(clear.Px, cloudy.Px); same != (id != "grassland") {
				t.Errorf("clouds changed the frame: %t", !same)
			}
		})
	}
}

func TestBigHeroArtStaysBelowCastleRoof(t *testing.T) {
	lv, err := level.LoadEmbedded("1-4")
	if err != nil {
		t.Fatal(err)
	}
	sm := engine.NewStateMachine()
	sm.StartRun()
	w := game.NewWorld(lv, &game.Default, sm, 3)
	w.SetPlaytest(game.PlaytestOptions{StartBig: true})
	if !w.StartAtColumn(45) {
		t.Fatal("missing foothold under the roof")
	}
	w.SetViewport(engine.Tile*24, engine.Tile*15)
	w.Input.KeyDown(engine.KeySpace)
	for range 30 {
		w.Step()
		if w.Player.LastHit.HitCeiling {
			break
		}
	}
	hit := w.Player.LastHit
	if !hit.HitCeiling || w.Player.Vel.Y != 0 || lv.TileAt(hit.CeilingX, hit.CeilingY).Glyph != 'H' {
		t.Fatalf("expected to stop at the solid roof, got %+v", hit)
	}
	scene := ForLevel(theme.Ink, "ink", lv, theme.TrueColor)
	for _, px := range []int{4, 8, 12, 16} {
		w.Player.Hidden = false
		hero := scene.Draw(w, px)
		w.Player.Hidden = true
		background := scene.Draw(w, px)
		roof := (hit.CeilingY+1)*px - WorldToArtScaled(w.CameraY, px)
		if end := roof * hero.W; !slices.Equal(hero.Px[:end], background.Px[:end]) || slices.Equal(hero.Px[end:], background.Px[end:]) {
			t.Errorf("at %d pixels per tile, expected the hero to draw only below the roof", px)
		}
	}
	w.Step()
	if lv.TileAt(hit.CeilingX, hit.CeilingY).Glyph != 'H' || len(w.Bumps) != 0 {
		t.Error("the hard roof moved or broke after the ceiling hit")
	}
}

// TestUnstyledLevelIsPixelIdentical is the "old levels look the same" rule: a
// scene built for a level with no style draws exactly what the plain scene did
// before levels had styles, at every zoom.
func TestUnstyledLevelIsPixelIdentical(t *testing.T) {
	w := castWorld(t)
	// The rule is about a level with no style; the built-in 1-1 has one now,
	// so take its terrain without it.
	w.Level.Style = level.Style{}
	for _, px := range []int{4, 8, 12, 16} {
		for _, base := range []theme.Tokens{theme.Ink, theme.Paper} {
			plain := (&Scene{T: base}).Draw(w, px)
			sc := ForLevel(base, base.Name, w.Level, theme.TrueColor)
			styled := sc.Draw(w, px)
			if plain.W != styled.W || plain.H != styled.H {
				t.Fatalf("%s at %d: canvas size changed", base.Name, px)
			}
			for i := range plain.Px {
				if plain.Px[i] != styled.Px[i] {
					t.Fatalf("%s at %d: pixel %d differs (%v vs %v)", base.Name, px, i, plain.Px[i], styled.Px[i])
				}
			}
		}
	}
}

// TestGradientSkyRunsTopToBottom checks the sky is the top colour at the top of
// the level and moves toward the bottom colour further down.
func TestGradientSkyRunsTopToBottom(t *testing.T) {
	top, bottom := level.RGB(0x10204a), level.RGB(0xf3a65a)
	w := styledWorld(t, level.Style{Sky: &level.Sky{Top: top, Bottom: bottom}})
	w.CameraY = 0
	sc := ForLevel(theme.Ink, "ink", w.Level, theme.TrueColor)
	c := sc.Draw(w, 8)

	topPx := c.Px[0]
	if topPx != tcell.NewHexColor(int32(top)) {
		t.Errorf("top-left sky pixel = %v, want the top colour", topPx)
	}
	// Row 2 of the level is open sky on the far left of 1-1.
	mid := c.Px[(2*8)*c.W]
	if mid == topPx {
		t.Error("the sky two tiles down is still the top colour")
	}
	if theme.Luminance(mid.(tcell.Color)) <= theme.Luminance(topPx.(tcell.Color)) {
		t.Error("the sky does not lighten towards the warm bottom colour")
	}
}

// TestStyledTerrainUsesTheStyle checks a blue-brick level draws blue bricks
// and not the theme's red ones.
func TestStyledTerrainUsesTheStyle(t *testing.T) {
	blue := level.RGB(0x3d6fb8)
	w := styledWorld(t, level.Style{Brick: &blue})
	w.CameraX = engine.Tile * 14 // the first brick row at 20..24
	sc := ForLevel(theme.Ink, "ink", w.Level, theme.TrueColor)
	for _, px := range []int{4, 8} {
		c := sc.Draw(w, px)
		if countColour(c, tcell.NewHexColor(int32(blue))) == 0 {
			t.Errorf("at %d px: no blue brick drawn", px)
		}
		if countColour(c, theme.Ink.Brick) != 0 {
			t.Errorf("at %d px: the theme's red brick is still drawn", px)
		}
	}
}

// TestHUDKeepsThePlainTheme checks the scene's palette changes but nothing
// outside it does: ForLevel never touches the theme it was given.
func TestHUDKeepsThePlainTheme(t *testing.T) {
	base := theme.Ink
	w := styledWorld(t, level.Style{Theme: "lava"})
	sc := ForLevel(base, "ink", w.Level, theme.TrueColor)
	if sc.T.HudText != base.HudText || sc.T.Sky == base.Sky {
		t.Error("ForLevel did not keep HUD colours or did not apply the sky")
	}
	if theme.Ink.Brick != base.Brick {
		t.Error("ForLevel modified the shared theme")
	}
}
