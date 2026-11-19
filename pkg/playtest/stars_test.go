package playtest

import (
	"testing"

	"super-mario/pkg/level"
)

// TestEditorStarsOnlyAtNight: the editor gets a star list for a night level
// and none for any other.
func TestEditorStarsOnlyAtNight(t *testing.T) {
	if n := len(Stars(level.Style{Theme: "night"}, 120, 15, 0)); n == 0 {
		t.Error("no stars for the night theme")
	}
	for _, st := range []level.Style{{}, {Theme: "grassland"}, {Theme: "underground"}, {Theme: "lava"}} {
		if n := len(Stars(st, 120, 15, 0)); n != 0 {
			t.Errorf("theme %q: %d stars, want none", st.Theme, n)
		}
	}
}

// TestEditorStarsMatchTheGame: every star the editor draws is a star the game
// draws in the same place, so the canvas looks like the playtest.
func TestEditorStarsMatchTheGame(t *testing.T) {
	data := oneOne(t)
	lv, err := level.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	lv.Style = level.Style{Theme: "night"}
	stars := Stars(lv.Style, lv.Width, lv.Height, 0)

	// Draw the whole level at four pixels a tile with nothing in it but sky,
	// at step 0 like the list above, and look for each lit star.
	f := SkyFrame(lv, 4)
	at := func(x, y int) [3]byte {
		i := (y*f.W + x) * 4
		return [3]byte{f.RGBA[i], f.RGBA[i+1], f.RGBA[i+2]}
	}
	missing := 0
	for _, s := range stars {
		if s.Color == "" || s.X >= f.W || s.Y >= f.H {
			continue
		}
		got := at(s.X, s.Y)
		if hex(int32(got[0]), int32(got[1]), int32(got[2])) != s.Color {
			missing++
		}
	}
	if missing > 0 {
		t.Errorf("%d of %d editor stars are not in the game's picture", missing, len(stars))
	}
}
