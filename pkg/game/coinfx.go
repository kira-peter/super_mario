package game

import "super-mario/pkg/engine"

// Coin effects: a collected coin flies to the coin counter, and the counter
// shows a green "+1" that grows and fades.
//
// The coin is counted the moment it is collected, so the score is never behind
// the game; what waits is only the counter on screen, which shows the coin when
// it lands (see CoinsShown).

// CoinFX is one coin on its way to the counter.
type CoinFX struct {
	// Pos is the coin's world position while it pops out of a block, and where
	// the flight starts from.
	Pos engine.Vec2
	Vel engine.Vec2
	// Pop counts down the steps spent jumping out of a block before the flight
	// starts. A coin picked up in the level flies at once.
	Pop int
	// Fly counts the steps of the flight, up to CoinFlySteps.
	Fly int
}

// Progress is how far along the flight the coin is, from 0 to 1.
func (c *CoinFX) Progress() float64 {
	if c.Pop > 0 {
		return 0
	}
	return float64(c.Fly) / CoinFlySteps
}

// CoinPlus is the "+1" shown at the counter when coins land. Coins that land
// close together add to the one already showing instead of stacking up.
type CoinPlus struct {
	Amount int
	Age    int
}

const (
	// CoinPopSteps is how long a coin from a block spends jumping out of it.
	CoinPopSteps = 22
	// CoinFlySteps is how long the flight to the counter takes.
	CoinFlySteps = 24
	// CoinPlusSteps is how long the "+1" stays: it grows, holds, then fades.
	CoinPlusSteps = 45
	// coinPlusMerge is how young a "+1" must be for a landing coin to add to it.
	coinPlusMerge = 12
	// goldStagger spaces out the gold mushroom's coins so they fly in a line.
	goldStagger = 4
)

// coinPopSpeed is the upward speed of a coin knocked out of a block: high
// enough to clear the block by a couple of tiles.
var coinPopSpeed = engine.FromFloat(-4.5)

// popCoin knocks a coin up out of the block at a cell.
func (w *World) popCoin(col, row int) {
	w.CoinFX = append(w.CoinFX, &CoinFX{
		Pos: engine.Vec2{X: engine.Fixed(col)*engine.Tile + engine.Tile/2, Y: engine.Fixed(row)*engine.Tile - engine.Tile/2},
		Vel: engine.Vec2{Y: coinPopSpeed},
		Pop: CoinPopSteps,
	})
}

// flyCoin sends a coin to the counter from a world position. delay holds it
// back a few steps, so several coins at once fly one after another.
func (w *World) flyCoin(at engine.Vec2, delay int) {
	fx := &CoinFX{Pos: at}
	fx.Fly = -delay
	w.CoinFX = append(w.CoinFX, fx)
}

// updateCoinFX advances the coins in flight and the "+1" at the counter.
func (w *World) updateCoinFX() {
	live := w.CoinFX[:0]
	for _, c := range w.CoinFX {
		if c.Pop > 0 {
			c.Pop--
			c.Vel.Y += w.T.ItemGravity
			c.Pos = c.Pos.Add(c.Vel)
			live = append(live, c)
			continue
		}
		c.Fly++
		if c.Fly < CoinFlySteps {
			live = append(live, c)
			continue
		}
		w.landCoin()
	}
	w.CoinFX = live

	plus := w.CoinPlus[:0]
	for _, p := range w.CoinPlus {
		p.Age++
		if p.Age < CoinPlusSteps {
			plus = append(plus, p)
		}
	}
	w.CoinPlus = plus
}

// landCoin shows a coin arriving at the counter.
func (w *World) landCoin() {
	if n := len(w.CoinPlus); n > 0 && w.CoinPlus[n-1].Age < coinPlusMerge {
		w.CoinPlus[n-1].Amount++
		w.CoinPlus[n-1].Age = 0
		return
	}
	w.CoinPlus = append(w.CoinPlus, &CoinPlus{Amount: 1})
}

// CoinsShown is the coin count the counter should show: the coins collected,
// less the ones still flying to it.
func (w *World) CoinsShown() int {
	n := w.Score.Coins - len(w.CoinFX)
	for n < 0 {
		// A coin in flight carried the count over a hundred; until it lands,
		// show the count from before.
		n += maxCoins
	}
	return n
}
