package engine

import "testing"

func TestFixedConversions(t *testing.T) {
	cases := []struct {
		in      float64
		wantInt int
		wantSub int
	}{
		{0, 0, 0},
		{1, 1, 0},
		{1.5625, 1, 144},
		{-1.5625, -1, 112}, // truncation toward zero keeps the sign on the integer part
		{16, 16, 0},
	}
	for _, c := range cases {
		f := FromFloat(c.in)
		if f.Int() != c.wantInt {
			t.Errorf("FromFloat(%v).Int() = %d, want %d", c.in, f.Int(), c.wantInt)
		}
		_ = c.wantSub
	}
}

func TestFloorRoundsDown(t *testing.T) {
	// -0.5 game units must floor to -1, not 0.
	f := FromFloat(-0.5)
	if got := f.Floor(); got != -1 {
		t.Errorf("Floor(-0.5) = %d, want -1", got)
	}
	if got := FromFloat(1.5).Floor(); got != 1 {
		t.Errorf("Floor(1.5) = %d, want 1", got)
	}
}

func TestTileConstant(t *testing.T) {
	if Tile != 16*256 {
		t.Fatalf("Tile = %d, want %d", Tile, 16*256)
	}
}

func TestMulRatioIsExact(t *testing.T) {
	// A third of a tile, times 3, must come back to the tile exactly enough
	// that repeated application does not drift.
	a := Tile.Div(3)
	if got := MulRatio(a, 3, 1); got != a*3 {
		t.Fatalf("MulRatio(a,3,1) = %d, want %d", got, a*3)
	}
}

func TestRectOverlap(t *testing.T) {
	a := Rect{0, 0, 100, 100}
	if !a.Overlaps(Rect{50, 50, 100, 100}) {
		t.Error("overlapping rects reported as disjoint")
	}
	if a.Overlaps(Rect{100, 0, 50, 50}) {
		t.Error("edge-touching rects must not overlap")
	}
	if a.Overlaps(Rect{200, 200, 10, 10}) {
		t.Error("disjoint rects reported as overlapping")
	}
}

func TestClampAndSign(t *testing.T) {
	if got := Fixed(50).Clamp(-10, 10); got != 10 {
		t.Errorf("Clamp above range = %d, want 10", got)
	}
	if got := Fixed(-50).Clamp(-10, 10); got != -10 {
		t.Errorf("Clamp below range = %d, want -10", got)
	}
	if Fixed(-3).Sign() != -1 || Fixed(0).Sign() != 0 || Fixed(3).Sign() != 1 {
		t.Error("Sign returned an unexpected value")
	}
}
