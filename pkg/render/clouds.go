package render

import (
	"math"
	"math/rand"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/game"
	"super-mario/pkg/render/theme"
)

// Clouds drift slowly left across the sky.
//
// They are scenery only: they live in the renderer, not the world, so they
// cannot touch the simulation, and they are not part of a replay. Their shapes,
// heights, speeds and spacing are random, picked fresh for every run.
//
// They hang in the level, not on the screen: when the view scrolls they pass by
// like the rest of the scenery, and walking back finds them where they drifted
// to. Each one drifts on its own, at its own pace, by the step count alone, so
// what the camera does never speeds a cloud up or slows it down.
//
// There are never more than three on screen. Clouds are kept in order from left
// to right, each a minimum distance behind the one ahead of it, and that
// distance is more than a third of what the screen can show. A faster cloud
// that catches up with a slower one falls in behind it at that distance instead
// of overtaking, the way clouds bunch up in a breeze. A cloud that drifts past
// the start of the level comes back at the far end as a new one.

// cloudShapes are the drawings. W is the cloud, w its shaded underside. Every
// row of a drawing is the same width.
var cloudShapes = []art{
	// tiny
	{".WW..", "WWWWW", ".www."},
	// small
	{"..WWW...", ".WWWWWW.", "WWWWWWWW", ".wwwwww."},
	// medium
	{"...WWW....", ".WWWWWWW..", "WWWWWWWWWW", ".wwwwwwww."},
	// tall
	{"...WWW...", "..WWWWW..", ".WWWWWWW.", "WWWWWWWWW", "WWWWWWWWW", ".wwwwwww."},
	// two puffs
	{"..WW....WW..", ".WWWW..WWWW.", "WWWWWWWWWWWW", ".wwwwwwwwww."},
	// three bumps
	{"..WW..WWW....", ".WWWWWWWWWW..", "WWWWWWWWWWWWW", ".wwwwwwwwwww."},
	// large
	{"....WWW.......", "..WWWWWWWWW...", ".WWWWWWWWWWWW.", "WWWWWWWWWWWWWW", ".wwwwwwwwwwww."},
	// long and flat
	{"...WWW...WW.....", ".WWWWWWWWWWWWW..", "WWWWWWWWWWWWWWWW", ".wwwwwwwwwwwwww."},
}

// maxCloudWidth is the widest drawing.
var maxCloudWidth = func() int {
	w := 0
	for _, s := range cloudShapes {
		if s.width() > w {
			w = s.width()
		}
	}
	return w
}()

// cloud is one cloud. Positions are in pixels of the four-pixel-tile drawing,
// measured in the level, so a cloud is the same size and in the same place at
// every zoom.
type cloud struct {
	shape int
	// x is the left edge, kept as a float so very slow drift still moves.
	x float64
	// y is the top edge.
	y int
	// speed is how far it moves left per simulation step.
	speed float64
	// gap is how close this cloud may come to the one ahead of it (on its
	// left), left edge to left edge. Each cloud has its own, so a bunch of
	// clouds is not evenly spaced either.
	gap float64
}

// Sky holds the clouds of the current run.
type Sky struct {
	rng *rand.Rand
	// clouds are in order from left to right.
	clouds []cloud
	// span is the level width and screen the screen width the clouds were laid
	// out for.
	span, screen int
	// last is the simulation step the clouds were last moved to.
	last int
}

const (
	// maxOnScreen is the most clouds a screen may show at once.
	maxOnScreen = 3
	// cloudTop and cloudBottom bound where a cloud's top edge may be, in
	// pixels from the top of the level: the upper sky, clear of the blocks the
	// player has to read.
	cloudTop, cloudBottom = 2, 14
	// cloudMinSpeed and cloudMaxSpeed are in pixels per step: at 60 steps a
	// second, a quarter to three quarters of a pixel a second, so a cloud takes
	// several seconds to move by one pixel.
	cloudMinSpeed, cloudMaxSpeed = 0.004, 0.012
	// cloudSpread is how much more than the minimum a cloud's gap may be, as a
	// fraction of it. The average comes out at about two clouds a screen.
	cloudSpread = 0.8
)

// NewSky returns a sky with its own random clouds. The seed makes a sky
// reproducible in tests; the game seeds it from the clock.
func NewSky(seed int64) *Sky {
	return &Sky{rng: rand.New(rand.NewSource(seed))}
}

// minGap is the least distance between two clouds' left edges that still keeps
// the screen to maxOnScreen.
//
// A cloud is on screen when its left edge is anywhere from one cloud width left
// of the screen to the screen's right edge: a stretch of screen plus the widest
// cloud. With every gap wider than a third of that stretch, a fourth cloud
// cannot fit in it. It is also never less than a cloud and a bit, so two clouds
// never overlap on a narrow terminal.
func (s *Sky) minGap() float64 {
	g := math.Floor(float64(s.screen+maxCloudWidth)/maxOnScreen) + 1
	if min := float64(maxCloudWidth + 4); g < min {
		g = min
	}
	return g
}

// newCloud makes a random cloud. It is not the same shape as the cloud before
// it, so two neighbours never look like copies.
func (s *Sky) newCloud(prevShape int) cloud {
	shape := s.rng.Intn(len(cloudShapes))
	if shape == prevShape {
		shape = (shape + 1 + s.rng.Intn(len(cloudShapes)-1)) % len(cloudShapes)
	}
	return cloud{
		shape: shape,
		y:     cloudTop + s.rng.Intn(cloudBottom-cloudTop+1),
		speed: cloudMinSpeed + s.rng.Float64()*(cloudMaxSpeed-cloudMinSpeed),
		gap:   s.minGap() * (1 + s.rng.Float64()*cloudSpread),
	}
}

// layout spreads clouds across the level, each its gap behind the last.
func (s *Sky) layout(span, screen int) {
	s.span, s.screen = span, screen
	s.clouds = s.clouds[:0]
	c := s.newCloud(-1)
	c.x = s.rng.Float64() * c.gap
	for c.x < float64(span) {
		s.clouds = append(s.clouds, c)
		next := s.newCloud(c.shape)
		next.x = c.x + next.gap
		c = next
	}
}

// advance moves the clouds to a simulation step. Driving them by step count,
// like every other animation, keeps their pace independent of the frame rate
// and of the camera, and they stop when the game is paused.
func (s *Sky) advance(step, span, screen int) {
	if span != s.span || screen != s.screen || s.clouds == nil {
		s.layout(span, screen)
		s.last = step
	}
	n := step - s.last
	if n < 0 || n > 600 {
		// A new run, or a long gap: do not fast-forward through it.
		n = 0
	}
	s.last = step
	for i := range s.clouds {
		c := &s.clouds[i]
		c.x -= c.speed * float64(n)
		// Left to right, so the cloud ahead has already moved: catching up
		// with it, this one falls in behind instead of passing.
		if i > 0 {
			if limit := s.clouds[i-1].x + c.gap; c.x < limit {
				c.x = limit
			}
		}
	}
	// Past the start of the level: a new random cloud at the far end, behind
	// the last one.
	for len(s.clouds) > 0 {
		first := s.clouds[0]
		if first.x+float64(cloudShapes[first.shape].width()) >= 0 {
			break
		}
		last := s.clouds[len(s.clouds)-1]
		c := s.newCloud(last.shape)
		c.x = math.Max(float64(span), last.x+c.gap)
		s.clouds = append(s.clouds[1:], c)
	}
}

// cloudPalette colours a cloud.
func cloudPalette(t theme.Tokens) palette {
	return func(l byte) (tcell.Color, bool) {
		switch l {
		case 'W':
			return t.Cast.Cloud, true
		case 'w':
			return t.Cast.CloudShade, true
		}
		return 0, false
	}
}

// clouds draws the sky's clouds behind everything else.
func (d drawer) clouds(w *game.World) {
	if d.s.Sky == nil {
		return
	}
	k := d.castScale()
	screen := (d.artW + k - 1) / k
	d.s.Sky.advance(w.StepCount(), w.Level.Width*heroDesignPx, screen)
	pal := cloudPalette(d.s.T)
	for _, c := range d.s.Sky.clouds {
		x := int(c.x)*k - d.camX
		if x >= d.artW || x+cloudShapes[c.shape].width()*k <= 0 {
			continue
		}
		d.c.blitArt(cloudShapes[c.shape], x, c.y*k-d.camY, k, false, pal)
	}
}
