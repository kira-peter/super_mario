package level

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// tiny is a minimal valid level body; tests splice extra fields in front of
// "spawn".
const tinyHead = `{"name":"t","world":"1-1","width":4,"height":3,"timeLimit":400,`
const tinyTail = `"spawn":{"x":0,"y":1},"flag":{"x":3,"topY":0,"bottomY":1},
	"rows":["    ","    ","XXXX"],"contents":[],"entities":[]}`

func tiny(extra string) []byte { return []byte(tinyHead + extra + tinyTail) }

func TestOldLevelWithoutMetaLoads(t *testing.T) {
	lv, err := Parse(tiny(""))
	if err != nil {
		t.Fatalf("a level with no meta must load: %v", err)
	}
	if lv.Meta != (Meta{}) {
		t.Errorf("meta = %+v, want empty", lv.Meta)
	}
}

func TestMetaIsRead(t *testing.T) {
	lv, err := Parse(tiny(`"meta":{"title":"天空之路","author":"tom","description":"跳","difficulty":3},`))
	if err != nil {
		t.Fatal(err)
	}
	want := Meta{Title: "天空之路", Author: "tom", Description: "跳", Difficulty: 3}
	if lv.Meta != want {
		t.Errorf("meta = %+v, want %+v", lv.Meta, want)
	}
}

func TestDifficultyOutOfRangeIsRejected(t *testing.T) {
	_, err := Parse(tiny(`"meta":{"difficulty":7},`))
	if err == nil {
		t.Fatal("difficulty 7 must be rejected")
	}
	if !strings.Contains(err.Error(), "between 1 and 5") {
		t.Errorf("error = %q, want it to give the allowed range", err)
	}
}

func TestOverlongMetaIsRejected(t *testing.T) {
	long := strings.Repeat("关", MaxTitleLen+1)
	_, err := Parse(tiny(`"meta":{"title":"` + long + `"},`))
	if err == nil {
		t.Fatal("an overlong title must be rejected")
	}
	if !strings.Contains(err.Error(), "meta.title") {
		t.Errorf("error = %q, want it to name the field", err)
	}
	// Exactly at the limit is fine, counted in characters, not bytes.
	ok := strings.Repeat("关", MaxTitleLen)
	if _, err := Parse(tiny(`"meta":{"title":"` + ok + `"},`)); err != nil {
		t.Errorf("a title of exactly %d characters was rejected: %v", MaxTitleLen, err)
	}
}

func TestFormatIsStable(t *testing.T) {
	data, err := assets.ReadFile("data/1-1.json")
	if err != nil {
		t.Fatal(err)
	}
	once, err := Format(data)
	if err != nil {
		t.Fatal(err)
	}
	twice, err := Format(once)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(once, twice) {
		t.Error("formatting a formatted level changed it")
	}
}

func TestFormatPreservesTheLevel(t *testing.T) {
	data, err := assets.ReadFile("data/1-1.json")
	if err != nil {
		t.Fatal(err)
	}
	out, err := Format(data)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := Parse(data)
	b, err := Parse(out)
	if err != nil {
		t.Fatalf("formatted level does not parse: %v", err)
	}
	if a.Width != b.Width || a.FlagX != b.FlagX || a.SpawnX != b.SpawnX || len(a.Spawns) != len(b.Spawns) {
		t.Error("formatting changed the level's shape")
	}
	for i := range a.tiles {
		if a.tiles[i] != b.tiles[i] {
			t.Fatalf("formatting changed tile %d", i)
		}
	}
	if len(a.Contents) != len(b.Contents) {
		t.Fatalf("formatting changed the block count")
	}
	for _, c := range a.Contents {
		i, ok := b.ContentAt(c.X, c.Y)
		if !ok || !reflect.DeepEqual(b.Contents[i], c) {
			t.Errorf("block at (%d,%d) changed", c.X, c.Y)
		}
	}
}

func TestFormatSortsAndOmitsDefaults(t *testing.T) {
	data := []byte(tinyHead + `"meta":{"title":"a<b"},` + `"spawn":{"x":0,"y":1},"flag":{"x":3,"topY":0,"bottomY":1},
	"rows":["?  ?","    ","XXXX"],
	"contents":[{"x":3,"y":0,"kind":"star","count":1,"hidden":false},{"x":0,"y":0,"kind":"coin","count":4}],
	"entities":[{"kind":"koopa","x":2,"y":1},{"kind":"goomba","x":1,"y":1}]}`)
	out, err := Format(data)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Index(s, `"kind": "coin"`) > strings.Index(s, `"kind": "star"`) {
		t.Error("contents are not sorted by position")
	}
	if strings.Index(s, `"goomba"`) > strings.Index(s, `"koopa"`) {
		t.Error("entities are not sorted by position")
	}
	if strings.Contains(s, `"hidden": false`) || strings.Contains(s, `"count": 1`) {
		t.Error("default values were written out")
	}
	if !strings.Contains(s, `"count": 4`) {
		t.Error("a non-default count was dropped")
	}
	if !strings.Contains(s, `"title": "a<b"`) {
		t.Errorf("title was escaped: %s", s)
	}
}

func TestFormatRejectsInvalidLevels(t *testing.T) {
	if _, err := Format([]byte(tinyHead + `"spawn":{"x":99,"y":1},"flag":{"x":3,"topY":0,"bottomY":1},"rows":["    ","    ","XXXX"]}`)); err == nil {
		t.Error("Format accepted a level the loader rejects")
	}
}
