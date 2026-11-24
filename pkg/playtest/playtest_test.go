package playtest

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"super-mario/pkg/engine"
	"super-mario/pkg/game"
	"super-mario/pkg/level"
)

func oneOne(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "level", "data", "1-1.json"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// script is a fixed run through the start of 1-1: run right, jumping at set
// moments. The same inputs feed both sides of the consistency test.
func script(step int) Input {
	in := InRight | InRun
	switch {
	case step%50 < 14:
		in |= InJump
	}
	if step > 600 && step < 640 {
		in = InLeft
	}
	return in
}

// TestPlaytestMatchesTheGame is the "a playtest is the real game" rule: the
// same level fed the same inputs ends the same way through the playtest
// session as through the game's own World, as the terminal drives it.
func TestPlaytestMatchesTheGame(t *testing.T) {
	data := oneOne(t)
	const steps = 900

	s, issues := Start(data, Options{FromColumn: -1, Seed: 77})
	if issues != nil {
		t.Fatal(issues)
	}

	lv, err := level.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	sm := engine.NewStateMachine()
	sm.StartRun()
	w := game.NewWorld(lv, &game.Default, sm, 3)
	w.SetSeed(77)
	w.SetViewport(engine.Tile*20, engine.Tile*15)
	// The playtest opens with his arrival, as the game does.
	w.StartEntrance()

	var held Input
	press := func(in Input) {
		pairs := []struct {
			bit Input
			key engine.Key
		}{{InLeft, engine.KeyLeft}, {InRight, engine.KeyRight}, {InDown, engine.KeyDown}, {InJump, engine.KeySpace}, {InRun, engine.KeyShift}}
		for _, p := range pairs {
			if in&p.bit != 0 && held&p.bit == 0 {
				w.Input.KeyDown(p.key)
			}
			if in&p.bit == 0 && held&p.bit != 0 {
				w.Input.KeyUp(p.key)
			}
		}
		held = in
	}

	for i := 0; i < steps; i++ {
		end := s.Step(script(i))
		press(script(i))
		w.Step()
		if s.World.Player.Pos != w.Player.Pos {
			t.Fatalf("step %d: playtest at %v, game at %v", i, s.World.Player.Pos, w.Player.Pos)
		}
		if end != Running {
			break
		}
	}
	if s.World.Score != w.Score {
		t.Errorf("score: playtest %+v, game %+v", s.World.Score, w.Score)
	}
	if s.World.Cleared != w.Cleared {
		t.Errorf("cleared: playtest %v, game %v", s.World.Cleared, w.Cleared)
	}
}

// TestPlaytestLeavesTheLevelAlone: smashing bricks and taking coins in a
// playtest does not change the level data the editor sent.
func TestPlaytestLeavesTheLevelAlone(t *testing.T) {
	data := oneOne(t)
	orig := append([]byte(nil), data...)
	s, _ := Start(data, Options{FromColumn: -1, StartBig: true, Invincible: true})
	for i := 0; i < 600; i++ {
		s.Step(script(i))
	}
	s.Stop()
	if !bytes.Equal(data, orig) {
		t.Error("the level bytes changed")
	}
	// And a fresh start sees the level as authored.
	again, _ := Start(data, Options{FromColumn: -1})
	lv, _ := level.Parse(data)
	for y := 0; y < lv.Height; y++ {
		for x := 0; x < lv.Width; x++ {
			if again.World.Level.TileAt(x, y) != lv.TileAt(x, y) {
				t.Fatalf("tile (%d,%d) differs in a new playtest", x, y)
			}
		}
	}
}

func TestStartRefusesALevelWithErrors(t *testing.T) {
	bad := bytes.Replace(oneOne(t), []byte(`"spawn": {"x": 2, "y": 12}`), []byte(`"spawn": {"x": 999, "y": 12}`), 1)
	s, issues := Start(bad, Options{FromColumn: -1})
	if s != nil || len(issues) == 0 {
		t.Fatal("a level without a valid spawn started")
	}
	if !strings.Contains(issues[0].Message, "spawn") {
		t.Errorf("the refusal does not say why: %s", issues[0].Message)
	}
}

func TestDeathEndsThePlaytest(t *testing.T) {
	s, _ := Start(oneOne(t), Options{FromColumn: -1})
	// Let him arrive first; nothing can happen while he is being assembled.
	for i := 0; i < game.RespawnDrop; i++ {
		s.Step(0)
	}
	s.World.Player.Pos.Y = engine.Fixed(20) * engine.Tile
	if end := s.Step(0); end != Died {
		t.Fatalf("end = %q after falling out of the level", end)
	}
	sum := s.Stop()
	if sum.End != Died || sum.Death == nil {
		t.Errorf("summary %+v", sum)
	}
}

func TestStartFromAColumn(t *testing.T) {
	s, _ := Start(oneOne(t), Options{FromColumn: 100})
	if c := int(s.World.Player.CenterX() / engine.Tile); c != 100 {
		t.Errorf("started in column %d", c)
	}
	// Column 70 is the first gap in 1-1.
	s, _ = Start(oneOne(t), Options{FromColumn: 70})
	if s.Notice == "" {
		t.Error("starting over a gap gave no notice")
	}
	if c := int(s.World.Player.CenterX() / engine.Tile); c != 2 {
		t.Errorf("a refused column did not fall back to the spawn (column %d)", c)
	}
}

func TestRenderGivesAFullFrame(t *testing.T) {
	s, _ := Start(oneOne(t), Options{FromColumn: -1, Zoom: 8, ViewW: 20, ViewH: 15})
	f := s.Render()
	if f.W != 160 || f.H != 120 || len(f.RGBA) != f.W*f.H*4 {
		t.Fatalf("frame %dx%d with %d bytes", f.W, f.H, len(f.RGBA))
	}
	for i := 3; i < len(f.RGBA); i += 4 {
		if f.RGBA[i] != 255 {
			t.Fatal("a game frame has a transparent pixel")
		}
	}
}

func TestShowDrawsListsRandomBlocks(t *testing.T) {
	data := bytes.Replace(oneOne(t),
		[]byte(`{"x": 16, "y": 9, "kind": "coin"}`),
		[]byte(`{"x": 16, "y": 9, "kind": "random", "pool": [{"kind": "star", "w": 1}]}`), 1)
	s, issues := Start(data, Options{FromColumn: -1, ShowDraws: true})
	if issues != nil {
		t.Fatal(issues)
	}
	f := s.Render()
	if len(f.Draws) != 1 || f.Draws[0].Kind != level.BlockStar || f.Draws[0].X != 16 {
		t.Errorf("draws = %+v", f.Draws)
	}
}

func TestSpritesCoverEveryElement(t *testing.T) {
	sp := Sprites(level.Style{}, 8)
	got := map[string]Sprite{}
	for _, s := range sp {
		got[s.ID] = s
		if len(s.RGBA) != s.W*s.H*4 {
			t.Errorf("%s: %d bytes for %dx%d", s.ID, len(s.RGBA), s.W, s.H)
		}
	}
	for _, c := range atlasCases {
		if _, ok := got[c.id]; !ok {
			t.Errorf("no sprite for %s", c.id)
		}
	}
	// A Koopa is wider than its tile and hangs over both sides, so the
	// sprite's offset has to say where it starts.
	if k := got["koopa"]; k.W <= 8 || k.OX >= 0 || k.OY+k.H != 8 {
		t.Errorf("koopa sprite %dx%d at offset %d,%d; want wider than a tile, feet on the tile's bottom", k.W, k.H, k.OX, k.OY)
	}
	// A brick fills its tile exactly.
	if b := got["brick"]; b.W != 8 || b.H != 8 || b.OX != 0 || b.OY != 0 {
		t.Errorf("brick sprite %dx%d at %d,%d", b.W, b.H, b.OX, b.OY)
	}
}

// TestSpritesFollowTheStyle: a blue-brick level gets blue brick sprites.
func TestSpritesFollowTheStyle(t *testing.T) {
	blue := level.RGB(0x3d6fb8)
	var brick Sprite
	for _, s := range Sprites(level.Style{Brick: &blue}, 8) {
		if s.ID == "brick" {
			brick = s
		}
	}
	found := false
	for i := 0; i+3 < len(brick.RGBA); i += 4 {
		if brick.RGBA[i] == 0x3d && brick.RGBA[i+1] == 0x6f && brick.RGBA[i+2] == 0xb8 {
			found = true
		}
	}
	if !found {
		t.Error("the brick sprite has no blue in it")
	}
}

func TestThemePreviewDrawsEveryTheme(t *testing.T) {
	for _, th := range level.Themes {
		f := ThemePreview(th.ID, 4)
		if f.W == 0 || len(f.RGBA) != f.W*f.H*4 {
			t.Errorf("theme %s: empty preview", th.ID)
		}
	}
}

func TestSkyColoursRunTopToBottom(t *testing.T) {
	sky := SkyColours(level.Style{Sky: &level.Sky{Top: 0x000000, Bottom: 0xffffff}}, 4)
	if len(sky) != 60 || sky[0] != "#000000" || sky[59] != "#ffffff" {
		t.Errorf("sky = %d rows, %s .. %s", len(sky), sky[0], sky[len(sky)-1])
	}
	plain := SkyColours(level.Style{}, 4)
	if plain[0] != plain[59] {
		t.Error("an unstyled sky is not flat")
	}
}
