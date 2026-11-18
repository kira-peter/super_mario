// Package lint gives a level designer the second tier of feedback: things that
// load fine but probably play badly (spec level-contribution / 两级设计检查).
//
// Errors come straight from level.Check, the same code the game loads with.
// Warnings come from here. They never stop a level being saved: a designer may
// want a jump that needs the mid-air second jump, or a block that is meant to
// be found by walking along its top.
package lint

import (
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/game"
	"super-mario/pkg/level"
	"super-mario/pkg/render/theme"
)

// Severity is how serious an issue is.
type Severity string

const (
	// Error stops the level from being saved or played: the game would
	// refuse to load it.
	Error Severity = "error"
	// Warning is a likely design problem. It is shown but stops nothing.
	Warning Severity = "warning"
)

// Issue is one finding, about one cell where it can say which.
type Issue struct {
	Severity Severity `json:"severity"`
	// X and Y are the cell, or -1 when the issue is about the whole level.
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Code    string `json:"code"`
	Message string `json:"message"`
	// Args are the values in the message, by name, for the editor's other
	// language.
	Args map[string]string `json:"args,omitempty"`
}

// withArgs names the values in an issue's message.
func withArgs(i Issue, kv ...interface{}) Issue {
	i.Args = map[string]string{}
	for k := 0; k+1 < len(kv); k += 2 {
		i.Args[kv[k].(string)] = fmt.Sprint(kv[k+1])
	}
	return i
}

// spawnSafeColumns is how far ahead of the start an enemy counts as "right at
// the start". 1-1 keeps this clear, and so should a level a new player opens.
const spawnSafeColumns = 8

// minDistinct is the smallest perceptual difference (CIEDE2000) a gameplay
// colour may have from the sky before it is flagged. Twelve is clearly visible
// at a glance; below about ten a small sprite starts to blend in.
const minDistinct = 12.0

// Run checks a level file and returns every error and warning. Errors come
// first; if there are any, warnings are not computed, since they would be
// about a level the game cannot load.
func Run(data []byte, t *game.Tuning) []Issue {
	if probs := level.Check(data); len(probs) > 0 {
		out := make([]Issue, len(probs))
		for i, p := range probs {
			out[i] = Issue{Severity: Error, X: p.X, Y: p.Y, Code: p.Code, Message: p.Message, Args: p.Args}
		}
		return out
	}
	lv, err := level.Parse(data)
	if err != nil {
		// Check and Parse share their rules; this cannot happen.
		return []Issue{{Severity: Error, X: -1, Y: -1, Code: "internal", Message: err.Error()}}
	}
	return Warnings(lv, t)
}

// Warnings runs every design check on a loaded level.
func Warnings(lv *level.Level, t *game.Tuning) []Issue {
	var out []Issue
	out = append(out, spawnChecks(lv)...)
	out = append(out, blockChecks(lv)...)
	out = append(out, enemyChecks(lv)...)
	out = append(out, contrastChecks(lv)...)
	out = append(out, Reach(lv, t)...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].X != out[j].X {
			return out[i].X < out[j].X
		}
		return out[i].Y < out[j].Y
	})
	return out
}

func warn(x, y int, code, format string, args ...interface{}) Issue {
	return Issue{Severity: Warning, X: x, Y: y, Code: code, Message: fmt.Sprintf(format, args...)}
}

// standable reports whether a body can stand in cell (x, y): the cell is open
// and the one below it is solid.
func standable(lv *level.Level, x, y int) bool {
	return !lv.SolidAt(x, y) && lv.SolidAt(x, y+1) && y+1 < lv.Height
}

func spawnChecks(lv *level.Level) []Issue {
	var out []Issue
	if !lv.SolidAt(lv.SpawnX, lv.SpawnY+1) || lv.SpawnY+1 >= lv.Height {
		out = append(out, warn(lv.SpawnX, lv.SpawnY, "spawn-no-ground",
			"出生点下方没有地面，玩家一开始就会掉下去"))
	}
	if lv.SolidAt(lv.SpawnX, lv.SpawnY) {
		out = append(out, warn(lv.SpawnX, lv.SpawnY, "spawn-in-solid",
			"出生点在实心格里，玩家会卡住"))
	}
	for _, s := range lv.Spawns {
		if d := s.X - lv.SpawnX; d >= 0 && d < spawnSafeColumns {
			out = append(out, withArgs(warn(s.X, s.Y, "spawn-enemy-near",
				"出生点前 %d 格内有敌人，玩家还没准备好就要应付它", spawnSafeColumns), "n", spawnSafeColumns))
		}
	}
	return out
}

func blockChecks(lv *level.Level) []Issue {
	var out []Issue
	for _, c := range lv.Contents {
		if c.Hidden {
			continue
		}
		// A block is hit from below: the cell under it must be open, or no
		// one can ever jump into it.
		if c.Y+1 >= lv.Height || lv.SolidAt(c.X, c.Y+1) {
			out = append(out, warn(c.X, c.Y, "block-sealed",
				"这个方块正下方被挡住，没法从下面顶到"))
		}
	}
	return out
}

func enemyChecks(lv *level.Level) []Issue {
	var out []Issue
	for _, s := range lv.Spawns {
		switch {
		case lv.SolidAt(s.X, s.Y):
			out = append(out, warn(s.X, s.Y, "enemy-in-solid",
				"敌人生成在实心格里，会卡住或被挤出"))
		case !lv.SolidAt(s.X, s.Y+1):
			// Floating enemies fall; that is sometimes intended (dropping onto
			// a platform), so say what will happen rather than call it wrong.
			out = append(out, warn(s.X, s.Y, "enemy-floating",
				"敌人脚下是空的，开局会直接掉下去"))
		}
	}
	return out
}

// contrastChecks flags gameplay colours that nearly match the sky. It judges
// the level's look as a true-colour terminal shows it, the most faithful one.
func contrastChecks(lv *level.Level) []Issue {
	if lv.Style.IsZero() {
		return nil
	}
	t, sky := theme.Apply(theme.Ink, lv.Style, theme.TrueColor)
	type pair struct {
		code, name string
		c          tcell.Color
	}
	pairs := []pair{
		{"contrast-brick", "砖块", t.Brick},
		{"contrast-question", "问号块", t.Question},
		{"contrast-coin", "金币", t.Coin},
		{"contrast-enemy", "敌人", t.Cast.GoombaBody},
		{"contrast-player", "玩家", t.PlayerBody},
		{"contrast-pipe", "水管", t.Pipe},
	}
	// The element names in English, for the editor's other language.
	english := map[string]string{
		"contrast-brick": "Bricks", "contrast-question": "Question blocks", "contrast-coin": "Coins",
		"contrast-enemy": "Enemies", "contrast-player": "Mario", "contrast-pipe": "Pipes",
	}
	var out []Issue
	for _, p := range pairs {
		// Against the top, middle and bottom of the sky, take the worst.
		worst := theme.Distinctness(p.c, sky.Top)
		if d := theme.Distinctness(p.c, sky.Bottom); d < worst {
			worst = d
		}
		if d := theme.Distinctness(p.c, sky.At(0.5)); d < worst {
			worst = d
		}
		if worst < minDistinct {
			out = append(out, withArgs(warn(-1, -1, p.code,
				"%s和天空颜色太接近（色差 %.0f，建议至少 %.0f），玩家可能看不清", p.name, worst, minDistinct),
				"nameZh", p.name, "nameEn", english[p.code],
				"d", fmt.Sprintf("%.0f", worst), "min", fmt.Sprintf("%.0f", minDistinct)))
		}
	}
	return out
}
