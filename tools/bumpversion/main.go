// Command bumpversion prints, or with -w writes, the next version number.
//
// The number lives in VERSION at the repository root; see VERSIONING.md. This
// tool exists so nobody has to add the numbers in their head, and -w also
// updates pkg/version/version.go's Default, which must match VERSION.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"super-mario/pkg/version"
)

func main() {
	write := flag.Bool("w", false, "write the new version into VERSION and pkg/version/version.go")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: bumpversion patch|minor|major [-w]\n")
	}
	// Accept -w before or after the kind: Go's flag package stops at the first
	// non-flag argument, so "minor -w" would otherwise ignore the -w.
	var args []string
	for _, a := range os.Args[1:] {
		if a == "-w" || a == "--w" {
			*write = true
			continue
		}
		args = append(args, a)
	}
	flag.CommandLine.Parse(args)
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	kind := flag.Arg(0)

	root, err := repoRoot()
	if err != nil {
		fail(err)
	}
	versionFile := filepath.Join(root, "VERSION")
	cur, err := os.ReadFile(versionFile)
	if err != nil {
		fail(err)
	}
	next, err := version.Bump(strings.TrimSpace(string(cur)), kind)
	if err != nil {
		fail(err)
	}
	fmt.Println(next)
	if !*write {
		return
	}

	if err := os.WriteFile(versionFile, []byte(next+"\n"), 0o644); err != nil {
		fail(err)
	}
	goFile := filepath.Join(root, "pkg", "version", "version.go")
	src, err := os.ReadFile(goFile)
	if err != nil {
		fail(err)
	}
	re := regexp.MustCompile(`(?m)^const Default = ".*"$`)
	if !re.Match(src) {
		fail(fmt.Errorf("%s has no `const Default = \"...\"` line to update", goFile))
	}
	out := re.ReplaceAll(src, []byte("const Default = \""+next+"\""))
	if err := os.WriteFile(goFile, out, 0o644); err != nil {
		fail(err)
	}
}

// repoRoot walks up from the working directory to the directory holding VERSION.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "VERSION")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no VERSION file above %s", dir)
		}
		dir = parent
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
