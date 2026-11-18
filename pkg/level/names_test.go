package level

import "testing"

func TestEveryPaletteEntryHasAnEnglishName(t *testing.T) {
	check := func(list string, ids []string) {
		for _, id := range ids {
			if _, ok := englishNames[list][id]; !ok {
				t.Errorf("%s %q has no English name", list, id)
			}
		}
	}
	ids := func(l []Swatch) (out []string) {
		for _, s := range l {
			out = append(out, s.ID)
		}
		return
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
	check("theme", th)
	check("sky", sk)
}
