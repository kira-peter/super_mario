package level

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// RGB is a 24-bit colour, 0xRRGGBB.
type RGB uint32

// Hex writes the colour the way level files do.
func (c RGB) Hex() string { return fmt.Sprintf("#%06x", uint32(c)&0xffffff) }

// Sky is the backdrop: one colour from the top of the level to the horizon, or
// a two-colour gradient between them. A solid sky has Top == Bottom.
type Sky struct {
	Top, Bottom RGB
}

// Gradient reports whether the sky changes colour from top to bottom.
func (s Sky) Gradient() bool { return s.Top != s.Bottom }

// Style is a level's appearance (spec level-appearance). Every field is
// optional: nil means "as the theme says", and with no theme, "as the game
// looks by default". A zero Style changes nothing.
type Style struct {
	// Theme names one of Themes. Empty means the game's own look.
	Theme  string
	Sky    *Sky
	Ground *RGB // soil
	Turf   *RGB // the grass on top of the ground
	Brick  *RGB
	Stair  *RGB // the hard blocks of a staircase
	Pipe   *RGB
}

// IsZero reports whether the style changes nothing.
func (s Style) IsZero() bool {
	return s.Theme == "" && s.Sky == nil && s.Ground == nil && s.Turf == nil &&
		s.Brick == nil && s.Stair == nil && s.Pipe == nil
}

// Resolved fills every field the level leaves out from its theme, so a renderer
// sees one complete answer per element. With no theme the unset fields stay
// nil, meaning "the game's default".
func (s Style) Resolved() Style {
	t, ok := ThemeByID(s.Theme)
	if !ok {
		return s
	}
	out := s
	if out.Sky == nil {
		sky := t.Sky
		out.Sky = &sky
	}
	pick := func(p *RGB, v RGB) *RGB {
		if p != nil {
			return p
		}
		return &v
	}
	out.Ground = pick(s.Ground, t.Ground)
	out.Turf = pick(s.Turf, t.Turf)
	out.Brick = pick(s.Brick, t.Brick)
	out.Stair = pick(s.Stair, t.Stair)
	out.Pipe = pick(s.Pipe, t.Pipe)
	return out
}

// Swatch is one named colour in a palette.
type Swatch struct {
	ID    string // what level files write
	Label string // what the editor shows
	RGB   RGB
}

// SkySwatch is one named sky preset.
type SkySwatch struct {
	ID    string
	Label string
	Sky   Sky
}

// Theme is a whole-level look: every element's colour, chosen to go together.
type Theme struct {
	ID    string
	Label string
	Sky   Sky
	// Ground is the soil, Turf the grass on it.
	Ground, Turf, Brick, Stair, Pipe RGB
}

// The built-in palettes. Order is the order the editor lists them in.
var (
	SkyPresets = []SkySwatch{
		{"day", "白天蓝", Sky{0x5c94fc, 0xa8d8ff}},
		{"dusk", "黄昏橙", Sky{0x3b2a5c, 0xf0954a}},
		{"night", "夜空", Sky{0x12141a, 0x12141a}},
		{"dawn", "黎明粉", Sky{0x6a7fc8, 0xf6b8c8}},
		{"overcast", "阴天灰", Sky{0x8a9099, 0xc4c9d0}},
		{"starry", "星空", Sky{0x05060f, 0x1f2550}},
	}
	GroundColors = []Swatch{
		{"brown", "棕土", 0xa8692f},
		{"red", "红土", 0xa8452f},
		{"gray", "灰岩", 0x6f7380},
		{"sand", "沙土", 0xc9a066},
		{"frozen", "冻土", 0x7d93a8},
		{"ash", "火山灰", 0x4a3a3a},
	}
	TurfColors = []Swatch{
		{"green", "草地绿", 0x58a64b},
		{"autumn", "秋草橙", 0xd98a2b},
		{"snow", "雪地白", 0xeef3f8},
		{"sand", "沙漠黄", 0xe6c76a},
		{"amethyst", "紫晶", 0x9a5fd0},
		{"ember", "余烬红", 0xd8452a},
	}
	BrickColors = []Swatch{
		{"red", "红砖", 0xa8643c},
		{"blue", "蓝砖", 0x3d6fb8},
		{"stone", "石砖", 0x8a8f99},
		{"gold", "金砖", 0xd9a92e},
	}
	StairColors = []Swatch{
		{"gray", "灰石", 0x8a909e},
		{"sand", "砂岩", 0xc4a473},
		{"slate", "青石", 0x4f6273},
		{"ice", "冰块", 0xa8d4e8},
		{"obsidian", "黑曜石", 0x3a2f4a},
	}
	PipeColors = []Swatch{
		{"green", "绿", 0x3f9a52},
		{"blue", "蓝", 0x3a7bd5},
		{"orange", "橙", 0xe0802e},
		{"gray", "灰", 0x7f8692},
	}
	Themes = []Theme{
		{"grassland", "草原", Sky{0x5c94fc, 0xa8d8ff}, 0xa8692f, 0x58a64b, 0xa8643c, 0x8a909e, 0x3f9a52},
		{"dusk", "黄昏", Sky{0x3b2a5c, 0xf0954a}, 0x8a4a2a, 0xd98a2b, 0xa8452f, 0x7a6a7f, 0x3f8a5a},
		{"night", "夜晚", Sky{0x05060f, 0x1f2550}, 0x6a4a2a, 0x2f6f4a, 0x5a4a8a, 0x4f6273, 0x2f6a5a},
		{"dawn", "黎明", Sky{0x6a7fc8, 0xf6b8c8}, 0xa86a5a, 0x7fbf6a, 0xc0705a, 0xa898b0, 0x4fa88a},
		{"snow", "雪原", Sky{0x9fbde0, 0xe8f0f8}, 0x7d93a8, 0xeef3f8, 0x3d6fb8, 0xa8d4e8, 0x3a7bd5},
		// A hazy blue sky fading to warm sand at the horizon. An all-gold
		// sky looked more like a desert but hid every coin and question block.
		{"desert", "沙漠", Sky{0x8cc4e8, 0xf3dcb0}, 0xc9a066, 0xe6c76a, 0xc07a3a, 0xc4a473, 0xe0802e},
		{"underground", "地下", Sky{0x000000, 0x0a0a14}, 0x3d5a8a, 0x5a7fb8, 0x3d6fb8, 0x4f6273, 0x3f9a52},
		// Cooled-rock bricks: a red brick vanished into the glowing sky.
		{"lava", "熔岩", Sky{0x1a0505, 0x6a1a0a}, 0x4a3a3a, 0xd8452a, 0x6a6070, 0x3a2f4a, 0x7f8692},
	}
)

// ThemeByID looks a theme up by the name level files use.
func ThemeByID(id string) (Theme, bool) {
	for _, t := range Themes {
		if t.ID == id {
			return t, true
		}
	}
	return Theme{}, false
}

func findSwatch(list []Swatch, id string) (RGB, bool) {
	for _, s := range list {
		if s.ID == id {
			return s.RGB, true
		}
	}
	return 0, false
}

func findSky(id string) (Sky, bool) {
	for _, s := range SkyPresets {
		if s.ID == id {
			return s.Sky, true
		}
	}
	return Sky{}, false
}

// nightSkies are the skies that mean night: the "night" and "starry" presets
// (the night theme uses the starry one). Underground and lava are dark too, but
// they are not night, and get no stars.
var nightSkies = []string{"night", "starry"}

// Night reports whether the level is set at night, so the renderer adds stars.
// It is: the night theme with its own sky, or a night or starry sky chosen
// directly. A custom colour is not night, however dark, so a level never gains
// stars it did not ask for.
func (s Style) Night() bool {
	r := s.Resolved()
	if r.Sky == nil {
		return false
	}
	for _, id := range nightSkies {
		if sky, ok := findSky(id); ok && sky == *r.Sky {
			return true
		}
	}
	return false
}

// --- on-disk shape -------------------------------------------------------

// styleSpec is "style" as written in a level file. Colours are kept as the
// author wrote them (a palette name or #rrggbb) so formatting a file does not
// turn "blue" into a hex code.
type styleSpec struct {
	Theme  string   `json:"theme,omitempty"`
	Sky    *skySpec `json:"sky,omitempty"`
	Ground string   `json:"ground,omitempty"`
	Turf   string   `json:"turf,omitempty"`
	Brick  string   `json:"brick,omitempty"`
	Stair  string   `json:"stair,omitempty"`
	Pipe   string   `json:"pipe,omitempty"`
}

func (s *styleSpec) empty() bool {
	return s == nil || (s.Theme == "" && s.Sky == nil && s.Ground == "" && s.Turf == "" && s.Brick == "" && s.Stair == "" && s.Pipe == "")
}

// skySpec is a sky preset name, one colour, or {"top": ..., "bottom": ...}.
type skySpec struct {
	Solid       string
	Top, Bottom string
}

func (s *skySpec) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) > 0 && b[0] == '"' {
		return json.Unmarshal(b, &s.Solid)
	}
	var g struct {
		Top    string `json:"top"`
		Bottom string `json:"bottom"`
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&g); err != nil {
		return fmt.Errorf("style.sky must be a colour or {\"top\", \"bottom\"}: %w", err)
	}
	s.Top, s.Bottom = g.Top, g.Bottom
	return nil
}

func (s skySpec) MarshalJSON() ([]byte, error) {
	if s.Solid != "" {
		return []byte(jsonString(s.Solid)), nil
	}
	return []byte("{\"top\": " + jsonString(s.Top) + ", \"bottom\": " + jsonString(s.Bottom) + "}"), nil
}

// parseHex reads #rrggbb.
func parseHex(s string) (RGB, bool) {
	if len(s) != 7 || s[0] != '#' {
		return 0, false
	}
	v, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		return 0, false
	}
	return RGB(v), true
}

// resolveColour turns a palette name or #rrggbb into a colour.
func resolveColour(field, v string, list []Swatch) (*RGB, error) {
	if v == "" {
		return nil, nil
	}
	if c, ok := parseHex(v); ok {
		return &c, nil
	}
	if c, ok := findSwatch(list, v); ok {
		return &c, nil
	}
	names := make([]string, len(list))
	for i, s := range list {
		names[i] = s.ID
	}
	return nil, fmt.Errorf("style.%s %q is neither #rrggbb nor one of: %s", field, v, strings.Join(names, ", "))
}

func validateStyle(s *styleSpec) (Style, error) {
	if s.empty() {
		return Style{}, nil
	}
	out := Style{Theme: s.Theme}
	if s.Theme != "" {
		if _, ok := ThemeByID(s.Theme); !ok {
			ids := make([]string, len(Themes))
			for i, t := range Themes {
				ids[i] = t.ID
			}
			return Style{}, fmt.Errorf("style.theme %q is unknown; themes are: %s", s.Theme, strings.Join(ids, ", "))
		}
	}
	if s.Sky != nil {
		sky, err := resolveSky(*s.Sky)
		if err != nil {
			return Style{}, err
		}
		out.Sky = &sky
	}
	var err error
	for _, f := range []struct {
		name string
		v    string
		list []Swatch
		dst  **RGB
	}{
		{"ground", s.Ground, GroundColors, &out.Ground},
		{"turf", s.Turf, TurfColors, &out.Turf},
		{"brick", s.Brick, BrickColors, &out.Brick},
		{"stair", s.Stair, StairColors, &out.Stair},
		{"pipe", s.Pipe, PipeColors, &out.Pipe},
	} {
		if *f.dst, err = resolveColour(f.name, f.v, f.list); err != nil {
			return Style{}, err
		}
	}
	return out, nil
}

func resolveSky(s skySpec) (Sky, error) {
	if s.Solid != "" {
		if c, ok := parseHex(s.Solid); ok {
			return Sky{c, c}, nil
		}
		if sky, ok := findSky(s.Solid); ok {
			return sky, nil
		}
		ids := make([]string, len(SkyPresets))
		for i, p := range SkyPresets {
			ids[i] = p.ID
		}
		return Sky{}, fmt.Errorf("style.sky %q is neither #rrggbb nor one of: %s", s.Solid, strings.Join(ids, ", "))
	}
	top, ok1 := parseHex(s.Top)
	bottom, ok2 := parseHex(s.Bottom)
	if !ok1 || !ok2 {
		return Sky{}, fmt.Errorf("style.sky top and bottom must both be #rrggbb (got %q, %q)", s.Top, s.Bottom)
	}
	return Sky{top, bottom}, nil
}

// formatStyle writes "style" on one line, fields in a fixed order.
func formatStyle(s *styleSpec) string {
	var parts []string
	add := func(k, v string) {
		if v != "" {
			parts = append(parts, jsonString(k)+": "+jsonString(v))
		}
	}
	add("theme", s.Theme)
	if s.Sky != nil {
		b, _ := s.Sky.MarshalJSON()
		parts = append(parts, "\"sky\": "+string(b))
	}
	add("ground", s.Ground)
	add("turf", s.Turf)
	add("brick", s.Brick)
	add("stair", s.Stair)
	add("pipe", s.Pipe)
	return "{" + strings.Join(parts, ", ") + "}"
}
