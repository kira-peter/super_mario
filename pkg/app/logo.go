package app

import "super-mario/pkg/render"

// glyphs is a 5x5 dot-matrix face for the title.
//
// Every stroke is exactly one dot thick and every letter is five dots tall, so
// the word reads as one typeface rather than a set of unrelated shapes. A dot
// is one art pixel; the renderer packs two rows of dots into each terminal row.
var glyphs = map[rune][5]string{
	'S': {
		".####",
		"#....",
		".###.",
		"....#",
		"####.",
	},
	'U': {
		"#...#",
		"#...#",
		"#...#",
		"#...#",
		".###.",
	},
	'P': {
		"####.",
		"#...#",
		"####.",
		"#....",
		"#....",
	},
	'E': {
		"#####",
		"#....",
		"###..",
		"#....",
		"#####",
	},
	'R': {
		"####.",
		"#...#",
		"####.",
		"#..#.",
		"#...#",
	},
	'M': {
		"#...#",
		"##.##",
		"#.#.#",
		"#...#",
		"#...#",
	},
	'A': {
		".###.",
		"#...#",
		"#####",
		"#...#",
		"#...#",
	},
	'I': {
		"#####",
		"..#..",
		"..#..",
		"..#..",
		"#####",
	},
	'O': {
		".###.",
		"#...#",
		"#...#",
		"#...#",
		".###.",
	},
}

// Logo slot letters. The title reuses the sprite palette so the logo changes
// with the theme like everything else:
//
//	R  body red        the top word
//	C  coin gold       the bottom word
//	K  outline         the drop shadow
const (
	logoTop    = 'R'
	logoBottom = 'C'
	logoShadow = 'K'
)

// buildWord lays a word out as a sprite, with a one-dot gap between letters and
// a shadow row underneath.
//
// The shadow sits straight below each letter rather than offset down and to
// the right. With strokes only one dot thick, a diagonal shadow is as thick as
// the stroke it follows and fills the gaps inside letters like S and R, so the
// word turns to mush. A shadow directly beneath leaves every outline intact and
// reads as the word standing on a surface.
func buildWord(word string, fill byte) render.Sprite {
	const h = 5
	runes := []rune(word)

	w := len(runes)*5 + (len(runes) - 1)
	grid := make([][]byte, h+1)
	for y := range grid {
		grid[y] = make([]byte, w)
		for x := range grid[y] {
			grid[y][x] = '.'
		}
	}

	x0 := 0
	for _, r := range runes {
		g, ok := glyphs[r]
		if !ok {
			x0 += 6
			continue
		}
		for x := 0; x < 5; x++ {
			// Shadow only under the bottom stroke of the glyph, so it never
			// intrudes into the counters of the letters above it.
			if g[h-1][x] == '#' {
				grid[h][x0+x] = logoShadow
			}
		}
		for y := 0; y < h; y++ {
			for x := 0; x < 5; x++ {
				if g[y][x] == '#' {
					grid[y][x0+x] = fill
				}
			}
		}
		x0 += 6
	}

	rows := make([]string, len(grid))
	for y := range grid {
		rows[y] = string(grid[y])
	}
	return render.NewSprite(rows...)
}

// LogoSprites returns the two words of the title logo.
func LogoSprites() (top, bottom render.Sprite) {
	return buildWord("SUPER", logoTop), buildWord("MARIO", logoBottom)
}
