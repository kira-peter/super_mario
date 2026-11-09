// Package save persists the leaderboard.
//
// The contract that matters is in spec game-flow / 最高分持久化: a broken,
// missing or unwritable file must never stop the game from starting.
package save

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// FileName is the on-disk name inside the app's config directory.
const FileName = "save.json"

// currentVersion is the format written today.
//
//	1  {"highScore": N}                       a single number
//	2  {"entries": [...]}                     a ranked board
const currentVersion = 2

// Data is the persisted state.
type Data struct {
	Version int     `json:"version"`
	Entries []Entry `json:"entries"`

	// HighScore is the version-1 field. It is read so an old save is not
	// thrown away, folded into Entries on load, and never written back.
	HighScore int `json:"highScore,omitempty"`
}

// Store reads and writes the save file.
//
// The directory is resolved lazily so tests can point it at a temp dir without
// touching the user's real config.
type Store struct {
	dir string
	// Warn is called when the file could not be read or written. It is for a
	// single non-blocking notice, never for aborting the game.
	Warn func(msg string)
}

// New returns a store rooted at the user's config directory.
//
// os.UserConfigDir resolves to the right place on Windows, macOS and Linux, so
// there is no hard-coded platform path anywhere.
func New() *Store {
	dir, err := os.UserConfigDir()
	if err != nil {
		return &Store{dir: ""}
	}
	return &Store{dir: filepath.Join(dir, "super-mario")}
}

// NewAt returns a store rooted at an explicit directory, used by tests.
func NewAt(dir string) *Store { return &Store{dir: dir} }

// Path returns the absolute save file path, or "" when no directory resolved.
func (s *Store) Path() string {
	if s.dir == "" {
		return ""
	}
	return filepath.Join(s.dir, FileName)
}

// Load returns the saved board.
//
// A missing file is not an error: it means a first run. A corrupt file yields
// an empty board plus a warning. A version-1 file is migrated, not discarded.
func (s *Store) Load() Board {
	path := s.Path()
	if path == "" {
		return Board{}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			s.warn("排行榜读取失败，本次从空榜开始")
		}
		return Board{}
	}
	var d Data
	if err := json.Unmarshal(raw, &d); err != nil {
		s.warn("排行榜文件已损坏，本次从空榜开始")
		return Board{}
	}

	var b Board
	for _, e := range d.Entries {
		if e.Score > 0 {
			b.Entries = append(b.Entries, e)
		}
	}
	// A version-1 save carried a single score and no run time. It becomes one
	// entry with no recorded time, so the old best is not lost. It is only
	// folded in when the file has no board of its own, to avoid counting it
	// again on every load.
	if d.HighScore > 0 && len(b.Entries) == 0 {
		b.Entries = append(b.Entries, Entry{
			Score: d.HighScore,
			Level: "1-1",
		})
	}
	b.sort()
	if len(b.Entries) > MaxEntries {
		b.Entries = b.Entries[:MaxEntries]
	}
	return b
}

// Save writes the board.
//
// A failure is reported through Warn and never returned: losing a score is not
// a reason to interrupt play.
func (s *Store) Save(b Board) {
	path := s.Path()
	if path == "" {
		s.warn("无法确定存档目录，排行榜不会保存")
		return
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		s.warn("无法创建存档目录，排行榜不会保存")
		return
	}
	entries := b.Entries
	if len(entries) > MaxEntries {
		entries = entries[:MaxEntries]
	}
	raw, err := json.MarshalIndent(Data{Version: currentVersion, Entries: entries}, "", "  ")
	if err != nil {
		s.warn("存档编码失败，排行榜不会保存")
		return
	}
	// Write to a temp file then rename, so a crash mid-write cannot leave a
	// truncated save behind.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		s.warn("存档写入失败，排行榜不会保存")
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		s.warn("存档替换失败，排行榜不会保存")
	}
}

func (s *Store) warn(msg string) {
	if s.Warn != nil {
		s.Warn(msg)
	}
}

// now is the clock used to stamp entries; tests replace it.
var now = func() time.Time { return time.Now().UTC() }

// NewEntry builds an entry for a run that just ended.
func NewEntry(score, steps int, level string, cleared bool) Entry {
	return Entry{Score: score, Steps: steps, Level: level, Cleared: cleared, At: now()}
}
