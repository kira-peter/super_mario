package level

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadmeStyleExampleLoads keeps the README honest: the style example it
// shows must be accepted by the loader, and every palette name the README
// lists must exist.
func TestReadmeStyleExampleLoads(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Skip("README not found")
	}
	doc := strings.ReplaceAll(string(data), "\r\n", "\n")
	i := strings.Index(doc, "<!-- style-example -->")
	if i < 0 {
		t.Fatal("README has no marked style example")
	}
	block := doc[i:]
	open := strings.Index(block, "```json\n")
	if open < 0 {
		t.Fatal("no json block after the style example marker")
	}
	body := block[open+len("```json\n"):]
	end := strings.Index(body, "```")
	if end < 0 {
		t.Fatal("the style example block is not closed")
	}
	example := strings.TrimSuffix(strings.TrimSpace(body[:end]), ",")

	if _, err := Parse(tiny(example + ",")); err != nil {
		t.Errorf("the README style example does not load: %v\n%s", err, example)
	}

	for _, id := range allPaletteIDs() {
		if !strings.Contains(doc, "`"+id+"`") {
			t.Errorf("README does not list %q", id)
		}
	}
}

func allPaletteIDs() []string {
	var ids []string
	for _, t := range Themes {
		ids = append(ids, t.ID)
	}
	for _, s := range SkyPresets {
		ids = append(ids, s.ID)
	}
	for _, l := range [][]Swatch{GroundColors, TurfColors, BrickColors, StairColors, PipeColors} {
		for _, s := range l {
			ids = append(ids, s.ID)
		}
	}
	return ids
}
