package render

import (
	"strings"
	"testing"

	"super-mario/pkg/engine"
	"super-mario/pkg/game"
	"super-mario/pkg/level"
	"super-mario/pkg/render/theme"
)

// heroWorld builds a world with the hero standing at the spawn, framed the same
// way the game frames it at the default zoom.
func heroWorld(t *testing.T, big bool, facing int) *game.World {
	t.Helper()
	lv, err := level.LoadEmbedded("1-1")
	if err != nil {
		t.Fatal(err)
	}
	sm := engine.NewStateMachine()
	sm.StartRun()
	w := game.NewWorld(lv, &game.Default, sm, 3)
	w.SetViewport(engine.Tile*30, engine.Tile*19)
	if big {
		w.Player.Grow(lv)
	}
	w.Player.Facing = facing
	w.Step()
	return w
}

// TestHeroSizes pins the dimensions the design was agreed at.
func TestHeroSizes(t *testing.T) {
	cases := []struct {
		big        bool
		pose       HeroPose
		wide, tall int
	}{
		{false, HeroStand, 6, 7},
		{false, HeroJump, 6, 7},
		{true, HeroStand, 7, 11},
		{true, HeroJump, 7, 11},
	}
	for _, c := range cases {
		h := Hero(c.big, c.pose, 0)
		if h.Sprite.W != c.wide || h.Sprite.H != c.tall {
			t.Errorf("big=%v pose=%v is %dx%d, want %dx%d", c.big, c.pose, h.Sprite.W, h.Sprite.H, c.wide, c.tall)
		}
		for i, r := range h.Sprite.Rows {
			if len(r) != c.wide {
				t.Errorf("big=%v pose=%v row %d is %d wide, want %d", c.big, c.pose, i, len(r), c.wide)
			}
		}
	}
}

// TestBigHeroHasTheAgreedBody checks the part of the design that was specified
// row by row: one hat row, two face rows, a single neck cell in the middle, one
// red row, two blue rows.
func TestBigHeroHasTheAgreedBody(t *testing.T) {
	rows := Hero(true, HeroStand, 0).Sprite.Rows
	want := []string{
		"..RRR..", // hat, one row
		"..SSS..", // face, two rows
		"..SSS..",
		"...S...", // neck: one skin cell, in the middle
		".RRRRRS", // red, one row, front hand at the shoulder
		".SBBBB.", // blue, two rows, back hand tucked in at the waist
		".BBBBB.",
		".......", // empty row: floating
	}
	for i, w := range want {
		if rows[i] != w {
			t.Errorf("row %d = %q, want %q", i, rows[i], w)
		}
	}
}

// handCells returns the skin cells that are hands: every skin cell in the body
// rows outside the head's columns. The head is columns 2-3 on the small figure
// and 2-4 on the big one, and the neck of the big figure sits in the middle of
// the head's columns, so it never counts.
func handCells(rows []string, big bool) [][2]int {
	lo, hi, bodyRows := 2, 3, 5 // small: five body rows above the wheel
	if big {
		lo, hi, bodyRows = 2, 4, 8
	}
	var out [][2]int
	for y := 0; y < bodyRows; y++ {
		for x, c := range rows[y] {
			if c == 'S' && (x < lo || x > hi) {
				out = append(out, [2]int{y, x})
			}
		}
	}
	return out
}

// TestJumpArms is the jump rule as drawn. The red row runs the full width, and
// each hand is a single skin cell at one end of it: the front hand moves up one
// row from where it was walking, in the same column, and the back hand stays on
// its waist row but moves one column out to the edge. Nothing is left floating.
func TestJumpArms(t *testing.T) {
	for _, big := range []bool{false, true} {
		stand := Hero(big, HeroStand, 0).Sprite.Rows
		jump := Hero(big, HeroJump, 0).Sprite.Rows
		hs, hj := handCells(stand, big), handCells(jump, big)

		if len(hs) != 2 || len(hj) != 2 {
			t.Fatalf("big=%v: %d hand cells standing and %d jumping, want 2 each (one per hand)",
				big, len(hs), len(hj))
		}
		// The right-hand cell is the front hand, the left-hand one the back hand.
		front := func(c [][2]int) [2]int {
			if c[0][1] > c[1][1] {
				return c[0]
			}
			return c[1]
		}
		back := func(c [][2]int) [2]int {
			if c[0][1] > c[1][1] {
				return c[1]
			}
			return c[0]
		}
		fs, fj := front(hs), front(hj)
		bs, bj := back(hs), back(hj)

		if fj[1] != fs[1] || fj[0] != fs[0]-1 {
			t.Errorf("big=%v: the front hand went from row %d col %d to row %d col %d, want one row up in the same column",
				big, fs[0], fs[1], fj[0], fj[1])
		}
		if bj[0] != bs[0] || bj[1] != bs[1]-1 {
			t.Errorf("big=%v: the back hand went from row %d col %d to row %d col %d, want the same row and one column out",
				big, bs[0], bs[1], bj[0], bj[1])
		}

		// Nothing floats: every hand touches the sleeve or the trousers.
		for _, cell := range hj {
			y, x := cell[0], cell[1]
			touches := false
			for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				ny, nx := y+d[0], x+d[1]
				if ny < 0 || ny >= len(jump) || nx < 0 || nx >= len(jump[ny]) {
					continue
				}
				if c := jump[ny][nx]; c == 'R' || c == 'B' {
					touches = true
				}
			}
			if !touches {
				t.Errorf("big=%v: the hand at row %d col %d touches no part of the body", big, y, x)
			}
		}
	}
}

// TestJumpSleeveRunsTheFullWidth checks the red row is unbroken on the jump.
func TestJumpSleeveRunsTheFullWidth(t *testing.T) {
	for _, c := range []struct {
		big bool
		row int
	}{{false, 2}, {true, 4}} {
		row := Hero(c.big, HeroJump, 0).Sprite.Rows[c.row]
		if strings.Trim(row, "R") != "" {
			t.Errorf("big=%v: the red row is %q, want it red from edge to edge", c.big, row)
		}
	}
}

// TestWheelHubIsHollow is the requirement that the middle of the big wheel is
// the same colour as the background, achieved by leaving it empty so whatever
// is behind shows through.
func TestWheelHubIsHollow(t *testing.T) {
	for phase := 0; phase < WheelPhases(true); phase++ {
		rows := Hero(true, HeroStand, phase).Sprite.Rows
		wheel := rows[len(rows)-3:]
		if centre := wheel[1][3]; centre != SlotNone {
			t.Errorf("phase %d: the hub is %q, want it empty so the background shows through", phase, string(centre))
		}
	}
}

// TestHubShowsWhateverIsBehindIt paints a brick behind the hero and checks the
// hub takes the brick's colour rather than a fixed sky colour.
func TestHubShowsWhateverIsBehindIt(t *testing.T) {
	tok := theme.Ink
	c := NewCanvas(40, 40)
	c.Clear(tok.Brick)

	h := Hero(true, HeroStand, 0)
	c.Draw(h.Sprite, 10, 10, tok)

	hubX := 10 + 3
	hubY := 10 + h.Sprite.H - 2
	if got := c.Px[hubY*c.W+hubX]; got != tok.Brick {
		t.Errorf("the hub shows %v, want the brick behind it (%v)", got, tok.Brick)
	}
}

// markCell finds where the marker is, as (row, column) within the wheel.
func markCell(rows []string, big bool) (int, int) {
	n := 2
	if big {
		n = 3
	}
	for y := len(rows) - n; y < len(rows); y++ {
		if x := strings.IndexByte(rows[y], SlotHeroMark); x >= 0 {
			return y - (len(rows) - n), x - wheelLeft(big, len(rows[y]))
		}
	}
	return -1, -1
}

// wheelLeft is the column where the wheel starts within a row of the given width.
func wheelLeft(big bool, width int) int {
	n := 2
	if big {
		n = 3
	}
	return (width - n) / 2
}

// TestWheelMarkerGoesClockwise is the direction rule, checked by the sign of the
// cross product around the wheel's centre (screen y points down, so positive is
// clockwise).
func TestWheelMarkerGoesClockwise(t *testing.T) {
	for _, big := range []bool{false, true} {
		n := WheelPhases(big)
		centre := 0.5
		if big {
			centre = 1
		}
		var total float64
		for p := 0; p < n; p++ {
			ay, ax := markCell(Hero(big, HeroStand, p).Sprite.Rows, big)
			by, bx := markCell(Hero(big, HeroStand, p+1).Sprite.Rows, big)
			if ay < 0 || by < 0 {
				t.Fatalf("big=%v: marker missing at phase %d", big, p)
			}
			total += (float64(ax)-centre)*(float64(by)-centre) - (float64(ay)-centre)*(float64(bx)-centre)
		}
		if total <= 0 {
			t.Errorf("big=%v: the marker goes anticlockwise rolling right (cross sum %.1f)", big, total)
		}
	}
}

// TestMirroredWheelTurnsTheOtherWay checks rolling left is anticlockwise without
// a second set of frames: mirroring puts the marker on the opposite side.
func TestMirroredWheelTurnsTheOtherWay(t *testing.T) {
	for _, big := range []bool{false, true} {
		for p := 0; p < WheelPhases(big); p++ {
			right := Hero(big, HeroStand, p)
			left := right.Mirrored()
			w := len(right.Sprite.Rows[0])
			_, rx := markCell(right.Sprite.Rows, big)
			_, lx := markCell(left.Sprite.Rows, big)
			n := 2
			if big {
				n = 3
			}
			if rx+lx != n-1 {
				t.Errorf("big=%v phase %d (width %d): marker at wheel column %d facing right and %d facing left; not mirror images",
					big, p, w, rx, lx)
			}
		}
	}
}

// TestMarkerIsAlwaysExactlyOne checks the one thing that tells the wheel is
// turning is always there, exactly once.
func TestMarkerIsAlwaysExactlyOne(t *testing.T) {
	for _, big := range []bool{false, true} {
		for p := 0; p < WheelPhases(big); p++ {
			rows := Hero(big, HeroStand, p).Sprite.Rows
			if n := strings.Count(strings.Join(rows, ""), string(SlotHeroMark)); n != 1 {
				t.Errorf("big=%v phase %d has %d markers, want 1", big, p, n)
			}
		}
	}
}

// TestHeroKeepsItsHeadStillWhenTurning is the hop-on-turn regression test. The
// body is not symmetrical, so anchoring on the whole drawing made the figure
// shift a pixel each time he turned round. The head is always centred.
func TestHeroKeepsItsHeadStillWhenTurning(t *testing.T) {
	for _, big := range []bool{false, true} {
		var mids [2]float64
		for i, face := range []int{1, -1} {
			w := heroWorld(t, big, face)
			c := (&Scene{T: theme.Ink}).Draw(w, 4)
			mids[i] = hatMid(t, c)
		}
		if mids[0] != mids[1] {
			t.Errorf("big=%v: the head moved from %.1f to %.1f when he turned round", big, mids[0], mids[1])
		}
	}
}

// hatMid returns the horizontal middle of the topmost row of hat-coloured
// pixels on the canvas.
func hatMid(t *testing.T, c *Canvas) float64 {
	t.Helper()
	red := theme.Ink.PlayerBody
	for y := 0; y < c.H; y++ {
		x0, x1 := -1, -1
		for x := 0; x < c.W; x++ {
			if c.Px[y*c.W+x] == red {
				if x0 < 0 {
					x0 = x
				}
				x1 = x
			}
		}
		if x0 >= 0 {
			return float64(x0+x1) / 2
		}
	}
	t.Fatal("no hat on the canvas")
	return 0
}

// TestHeroStandsOnTheGround checks the wheel sits directly on the floor, with
// neither a gap nor an overlap.
func TestHeroStandsOnTheGround(t *testing.T) {
	for _, big := range []bool{false, true} {
		w := heroWorld(t, big, 1)
		c := (&Scene{T: theme.Ink}).Draw(w, 4)

		bottom := -1
		tk := theme.Ink
		for y := 0; y < c.H; y++ {
			for x := 0; x < c.W; x++ {
				if px := c.Px[y*c.W+x]; px == tk.HeroWheel || px == tk.HeroMark {
					bottom = y
				}
			}
		}
		groundTop := roundArt(engine.Fixed(w.Level.SpawnY+1)*engine.Tile, 4) - WorldToArtScaled(w.CameraY, 4)
		if bottom+1 != groundTop {
			t.Errorf("big=%v: the wheel ends on row %d but the ground starts on row %d", big, bottom, groundTop)
		}
	}
}
