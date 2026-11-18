package theme

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/level"
)

// TestGradientBandsIn256Colours checks a gradient becomes several palette
// bands that only ever step one way, never one flat colour and never stripes.
func TestGradientBandsIn256Colours(t *testing.T) {
	st := level.Style{Sky: &level.Sky{Top: 0x10204a, Bottom: 0xf3a65a}}
	_, sky := Apply(Ink, st, Color256)
	if !sky.Gradient() {
		t.Fatal("the gradient collapsed to one colour")
	}
	distinct := map[tcell.Color]bool{}
	prev := -1.0
	for i := 0; i <= 100; i++ {
		c := sky.At(float64(i) / 100)
		if Quantize256(c) != c {
			t.Fatalf("band %v is not a palette colour", c)
		}
		distinct[c] = true
		l := Luminance(c)
		if l+1e-9 < prev {
			t.Fatalf("the bands get darker again at %d%%", i)
		}
		prev = l
	}
	if len(distinct) < 3 {
		t.Errorf("only %d bands; a gradient should show several", len(distinct))
	}
}

// TestSceneryNeverMatchesTheSkyIn256 is the "do not melt into the sky" rule.
func TestSceneryNeverMatchesTheSkyIn256(t *testing.T) {
	// A brick deliberately a hair away from the sky: in true colour they
	// differ, in 256 colours they would land on the same entry.
	sky := level.RGB(0x5f87d7)
	brick := level.RGB(0x5c88d5)
	if Quantize256(rgb(sky)) != Quantize256(rgb(brick)) {
		t.Skip("the chosen colours no longer collide in the palette")
	}
	got, fill := Apply(Ink, level.Style{Sky: &level.Sky{Top: sky, Bottom: sky}, Brick: &brick}, Color256)
	if Quantize256(got.Brick) == fill.Top {
		t.Error("the brick still quantises to the sky colour")
	}
	// Moved, but not far: it is still a blue.
	r, _, b := got.Brick.RGB()
	if b <= r {
		t.Errorf("the brick was moved to %v, which is no longer blue", got.Brick)
	}
}

func TestTrueColourIsNotQuantised(t *testing.T) {
	st := level.Style{Sky: &level.Sky{Top: 0x10204a, Bottom: 0xf3a65a}}
	_, sky := Apply(Ink, st, TrueColor)
	if sky.Top != rgb(0x10204a) {
		t.Error("true colour changed the sky's own colour")
	}
}

// TestEveryThemeSurvives256 checks each built-in theme keeps its scenery off
// the sky on a 256-colour terminal.
func TestEveryThemeSurvives256(t *testing.T) {
	for _, th := range level.Themes {
		got, fill := Apply(Ink, level.Style{Theme: th.ID}, Color256)
		skies := skyColours(fill)
		for name, c := range map[string]tcell.Color{
			"brick": got.Brick, "pipe": got.Pipe, "stone": got.Stone,
			"ground": got.Ground, "turf": got.Cast.Grass,
		} {
			q := Quantize256(c)
			for _, s := range skies {
				if q == s {
					t.Errorf("theme %s: %s is the sky's palette colour", th.ID, name)
				}
			}
		}
	}
}
