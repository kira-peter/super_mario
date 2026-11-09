package app

import "super-mario/pkg/engine"

// keyProbe reports whether a key is physically held down right now.
//
// known is false when the platform cannot say for that key, in which case the
// caller falls back to inferring the state from the presses it has seen.
type keyProbe func(k engine.Key) (down bool, known bool)
