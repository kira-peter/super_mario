package catalog

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func oneOne(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "level", "data", "1-1.json"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func withTitle(t *testing.T, title string) []byte {
	return []byte(strings.Replace(string(oneOne(t)), `"timeLimit": 400,`, `"timeLimit": 400,
  "meta": {"title": "`+title+`"},`, 1))
}

// inRepo is a catalog as it is inside the repository: built-in levels on disk.
func inRepo(t *testing.T) (*Catalog, string, string) {
	t.Helper()
	b, l := t.TempDir(), filepath.Join(t.TempDir(), "levels")
	os.WriteFile(filepath.Join(b, "1-1.json"), oneOne(t), 0o644)
	return &Catalog{BuiltinDir: b, LocalDir: l}, b, l
}

// shipped is a catalog as it is on a friend's computer: only the program.
func shipped(t *testing.T) (*Catalog, string) {
	t.Helper()
	l := filepath.Join(t.TempDir(), "levels")
	return &Catalog{LocalDir: l, Embedded: fstest.MapFS{"1-1.json": {Data: oneOne(t)}}}, l
}

func TestNewLevelsGoToTheRepoInsideItAndLocalOutside(t *testing.T) {
	c, _, _ := inRepo(t)
	if c.NewLevelSource() != Builtin {
		t.Error("inside the repository a new level should go to pkg/level/data")
	}
	s, _ := shipped(t)
	if s.NewLevelSource() != Local {
		t.Error("with only the program a new level should go to levels/")
	}
}

func TestSameNameInBothSourcesListsBoth(t *testing.T) {
	c, _, local := inRepo(t)
	os.MkdirAll(local, 0o755)
	os.WriteFile(filepath.Join(local, "1-1.json"), withTitle(t, "my 1-1"), 0o644)
	list := c.List()
	if len(list) != 2 {
		t.Fatalf("list = %+v", list)
	}
	if list[0].Source != Builtin || list[1].Source != Local || list[0].ID != "1-1" || list[1].ID != "1-1" {
		t.Errorf("want built-in 1-1 then local 1-1, got %+v", list)
	}
	if list[0].Key() == list[1].Key() {
		t.Error("the two keep the same leaderboard key")
	}
}

func TestWriteGoesBackWhereItCameFrom(t *testing.T) {
	c, builtin, local := inRepo(t)
	path, err := c.Write(Local, "c-tom-sky", withTitle(t, "sky"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(path) != local {
		t.Errorf("wrote to %s, want the local folder %s", path, local)
	}
	if _, err := os.Stat(filepath.Join(builtin, "c-tom-sky.json")); err == nil {
		t.Error("a local save touched the built-in directory")
	}
	if !c.Exists(Local, "c-tom-sky") || c.Exists(Builtin, "c-tom-sky") {
		t.Error("Exists disagrees with where the file is")
	}
}

func TestLocalFolderIsCreatedOnFirstSave(t *testing.T) {
	c, local := shipped(t)
	if _, err := os.Stat(local); err == nil {
		t.Fatal("setup: the folder already exists")
	}
	if _, err := c.Write(Local, "c-amy-new", oneOne(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(local, "c-amy-new.json")); err != nil {
		t.Errorf("not written: %v", err)
	}
}

func TestBuiltinInsideTheProgramIsReadOnly(t *testing.T) {
	c, _ := shipped(t)
	if c.Writable(Builtin) {
		t.Error("compiled-in levels reported writable")
	}
	if _, err := c.Write(Builtin, "1-1", oneOne(t)); !errors.Is(err, ErrReadOnly) {
		t.Errorf("err = %v, want ErrReadOnly", err)
	}
	if _, err := c.Read(Builtin, "1-1"); err != nil {
		t.Errorf("compiled-in 1-1 cannot be read: %v", err)
	}
}

func TestWriteRefusesBadLevelsAndNames(t *testing.T) {
	c, _, local := inRepo(t)
	if _, err := c.Write(Local, "c-x-y", []byte(`{"name":"t"}`)); err == nil {
		t.Error("a broken level was written")
	}
	for _, bad := range []string{"../evil", "a/b", "UP", ""} {
		if _, err := c.Write(Local, bad, oneOne(t)); err == nil {
			t.Errorf("name %q accepted", bad)
		}
		if _, err := c.Read(Local, bad); err == nil {
			t.Errorf("read of %q accepted", bad)
		}
	}
	if entries, _ := os.ReadDir(filepath.Dir(local)); len(entries) > 1 {
		t.Errorf("stray files: %v", entries)
	}
}

func TestBrokenLocalFileIsMarkedNotFatal(t *testing.T) {
	c, _, local := inRepo(t)
	os.MkdirAll(local, 0o755)
	os.WriteFile(filepath.Join(local, "c-bad-file.json"), []byte("{not json"), 0o644)
	os.WriteFile(filepath.Join(local, "Not A Level.json"), oneOne(t), 0o644) // bad name: skipped
	var bad *Entry
	for _, e := range c.List() {
		if e.ID == "c-bad-file" {
			e := e
			bad = &e
		}
		if strings.Contains(e.ID, " ") {
			t.Error("a file with an unsafe name was listed")
		}
	}
	if bad == nil || bad.Err == "" {
		t.Errorf("broken file: %+v", bad)
	}
}

func TestKeysRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		src Source
		id  string
	}{{Builtin, "1-1"}, {Local, "1-1"}, {Local, "c-a-b"}} {
		s, id := ParseKey(Key(tc.src, tc.id))
		if s != tc.src || id != tc.id {
			t.Errorf("%v %s -> %v %s", tc.src, tc.id, s, id)
		}
	}
	// Old leaderboard entries name 1-1 plainly; they stay built-in.
	if s, id := ParseKey("1-1"); s != Builtin || id != "1-1" {
		t.Error("an old entry key changed meaning")
	}
}

func TestFindRepoNeedsTheLevelDirectory(t *testing.T) {
	if _, err := FindRepo("."); err != nil {
		t.Fatalf("from inside the repo: %v", err)
	}
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "go.mod"), []byte("module super-mario\n"), 0o644)
	if _, err := FindRepo(d); err == nil {
		t.Error("a go.mod without pkg/level/data was taken for the repository")
	}
}
