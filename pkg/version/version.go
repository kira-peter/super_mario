// Package version is the one place the game's version number lives.
//
// The number itself is in the file VERSION at the repository root, which is
// the only thing a release edits. The build copies it in here with
// -ldflags "-X super-mario/pkg/version.Number=..." (see the Makefile); a plain
// `go run` or `go build` without the Makefile falls back to the default below,
// so the game always has a version to print either way.
//
// The numbering rule is in VERSIONING.md: x.y.z, where a major bump means
// players' old saves or level files stop working, a minor bump means new
// things they can use, and a patch bump means nothing they wrote breaks.
package version

import (
	"fmt"
	"strconv"
	"strings"
)

// Default is the version a build without -ldflags reports. It must match the
// VERSION file; TestVERSIONFile keeps the two in step.
const Default = "1.0.0"

// Number is the version this binary reports: "x.y.z", with an optional
// "-suffix" for a pre-release. The Makefile overrides it from the VERSION file.
var Number = Default

// Commit is the git commit the binary was built from, short form. The Makefile
// sets it; a build without the Makefile leaves it empty.
var Commit = ""

// String renders the version for display: "v1.0.0" and, when the build knows
// it, the commit in brackets: "v1.0.0 (a1b2c3d)".
func String() string {
	n := Number
	if n == "" {
		n = Default
	}
	s := "v" + n
	if Commit != "" {
		s += " (" + Commit + ")"
	}
	return s
}

// Parsed is a version number broken into its parts.
type Parsed struct {
	Major, Minor, Patch int
	// Pre is the "-rc.1" part, empty for a release.
	Pre string
}

// String puts a Parsed back into the form the VERSION file uses.
func (p Parsed) String() string {
	s := fmt.Sprintf("%d.%d.%d", p.Major, p.Minor, p.Patch)
	if p.Pre != "" {
		s += "-" + p.Pre
	}
	return s
}

// Parse reads "x.y.z" or "x.y.z-suffix". Anything else is an error, so a
// mistyped VERSION file stops the build rather than shipping.
func Parse(s string) (Parsed, error) {
	var p Parsed
	if s == "" {
		return p, fmt.Errorf("empty version")
	}
	core, pre, hasPre := strings.Cut(s, "-")
	if hasPre {
		if pre == "" {
			return p, fmt.Errorf("version %q: empty pre-release suffix", s)
		}
		if !validSuffix(pre) {
			return p, fmt.Errorf("version %q: pre-release suffix must be letters, digits and dots", s)
		}
		p.Pre = pre
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return p, fmt.Errorf("version %q: want x.y.z", s)
	}
	out := make([]int, 3)
	for i, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return p, fmt.Errorf("version %q: %q is not a plain number (no leading zeros)", s, part)
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return p, fmt.Errorf("version %q: %q is not a non-negative number", s, part)
		}
		out[i] = n
	}
	p.Major, p.Minor, p.Patch = out[0], out[1], out[2]
	return p, nil
}

func validSuffix(s string) bool {
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.':
		default:
			return false
		}
	}
	return true
}

// Bump raises a version by one step of the numbering rule: patch for a fix,
// minor for a new feature (which also resets patch), major for a breaking
// change (which resets minor and patch). A pre-release suffix is dropped: the
// release it was working toward is the one being cut.
func Bump(s, kind string) (string, error) {
	p, err := Parse(s)
	if err != nil {
		return "", err
	}
	p.Pre = ""
	switch kind {
	case "patch":
		p.Patch++
	case "minor":
		p.Minor++
		p.Patch = 0
	case "major":
		p.Major++
		p.Minor, p.Patch = 0, 0
	default:
		return "", fmt.Errorf("bump %q: want patch, minor or major", kind)
	}
	return p.String(), nil
}
