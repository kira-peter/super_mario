package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFmtLevelsLeavesFormattedFilesAlone(t *testing.T) {
	dir := t.TempDir()
	src, err := os.ReadFile(filepath.Join("..", "..", builtinLevelDir, "1-1.json"))
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "1-1.json")
	if err := os.WriteFile(p, src, 0o644); err != nil {
		t.Fatal(err)
	}

	// The repository copy is already formatted, so nothing changes.
	n, err := fmtLevels(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("fmt-levels changed %d already-formatted files", n)
	}
	after, _ := os.ReadFile(p)
	if string(after) != string(src) {
		t.Error("the file changed")
	}
}

func TestFmtLevelsCheckReportsWithoutWriting(t *testing.T) {
	dir := t.TempDir()
	messy := []byte(`{"name":"t","world":"1-1","width":4,"height":3,"timeLimit":400,
	"spawn":{"x":0,"y":1},"flag":{"x":3,"topY":0,"bottomY":1},
	"rows":["    ","    ","XXXX"],"contents":[],"entities":[]}`)
	p := filepath.Join(dir, "t.json")
	if err := os.WriteFile(p, messy, 0o644); err != nil {
		t.Fatal(err)
	}

	n, err := fmtLevels(dir, true)
	if err != nil || n != 1 {
		t.Fatalf("check = %d, %v; want 1 file reported", n, err)
	}
	if got, _ := os.ReadFile(p); string(got) != string(messy) {
		t.Error("-check wrote the file")
	}

	// Without -check it is rewritten, and a second run finds nothing to do.
	if n, err := fmtLevels(dir, false); err != nil || n != 1 {
		t.Fatalf("format = %d, %v", n, err)
	}
	if n, err := fmtLevels(dir, false); err != nil || n != 0 {
		t.Errorf("second run changed %d files (%v)", n, err)
	}
}
