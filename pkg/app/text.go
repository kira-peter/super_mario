package app

import (
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

// All text drawing goes through this file.
//
// The terminal gives every character a width of one or two cells, and the
// library that talks to the terminal decides which with go-runewidth. Using the
// same table here is what keeps layout and drawing in agreement. Earlier code
// guessed with "anything above U+1100 is two cells wide", which is wrong in both
// directions: it called arrows and box-drawing characters wide, and it advanced
// by one cell after drawing a wide character, so the next character landed on
// top of the wide one's second cell and cut it off.

// runeWidth returns how many cells a character occupies.
func runeWidth(r rune) int {
	w := runewidth.RuneWidth(r)
	if w < 1 {
		// Zero-width and control characters take no room of their own; give
		// them one cell so the cursor still advances and nothing overlaps.
		return 1
	}
	return w
}

// displayWidth returns the number of cells a run of text occupies.
func displayWidth(rs []rune) int {
	n := 0
	for _, r := range rs {
		n += runeWidth(r)
	}
	return n
}

// writeRunes writes text starting at an exact cell, advancing by each
// character's real width, and returns the column after the last one.
func writeRunes(s tcell.Screen, x, y int, text string, style tcell.Style) int {
	for _, r := range text {
		s.SetContent(x, y, r, nil, style)
		x += runeWidth(r)
	}
	return x
}

// writeCentered writes text centred in a span of w cells.
func writeCentered(s tcell.Screen, y, w int, text string, style tcell.Style) {
	if y < 0 {
		return
	}
	x := (w - displayWidth([]rune(text))) / 2
	if x < 0 {
		x = 0
	}
	writeRunesClipped(s, x, y, w, text, style)
}

// writeRunesClipped is writeRunes that stops at the edge instead of drawing
// past it. A wide character that would straddle the edge is dropped whole.
func writeRunesClipped(s tcell.Screen, x, y, limit int, text string, style tcell.Style) {
	for _, r := range text {
		rw := runeWidth(r)
		if x+rw > limit {
			return
		}
		s.SetContent(x, y, r, nil, style)
		x += rw
	}
}

// writeRight writes text so that it ends at column right (exclusive).
func writeRight(s tcell.Screen, right, y int, text string, style tcell.Style) {
	writeRunes(s, right-displayWidth([]rune(text)), y, text, style)
}

// writeAt writes text on one row, optionally centred within a span.
func writeAt(s tcell.Screen, x0, y, span int, text string, style tcell.Style, center bool) {
	x := x0 + 1
	if center {
		x = x0 + (span-displayWidth([]rune(text)))/2
	}
	writeRunes(s, x, y, text, style)
}

// fillRow paints a row segment with a style, used for highlight bars.
func fillRow(s tcell.Screen, x, y, w int, style tcell.Style) {
	for i := 0; i < w; i++ {
		s.SetContent(x+i, y, ' ', nil, style)
	}
}
