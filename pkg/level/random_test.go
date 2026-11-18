package level

import (
	"strings"
	"testing"
)

// withBlocks is a level with question blocks at (0,0) and (3,0) and the given
// contents.
func withBlocks(contents string) []byte {
	return []byte(`{"name":"t","world":"1-1","width":4,"height":3,"timeLimit":400,
	"spawn":{"x":0,"y":1},"flag":{"x":3,"topY":0,"bottomY":1},
	"rows":["?  ?","    ","XXXX"],"contents":[` + contents + `],"entities":[]}`)
}

func TestRandomBlockLoads(t *testing.T) {
	lv, err := Parse(withBlocks(`{"x":0,"y":0,"kind":"random","pool":[{"kind":"mushroom","w":3},{"kind":"oneup","w":1}]},
		{"x":3,"y":0,"kind":"coin"}`))
	if err != nil {
		t.Fatal(err)
	}
	i, _ := lv.ContentAt(0, 0)
	c := lv.Contents[i]
	if !c.Random || c.Kind != BlockRandom || len(c.Pool) != 2 || c.Pool[0] != (Choice{BlockMushroom, 3}) {
		t.Errorf("random block = %+v", c)
	}
	j, _ := lv.ContentAt(3, 0)
	if lv.Contents[j].Random || lv.Contents[j].Pool != nil {
		t.Error("a fixed block came out random")
	}
}

func TestRandomBlockRejections(t *testing.T) {
	for _, tc := range []struct{ contents, want string }{
		{`{"x":0,"y":0,"kind":"random"},{"x":3,"y":0,"kind":"coin"}`, "empty pool"},
		{`{"x":0,"y":0,"kind":"random","pool":[]},{"x":3,"y":0,"kind":"coin"}`, "empty pool"},
		{`{"x":0,"y":0,"kind":"random","pool":[{"kind":"coin","w":0},{"kind":"star","w":0}]},{"x":3,"y":0,"kind":"coin"}`, "add up to 0"},
		{`{"x":0,"y":0,"kind":"random","pool":[{"kind":"coin","w":-1},{"kind":"star","w":2}]},{"x":3,"y":0,"kind":"coin"}`, "negative weight"},
		{`{"x":0,"y":0,"kind":"random","pool":[{"kind":"fireflower","w":1}]},{"x":3,"y":0,"kind":"coin"}`, "unknown kind"},
		{`{"x":0,"y":0,"kind":"random","pool":[{"kind":"random","w":1}]},{"x":3,"y":0,"kind":"coin"}`, "unknown kind"},
		{`{"x":0,"y":0,"kind":"random","pool":[{"kind":"coin","w":1},{"kind":"coin","w":2}]},{"x":3,"y":0,"kind":"coin"}`, "twice"},
		{`{"x":0,"y":0,"kind":"coin","pool":[{"kind":"star","w":1}]},{"x":3,"y":0,"kind":"coin"}`, "not \"random\""},
	} {
		_, err := Parse(withBlocks(tc.contents))
		if err == nil {
			t.Errorf("accepted: %s", tc.contents)
			continue
		}
		msg := err.Error()
		if !strings.Contains(msg, tc.want) || !strings.Contains(msg, "(0,0)") {
			t.Errorf("error %q should mention %q and the block's position", msg, tc.want)
		}
	}
}

// TestOldBlocksUnchanged checks a level with only fixed blocks reads exactly as
// before random blocks existed.
func TestOldBlocksUnchanged(t *testing.T) {
	lv, err := LoadEmbedded("1-1")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range lv.Contents {
		if c.Random || c.Pool != nil || !c.Kind.Concrete() {
			t.Errorf("1-1 block at (%d,%d) is %+v", c.X, c.Y, c)
		}
	}
}

func TestRestorePutsRandomBack(t *testing.T) {
	lv, err := Parse(withBlocks(`{"x":0,"y":0,"kind":"random","pool":[{"kind":"star","w":1}]},{"x":3,"y":0,"kind":"coin"}`))
	if err != nil {
		t.Fatal(err)
	}
	i, _ := lv.ContentAt(0, 0)
	lv.Contents[i].Kind = BlockStar // what a world does when it draws
	lv.Restore()
	if lv.Contents[i].Kind != BlockRandom {
		t.Errorf("after Restore the block is %q, want random again", lv.Contents[i].Kind)
	}
}

func TestFormatWritesPool(t *testing.T) {
	out, err := Format(withBlocks(`{"x":0,"y":0,"kind":"random","pool":[{"kind":"mushroom","w":3},{"kind":"oneup","w":1}]},{"x":3,"y":0,"kind":"coin"}`))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"x": 0, "y": 0, "kind": "random", "pool": [{"kind": "mushroom", "w": 3}, {"kind": "oneup", "w": 1}]}`
	if !strings.Contains(string(out), want) {
		t.Errorf("formatted:\n%s\nwant a line\n%s", out, want)
	}
}
