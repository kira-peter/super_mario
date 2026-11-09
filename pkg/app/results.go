package app

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/save"
)

// groupDigits renders 17550 as "17,550".
func groupDigits(n int) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// signed renders a difference with an explicit sign, "+2,310" or "-480".
func signed(n int) string {
	if n >= 0 {
		return "+" + groupDigits(n)
	}
	return groupDigits(n)
}

// resultsLine is one row of the report card: a label on the left edge and a
// value on the right edge, so the figures line up as a column.
type resultsLine struct {
	label string
	note  string // small text after the label, e.g. "× 259"
	value string
	style tcell.Style
	rule  bool // a divider instead of text
	gap   bool // a blank row
}

// drawResults paints the end-of-level report card.
//
// It is laid out as a table, not as centred lines. Labels share a left edge and
// every figure shares one right edge, which is what lets the eye add the rows
// up: base score, plus the time bonus, equals the total.
func (a *App) drawResults(w, h int) {
	a.fillBackground(w, h)

	base := tcell.StyleDefault.Foreground(a.theme.HudText).Background(a.theme.Sky)
	muted := tcell.StyleDefault.Foreground(a.theme.HudMuted).Background(a.theme.Sky)
	accent := tcell.StyleDefault.Foreground(a.theme.PlayerBody).Background(a.theme.Sky)
	gold := tcell.StyleDefault.Foreground(a.theme.Question).Background(a.theme.Sky)
	good := tcell.StyleDefault.Foreground(a.theme.Pipe).Background(a.theme.Sky)
	bad := tcell.StyleDefault.Foreground(a.theme.PlayerBody).Background(a.theme.Sky)

	cleared := a.resultCleared
	t := a.world.Tally

	// Heading.
	heading, headStyle := "G A M E   O V E R", bad
	if cleared {
		heading, headStyle = "通 关", gold
	}

	// Score breakdown. The base is what the player had when they reached the
	// flag; the bonus is what the remaining time was worth.
	baseScore := t.Base
	if !cleared || baseScore == 0 && t.Awarded == 0 {
		baseScore = a.world.Score.Points - t.Awarded
	}
	if baseScore < 0 {
		baseScore = 0
	}

	var lines []resultsLine
	lines = append(lines, resultsLine{label: "基础得分", value: groupDigits(baseScore), style: base})
	if cleared {
		note := ""
		if t.Units > 0 {
			note = fmt.Sprintf("× %d", t.Units)
		}
		lines = append(lines, resultsLine{label: "时间奖励", note: note,
			value: "+" + groupDigits(t.Awarded), style: muted})
	}
	lines = append(lines,
		resultsLine{rule: true},
		resultsLine{label: "总  分", value: groupDigits(a.world.Score.Points), style: gold},
		resultsLine{gap: true},
		resultsLine{label: "用  时", value: save.FormatDuration(a.world.RunSteps()), style: base},
	)

	// Standing on the board. Only shown once the tally has finished, so the
	// rank does not flicker while the bonus is still counting up.
	settled := t.Done || !cleared
	if settled {
		switch {
		case a.resultRank == 1 && !a.resultHadPrev:
			lines = append(lines, resultsLine{label: "排  名", value: "第 1 名  ★ 首次上榜", style: gold})
		case a.resultRank == 1:
			lines = append(lines, resultsLine{label: "排  名", value: "第 1 名  ★ 新纪录", style: gold})
			lines = append(lines, resultsLine{label: "超过原纪录",
				value: signed(a.world.Score.Points - a.resultPrevBest.Score), style: good})
		case a.resultRank > 0:
			lines = append(lines, resultsLine{label: "排  名", value: fmt.Sprintf("第 %d 名", a.resultRank), style: base})
			if a.resultHadPrev {
				lines = append(lines, resultsLine{label: "距第 1 名",
					value: signed(a.world.Score.Points - a.resultPrevBest.Score), style: muted})
			}
		default:
			lines = append(lines, resultsLine{label: "排  名", value: "未上榜", style: muted})
		}
	}

	// Measure the card. The width is the widest label-plus-value row with room
	// between them, kept to a sensible minimum so short rows do not look lost.
	const minCard = 34
	card := minCard
	for _, ln := range lines {
		if ln.rule || ln.gap {
			continue
		}
		need := displayWidth([]rune(ln.label)) + displayWidth([]rune(ln.note)) +
			displayWidth([]rune(ln.value)) + 6
		if ln.note != "" {
			need += 2
		}
		if need > card {
			card = need
		}
	}
	if card > w-2 {
		card = w - 2
	}
	left := (w - card) / 2
	right := left + card

	// Vertical placement: heading, gap, body, gap, footer.
	total := 2 + len(lines) + 2
	y := (h - total) / 2
	if y < 1 {
		y = 1
	}

	writeCentered(a.screen, y, w, heading, headStyle)
	if cleared {
		writeCentered(a.screen, y+1, w, "WORLD "+a.lv.World, muted)
	}
	y += 3

	for _, ln := range lines {
		switch {
		case ln.gap:
			// blank row
		case ln.rule:
			for x := left; x < right; x++ {
				a.screen.SetContent(x, y, '─', nil, muted)
			}
		default:
			x := writeRunes(a.screen, left, y, ln.label, ln.style)
			if ln.note != "" {
				writeRunes(a.screen, x+2, y, ln.note, muted)
			}
			writeRight(a.screen, right, y, ln.value, ln.style)
		}
		y++
	}

	// Footer: only once there is something to continue to.
	y++
	if settled {
		writeCentered(a.screen, y, w, "空格 继续", muted)
	} else {
		writeCentered(a.screen, y, w, "结算中…", muted)
	}
	_ = accent
}
