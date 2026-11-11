package game

import (
	"testing"

	"super-mario/pkg/engine"
	"super-mario/pkg/level"
)

// TestBlockCoinPopsThenFlies checks a coin from a block jumps up out of it,
// then flies to the counter, and the counter only shows it once it lands.
func TestBlockCoinPopsThenFlies(t *testing.T) {
	w := newTestWorld(t)
	col, row := w.Level.SpawnX+2, w.Level.SpawnY-3
	w.Level.SetTile(col, row, level.Tile{Kind: level.KindQuestion, Glyph: '?'})
	w.Level.Contents = append(w.Level.Contents, level.Content{X: col, Y: row, Kind: level.BlockCoin, Count: 1})
	before := w.Score.Coins

	w.dispense(col, row)
	if w.Score.Coins != before+1 {
		t.Fatalf("coins %d -> %d, want the coin counted at once", before, w.Score.Coins)
	}
	if len(w.CoinFX) != 1 || w.CoinFX[0].Pop == 0 {
		t.Fatal("a coin from a block should start by popping out of it")
	}
	if w.CoinsShown() != before {
		t.Errorf("the counter shows %d while the coin is still in the air, want %d", w.CoinsShown(), before)
	}

	startY := w.CoinFX[0].Pos.Y
	for i := 0; i < CoinPopSteps/2; i++ {
		w.updateCoinFX()
	}
	if w.CoinFX[0].Pos.Y >= startY {
		t.Error("the coin did not rise out of the block")
	}

	for i := 0; i < CoinPopSteps+CoinFlySteps; i++ {
		w.updateCoinFX()
	}
	if len(w.CoinFX) != 0 {
		t.Fatal("the coin never reached the counter")
	}
	if w.CoinsShown() != before+1 {
		t.Errorf("the counter shows %d after the coin landed, want %d", w.CoinsShown(), before+1)
	}
	if len(w.CoinPlus) != 1 || w.CoinPlus[0].Amount != 1 {
		t.Errorf("want one \"+1\" at the counter, got %v", w.CoinPlus)
	}
}

// TestPickedUpCoinFliesToo checks a coin taken in the level flies at once.
func TestPickedUpCoinFliesToo(t *testing.T) {
	w, cx, cy := coinWorld(t)
	w.Player.Pos.X = engine.Fixed(cx)*engine.Tile + 2
	w.Player.Pos.Y = engine.Fixed(cy+1)*engine.Tile - w.Player.H
	w.collectCoins()
	if len(w.CoinFX) != 1 || w.CoinFX[0].Pop != 0 {
		t.Fatal("a coin picked up in the level should fly straight to the counter")
	}
}

// TestCoinPlusFadesAway checks the "+1" goes away on its own.
func TestCoinPlusFadesAway(t *testing.T) {
	w := newTestWorld(t)
	w.landCoin()
	for i := 0; i < CoinPlusSteps; i++ {
		w.updateCoinFX()
	}
	if len(w.CoinPlus) != 0 {
		t.Error("the \"+1\" is still showing after its time")
	}
}

// TestCoinsLandingTogetherAddUp checks the gold mushroom's five coins fly in a
// line and show as one count going up, not five labels on top of each other.
func TestCoinsLandingTogetherAddUp(t *testing.T) {
	w := newTestWorld(t)
	pickUp(t, w, ItemGold)
	if len(w.CoinFX) != goldCoins {
		t.Fatalf("%d coins flying after a gold mushroom, want %d", len(w.CoinFX), goldCoins)
	}
	for i := 0; i < CoinFlySteps+goldCoins*goldStagger; i++ {
		w.updateCoinFX()
	}
	if len(w.CoinPlus) != 1 || w.CoinPlus[0].Amount != goldCoins {
		t.Errorf("want one \"+%d\", got %d labels", goldCoins, len(w.CoinPlus))
	}
}

// TestCounterDoesNotGoNegativeOverAHundred checks the count shown while a coin
// that carried it past a hundred is still flying.
func TestCounterDoesNotGoNegativeOverAHundred(t *testing.T) {
	w := newTestWorld(t)
	w.Score.Coins = 99
	w.Score.addCoins(1)
	w.flyCoin(engine.Vec2{}, 0)
	if got := w.CoinsShown(); got != 99 {
		t.Errorf("counter shows %d while the hundredth coin flies, want 99", got)
	}
}

// TestMushroomTurnsAtAPipe is the regression test for a mushroom that stuck to
// the first wall it met: the collision resolve zeroes its speed, and negating
// zero left it there.
func TestMushroomTurnsAtAPipe(t *testing.T) {
	w := newTestWorld(t)
	col, row := w.Level.SpawnX+4, w.Level.SpawnY
	w.Level.SetTile(col, row, level.Tile{Kind: level.KindSolid, Glyph: '['})
	it := NewItem(ItemMushroom, col-2, row)
	it.Emerging = 0
	it.Pos.Y = engine.Fixed(row+1)*engine.Tile - it.H
	it.Vel.X = w.T.ItemWalkSpeed
	it.Facing = 1
	// Long enough to reach the pipe and turn, short enough not to reach the
	// left edge of the level and turn again.
	for i := 0; i < 40; i++ {
		it.update(w.Level, w.T)
	}
	if it.Vel.X >= 0 {
		t.Fatalf("mushroom speed %v after meeting the pipe, want it walking left", it.Vel.X)
	}
	if it.Right() > engine.Fixed(col)*engine.Tile {
		t.Error("the mushroom went into the pipe")
	}
	if it.Pos.X >= engine.Fixed(col-2)*engine.Tile {
		t.Error("the mushroom turned but did not walk away from the pipe")
	}
}

// shellWorld is a test world with a resting shell on the ground in front of
// the player.
func shellWorld(t *testing.T) (*World, *Enemy) {
	t.Helper()
	w := newTestWorld(t)
	e := NewKoopa(w.Level.SpawnX+4, w.Level.SpawnY)
	e.Active = true
	e.toShell()
	e.Grace = 0
	w.Enemies = []*Enemy{e}
	return w, e
}

// landOn drops the player onto the enemy at a horizontal offset from its centre.
func landOn(w *World, e *Enemy, dx engine.Fixed) {
	p := w.Player
	p.Pos.X = e.CenterX() + dx - p.W/2
	p.Pos.Y = e.Pos.Y - p.H + 2
	p.Vel.Y = engine.FromFloat(2)
	w.resolveEnemyContacts()
}

// TestStompingAShellKicksItAwayFromTheFoot checks the second stomp: landing on
// the left half sends the shell right, on the right half sends it left.
func TestStompingAShellKicksItAwayFromTheFoot(t *testing.T) {
	w, e := shellWorld(t)
	landOn(w, e, -engine.FromFloat(4))
	if e.Shell != KoopaSliding || e.Vel.X <= 0 {
		t.Errorf("landed on the left of the shell: state %v speed %v, want sliding right", e.Shell, e.Vel.X)
	}

	w, e = shellWorld(t)
	landOn(w, e, engine.FromFloat(4))
	if e.Shell != KoopaSliding || e.Vel.X >= 0 {
		t.Errorf("landed on the right of the shell: state %v speed %v, want sliding left", e.Shell, e.Vel.X)
	}
}

// TestFirstStompOnlyMakesAShell checks the first stomp turns the Koopa into a
// resting shell, and the bounce off it does not kick it straight away.
func TestFirstStompOnlyMakesAShell(t *testing.T) {
	w := newTestWorld(t)
	e := NewKoopa(w.Level.SpawnX+4, w.Level.SpawnY)
	e.Active = true
	w.Enemies = []*Enemy{e}
	landOn(w, e, 0)
	if e.Shell != KoopaShell {
		t.Fatalf("state %v after the first stomp, want a resting shell", e.Shell)
	}
	// The player is still overlapping the shell on the next steps.
	for i := 0; i < 3; i++ {
		w.resolveEnemyContacts()
	}
	if e.Shell != KoopaShell {
		t.Errorf("the fresh shell was kicked by the bounce that made it (state %v)", e.Shell)
	}
	if w.Player.Dead {
		t.Error("the player died on the shell he just made")
	}
}

// TestKickedShellDoesNotHurtTheKicker checks the shell that was just kicked
// does not hurt the player it is still touching.
func TestKickedShellDoesNotHurtTheKicker(t *testing.T) {
	w, e := shellWorld(t)
	landOn(w, e, -engine.FromFloat(4))
	w.Player.Vel.Y = 0
	w.resolveEnemyContacts()
	if w.Player.Dead {
		t.Error("the player died to the shell he just kicked")
	}
}
