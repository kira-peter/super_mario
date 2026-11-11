package render

import (
	"testing"

	"super-mario/pkg/engine"
	"super-mario/pkg/render/theme"
)

const (
	testSpan   = 212 * 4 // the 1-1 level, in four-pixel-tile pixels
	testScreen = 160     // castWorld's forty-tile view at the default zoom
)

// TestCloudsDriftLeft checks the clouds move left as the steps go by.
func TestCloudsDriftLeft(t *testing.T) {
	s := NewSky(1)
	s.advance(0, testSpan, testScreen)
	before := s.clouds[0].x
	s.advance(300, testSpan, testScreen)
	if s.clouds[0].x >= before {
		t.Errorf("the first cloud went from %.3f to %.3f, want it further left", before, s.clouds[0].x)
	}
}

// TestCloudsDriftSlowly pins the pace: no cloud moves more than one pixel in a
// second, so they read as drifting, not sliding.
func TestCloudsDriftSlowly(t *testing.T) {
	s := NewSky(1)
	s.advance(0, testSpan, testScreen)
	for _, c := range s.clouds {
		if perSecond := c.speed * 60; perSecond > 1 {
			t.Errorf("a cloud moves %.2f pixels a second, want at most 1", perSecond)
		}
	}
}

// visible counts the clouds any part of which is on a screen at camX.
func visible(s *Sky, camX, screen float64) int {
	n := 0
	for _, c := range s.clouds {
		left := float64(int(c.x))
		if left < camX+screen && left+float64(cloudShapes[c.shape].width()) > camX {
			n++
		}
	}
	return n
}

// TestNeverMoreThanThreeCloudsOnScreen is the rule asked for, checked the hard
// way: for several screen widths, at every camera position across the level,
// over a whole run and beyond, while fast clouds catch up with slow ones.
func TestNeverMoreThanThreeCloudsOnScreen(t *testing.T) {
	for _, screen := range []int{40, 60, 100, 120, 160, 240} {
		for seed := int64(1); seed <= 5; seed++ {
			s := NewSky(seed)
			for step := 0; step <= 200000; step += 500 {
				s.advance(step, testSpan, screen)
				for cam := 0; cam+screen <= testSpan; cam += 2 {
					if n := visible(s, float64(cam), float64(screen)); n > maxOnScreen {
						t.Fatalf("screen %d, seed %d, step %d, camera %d: %d clouds on screen, want at most %d",
							screen, seed, step, cam, n, maxOnScreen)
					}
				}
			}
		}
	}
}

// TestCloudsAreNotTooFewEither checks the sky does not go bare: across the
// level there is on average more than one cloud per screen.
func TestCloudsAreNotTooFewEither(t *testing.T) {
	s := NewSky(1)
	s.advance(0, testSpan, testScreen)
	perScreen := float64(len(s.clouds)) * testScreen / testSpan
	if perScreen < 1.2 {
		t.Errorf("%.2f clouds a screen on average, the sky looks bare", perScreen)
	}
}

// TestCloudsNeverOverlap checks no two clouds are drawn on top of each other.
func TestCloudsNeverOverlap(t *testing.T) {
	for _, screen := range []int{20, 40, 160} {
		s := NewSky(2)
		for step := 0; step <= 200000; step += 500 {
			s.advance(step, testSpan, screen)
			for i := 1; i < len(s.clouds); i++ {
				prev := s.clouds[i-1]
				if s.clouds[i].x < prev.x+float64(cloudShapes[prev.shape].width()) {
					t.Fatalf("screen %d step %d: clouds %d and %d overlap", screen, step, i-1, i)
				}
			}
		}
	}
}

// TestThereAreManyCloudShapes checks the variety asked for, and that every
// drawing is rectangular with only cloud letters in it.
func TestThereAreManyCloudShapes(t *testing.T) {
	if len(cloudShapes) < 6 {
		t.Errorf("%d cloud shapes, want at least 6", len(cloudShapes))
	}
	pal := cloudPalette(theme.Ink)
	for i, s := range cloudShapes {
		for r, row := range s {
			if len(row) != s.width() {
				t.Errorf("shape %d row %d is %d wide, want %d", i, r, len(row), s.width())
			}
			for _, ch := range []byte(row) {
				if _, ok := pal(ch); ch != '.' && !ok {
					t.Errorf("shape %d uses %q", i, string(ch))
				}
			}
		}
	}
	// A run should actually use most of them.
	used := map[int]bool{}
	s := NewSky(3)
	s.advance(0, testSpan, testScreen)
	for _, c := range s.clouds {
		used[c.shape] = true
	}
	if len(used) < len(cloudShapes)/2 {
		t.Errorf("a whole level of clouds used only %d of %d shapes", len(used), len(cloudShapes))
	}
}

// TestNeighbouringCloudsDiffer checks two clouds side by side are never the
// same drawing.
func TestNeighbouringCloudsDiffer(t *testing.T) {
	s := NewSky(4)
	for step := 0; step <= 200000; step += 500 {
		s.advance(step, testSpan, testScreen)
		for i := 1; i < len(s.clouds); i++ {
			if s.clouds[i].shape == s.clouds[i-1].shape {
				t.Fatalf("step %d: clouds %d and %d are the same shape", step, i-1, i)
			}
		}
	}
}

// TestCloudsAreRandomPerRun checks two runs do not get the same sky, and the
// clouds of one sky differ in speed.
func TestCloudsAreRandomPerRun(t *testing.T) {
	a, b := NewSky(1), NewSky(2)
	a.advance(0, testSpan, testScreen)
	b.advance(0, testSpan, testScreen)
	same := len(a.clouds) == len(b.clouds)
	for i := 0; same && i < len(a.clouds); i++ {
		if a.clouds[i] != b.clouds[i] {
			same = false
		}
	}
	if same {
		t.Error("two different seeds gave the same clouds")
	}
	speeds := map[float64]bool{}
	for _, c := range a.clouds {
		speeds[c.speed] = true
	}
	if len(speeds) < 2 {
		t.Error("every cloud drifts at the same speed")
	}
}

// TestCloudsComeBackAtTheFarEnd checks a cloud that drifts past the start of
// the level is replaced by one at the far end, so the sky never empties.
func TestCloudsComeBackAtTheFarEnd(t *testing.T) {
	s := NewSky(3)
	s.advance(0, testSpan, testScreen)
	n := len(s.clouds)
	for step := 0; step < 2000000; step += 600 {
		s.advance(step, testSpan, testScreen)
		if len(s.clouds) < n-1 || len(s.clouds) > n+1 {
			t.Fatalf("cloud count went from %d to %d", n, len(s.clouds))
		}
		for _, c := range s.clouds {
			if c.x+float64(cloudShapes[c.shape].width()) < 0 {
				t.Fatalf("a cloud is stuck past the start of the level at %.2f", c.x)
			}
		}
	}
}

// TestCloudsStayInTheUpperSky checks no cloud hangs low, where it would sit
// behind the blocks the player has to read.
func TestCloudsStayInTheUpperSky(t *testing.T) {
	s := NewSky(4)
	for step := 0; step < 2000000; step += 600 {
		s.advance(step, testSpan, testScreen)
		for _, c := range s.clouds {
			if c.y < cloudTop || c.y > cloudBottom {
				t.Fatalf("a cloud's top is at %d, want %d..%d", c.y, cloudTop, cloudBottom)
			}
		}
	}
}

// cloudLeftEdge renders and returns the leftmost column with cloud on it.
func cloudLeftEdge(t *testing.T, scene *Scene, camX engine.Fixed) int {
	t.Helper()
	w := castWorld(t)
	w.CameraX, w.CameraY = camX, 0
	c := scene.Draw(w, 4)
	for x := 0; x < c.W; x++ {
		for y := 0; y < c.H; y++ {
			if c.Px[y*c.W+x] == theme.Ink.Cast.Cloud {
				return x
			}
		}
	}
	return -1
}

// TestCloudsScrollWithTheLevel checks the clouds hang in the level: moving the
// camera moves a cloud on screen by exactly the same amount, the other way.
func TestCloudsScrollWithTheLevel(t *testing.T) {
	sky := NewSky(5)
	sky.advance(0, testSpan, testScreen)
	// Frame a cloud eight pixels in from the left, then look again two tiles
	// further right. The cloud before it is a whole gap further left, off
	// screen, so the leftmost cloud on screen is this one both times.
	var target cloud
	for _, c := range sky.clouds {
		if c.x >= 16 {
			target = c
			break
		}
	}
	cam := engine.Fixed(int(target.x)/4-2) * engine.Tile
	scene := &Scene{T: theme.Ink, Sky: sky}
	a := cloudLeftEdge(t, scene, cam)
	b := cloudLeftEdge(t, scene, cam+2*engine.Tile)
	if a < 0 || b < 0 {
		t.Fatal("no cloud on screen")
	}
	if a-b != 8 {
		t.Errorf("scrolling two tiles moved the cloud %d pixels, want 8", a-b)
	}
}

// TestCloudPaceIgnoresTheCamera checks the clouds drift the same distance in
// the same number of steps whether the camera stands still or runs back and
// forth across the level meanwhile.
func TestCloudPaceIgnoresTheCamera(t *testing.T) {
	still, moving := NewSky(6), NewSky(6)
	ws, wm := castWorld(t), castWorld(t)
	ss, sm := &Scene{T: theme.Ink, Sky: still}, &Scene{T: theme.Ink, Sky: moving}
	for i := 0; i < 1200; i++ {
		ws.Step()
		wm.Step()
		ws.CameraX = 0
		wm.CameraX = engine.Fixed((i/10)%60) * engine.Tile
		if (i/600)%2 == 1 {
			wm.CameraX = engine.Fixed(60-(i/10)%60) * engine.Tile
		}
		ss.Draw(ws, 4)
		sm.Draw(wm, 4)
	}
	for i := range still.clouds {
		if still.clouds[i] != moving.clouds[i] {
			t.Fatalf("cloud %d ended at %.3f with a still camera and %.3f with a moving one",
				i, still.clouds[i].x, moving.clouds[i].x)
		}
	}
}

// TestCloudsAreDrawnBehindTheTerrain renders the real level with a sky and
// checks clouds appear, and never over the ground.
func TestCloudsAreDrawnBehindTheTerrain(t *testing.T) {
	w := castWorld(t)
	w.CameraX, w.CameraY = 0, 0
	scene := &Scene{T: theme.Ink, Sky: NewSky(5)}
	c := scene.Draw(w, 4)
	if countColour(c, theme.Ink.Cast.Cloud) == 0 {
		t.Fatal("a sky with clouds drew none")
	}
	groundTop := 13 * 4
	for x := 0; x < c.W; x++ {
		if p := c.Px[groundTop*c.W+x]; p == theme.Ink.Cast.Cloud || p == theme.Ink.Cast.CloudShade {
			t.Fatalf("a cloud is drawn over the ground at x=%d", x)
		}
	}
}

// TestCloudsAreWhite pins the colour asked for.
func TestCloudsAreWhite(t *testing.T) {
	if r, g, b := theme.Ink.Cast.Cloud.RGB(); r != 255 || g != 255 || b != 255 {
		t.Errorf("cloud is #%02x%02x%02x, want pure white", r, g, b)
	}
}

// TestNoSkyDrawsNoClouds keeps the pixel-counting tests honest: a scene with no
// sky set draws a clear one.
func TestNoSkyDrawsNoClouds(t *testing.T) {
	w := castWorld(t)
	c := (&Scene{T: theme.Ink}).Draw(w, 4)
	if countColour(c, theme.Ink.Cast.Cloud) != 0 {
		t.Error("a scene without a sky drew clouds")
	}
}
