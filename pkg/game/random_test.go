package game

import (
	"math"
	"strings"
	"testing"

	"super-mario/pkg/engine"
	"super-mario/pkg/level"
)

// randomLevel is a flat level with a row of question blocks along row 9 at the
// given columns, every one random with the given pool.
func randomLevel(t *testing.T, cols []int, pool string) *level.Level {
	t.Helper()
	const width = 60
	rows := make([]string, 15)
	for y := range rows {
		b := []byte(strings.Repeat(" ", width))
		if y >= 13 {
			b = []byte(strings.Repeat("X", width))
		}
		if y == 9 {
			for _, c := range cols {
				b[c] = '?'
			}
		}
		rows[y] = string(b)
	}
	var contents []string
	for _, c := range cols {
		contents = append(contents, `{"x":`+itoa(c)+`,"y":9,"kind":"random","pool":`+pool+`}`)
	}
	data := `{"name":"t","world":"1-1","width":60,"height":15,"timeLimit":400,` +
		`"spawn":{"x":2,"y":12},"flag":{"x":58,"topY":2,"bottomY":12},` +
		`"rows":[` + quoteRows(rows) + `],"contents":[` + strings.Join(contents, ",") + `],"entities":[]}`
	lv, err := level.Parse([]byte(data))
	if err != nil {
		t.Fatalf("build level: %v", err)
	}
	return lv
}

const wideRandomPool = `[{"kind":"coin","w":1},{"kind":"mushroom","w":1},{"kind":"star","w":1},{"kind":"oneup","w":1},{"kind":"poison","w":1},{"kind":"gold","w":1}]`

func draws(w *World) map[int]level.BlockKind {
	out := map[int]level.BlockKind{}
	for _, c := range w.Level.Contents {
		if c.Random {
			out[c.X] = c.Kind
		}
	}
	return out
}

func randomWorld(t *testing.T, cols []int, pool string, seed int64) *World {
	t.Helper()
	sm := engine.NewStateMachine()
	sm.StartRun()
	w := NewWorld(randomLevel(t, cols, pool), &Default, sm, 3)
	w.SetSeed(seed)
	return w
}

func TestRandomBlocksAreDrawnAtStart(t *testing.T) {
	w := randomWorld(t, []int{10, 12, 14}, wideRandomPool, 7)
	for x, k := range draws(w) {
		if !k.Concrete() {
			t.Errorf("block %d holds %q, not a real content", x, k)
		}
	}
}

func TestSameSeedSameDraws(t *testing.T) {
	cols := []int{10, 12, 14, 16, 18, 20}
	a := draws(randomWorld(t, cols, wideRandomPool, 42))
	b := draws(randomWorld(t, cols, wideRandomPool, 42))
	for x := range a {
		if a[x] != b[x] {
			t.Errorf("block %d: %q then %q with the same seed", x, a[x], b[x])
		}
	}
}

// TestRespawnRedraws: a new life draws again, and it is not stuck on the same
// result every life.
func TestRespawnRedraws(t *testing.T) {
	cols := []int{10, 12, 14, 16, 18, 20, 22, 24}
	w := randomWorld(t, cols, wideRandomPool, 3)
	first := draws(w)

	w.life++
	w.Reset()
	second := draws(w)
	if w.Life() != 1 {
		t.Fatalf("life = %d", w.Life())
	}
	differs := false
	for x := range first {
		if first[x] != second[x] {
			differs = true
		}
	}
	if !differs {
		t.Error("every block drew the same on the next life; the life is not part of the draw")
	}

	// And a new run starts back at life 0 with the first life's draws.
	w.ResetRun()
	w.Reset()
	again := draws(w)
	for x := range first {
		if first[x] != again[x] {
			t.Errorf("block %d: a new run with the same seed drew %q, then %q", x, first[x], again[x])
		}
	}
}

// TestDrawsIgnoreOtherBlocks is the "adding a block elsewhere changes nothing"
// rule.
func TestDrawsIgnoreOtherBlocks(t *testing.T) {
	few := draws(randomWorld(t, []int{10, 20}, wideRandomPool, 99))
	many := draws(randomWorld(t, []int{5, 10, 15, 20, 25, 30}, wideRandomPool, 99))
	for _, x := range []int{10, 20} {
		if few[x] != many[x] {
			t.Errorf("block %d drew %q alone and %q with others around", x, few[x], many[x])
		}
	}
}

// TestDrawsFollowTheWeights draws ten thousand times and checks the share.
func TestDrawsFollowTheWeights(t *testing.T) {
	pool := []level.Choice{{Kind: level.BlockCoin, Weight: 8}, {Kind: level.BlockOneUp, Weight: 2}}
	const n = 10000
	ones := 0
	for i := 0; i < n; i++ {
		if DrawRandom(int64(i), 0, 21, 9, pool) == level.BlockOneUp {
			ones++
		}
	}
	share := float64(ones) / n
	// Four standard deviations of a binomial at p = 0.2: a false failure is
	// vanishingly rare, a real bias of a few percent is caught.
	sd := math.Sqrt(0.2 * 0.8 / n)
	if math.Abs(share-0.2) > 4*sd {
		t.Errorf("1UP came up %.1f%% of the time, want about 20%%", share*100)
	}
}

func TestZeroWeightIsNeverDrawn(t *testing.T) {
	pool := []level.Choice{{Kind: level.BlockStar, Weight: 0}, {Kind: level.BlockCoin, Weight: 1}}
	for i := 0; i < 2000; i++ {
		if DrawRandom(int64(i), i%3, i%50, 9, pool) == level.BlockStar {
			t.Fatal("a weight-0 entry was drawn")
		}
	}
}

// TestMultiCoinBlockKeepsItsDraw: a random block with several uses gives the
// same thing on every hit of a life, and hitting it does not redraw it.
func TestMultiCoinBlockKeepsItsDraw(t *testing.T) {
	w := randomWorld(t, []int{10}, `[{"kind":"coin","w":1},{"kind":"gold","w":1}]`, 5)
	i, _ := w.Level.ContentAt(10, 9)
	w.Level.Contents[i].Count = 5
	drawn := w.Level.Contents[i].Kind
	for hit := 0; hit < 3; hit++ {
		if w.Level.Contents[i].Kind != drawn {
			t.Fatalf("hit %d: the block changed from %q to %q", hit, drawn, w.Level.Contents[i].Kind)
		}
		if drawn != level.BlockCoin {
			break // a gold block is used up by one hit
		}
		w.dispense(10, 9)
	}
}

// TestFixedSeedRunReplays: with a fixed seed, two runs fed the same inputs end
// in the same place with the same score, random blocks and all.
func TestFixedSeedRunReplays(t *testing.T) {
	play := func() (engine.Fixed, int, map[int]level.BlockKind) {
		w := randomWorld(t, []int{8, 10, 12}, wideRandomPool, 2024)
		for i := 0; i < 240; i++ {
			w.Input.KeyDown(engine.KeyRight)
			if i%40 == 0 {
				w.Input.KeyDown(engine.KeySpace)
			}
			w.Step()
			w.Input.KeyUp(engine.KeySpace)
		}
		return w.Player.Pos.X, w.Score.Points, draws(w)
	}
	x1, s1, d1 := play()
	x2, s2, d2 := play()
	if x1 != x2 || s1 != s2 {
		t.Errorf("replay diverged: x %d vs %d, score %d vs %d", x1, x2, s1, s2)
	}
	for k := range d1 {
		if d1[k] != d2[k] {
			t.Errorf("block %d drew %q then %q", k, d1[k], d2[k])
		}
	}
}

// TestFixedBlocksAreUntouched: a level with no random blocks plays exactly as
// before, whatever the seed.
func TestFixedBlocksAreUntouched(t *testing.T) {
	lv, err := level.LoadEmbedded("1-1")
	if err != nil {
		t.Fatal(err)
	}
	before := append([]level.Content(nil), lv.Contents...)
	sm := engine.NewStateMachine()
	sm.StartRun()
	w := NewWorld(lv, &Default, sm, 3)
	w.SetSeed(12345)
	w.life = 2
	w.Reset()
	for i, c := range w.Level.Contents {
		if c.Kind != before[i].Kind {
			t.Errorf("1-1 block (%d,%d) changed from %q to %q", c.X, c.Y, before[i].Kind, c.Kind)
		}
	}
}
