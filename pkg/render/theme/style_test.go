package theme

import (
	"reflect"
	"testing"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/level"
)

func ptr(v level.RGB) *level.RGB { return &v }

func TestApplyZeroStyleChangesNothing(t *testing.T) {
	got, sky := Apply(Ink, level.Style{}, TrueColor)
	if !reflect.DeepEqual(got, Ink) {
		t.Error("an empty style changed the palette")
	}
	if sky.Gradient() || sky.Top != Ink.Sky {
		t.Errorf("sky = %+v, want the plain theme sky", sky)
	}
}

// TestApplyLeavesTheRestAlone is the "only this level's picture" rule: nothing
// but the scenery tokens may change.
func TestApplyLeavesTheRestAlone(t *testing.T) {
	st := level.Style{Theme: "lava"}
	got, _ := Apply(Ink, st, TrueColor)

	same := []struct {
		name string
		a, b tcell.Color
	}{
		{"HudText", got.HudText, Ink.HudText},
		{"HudMuted", got.HudMuted, Ink.HudMuted},
		{"Backdrop", got.Backdrop, Ink.Backdrop},
		{"PlayerBody", got.PlayerBody, Ink.PlayerBody},
		{"PlayerAccent", got.PlayerAccent, Ink.PlayerAccent},
		{"EnemyBody", got.EnemyBody, Ink.EnemyBody},
		{"Coin", got.Coin, Ink.Coin},
		{"Question", got.Question, Ink.Question},
		{"Outline", got.Outline, Ink.Outline},
		{"GoombaBody", got.Cast.GoombaBody, Ink.Cast.GoombaBody},
		{"KoopaShell", got.Cast.KoopaShell, Ink.Cast.KoopaShell},
		{"ShroomRed", got.Cast.ShroomRed, Ink.Cast.ShroomRed},
		{"CastleWall", got.Cast.CastleWall, Ink.Cast.CastleWall},
	}
	for _, s := range same {
		if s.a != s.b {
			t.Errorf("%s changed from %v to %v", s.name, s.b, s.a)
		}
	}

	lava, _ := level.ThemeByID("lava")
	changed := []struct {
		name string
		a    tcell.Color
		want level.RGB
	}{
		{"Brick", got.Brick, lava.Brick},
		{"Pipe", got.Pipe, lava.Pipe},
		{"Ground", got.Ground, lava.Ground},
		{"Grass", got.Cast.Grass, lava.Turf},
		{"Stone", got.Stone, lava.Stair},
	}
	for _, c := range changed {
		if c.a != rgb(c.want) {
			t.Errorf("%s = %v, want the lava colour %06x", c.name, c.a, c.want)
		}
	}
}

func TestApplyThemePlusOverride(t *testing.T) {
	blue := level.RGB(0x3a7bd5)
	got, _ := Apply(Ink, level.Style{Theme: "desert", Pipe: &blue}, TrueColor)
	desert, _ := level.ThemeByID("desert")
	if got.Pipe != rgb(blue) {
		t.Errorf("pipe = %v, want the override", got.Pipe)
	}
	if got.Brick != rgb(desert.Brick) {
		t.Errorf("brick = %v, want the desert brick", got.Brick)
	}
}

// TestDerivedShadesFollowTheMainColour checks the companion colours are lighter
// and darker versions of the chosen colour, not leftovers from the old theme.
func TestDerivedShadesFollowTheMainColour(t *testing.T) {
	blue := level.RGB(0x3d6fb8)
	got, _ := Apply(Ink, level.Style{Brick: &blue}, TrueColor)
	if !(Luminance(got.BrickEdge) > Luminance(got.Brick) && Luminance(got.Brick) > Luminance(got.BrickMortar)) {
		t.Error("brick edge, face and mortar are not in light-to-dark order")
	}
	// The mortar must be a dark blue, not the red theme's mortar.
	r, _, b := got.BrickMortar.RGB()
	if b <= r {
		t.Errorf("mortar %v is not bluish", got.BrickMortar)
	}

	pink := level.RGB(0xf080c0)
	got, _ = Apply(Ink, level.Style{Turf: &pink}, TrueColor)
	if Luminance(got.Cast.GrassDark) >= Luminance(got.Cast.Grass) {
		t.Error("the dark turf is not darker")
	}
	r, g, _ := got.Cast.GrassDark.RGB()
	if r <= g {
		t.Errorf("the dark pink turf %v lost its hue", got.Cast.GrassDark)
	}
}

func TestGradientSky(t *testing.T) {
	st := level.Style{Sky: &level.Sky{Top: 0x10204a, Bottom: 0xf3a65a}}
	tok, sky := Apply(Ink, st, TrueColor)
	if !sky.Gradient() {
		t.Fatal("a two-colour sky is not a gradient")
	}
	if sky.At(0) != rgb(0x10204a) || sky.At(1) != rgb(0xf3a65a) {
		t.Errorf("ends = %v, %v", sky.At(0), sky.At(1))
	}
	// It changes steadily: each step down is at least as light as the last.
	prev := Luminance(sky.At(0))
	for i := 1; i <= 20; i++ {
		l := Luminance(sky.At(float64(i) / 20))
		if l+1e-9 < prev {
			t.Fatalf("the gradient gets darker at step %d", i)
		}
		prev = l
	}
	if tok.Sky != sky.Bottom {
		t.Error("the flat sky token is not the horizon colour")
	}
}

func TestCloudsStayVisible(t *testing.T) {
	for _, s := range level.SkyPresets {
		s := s
		tok, _ := Apply(Ink, level.Style{Sky: &s.Sky}, TrueColor)
		_, fill := Apply(Ink, level.Style{Sky: &s.Sky}, TrueColor)
		if c := Contrast(tok.Cast.Cloud, fill.At(0.3)); c < 1.25 {
			t.Errorf("sky %s: clouds have contrast %.2f against it", s.ID, c)
		}
	}
}

func TestMonoIgnoresStyle(t *testing.T) {
	mono := Degrade(Ink, Mono)
	got, sky := Apply(mono, level.Style{Theme: "lava"}, Mono)
	if !reflect.DeepEqual(got, mono) || sky.Gradient() {
		t.Error("a mono terminal took on level colours")
	}
}
