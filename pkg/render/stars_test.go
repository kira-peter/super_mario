package render

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/engine"
	"super-mario/pkg/game"
	"super-mario/pkg/level"
	"super-mario/pkg/render/theme"
)

// starPixels counts the pixels a styled scene draws in star colours that the
// same scene without stars does not, across the whole level one screen at a
// time, and reports whether any of them landed on something other than sky.
func starPixels(t *testing.T, w *game.World, st level.Style, px int) (n int, onTerrain bool) {
	t.Helper()
	w.Level.Style = st
	sc := ForLevel(theme.Ink, "ink", w.Level, theme.TrueColor)
	night := sc.Stars
	lvW := engine.Fixed(w.Level.Width) * engine.Tile
	for x := engine.Fixed(0); x < lvW; x += w.ViewW {
		w.CameraX = x
		sc.Stars = night
		with := sc.Draw(w, px)
		sc.Stars = false
		without := sc.Draw(w, px)
		for i := range with.Px {
			if with.Px[i] == without.Px[i] {
				continue
			}
			n++
			if c, ok := without.Px[i].(tcell.Color); !ok || luminance(c) > starDarkLimit {
				onTerrain = true
			}
		}
	}
	return n, onTerrain
}

func TestNightThemeHasStars(t *testing.T) {
	w := castWorld(t)
	n, onTerrain := starPixels(t, w, level.Style{Theme: "night"}, 4)
	if n == 0 {
		t.Fatal("the night theme drew no stars")
	}
	if onTerrain {
		t.Error("a star was drawn over something that is not dark sky")
	}
	t.Logf("%d star pixels on one screen", n)
}

func TestStarrySkyHasStars(t *testing.T) {
	w := castWorld(t)
	sky, _ := findSkyForTest("starry")
	if n, _ := starPixels(t, w, level.Style{Sky: &sky}, 4); n == 0 {
		t.Fatal("the starry sky drew no stars")
	}
}

// TestOnlyNightHasStars: day, and the dark skies that are not night
// (underground, lava, a plain unstyled level, a custom dark colour), stay
// without stars.
func TestOnlyNightHasStars(t *testing.T) {
	dark := level.Sky{Top: 0x05060f, Bottom: 0x05060f}
	for name, st := range map[string]level.Style{
		"unstyled":    {},
		"grassland":   {Theme: "grassland"},
		"underground": {Theme: "underground"},
		"lava":        {Theme: "lava"},
		"custom dark": {Sky: &dark},
	} {
		w := castWorld(t)
		if n, _ := starPixels(t, w, st, 4); n != 0 {
			t.Errorf("%s: %d star pixels, want none", name, n)
		}
	}
}

// TestStarsStayPut: the same level draws the same stars every time, and they
// move with the level when the camera scrolls, not with the screen.
func TestStarsStayPut(t *testing.T) {
	w := castWorld(t)
	w.Level.Style = level.Style{Theme: "night"}
	sc := ForLevel(theme.Ink, "ink", w.Level, theme.TrueColor)
	a, b := sc.Draw(w, 4), sc.Draw(w, 4)
	for i := range a.Px {
		if a.Px[i] != b.Px[i] {
			t.Fatal("two draws of the same frame differ")
		}
	}
}

// TestNoStarsWithoutColour: a monochrome terminal has no dark sky to put them
// on, so it draws none.
func TestNoStarsWithoutColour(t *testing.T) {
	w := castWorld(t)
	w.Level.Style = level.Style{Theme: "night"}
	sc := ForLevel(theme.Degrade(theme.Ink, theme.Mono), "ink", w.Level, theme.Mono)
	with := sc.Draw(w, 4)
	sc.Stars = false
	without := sc.Draw(w, 4)
	for i := range with.Px {
		if with.Px[i] != without.Px[i] {
			t.Fatal("stars drawn on a monochrome terminal")
		}
	}
}

// TestStarsOnlyInTheUpperThird: no star sits below the top third of the
// level, in the editor's list or in the game's picture.
func TestStarsOnlyInTheUpperThird(t *testing.T) {
	w := castWorld(t)
	w.Level.Style = level.Style{Theme: "night"}
	band := w.Level.Height * heroDesignPx / 3
	_, fill := theme.Apply(theme.Ink, w.Level.Style, theme.TrueColor)
	for _, s := range StarField(w.Level.Style, w.Level.Width, w.Level.Height, fill.At) {
		if s.Y >= band {
			t.Errorf("star at row %d, below the upper third (rows 0..%d)", s.Y, band-1)
		}
	}

	// The game's own picture: at four pixels a tile with the camera at the
	// top, every pixel the stars add is above the band.
	w.CameraY = 0
	sc := ForLevel(theme.Ink, "ink", w.Level, theme.TrueColor)
	with := sc.Draw(w, 4)
	sc.Stars = false
	without := sc.Draw(w, 4)
	for i := range with.Px {
		if with.Px[i] != without.Px[i] && i/with.W >= band {
			t.Fatalf("a star pixel at row %d, below the upper third", i/with.W)
		}
	}
}

// TestStarsAreSparse: a sparse sky, about two stars a default screen, as tuned
// by eye: some, not a field of them.
func TestStarsAreSparse(t *testing.T) {
	lv := castWorld(t).Level
	lv.Style = level.Style{Theme: "night"}
	_, fill := theme.Apply(theme.Ink, lv.Style, theme.TrueColor)
	all := StarField(lv.Style, lv.Width, lv.Height, fill.At)
	// A default 120x40 terminal at four columns a tile shows about 30 tiles.
	const screen = 30
	per := float64(len(all)) * screen / float64(lv.Width)
	t.Logf("%d stars across %d tiles: %.1f a screen", len(all), lv.Width, per)
	if per < 1 || per > 4 {
		t.Errorf("%.1f stars a screen, want about two", per)
	}
}

// TestStarsTwinkleAtRandom: over a few seconds every star changes brightness,
// and they do not all change on the same steps.
func TestStarsTwinkleAtRandom(t *testing.T) {
	w := castWorld(t)
	w.Level.Style = level.Style{Theme: "night"}
	sc := ForLevel(theme.Ink, "ink", w.Level, theme.TrueColor)
	d := drawer{s: &sc}

	stars := StarField(w.Level.Style, w.Level.Width, w.Level.Height, sc.Fill.At)
	if len(stars) < 5 {
		t.Fatalf("only %d stars to watch", len(stars))
	}
	changes := map[int]int{} // step -> how many stars changed on it
	for _, s := range stars {
		prev, prevLit, _ := d.starColour(s.hash, 0)
		changed := false
		for step := 1; step < 300; step++ {
			c, lit, _ := d.starColour(s.hash, step)
			if c != prev || lit != prevLit {
				changes[step]++
				changed = true
			}
			prev, prevLit = c, lit
		}
		if !changed {
			t.Errorf("star at (%d,%d) held steady for five seconds", s.X, s.Y)
		}
	}
	for step, n := range changes {
		if n == len(stars) {
			t.Errorf("every star changed together on step %d", step)
		}
	}
}

func findSkyForTest(id string) (level.Sky, bool) {
	for _, s := range level.SkyPresets {
		if s.ID == id {
			return s.Sky, true
		}
	}
	return level.Sky{}, false
}
