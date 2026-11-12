package game

import (
	"super-mario/pkg/engine"
	"super-mario/pkg/level"
)

// The end-of-level sequence, after the original game:
//
//  1. Touching the pole grabs it, and scores by how high up it was grabbed.
//  2. Mario slides down while the flag slides down with him.
//  3. At the bottom he steps round to the far side of the pole and walks to the
//     castle, and disappears into it.
//  4. Only then does the results screen start counting the time bonus.

type flagPhase int

const (
	// flagSliding is the slide down the pole.
	flagSliding flagPhase = iota
	// flagWalking is the walk to the castle.
	flagWalking
	// flagGone is the pause after he has gone in, before the results.
	flagGone
	// flagCatchUp is the flag racing down to where he grabbed the pole, before
	// the two slide down together. It is listed last so flagSliding stays the
	// zero value.
	flagCatchUp
)

const (
	// flagSlideSpeed is how fast both Mario and the flag slide, per step.
	flagSlideSpeed = 2
	// flagCatchUpSpeed is how fast the flag drops to meet him, per step: four
	// times the slide, so it arrives in a few steps and then goes down with him.
	flagCatchUpSpeed = 8
	// flagWalkSpeed is his walking speed to the castle, per step.
	flagWalkSpeed = 1
	// flagClothHeight is the height of the flag cloth, in game units: three
	// pixels of the four-pixel drawing.
	flagClothHeight = 12
	// CastleDistance is how far right of the pole the castle door is, in tiles:
	// the same place the renderer draws the castle's door.
	CastleDistance = 7
	// flagGoneSteps is the pause once he is inside.
	flagGoneSteps = 30
	// FlagBonusSteps is how long the score from the pole stays on screen.
	FlagBonusSteps = 90
)

// flagTiers are the pole scores, highest first. frac is the share of the pole
// below the grab point, in percent: grabbing near the top leaves nearly all of
// it below him. A grab lower than every tier is worth flagFloorScore.
var flagTiers = []struct{ frac, score int }{
	{85, 5000},
	{65, 2000},
	{45, 800},
	{25, 400},
}

// flagFloorScore is the score for grabbing the pole at the very bottom.
const flagFloorScore = 100

// FlagBonus is the score that floats up the pole after a grab.
type FlagBonus struct {
	Amount int
	// Y is where he grabbed the pole, in game units: where the number starts.
	Y   engine.Fixed
	Age int
}

// flagpoleScore rewards grabbing the pole higher up.
func flagpoleScore(y engine.Fixed, lv *level.Level) int {
	height := engine.Fixed(lv.FlagBot-lv.FlagTop+1) * engine.Tile
	if height <= 0 {
		return flagFloorScore
	}
	frac := int((engine.Fixed(lv.FlagBot+1)*engine.Tile - y) * 100 / height)
	for _, t := range flagTiers {
		if frac > t.frac {
			return t.score
		}
	}
	return flagFloorScore
}

// startFlagpole grabs the pole.
func (w *World) startFlagpole() {
	if !w.SM.TouchFlag() {
		return
	}
	p := w.Player
	p.SlideDown = true
	w.Cleared = true
	w.ClearedAt = w.stepCount
	p.Vel = engine.Vec2{}
	p.Pos.X = engine.Fixed(w.Level.FlagX)*engine.Tile - p.W/2

	points := flagpoleScore(p.Pos.Y, w.Level)
	w.Score.add(points)
	w.FlagBonus = &FlagBonus{Amount: points, Y: p.Pos.Y}
	w.FlagDrop = 0
	w.flagPhase = flagCatchUp
	w.flagWait = 0
}

// stepFlagpole advances the end-of-level sequence.
func (w *World) stepFlagpole() {
	w.updateCoinFX()
	if w.FlagBonus != nil {
		w.FlagBonus.Age++
		if w.FlagBonus.Age >= FlagBonusSteps {
			w.FlagBonus = nil
		}
	}
	p := w.Player
	lv := w.Level

	switch w.flagPhase {
	case flagCatchUp:
		// He hangs on the pole while the flag comes down to his height: the top
		// of the cloth stops level with the top of his head.
		floor := engine.Fixed(lv.FlagBot+1) * engine.Tile
		maxDrop := floor - engine.FromFloat(flagClothHeight) - engine.Fixed(lv.FlagTop+1)*engine.Tile
		target := p.Pos.Y - engine.Fixed(lv.FlagTop+1)*engine.Tile
		if target < 0 {
			target = 0
		}
		if target > maxDrop {
			target = maxDrop
		}
		w.FlagDrop += engine.FromFloat(flagCatchUpSpeed)
		if w.FlagDrop >= target {
			w.FlagDrop = target
			w.flagPhase = flagSliding
		}

	case flagSliding:
		floor := engine.Fixed(lv.FlagBot+1) * engine.Tile
		speed := engine.FromFloat(flagSlideSpeed)
		if p.Pos.Y+p.H < floor {
			p.Pos.Y += speed
			if p.Pos.Y+p.H > floor {
				p.Pos.Y = floor - p.H
			}
		}
		// The cloth starts just under the ball and stops on the last row.
		maxDrop := floor - engine.FromFloat(flagClothHeight) - engine.Fixed(lv.FlagTop+1)*engine.Tile
		if w.FlagDrop < maxDrop {
			w.FlagDrop += speed
			if w.FlagDrop > maxDrop {
				w.FlagDrop = maxDrop
			}
		}
		if p.Pos.Y+p.H >= floor && w.FlagDrop >= maxDrop {
			// Step round to the far side of the pole, facing the castle.
			w.flagPhase = flagWalking
			p.SlideDown = false
			p.Facing = 1
			p.Pos.X = engine.Fixed(lv.FlagX+1) * engine.Tile
		}

	case flagWalking:
		p.Pos.X += engine.FromFloat(flagWalkSpeed)
		p.advanceWheel(engine.FromFloat(flagWalkSpeed))
		if p.Pos.X >= engine.Fixed(lv.FlagX+CastleDistance)*engine.Tile {
			p.Hidden = true
			w.flagPhase = flagGone
		}

	case flagGone:
		w.flagWait++
		if w.flagWait >= flagGoneSteps {
			w.SM.ShowResults()
			w.startTally()
		}
	}
}
