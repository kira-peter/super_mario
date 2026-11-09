package save

import (
	"sort"
	"time"
)

// MaxEntries is how many scores a board keeps. A short board stays readable on
// one screen and keeps the save file tiny.
const MaxEntries = 10

// Entry is one finished run on the leaderboard.
type Entry struct {
	// Score is the final score, including the time bonus.
	Score int `json:"score"`
	// Steps is how long the run took in simulation steps, at 60 per second.
	//
	// Time is stored as steps rather than seconds because the simulation is
	// deterministic and steps are exact: the same run always records the same
	// number, and it is converted for display only.
	Steps int `json:"steps"`
	// Level names the stage, so a board can span several stages later.
	Level string `json:"level"`
	// Cleared reports whether the run reached the flagpole. A run that ran out
	// of lives is still recorded, but is marked so it reads differently.
	Cleared bool `json:"cleared"`
	// At is when the run ended, in UTC.
	At time.Time `json:"at"`
}

// Seconds returns the run time in seconds.
func (e Entry) Seconds() float64 { return float64(e.Steps) / 60.0 }

// Board is the ranked list of runs, best first.
type Board struct {
	Entries []Entry
}

// better reports whether a ranks above b: higher score wins, and on a tie the
// faster run does, and on a further tie the earlier one, so ranking is stable.
func better(a, b Entry) bool {
	if a.Score != b.Score {
		return a.Score > b.Score
	}
	if a.Steps != b.Steps {
		return a.Steps < b.Steps
	}
	return a.At.Before(b.At)
}

// Add inserts a run and returns its 1-based rank, or 0 if it did not make the
// board. The board is trimmed to MaxEntries.
func (b *Board) Add(e Entry) int {
	if e.Score <= 0 {
		return 0
	}
	b.Entries = append(b.Entries, e)
	b.sort()
	if len(b.Entries) > MaxEntries {
		b.Entries = b.Entries[:MaxEntries]
	}
	for i, got := range b.Entries {
		if got == e {
			return i + 1
		}
	}
	return 0
}

// sort orders the entries best first.
func (b *Board) sort() {
	sort.SliceStable(b.Entries, func(i, j int) bool {
		return better(b.Entries[i], b.Entries[j])
	})
}

// Best returns the top entry, if any.
func (b *Board) Best() (Entry, bool) {
	if len(b.Entries) == 0 {
		return Entry{}, false
	}
	return b.Entries[0], true
}

// Qualifies reports whether a score would earn a place, without adding it.
func (b *Board) Qualifies(score int) bool {
	if score <= 0 {
		return false
	}
	if len(b.Entries) < MaxEntries {
		return true
	}
	return score > b.Entries[len(b.Entries)-1].Score
}

// FormatDuration renders a step count as m:ss.t, for example 1:23.4.
//
// Tenths are kept because two clears a second apart are common, and a board
// that cannot tell them apart is not much of a board.
func FormatDuration(steps int) string {
	if steps < 0 {
		steps = 0
	}
	tenths := steps * 10 / 60
	m := tenths / 600
	s := (tenths / 10) % 60
	t := tenths % 10
	return formatMST(m, s, t)
}

func formatMST(m, s, t int) string {
	return itoa(m) + ":" + pad2(s) + "." + itoa(t)
}

func pad2(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
