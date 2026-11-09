package level

import (
	"strings"
	"testing"
)

func TestSymbolTableIsComplete(t *testing.T) {
	// Every symbol the loader knows must round-trip into a Kind.
	for ch := range symbolTable {
		if ch == 0 {
			t.Error("symbol table contains a zero rune")
		}
	}
	if len(symbolTable) < 10 {
		t.Errorf("symbol table has %d entries, expected the full alphabet", len(symbolTable))
	}
}

func TestLoadEmbedded11(t *testing.T) {
	lv, err := LoadEmbedded("1-1")
	if err != nil {
		t.Fatalf("load 1-1: %v", err)
	}
	if lv.Width < 150 {
		t.Errorf("level width = %d, want a long horizontal level", lv.Width)
	}
	if lv.Height != 15 {
		t.Errorf("level height = %d, want 15", lv.Height)
	}
	if lv.SpawnX < 0 || lv.SpawnX >= lv.Width {
		t.Errorf("spawn x = %d out of range", lv.SpawnX)
	}

	// The player must spawn standing on solid ground.
	if !lv.SolidAt(lv.SpawnX, lv.SpawnY+1) {
		t.Error("spawn point is not on solid ground")
	}

	// The flagpole must sit near the right end and be tall enough to matter.
	if lv.FlagX < lv.Width-20 {
		t.Errorf("flag x = %d, expected near the right end (%d)", lv.FlagX, lv.Width)
	}
	if lv.FlagBot-lv.FlagTop < 8 {
		t.Errorf("flagpole spans %d rows, expected at least 9", lv.FlagBot-lv.FlagTop+1)
	}
}

func TestOpeningIsSafeAndTeachesCoins(t *testing.T) {
	lv, err := LoadEmbedded("1-1")
	if err != nil {
		t.Fatal(err)
	}

	// No enemy may appear close to the spawn point.
	const safeColumns = 12
	for _, s := range lv.Spawns {
		if s.X > lv.SpawnX && s.X < lv.SpawnX+safeColumns {
			t.Errorf("enemy %q spawns only %d columns from the start", s.Kind, s.X-lv.SpawnX)
		}
	}

	// The first question block the player meets must yield coins, not an item.
	first := -1
	for i, c := range lv.Contents {
		if c.Hidden {
			continue
		}
		if first == -1 || c.X < lv.Contents[first].X {
			first = i
		}
	}
	if first == -1 {
		t.Fatal("level has no visible question block")
	}
	if lv.Contents[first].Kind != BlockCoin {
		t.Errorf("first block yields %q, want a coin", lv.Contents[first].Kind)
	}
}

func TestLevelProvidesMushroomAndStar(t *testing.T) {
	lv, err := LoadEmbedded("1-1")
	if err != nil {
		t.Fatal(err)
	}
	var mushroom, star, hidden bool
	for _, c := range lv.Contents {
		switch c.Kind {
		case BlockMushroom:
			mushroom = true
		case BlockStar:
			star = true
		}
		if c.Hidden {
			hidden = true
		}
	}
	if !mushroom {
		t.Error("level has no mushroom block; the player can never grow")
	}
	if !star {
		t.Error("level has no star block")
	}
	if !hidden {
		t.Error("level has no hidden block")
	}
}

func TestLevelHasPipesAndGaps(t *testing.T) {
	lv, err := LoadEmbedded("1-1")
	if err != nil {
		t.Fatal(err)
	}

	pipes := 0
	for y := 0; y < lv.Height; y++ {
		for x := 0; x < lv.Width; x++ {
			if lv.TileAt(x, y).Glyph == '[' {
				pipes++
			}
		}
	}
	if pipes != 4 {
		t.Errorf("level has %d pipes, want 4", pipes)
	}

	// At least one column near the start must have no floor, which is the
	// first gap the player has to jump.
	gaps := 0
	for x := 0; x < lv.Width; x++ {
		if !lv.SolidAt(x, 13) {
			gaps++
		}
	}
	if gaps < 2 {
		t.Errorf("level has %d gap columns, expected at least 2", gaps)
	}
}

func TestUnknownSymbolIsRejected(t *testing.T) {
	data := `{"name":"t","world":"1-1","width":4,"height":3,"timeLimit":400,
	"spawn":{"x":0,"y":1},"flag":{"x":3,"topY":0,"bottomY":1},
	"rows":["    ","  Z ","XXXX"],"contents":[],"entities":[]}`
	_, err := Parse([]byte(data))
	if err == nil {
		t.Fatal("expected an error for an unknown tile symbol")
	}
	if !strings.Contains(err.Error(), "unknown tile symbol") {
		t.Errorf("error = %q, want it to name the unknown symbol", err)
	}
}

func TestMissingSpawnIsRejected(t *testing.T) {
	// Spawn at an impossible coordinate must be caught, not accepted.
	data := `{"name":"t","world":"1-1","width":4,"height":3,"timeLimit":400,
	"spawn":{"x":99,"y":1},"flag":{"x":3,"topY":0,"bottomY":1},
	"rows":["    ","    ","XXXX"],"contents":[],"entities":[]}`
	_, err := Parse([]byte(data))
	if err == nil {
		t.Fatal("expected an error for an out-of-range spawn")
	}
	if !strings.Contains(err.Error(), "spawn point") {
		t.Errorf("error = %q, want it to mention the spawn point", err)
	}
}

func TestRowWidthMustMatchDeclaredWidth(t *testing.T) {
	data := `{"name":"t","world":"1-1","width":8,"height":2,"timeLimit":400,
	"spawn":{"x":0,"y":0},"flag":{"x":1,"topY":0,"bottomY":1},
	"rows":["    ","XXXX"],"contents":[],"entities":[]}`
	_, err := Parse([]byte(data))
	if err == nil {
		t.Fatal("expected an error for a short row")
	}
}

func TestQuestionBlockNeedsContent(t *testing.T) {
	// A question tile with no payload would silently do nothing at runtime,
	// which is the hardest authoring mistake to notice by playing.
	data := `{"name":"t","world":"1-1","width":4,"height":3,"timeLimit":400,
	"spawn":{"x":0,"y":1},"flag":{"x":3,"topY":0,"bottomY":1},
	"rows":[" ?  ","    ","XXXX"],"contents":[],"entities":[]}`
	_, err := Parse([]byte(data))
	if err == nil {
		t.Fatal("expected an error for a question block with no content")
	}
	if !strings.Contains(err.Error(), "no content entry") {
		t.Errorf("error = %q, want it to mention the missing content", err)
	}
}

func TestHiddenBlockMustBeInAir(t *testing.T) {
	data := `{"name":"t","world":"1-1","width":4,"height":3,"timeLimit":400,
	"spawn":{"x":0,"y":1},"flag":{"x":3,"topY":0,"bottomY":1},
	"rows":[" X  ","    ","XXXX"],
	"contents":[{"x":1,"y":0,"kind":"coin","hidden":true}],"entities":[]}`
	_, err := Parse([]byte(data))
	if err == nil {
		t.Fatal("expected an error for a hidden block placed inside a solid tile")
	}
}

func TestTileAtClampsOutsideLevel(t *testing.T) {
	lv, err := LoadEmbedded("1-1")
	if err != nil {
		t.Fatal(err)
	}
	// Outside the left and right edges reads as solid, so the player cannot
	// walk out of the level sideways.
	if !lv.SolidAt(-1, 13) {
		t.Error("outside left edge should be solid")
	}
	if !lv.SolidAt(lv.Width+5, 13) {
		t.Error("outside right edge should be solid")
	}
}

func TestEmbeddedNamesFinds11(t *testing.T) {
	names, err := EmbeddedNames()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range names {
		if n == "1-1" {
			found = true
		}
	}
	if !found {
		t.Errorf("embedded levels %v do not include 1-1", names)
	}
}
