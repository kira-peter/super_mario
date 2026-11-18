package level

import (
	"strings"
	"testing"
)

func TestNoStyleIsTheDefaultLook(t *testing.T) {
	lv, err := Parse(tiny(""))
	if err != nil {
		t.Fatal(err)
	}
	if !lv.Style.IsZero() {
		t.Errorf("a level without style has %+v", lv.Style)
	}
	if !lv.Style.Resolved().IsZero() {
		t.Error("resolving an empty style invented colours")
	}
}

func TestStyleAcceptsNamesAndHex(t *testing.T) {
	lv, err := Parse(tiny(`"style":{"theme":"snow","sky":{"top":"#1b2a4a","bottom":"#f3a65a"},
		"turf":"#e8eef5","brick":"blue","pipe":"orange"},`))
	if err != nil {
		t.Fatal(err)
	}
	s := lv.Style
	if s.Theme != "snow" {
		t.Errorf("theme = %q", s.Theme)
	}
	if s.Sky == nil || s.Sky.Top != 0x1b2a4a || s.Sky.Bottom != 0xf3a65a || !s.Sky.Gradient() {
		t.Errorf("sky = %+v", s.Sky)
	}
	if s.Turf == nil || *s.Turf != 0xe8eef5 {
		t.Errorf("turf = %v", s.Turf)
	}
	if blue, _ := findSwatch(BrickColors, "blue"); s.Brick == nil || *s.Brick != blue {
		t.Errorf("brick = %v", s.Brick)
	}
	if s.Stair != nil {
		t.Error("an unset field was filled in at parse time")
	}
}

func TestSkyPresetAndSolidColour(t *testing.T) {
	lv, err := Parse(tiny(`"style":{"sky":"dusk"},`))
	if err != nil {
		t.Fatal(err)
	}
	if dusk, _ := findSky("dusk"); *lv.Style.Sky != dusk {
		t.Errorf("sky = %+v, want the dusk preset", lv.Style.Sky)
	}
	lv, err = Parse(tiny(`"style":{"sky":"#336699"},`))
	if err != nil {
		t.Fatal(err)
	}
	if *lv.Style.Sky != (Sky{0x336699, 0x336699}) || lv.Style.Sky.Gradient() {
		t.Errorf("a solid sky = %+v", lv.Style.Sky)
	}
}

func TestThemePlusOverride(t *testing.T) {
	lv, err := Parse(tiny(`"style":{"theme":"desert","pipe":"blue"},`))
	if err != nil {
		t.Fatal(err)
	}
	r := lv.Style.Resolved()
	desert, _ := ThemeByID("desert")
	blue, _ := findSwatch(PipeColors, "blue")
	if *r.Pipe != blue {
		t.Errorf("pipe = %v, want the override", r.Pipe)
	}
	if *r.Brick != desert.Brick || *r.Turf != desert.Turf || *r.Sky != desert.Sky {
		t.Error("the other elements did not keep the desert colours")
	}
}

func TestStyleRejectsBadValues(t *testing.T) {
	for _, tc := range []struct{ style, want string }{
		{`{"theme":"moon"}`, "style.theme"},
		{`{"brick":"purple"}`, "style.brick"},
		{`{"pipe":"#12345"}`, "style.pipe"},
		{`{"sky":"#zzzzzz"}`, "style.sky"},
		{`{"sky":{"top":"#000000"}}`, "style.sky"},
		{`{"sky":{"top":"#000000","bottom":"#ffffff","middle":"#888888"}}`, "style.sky"},
		{`{"colour":"red"}`, "colour"},
	} {
		_, err := Parse(tiny(`"style":` + tc.style + `,`))
		if err == nil {
			t.Errorf("style %s was accepted", tc.style)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("style %s: error %q does not mention %q", tc.style, err, tc.want)
		}
	}
}

func TestFormatKeepsStyleAsWritten(t *testing.T) {
	data := tiny(`"style":{"pipe":"blue","theme":"snow","sky":{"top":"#000000","bottom":"#ffffff"}},`)
	out, err := Format(data)
	if err != nil {
		t.Fatal(err)
	}
	want := `"style": {"theme": "snow", "sky": {"top": "#000000", "bottom": "#ffffff"}, "pipe": "blue"},`
	if !strings.Contains(string(out), want) {
		t.Errorf("formatted style:\n%s\nwant a line containing\n%s", out, want)
	}
	again, err := Format(out)
	if err != nil || string(again) != string(out) {
		t.Error("formatting the style is not stable")
	}
}

func TestEveryPaletteEntryHasAUniqueID(t *testing.T) {
	check := func(name string, ids []string) {
		seen := map[string]bool{}
		for _, id := range ids {
			if id == "" || seen[id] || strings.HasPrefix(id, "#") {
				t.Errorf("%s: bad or duplicate id %q", name, id)
			}
			seen[id] = true
		}
	}
	ids := func(l []Swatch) []string {
		out := make([]string, len(l))
		for i, s := range l {
			out[i] = s.ID
		}
		return out
	}
	check("ground", ids(GroundColors))
	check("turf", ids(TurfColors))
	check("brick", ids(BrickColors))
	check("stair", ids(StairColors))
	check("pipe", ids(PipeColors))
	var th, sk []string
	for _, x := range Themes {
		th = append(th, x.ID)
	}
	for _, x := range SkyPresets {
		sk = append(sk, x.ID)
	}
	check("themes", th)
	check("skies", sk)
	if len(Themes) < 8 {
		t.Errorf("%d themes, the spec asks for at least 8", len(Themes))
	}
}
