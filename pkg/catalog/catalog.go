// Package catalog is where levels live: the built-in ones and the local ones
// beside the game program (spec level-select / 本地关卡不用重新编译, spec
// level-editor / 保存位置).
//
// Two sources:
//
//   - Builtin: the levels that ship with the game. Inside the repository this
//     is pkg/level/data on disk, so a level the editor just saved can be played
//     without rebuilding. Outside it, it is the copy compiled into the program,
//     and it is read-only.
//   - Local: the levels/ folder next to the program, read at run time. A level
//     someone sends you goes here; copy the program and this folder together
//     to share them.
//
// A level is named by its source and id. A built-in and a local level may share
// an id; both are listed.
package catalog

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"super-mario/pkg/level"
)

// Source says where a level is kept.
type Source string

const (
	Builtin Source = "builtin"
	Local   Source = "local"
)

// LocalDirName is the folder beside the program that holds local levels.
const LocalDirName = "levels"

// RepoLevelDir is where the built-in levels live inside the repository.
const RepoLevelDir = "pkg/level/data"

// moduleName is the go.mod module the repository declares.
const moduleName = "super-mario"

// ErrReadOnly is returned for a write to built-in levels that are compiled into
// the program and not on disk.
var ErrReadOnly = errors.New("built-in levels are part of the program and cannot be changed")

// ErrNotFound is returned when no such level exists.
var ErrNotFound = errors.New("no such level")

// Entry is one level in the catalog.
type Entry struct {
	Source Source `json:"source"`
	ID     string `json:"id"`
	// Path is the file on disk, or "" for a level inside the program.
	Path string `json:"path,omitempty"`
	// Meta is the level's description; zero when it did not load.
	Title       string `json:"title,omitempty"`
	Author      string `json:"author,omitempty"`
	Difficulty  int    `json:"difficulty,omitempty"`
	Description string `json:"description,omitempty"`
	Official    bool   `json:"official"`
	// Err is why the file does not load, or "".
	Err string `json:"error,omitempty"`
}

// Key is the level's unique name across both sources: the id for a built-in
// level, "local:" and the id for a local one. Leaderboard entries use it, so a
// local copy of 1-1 keeps its own scores.
func (e Entry) Key() string { return Key(e.Source, e.ID) }

// Key builds an entry key.
func Key(src Source, id string) string {
	if src == Local {
		return "local:" + id
	}
	return id
}

// ParseKey splits a key into source and id.
func ParseKey(k string) (Source, string) {
	if id, ok := strings.CutPrefix(k, "local:"); ok {
		return Local, id
	}
	return Builtin, k
}

// Catalog is the two level directories.
type Catalog struct {
	// BuiltinDir is pkg/level/data on disk, or "" to use the levels compiled
	// into the program.
	BuiltinDir string
	// LocalDir is the levels/ folder beside the program.
	LocalDir string
	// Embedded is the compiled-in levels, used when BuiltinDir is "".
	Embedded fs.FS
}

// Default finds the two directories for the running program: the repository
// from the working directory or the program's own directory, and the levels/
// folder beside the program.
func Default() *Catalog {
	c := &Catalog{Embedded: level.Embedded()}
	exe, _ := os.Executable()
	exeDir := ""
	if exe != "" {
		if r, err := filepath.EvalSymlinks(exe); err == nil {
			exe = r
		}
		exeDir = filepath.Dir(exe)
	}
	for _, start := range []string{".", exeDir} {
		if start == "" {
			continue
		}
		if root, err := FindRepo(start); err == nil {
			c.BuiltinDir = filepath.Join(root, filepath.FromSlash(RepoLevelDir))
			break
		}
	}
	if exeDir != "" && !isGoBuildCache(exeDir) {
		c.LocalDir = filepath.Join(exeDir, LocalDirName)
	} else if wd, err := os.Getwd(); err == nil {
		// `go run` builds into a temporary directory that is deleted on
		// exit; keep local levels beside where it was run instead.
		c.LocalDir = filepath.Join(wd, LocalDirName)
	}
	return c
}

// isGoBuildCache reports whether a directory is where `go run` and `go test`
// put their temporary programs.
func isGoBuildCache(dir string) bool {
	tmp := os.TempDir()
	rel, err := filepath.Rel(tmp, dir)
	return err == nil && !strings.HasPrefix(rel, "..") && strings.Contains(strings.ToLower(dir), "go-build")
}

// InRepo reports whether built-in levels are read from the repository on disk.
func (c *Catalog) InRepo() bool { return c.BuiltinDir != "" }

// Writable reports whether levels can be saved to a source.
func (c *Catalog) Writable(src Source) bool {
	switch src {
	case Builtin:
		return c.BuiltinDir != ""
	case Local:
		return c.LocalDir != ""
	}
	return false
}

// NewLevelSource is where a new level is saved: the repository when running in
// it (so it can be contributed), otherwise the local folder.
func (c *Catalog) NewLevelSource() Source {
	if c.InRepo() {
		return Builtin
	}
	return Local
}

// Dir is a source's directory on disk, or "".
func (c *Catalog) Dir(src Source) string {
	if src == Builtin {
		return c.BuiltinDir
	}
	return c.LocalDir
}

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
var officialPattern = regexp.MustCompile(`^[0-9]+-[0-9]+$`)
var communityPattern = regexp.MustCompile(`^c-[a-z0-9]+(-[a-z0-9]+)+$`)

// ValidName reports whether id is safe to use as a file name. It is checked
// before every read and write, which is what stops a name like "../x" from
// leaving the level directory.
func ValidName(id string) bool { return idPattern.MatchString(id) }

// ValidID reports whether id follows the naming rule for a saved level: an
// official world-stage name, or a community c-author-name.
func ValidID(id string) bool {
	return ValidName(id) && (officialPattern.MatchString(id) || communityPattern.MatchString(id))
}

// IsOfficial reports whether id is an official world-stage name.
func IsOfficial(id string) bool { return officialPattern.MatchString(id) }

// List returns every level in both sources: built-in ones first, then local
// ones. Files that do not load are included with Err set, so the editor can
// say why; the game leaves them out.
func (c *Catalog) List() []Entry {
	var out []Entry
	out = append(out, c.list(Builtin)...)
	out = append(out, c.list(Local)...)
	return out
}

func (c *Catalog) list(src Source) []Entry {
	var out []Entry
	add := func(id, path string, data []byte, err error) {
		e := Entry{Source: src, ID: id, Path: path, Official: IsOfficial(id)}
		if err == nil {
			var lv *level.Level
			lv, err = level.Parse(data)
			if err == nil {
				e.Title, e.Author, e.Difficulty, e.Description = lv.Meta.Title, lv.Meta.Author, lv.Meta.Difficulty, lv.Meta.Description
			}
		}
		if err != nil {
			e.Err = err.Error()
		}
		out = append(out, e)
	}
	dir := c.Dir(src)
	switch {
	case dir != "":
		paths, _ := filepath.Glob(filepath.Join(dir, "*.json"))
		for _, p := range paths {
			id := strings.TrimSuffix(filepath.Base(p), ".json")
			if !ValidName(id) {
				continue
			}
			data, err := os.ReadFile(p)
			add(id, p, data, err)
		}
	case src == Builtin && c.Embedded != nil:
		entries, _ := fs.ReadDir(c.Embedded, ".")
		for _, en := range entries {
			id, ok := strings.CutSuffix(en.Name(), ".json")
			if !ok || !ValidName(id) {
				continue
			}
			data, err := fs.ReadFile(c.Embedded, en.Name())
			add(id, "", data, err)
		}
	}
	sortEntries(out)
	return out
}

// sortEntries orders one source: official levels by world and stage as numbers
// (so 1-10 comes after 1-9), then community levels by title.
func sortEntries(l []Entry) {
	sort.SliceStable(l, func(i, j int) bool {
		a, b := l[i], l[j]
		if a.Official != b.Official {
			return a.Official
		}
		if a.Official {
			aw, as := worldStage(a.ID)
			bw, bs := worldStage(b.ID)
			if aw != bw {
				return aw < bw
			}
			return as < bs
		}
		return strings.ToLower(a.Label()) < strings.ToLower(b.Label())
	})
}

// Label is the level's title, or its id when it has none.
func (e Entry) Label() string {
	if e.Title != "" {
		return e.Title
	}
	return e.ID
}

func worldStage(id string) (int, int) {
	var w, s int
	fmt.Sscanf(id, "%d-%d", &w, &s)
	return w, s
}

// Read returns a level file's bytes.
func (c *Catalog) Read(src Source, id string) ([]byte, error) {
	if !ValidName(id) {
		return nil, ErrNotFound
	}
	if dir := c.Dir(src); dir != "" {
		data, err := os.ReadFile(filepath.Join(dir, id+".json"))
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrNotFound
		}
		return data, err
	}
	if src == Builtin && c.Embedded != nil {
		data, err := fs.ReadFile(c.Embedded, id+".json")
		if err != nil {
			return nil, ErrNotFound
		}
		return data, nil
	}
	return nil, ErrNotFound
}

// Load reads and parses a level.
func (c *Catalog) Load(src Source, id string) (*level.Level, error) {
	data, err := c.Read(src, id)
	if err != nil {
		return nil, err
	}
	return level.Parse(data)
}

// Exists reports whether a source already has a level by this id.
func (c *Catalog) Exists(src Source, id string) bool {
	_, err := c.Read(src, id)
	return err == nil
}

// Write saves a level file in the canonical layout and returns its path. The
// level is checked first: a level the game would refuse is never written. The
// local folder is created when it does not exist yet.
func (c *Catalog) Write(src Source, id string, data []byte) (string, error) {
	if !ValidName(id) {
		return "", fmt.Errorf("bad level name %q", id)
	}
	dir := c.Dir(src)
	if dir == "" {
		return "", ErrReadOnly
	}
	formatted, err := level.Format(data)
	if err != nil {
		return "", err
	}
	if src == Local {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}
	path := filepath.Join(dir, id+".json")
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, formatted) {
		return path, nil
	}
	return path, WriteAtomic(path, formatted)
}

// WriteAtomic writes next to the target and renames over it, so the old file
// is either untouched or fully replaced, never half-written.
func WriteAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-"+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

// FindRepo walks up from dir looking for this project's go.mod and returns the
// repository root, or an error when dir is not inside it.
func FindRepo(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && declaresModule(data, moduleName) {
			if st, err := os.Stat(filepath.Join(dir, filepath.FromSlash(RepoLevelDir))); err == nil && st.IsDir() {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("not inside the super-mario repository")
		}
		dir = parent
	}
}

func declaresModule(gomod []byte, name string) bool {
	for _, line := range strings.Split(string(gomod), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && f[0] == "module" && f[1] == name {
			return true
		}
	}
	return false
}
