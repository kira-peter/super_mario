package level

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Format validates a level file and rewrites it in the one canonical layout.
//
// The layout is chosen for review, since community levels arrive as pull
// requests: every terrain row is one line, every block and every enemy is one
// line, lists are sorted by position, and field order never changes. Two
// people saving the same level get the same bytes, and a one-tile edit is a
// one-line diff (design decision 3).
//
// The editor, the fmt-levels command and the repository test all call this,
// so there is exactly one formatter.
func Format(data []byte) ([]byte, error) {
	f, err := decode(data)
	if err != nil {
		return nil, err
	}
	if _, err := validate(f); err != nil {
		return nil, err
	}
	return formatFile(f), nil
}

// decode reads the on-disk shape strictly: an unknown field is an error, so a
// typo such as "contnets" cannot silently drop every block.
func decode(data []byte) (*file, error) {
	var f file
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("parse level json: %w", err)
	}
	return &f, nil
}

func formatFile(f *file) []byte {
	var b strings.Builder
	w := func(format string, args ...interface{}) { fmt.Fprintf(&b, format, args...) }

	w("{\n")
	w("  \"name\": %s,\n", jsonString(f.Name))
	w("  \"world\": %s,\n", jsonString(f.World))
	w("  \"width\": %d,\n", f.Width)
	w("  \"height\": %d,\n", f.Height)
	w("  \"timeLimit\": %d,\n", f.TimeLimit)
	if !f.Meta.empty() {
		w("  \"meta\": %s,\n", formatMeta(f.Meta))
	}
	if !f.Style.empty() {
		w("  \"style\": %s,\n", formatStyle(f.Style))
	}
	w("  \"spawn\": {\"x\": %d, \"y\": %d},\n", f.Spawn.X, f.Spawn.Y)
	w("  \"flag\": {\"x\": %d, \"topY\": %d, \"bottomY\": %d},\n", f.Flag.X, f.Flag.TopY, f.Flag.BottomY)

	w("  \"rows\": [\n")
	for i, r := range f.Rows {
		w("    %s%s\n", jsonString(r), comma(i, len(f.Rows)))
	}
	w("  ],\n")

	contents := append([]content(nil), f.Contents...)
	sort.SliceStable(contents, func(i, j int) bool {
		return lessXY(contents[i].X, contents[i].Y, contents[j].X, contents[j].Y)
	})
	writeList(&b, "contents", len(contents), func(i int) string { return formatContent(contents[i]) }, true)

	entities := append([]entity(nil), f.Entities...)
	sort.SliceStable(entities, func(i, j int) bool {
		a, c := entities[i], entities[j]
		if a.X != c.X || a.Y != c.Y {
			return lessXY(a.X, a.Y, c.X, c.Y)
		}
		return a.Kind < c.Kind
	})
	writeList(&b, "entities", len(entities), func(i int) string { return formatEntity(entities[i]) }, false)

	w("}\n")
	return []byte(b.String())
}

// writeList writes a named array with one item per line, or [] when empty.
func writeList(b *strings.Builder, name string, n int, item func(int) string, more bool) {
	tail := ""
	if more {
		tail = ","
	}
	if n == 0 {
		fmt.Fprintf(b, "  %q: []%s\n", name, tail)
		return
	}
	fmt.Fprintf(b, "  %q: [\n", name)
	for i := 0; i < n; i++ {
		fmt.Fprintf(b, "    %s%s\n", item(i), comma(i, n))
	}
	fmt.Fprintf(b, "  ]%s\n", tail)
}

func formatMeta(m *metaSpec) string {
	var parts []string
	if m.Title != "" {
		parts = append(parts, "\"title\": "+jsonString(m.Title))
	}
	if m.Author != "" {
		parts = append(parts, "\"author\": "+jsonString(m.Author))
	}
	if m.Description != "" {
		parts = append(parts, "\"description\": "+jsonString(m.Description))
	}
	if m.Difficulty != 0 {
		parts = append(parts, "\"difficulty\": "+strconv.Itoa(m.Difficulty))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// formatContent writes a block payload, leaving out fields at their default
// (one use, visible) so the common case reads as just a position and a kind.
func formatContent(c content) string {
	s := fmt.Sprintf("{\"x\": %d, \"y\": %d, \"kind\": %s", c.X, c.Y, jsonString(c.Kind))
	if c.Count > 1 {
		s += fmt.Sprintf(", \"count\": %d", c.Count)
	}
	if c.Hidden {
		s += ", \"hidden\": true"
	}
	if len(c.Pool) > 0 {
		parts := make([]string, len(c.Pool))
		for i, p := range c.Pool {
			parts[i] = fmt.Sprintf("{\"kind\": %s, \"w\": %d}", jsonString(p.Kind), p.Weight)
		}
		s += ", \"pool\": [" + strings.Join(parts, ", ") + "]"
	}
	return s + "}"
}

func formatEntity(e entity) string {
	return fmt.Sprintf("{\"kind\": %s, \"x\": %d, \"y\": %d}", jsonString(e.Kind), e.X, e.Y)
}

func lessXY(ax, ay, bx, by int) bool {
	if ax != bx {
		return ax < bx
	}
	return ay < by
}

func comma(i, n int) string {
	if i < n-1 {
		return ","
	}
	return ""
}

// jsonString quotes a string the JSON way, without escaping <, > and & the way
// encoding/json does by default: a level title may well contain them.
func jsonString(s string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return strings.TrimSuffix(buf.String(), "\n")
}
