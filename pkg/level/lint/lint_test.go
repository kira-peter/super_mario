package lint

import (
	"strings"
	"testing"

	"super-mario/pkg/engine"
	"super-mario/pkg/game"
	"super-mario/pkg/level"
)

// flat builds a 15-row level from a terrain sketch: ground rows 13 and 14
// everywhere except where gaps names columns, extra lines patched in by row.
type sketch struct {
	width   int
	gaps    [][2]int // [from, to) columns with no ground
	patches map[int]map[int]byte
	spawn   [2]int
	flagX   int
	extra   string // extra JSON fields, each ending in a comma
	content string
	enemies string
}

func (s sketch) json() []byte {
	if s.width == 0 {
		s.width = 60
	}
	if s.spawn == [2]int{} {
		s.spawn = [2]int{2, 12}
	}
	if s.flagX == 0 {
		s.flagX = s.width - 3
	}
	rows := make([]string, 15)
	for y := range rows {
		b := []byte(strings.Repeat(" ", s.width))
		if y >= 13 {
			for x := range b {
				b[x] = 'X'
			}
			for _, g := range s.gaps {
				for x := g[0]; x < g[1]; x++ {
					b[x] = ' '
				}
			}
		}
		for x, c := range s.patches[y] {
			b[x] = c
		}
		if y >= 2 && y <= 12 {
			if y == 2 {
				b[s.flagX] = 'T'
			} else {
				b[s.flagX] = 'F'
			}
		}
		rows[y] = `"` + string(b) + `"`
	}
	return []byte(`{"name":"t","world":"1-1","width":` + itoa(s.width) + `,"height":15,"timeLimit":400,` + s.extra +
		`"spawn":{"x":` + itoa(s.spawn[0]) + `,"y":` + itoa(s.spawn[1]) + `},` +
		`"flag":{"x":` + itoa(s.flagX) + `,"topY":2,"bottomY":12},` +
		`"rows":[` + strings.Join(rows, ",") + `],"contents":[` + s.content + `],"entities":[` + s.enemies + `]}`)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

func codes(issues []Issue) map[string]Issue {
	out := map[string]Issue{}
	for _, i := range issues {
		out[i.Code] = i
	}
	return out
}

func run(t *testing.T, s sketch) map[string]Issue {
	t.Helper()
	is := Run(s.json(), &game.Default)
	for _, i := range is {
		if i.Severity == Error {
			t.Fatalf("sketch does not load: %s", i.Message)
		}
	}
	return codes(is)
}

func TestErrorsComeFromTheLoader(t *testing.T) {
	is := Run([]byte(`{"name":"t"}`), &game.Default)
	if len(is) == 0 || is[0].Severity != Error {
		t.Fatalf("a broken level gave %+v", is)
	}
}

func TestCleanLevelHasNoWarnings(t *testing.T) {
	if got := run(t, sketch{}); len(got) != 0 {
		t.Errorf("a plain flat level warned: %+v", got)
	}
}

func TestSpawnWithoutGround(t *testing.T) {
	got := run(t, sketch{gaps: [][2]int{{0, 6}}})
	if _, ok := got["spawn-no-ground"]; !ok {
		t.Errorf("no warning for a spawn over a pit: %+v", got)
	}
}

func TestEnemyNearSpawn(t *testing.T) {
	got := run(t, sketch{enemies: `{"kind":"goomba","x":5,"y":12}`})
	if i, ok := got["spawn-enemy-near"]; !ok || i.X != 5 {
		t.Errorf("no warning for an enemy right at the start: %+v", got)
	}
	got = run(t, sketch{enemies: `{"kind":"goomba","x":30,"y":12}`})
	if _, ok := got["spawn-enemy-near"]; ok {
		t.Error("an enemy far from the start was flagged")
	}
}

func TestSealedBlock(t *testing.T) {
	got := run(t, sketch{
		patches: map[int]map[int]byte{9: {20: '?'}, 10: {20: 'B'}},
		content: `{"x":20,"y":9,"kind":"coin"}`,
	})
	if i, ok := got["block-sealed"]; !ok || i.X != 20 || i.Y != 9 {
		t.Errorf("no warning for a block with a brick under it: %+v", got)
	}
}

func TestEnemyInSolidAndFloating(t *testing.T) {
	got := run(t, sketch{
		patches: map[int]map[int]byte{12: {30: 'H'}},
		enemies: `{"kind":"goomba","x":30,"y":12},{"kind":"koopa","x":40,"y":5}`,
	})
	if i, ok := got["enemy-in-solid"]; !ok || i.X != 30 {
		t.Errorf("no warning for an enemy inside a block: %+v", got)
	}
	if i, ok := got["enemy-floating"]; !ok || i.X != 40 {
		t.Errorf("no warning for an enemy in mid-air: %+v", got)
	}
}

func TestLowContrastColours(t *testing.T) {
	// A sky the colour of a coin.
	got := run(t, sketch{extra: `"style":{"sky":"#f5c842"},`})
	if _, ok := got["contrast-coin"]; !ok {
		t.Errorf("no warning for coins on a coin-coloured sky: %+v", got)
	}
	// Gold on pale blue is a big hue difference with almost no lightness
	// difference. It reads at a glance and must not be flagged.
	got = run(t, sketch{extra: `"style":{"sky":"#a8d8ff"},`})
	if i, ok := got["contrast-coin"]; ok {
		t.Errorf("a coin on a pale blue sky was flagged: %s", i.Message)
	}
}

// TestBuiltInLooksPassTheirOwnCheck: every theme and every sky preset the
// editor offers keeps the gameplay colours visible.
func TestBuiltInLooksPassTheirOwnCheck(t *testing.T) {
	var styles []string
	for _, th := range level.Themes {
		styles = append(styles, `{"theme":"`+th.ID+`"}`)
	}
	for _, s := range level.SkyPresets {
		styles = append(styles, `{"sky":"`+s.ID+`"}`)
	}
	for _, st := range styles {
		for code, i := range run(t, sketch{extra: `"style":` + st + `,`}) {
			if strings.HasPrefix(code, "contrast-") {
				t.Errorf("%s: %s", st, i.Message)
			}
		}
	}
}

// --- reachability --------------------------------------------------------

func TestEnvelopeIsSane(t *testing.T) {
	e := EnvelopeFor(&game.Default)
	// The tallest pipe in 1-1 is four tiles and must be climbable.
	if e.MaxRise < 4 {
		t.Errorf("a jump rises %d tiles; 1-1's pipes need 4", e.MaxRise)
	}
	if r := e.Reach(); r < 4 || r > 14 {
		t.Errorf("a running jump reaches %d tiles; expected a handful", r)
	}
	if !e.CanLand(3, 0) || !e.CanLand(2, 2) || e.CanLand(2, e.MaxRise+2) {
		t.Error("the envelope answers simple cases wrongly")
	}
}

func TestOneOneIsReachable(t *testing.T) {
	lv, err := level.LoadEmbedded("1-1")
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range Reach(lv, &game.Default) {
		t.Errorf("1-1: %s at (%d,%d)", i.Message, i.X, i.Y)
	}
}

// TestBuiltinWorldsAreClean: the eight built-in levels raise no design
// warnings at all, the reach check included, so each can be cleared from its
// start under the game's own jump physics.
func TestBuiltinWorldsAreClean(t *testing.T) {
	for _, n := range []string{"1-1", "1-2", "1-3", "1-4", "2-1", "2-2", "2-3", "2-4"} {
		lv, err := level.LoadEmbedded(n)
		if err != nil {
			t.Errorf("%s: %v", n, err)
			continue
		}
		for _, i := range Warnings(lv, &game.Default) {
			t.Errorf("%s: %s at (%d,%d): %s", n, i.Code, i.X, i.Y, i.Message)
		}
	}
}

func TestTwelveTileGapIsFlagged(t *testing.T) {
	got := run(t, sketch{width: 80, gaps: [][2]int{{30, 42}}})
	i, ok := got["gap-too-wide"]
	if !ok {
		t.Fatalf("no warning for a 12-tile gap: %+v", got)
	}
	if i.X < 29 || i.X > 31 {
		t.Errorf("the warning points at column %d, not the gap at 30", i.X)
	}
	if !strings.Contains(i.Message, "12") {
		t.Errorf("the message does not give the gap's width: %s", i.Message)
	}
}

func TestNarrowGapIsFine(t *testing.T) {
	got := run(t, sketch{gaps: [][2]int{{20, 23}}})
	if i, ok := got["gap-too-wide"]; ok {
		t.Errorf("a 3-tile gap was flagged: %s", i.Message)
	}
}

func TestHighWallIsFlagged(t *testing.T) {
	wall := map[int]map[int]byte{}
	for y := 2; y <= 12; y++ {
		wall[y] = map[int]byte{30: 'H', 31: 'H'}
	}
	got := run(t, sketch{patches: wall})
	if _, ok := got["wall-too-high"]; !ok {
		t.Errorf("no warning for an eleven-tile wall: %+v", got)
	}
}

// TestTuningChangesTheAnswer is the rule that the checks follow the physics:
// a gap too wide for the default jump becomes fine with a stronger one.
func TestTuningChangesTheAnswer(t *testing.T) {
	s := sketch{width: 80, gaps: [][2]int{{30, 42}}}
	if _, ok := codes(Run(s.json(), &game.Default))["gap-too-wide"]; !ok {
		t.Fatal("setup: the gap should be too wide with the default tuning")
	}
	strong := game.Default
	strong.RunMax = engine.FromFloat(6)
	strong.RunAccel = engine.FromFloat(0.5)
	strong.AirAccel = engine.FromFloat(0.3)
	if i, ok := codes(Run(s.json(), &strong))["gap-too-wide"]; ok {
		t.Errorf("with a much faster run the gap is still flagged: %s", i.Message)
	}
}
