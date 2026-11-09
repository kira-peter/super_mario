package render

import (
	"testing"

	"super-mario/pkg/render/theme"
)

func TestHalfBlockConversion(t *testing.T) {
	// A 2x4 art buffer: top row red, bottom row blue, so each output row has
	// a different colour above and below.
	c := NewCanvas(2, 4)
	red := theme.Ink.PlayerBody
	blue := theme.Ink.PlayerAccent
	for x := 0; x < 2; x++ {
		c.Set(x, 0, red)
		c.Set(x, 1, blue)
		c.Set(x, 2, red)
		c.Set(x, 3, blue)
	}

	cells := c.Convert(2, 2, true)
	if len(cells) != 4 {
		t.Fatalf("got %d cells, want 4", len(cells))
	}
	for i, cell := range cells {
		if cell.Rune != '▀' {
			t.Errorf("cell %d rune = %q, want the upper half block", i, string(cell.Rune))
		}
		if cell.FG != red || cell.BG != blue {
			t.Errorf("cell %d colours = %v/%v, want red above blue", i, cell.FG, cell.BG)
		}
	}
}

func TestIdenticalHalvesCollapseToSpace(t *testing.T) {
	c := NewCanvas(2, 4)
	green := theme.Ink.Pipe
	c.Clear(green)

	cells := c.Convert(2, 2, true)
	for i, cell := range cells {
		if cell.Rune != ' ' {
			t.Errorf("cell %d rune = %q, want a space when both halves match",
				i, string(cell.Rune))
		}
		if cell.BG != green {
			t.Errorf("cell %d background = %v, want the shared colour", i, cell.BG)
		}
	}
}

func TestASCIIFallbackKeepsOneCellPerCell(t *testing.T) {
	c := NewCanvas(4, 4)
	c.Set(0, 0, theme.Ink.PlayerBody)
	c.Set(1, 1, theme.Ink.PlayerBody)

	cells := c.Convert(4, 2, false)
	if len(cells) != 8 {
		t.Fatalf("got %d cells, want 8 in ASCII mode too", len(cells))
	}
	for i, cell := range cells {
		switch cell.Rune {
		case ' ', '.', '=', '#':
		default:
			t.Errorf("cell %d rune = %q, not from the ASCII set", i, string(cell.Rune))
		}
	}
}

func TestSpriteSlotsResolveThroughTheme(t *testing.T) {
	if _, ok := SlotColor(theme.Ink, SlotBody); !ok {
		t.Error("body slot did not resolve")
	}
	if _, ok := SlotColor(theme.Ink, SlotNone); ok {
		t.Error("empty slot must not resolve to a colour")
	}
}

func TestSpriteMirrorIsHorizontalFlip(t *testing.T) {
	s := sprite(
		"AB.",
		".CD",
	)
	m := mirror(s)
	if m.Rows[0] != ".BA" {
		t.Errorf("mirrored row 0 = %q, want %q", m.Rows[0], ".BA")
	}
	if m.Rows[1] != "DC." {
		t.Errorf("mirrored row 1 = %q, want %q", m.Rows[1], "DC.")
	}
}

func TestEnemySilhouettesDifferInMonochrome(t *testing.T) {
	// Colour alone must not be what tells a Koopa from a Goomba: their heights
	// and outlines have to differ so a mono terminal stays readable.
	if Goomba.H == Koopa.H {
		t.Errorf("Goomba and Koopa are both %d rows tall; they would look alike without colour", Goomba.H)
	}
	if Shell.H >= Koopa.H {
		t.Error("a shell should be visibly shorter than a walking Koopa")
	}
}

func TestCoinFramesDiffer(t *testing.T) {
	// Compare whole silhouettes, not just the first row: a spin that only
	// moves its highlight still differs, and that difference is the point.
	seen := map[string]bool{}
	for i, f := range CoinFrames {
		key := ""
		for _, row := range f.Rows {
			key += row
		}
		if seen[key] {
			t.Errorf("coin frame %d is identical to an earlier frame", i)
		}
		seen[key] = true
	}
}

func TestDegradeToMonoDropsColour(t *testing.T) {
	mono := theme.Degrade(theme.Ink, theme.Mono)
	if mono.Full {
		t.Error("mono tokens should not claim full colour")
	}
	if mono.PlayerBody == theme.Ink.PlayerBody {
		t.Error("mono palette should not reuse the true-colour body colour")
	}
}

func TestConvertHandlesArtWiderThanTerminal(t *testing.T) {
	// A 160-pixel art buffer into a 120-column terminal must downsample, not
	// crop: the rightmost art pixels still have to influence the last cell.
	c := NewCanvas(160, 100)
	c.Clear(nil)
	c.Set(159, 99, theme.Ink.PlayerBody)

	cells := c.Convert(120, 50, true)
	last := cells[len(cells)-1]
	if last.BG == nil && last.FG == nil {
		t.Error("the bottom-right art pixel did not reach the last terminal cell")
	}
}
