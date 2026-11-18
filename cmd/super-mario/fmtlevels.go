package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"super-mario/pkg/level"
)

// fmtLevels rewrites every level file in dir in the canonical layout.
//
// With check set it writes nothing and reports the files that are not already
// formatted, for CI and for a contributor who wants to know before pushing.
// It returns how many files were (or, with check, would be) changed.
func fmtLevels(dir string, check bool) (int, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return 0, err
	}
	sort.Strings(paths)
	changed := 0
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return changed, err
		}
		out, err := level.Format(data)
		if err != nil {
			return changed, fmt.Errorf("%s: %w", p, err)
		}
		if bytes.Equal(data, out) {
			continue
		}
		changed++
		if check {
			fmt.Printf("排版不一致：%s\n", p)
			continue
		}
		if err := writeFileAtomic(p, out); err != nil {
			return changed, err
		}
		fmt.Printf("已规整：%s\n", p)
	}
	return changed, nil
}

// writeFileAtomic writes next to the target and renames over it, so a failed
// write never leaves a half-written level behind.
func writeFileAtomic(path string, data []byte) error {
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
