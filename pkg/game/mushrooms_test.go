package game

import (
	"testing"

	"super-mario/pkg/engine"
	"super-mario/pkg/level"
)

// pickUp places a mushroom of a kind on the player and runs the contact check.
func pickUp(t *testing.T, w *World, kind ItemKind) {
	t.Helper()
	it := NewItem(kind, 0, 0)
	it.Emerging = 0
	it.Pos = w.Player.Pos
	w.Items = append(w.Items, it)
	w.resolveItemContacts()
	if !it.Removed {
		t.Fatalf("mushroom %d was not picked up", kind)
	}
}

func TestRedMushroomGrows(t *testing.T) {
	w := newTestWorld(t)
	pickUp(t, w, ItemMushroom)
	if w.Player.Form != FormBig {
		t.Errorf("form %v after a red mushroom, want big", w.Player.Form)
	}
}

func TestBlueMushroomMakesInvincible(t *testing.T) {
	w := newTestWorld(t)
	pickUp(t, w, ItemStar)
	if !w.Player.Invulnerable() {
		t.Error("not invincible after a blue mushroom")
	}
}

func TestGreenMushroomGivesALife(t *testing.T) {
	w := newTestWorld(t)
	before := w.Score.Lives
	pickUp(t, w, ItemOneUp)
	if w.Score.Lives != before+1 {
		t.Errorf("lives %d -> %d after a green mushroom, want one more", before, w.Score.Lives)
	}
}

func TestGoldMushroomGivesFiveCoins(t *testing.T) {
	w := newTestWorld(t)
	before := w.Score.Coins
	pickUp(t, w, ItemGold)
	if w.Score.Coins != before+goldCoins {
		t.Errorf("coins %d -> %d after a gold mushroom, want +%d", before, w.Score.Coins, goldCoins)
	}
}

// TestPurpleMushroomHurts checks the poison mushroom costs a form, and kills a
// small Mario, the same as touching an enemy.
func TestPurpleMushroomHurts(t *testing.T) {
	w := newTestWorld(t)
	w.Player.Grow(w.Level)
	w.Player.GrowFlash = 0
	pickUp(t, w, ItemPoison)
	if w.Player.Form != FormSmall {
		t.Errorf("form %v after poison while big, want small", w.Player.Form)
	}

	w = newTestWorld(t)
	pickUp(t, w, ItemPoison)
	if !w.Player.Dead {
		t.Error("a small Mario survived the poison mushroom")
	}
}

// TestInvincibilityProtectsFromPoison checks the blue mushroom also covers the
// purple one.
func TestInvincibilityProtectsFromPoison(t *testing.T) {
	w := newTestWorld(t)
	pickUp(t, w, ItemStar)
	pickUp(t, w, ItemPoison)
	if w.Player.Dead {
		t.Error("an invincible Mario was killed by the poison mushroom")
	}
}

// TestEveryBlockKindReleasesItsMushroom checks each payload in the level data
// turns into the matching pickup when the block is hit.
func TestEveryBlockKindReleasesItsMushroom(t *testing.T) {
	for kind, want := range map[level.BlockKind]ItemKind{
		level.BlockMushroom: ItemMushroom,
		level.BlockStar:     ItemStar,
		level.BlockOneUp:    ItemOneUp,
		level.BlockPoison:   ItemPoison,
		level.BlockGold:     ItemGold,
	} {
		w := newTestWorld(t)
		col, row := w.Level.SpawnX+2, w.Level.SpawnY-3
		w.Level.SetTile(col, row, level.Tile{Kind: level.KindQuestion, Glyph: '?'})
		w.Level.Contents = append(w.Level.Contents, level.Content{X: col, Y: row, Kind: kind, Count: 1})
		w.dispense(col, row)
		if len(w.Items) != 1 || w.Items[0].Kind != want {
			t.Errorf("a %q block released %v, want one item of kind %d", kind, w.Items, want)
		}
	}
}

// TestLevelHasEveryMushroom checks 1-1 actually offers all five.
func TestLevelHasEveryMushroom(t *testing.T) {
	lv, err := level.LoadEmbedded("1-1")
	if err != nil {
		t.Fatal(err)
	}
	have := map[ItemKind]bool{}
	for _, c := range lv.Contents {
		if k, ok := itemFor(c.Kind); ok {
			have[k] = true
		}
	}
	for _, k := range []ItemKind{ItemMushroom, ItemStar, ItemOneUp, ItemPoison, ItemGold} {
		if !have[k] {
			t.Errorf("1-1 has no block that releases mushroom kind %d", k)
		}
	}
}

// TestKoopaBoxMatchesItsDrawing pins the box to the drawing: a tile and a half
// long, one tile tall, so it can be stomped where it is seen.
func TestKoopaBoxMatchesItsDrawing(t *testing.T) {
	e := NewKoopa(10, 12)
	if e.W <= engine.Tile || e.H > engine.Tile {
		t.Errorf("koopa box is %v x %v, want wider than a tile and no taller than one", e.W, e.H)
	}
	bottom, centre := e.Bottom(), e.CenterX()
	e.toShell()
	if e.Bottom() != bottom {
		t.Error("the shell did not keep the Koopa's feet where they were")
	}
	if d := e.CenterX() - centre; d < -1 || d > 1 {
		t.Errorf("the shell moved sideways by %v when the Koopa tucked in", d)
	}
	if e.W >= NewKoopa(10, 12).W {
		t.Error("the shell is not narrower than the walking Koopa")
	}
}
