package save

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func entry(score, steps int) Entry {
	return Entry{Score: score, Steps: steps, Level: "1-1", Cleared: true,
		At: time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)}
}

func TestBoardRanksByScoreThenSpeed(t *testing.T) {
	var b Board
	b.Add(entry(1000, 5000))
	b.Add(entry(3000, 9000))
	b.Add(entry(3000, 4000)) // same score, faster: ranks above
	b.Add(entry(2000, 1000))

	want := []struct{ score, steps int }{{3000, 4000}, {3000, 9000}, {2000, 1000}, {1000, 5000}}
	for i, w := range want {
		if b.Entries[i].Score != w.score || b.Entries[i].Steps != w.steps {
			t.Errorf("rank %d = %d pts / %d steps, want %d / %d",
				i+1, b.Entries[i].Score, b.Entries[i].Steps, w.score, w.steps)
		}
	}
}

func TestAddReturnsTheRank(t *testing.T) {
	var b Board
	if r := b.Add(entry(500, 100)); r != 1 {
		t.Errorf("first entry rank = %d, want 1", r)
	}
	if r := b.Add(entry(900, 100)); r != 1 {
		t.Errorf("a better entry rank = %d, want 1", r)
	}
	if r := b.Add(entry(100, 100)); r != 3 {
		t.Errorf("the worst entry rank = %d, want 3", r)
	}
}

func TestBoardKeepsOnlyTheBest(t *testing.T) {
	var b Board
	for i := 1; i <= MaxEntries+5; i++ {
		b.Add(entry(i*100, 1000))
	}
	if len(b.Entries) != MaxEntries {
		t.Fatalf("board holds %d entries, want %d", len(b.Entries), MaxEntries)
	}
	if got := b.Entries[len(b.Entries)-1].Score; got != 600 {
		t.Errorf("lowest kept score = %d, want 600", got)
	}
	if r := b.Add(entry(50, 1000)); r != 0 {
		t.Errorf("a score below the board got rank %d, want 0 (not placed)", r)
	}
}

func TestZeroScoreIsNotRecorded(t *testing.T) {
	var b Board
	if r := b.Add(entry(0, 100)); r != 0 || len(b.Entries) != 0 {
		t.Errorf("a zero score was recorded (rank %d, %d entries)", r, len(b.Entries))
	}
}

func TestQualifies(t *testing.T) {
	var b Board
	if !b.Qualifies(1) {
		t.Error("any positive score should qualify on an empty board")
	}
	if b.Qualifies(0) {
		t.Error("zero should never qualify")
	}
	for i := 1; i <= MaxEntries; i++ {
		b.Add(entry(i*100, 1))
	}
	if b.Qualifies(100) {
		t.Error("a score equal to the lowest should not displace it")
	}
	if !b.Qualifies(101) {
		t.Error("a score above the lowest should qualify")
	}
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		steps int
		want  string
	}{
		{0, "0:00.0"},
		{60, "0:01.0"},
		{90, "0:01.5"},
		{3600, "1:00.0"},
		{4994, "1:23.2"}, // 83.23 s
		{-5, "0:00.0"},
	}
	for _, c := range cases {
		if got := FormatDuration(c.steps); got != c.want {
			t.Errorf("FormatDuration(%d) = %q, want %q", c.steps, got, c.want)
		}
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	s := NewAt(t.TempDir())
	var b Board
	b.Add(entry(12345, 4994))
	b.Add(entry(8000, 3000))
	s.Save(b)

	got := s.Load()
	if len(got.Entries) != 2 {
		t.Fatalf("loaded %d entries, want 2", len(got.Entries))
	}
	if got.Entries[0].Score != 12345 || got.Entries[0].Steps != 4994 {
		t.Errorf("top entry = %+v, want 12345 pts / 4994 steps", got.Entries[0])
	}
	if !got.Entries[0].At.Equal(b.Entries[0].At) {
		t.Errorf("timestamp changed across save/load: %v vs %v", got.Entries[0].At, b.Entries[0].At)
	}
}

// TestVersionOneSaveIsMigrated checks that a score saved by the previous
// release survives the format change.
func TestVersionOneSaveIsMigrated(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName),
		[]byte(`{"highScore":9876,"version":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got := NewAt(dir).Load()
	if len(got.Entries) != 1 || got.Entries[0].Score != 9876 {
		t.Fatalf("old high score was lost: %+v", got.Entries)
	}
	if got.Entries[0].Steps != 0 {
		t.Errorf("a migrated entry has no recorded time, got %d steps", got.Entries[0].Steps)
	}
}

// TestMigrationDoesNotRepeat checks the old score is not counted again once the
// file has been rewritten in the new format.
func TestMigrationDoesNotRepeat(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, FileName), []byte(`{"highScore":500,"version":1}`), 0o644)
	s := NewAt(dir)
	b := s.Load()
	b.Add(entry(900, 100))
	s.Save(b)

	got := s.Load()
	count := 0
	for _, e := range got.Entries {
		if e.Score == 500 {
			count++
		}
	}
	if count != 1 {
		t.Errorf("the migrated score appears %d times, want 1", count)
	}
}

func TestMissingFileGivesAnEmptyBoardWithoutWarning(t *testing.T) {
	s := NewAt(t.TempDir())
	warned := false
	s.Warn = func(string) { warned = true }
	if got := s.Load(); len(got.Entries) != 0 {
		t.Errorf("a first run has %d entries, want 0", len(got.Entries))
	}
	if warned {
		t.Error("a missing file is the normal first run and must not warn")
	}
}

func TestCorruptFileFallsBackToAnEmptyBoard(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, FileName), []byte("{not json"), 0o644)
	s := NewAt(dir)
	warned := false
	s.Warn = func(string) { warned = true }
	if got := s.Load(); len(got.Entries) != 0 {
		t.Errorf("a corrupt file produced %d entries", len(got.Entries))
	}
	if !warned {
		t.Error("a corrupt file should produce one notice")
	}
}

func TestNonPositiveScoresInTheFileAreDropped(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, FileName),
		[]byte(`{"version":2,"entries":[{"score":-5,"steps":1},{"score":300,"steps":2}]}`), 0o644)
	got := NewAt(dir).Load()
	if len(got.Entries) != 1 || got.Entries[0].Score != 300 {
		t.Errorf("entries = %+v, want only the 300 entry", got.Entries)
	}
}

func TestUnwritableDirectoryDoesNotPanic(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	_ = os.WriteFile(blocker, []byte("x"), 0o644)

	s := NewAt(filepath.Join(blocker, "nested"))
	warned := false
	s.Warn = func(string) { warned = true }
	var b Board
	b.Add(entry(99, 1))
	s.Save(b)
	if !warned {
		t.Error("an unwritable directory should warn")
	}
	if got := s.Load(); len(got.Entries) != 0 {
		t.Error("loading from an unwritable directory should give an empty board")
	}
}

func TestSaveIsAtomic(t *testing.T) {
	dir := t.TempDir()
	s := NewAt(dir)
	var b Board
	b.Add(entry(10, 1))
	s.Save(b)
	b.Add(entry(20, 1))
	s.Save(b)
	if _, err := os.Stat(filepath.Join(dir, FileName+".tmp")); err == nil {
		t.Error("a temp file was left behind after saving")
	}
	if got := s.Load(); got.Entries[0].Score != 20 {
		t.Errorf("top score = %d, want the latest 20", got.Entries[0].Score)
	}
}

func TestNoConfigDirIsHarmless(t *testing.T) {
	s := &Store{dir: ""}
	warned := false
	s.Warn = func(string) { warned = true }
	if got := s.Load(); len(got.Entries) != 0 {
		t.Error("expected an empty board")
	}
	var b Board
	b.Add(entry(5, 1))
	s.Save(b)
	if !warned {
		t.Error("saving with no directory should warn")
	}
}
