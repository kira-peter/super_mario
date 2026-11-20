package version

import "testing"

func TestParse(t *testing.T) {
	for _, c := range []struct {
		in                  string
		major, minor, patch int
		pre                 string
	}{
		{"0.1.0", 0, 1, 0, ""},
		{"1.0.0", 1, 0, 0, ""},
		{"12.34.56", 12, 34, 56, ""},
		{"1.2.3-rc.1", 1, 2, 3, "rc.1"},
		{"0.0.1-beta", 0, 0, 1, "beta"},
	} {
		p, err := Parse(c.in)
		if err != nil {
			t.Errorf("Parse(%q): %v", c.in, err)
			continue
		}
		if p.Major != c.major || p.Minor != c.minor || p.Patch != c.patch || p.Pre != c.pre {
			t.Errorf("Parse(%q) = %+v, want %d.%d.%d-%q", c.in, p, c.major, c.minor, c.patch, c.pre)
		}
		if got := p.String(); got != c.in {
			t.Errorf("Parse(%q).String() = %q", c.in, got)
		}
	}
}

func TestParseRejects(t *testing.T) {
	for _, in := range []string{
		"", "1", "1.2", "1.2.3.4", "v1.2.3", "1.2.x", "1.2.3-", "1.2.3-a b",
		"01.2.3", "1.02.3", "1.2.03", "-1.2.3", "1..2", "1.2.-3",
	} {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) accepted a version that is not x.y.z", in)
		}
	}
}

func TestBump(t *testing.T) {
	for _, c := range []struct{ in, kind, want string }{
		{"0.1.0", "patch", "0.1.1"},
		{"0.1.0", "minor", "0.2.0"},
		{"0.1.0", "major", "1.0.0"},
		{"1.2.3", "patch", "1.2.4"},
		{"1.2.3", "minor", "1.3.0"},
		{"1.2.3", "major", "2.0.0"},
		{"9.99.99", "patch", "9.99.100"},
		// A pre-release becomes the release it was working toward.
		{"1.0.0-rc.1", "patch", "1.0.1"},
	} {
		got, err := Bump(c.in, c.kind)
		if err != nil {
			t.Errorf("Bump(%q, %q): %v", c.in, c.kind, err)
			continue
		}
		if got != c.want {
			t.Errorf("Bump(%q, %q) = %q, want %q", c.in, c.kind, got, c.want)
		}
	}
	if _, err := Bump("1.0.0", "huge"); err == nil {
		t.Error("Bump accepted an unknown kind")
	}
}

// TestString checks the display form, and that a build without -ldflags still
// reports the default rather than an empty string.
func TestString(t *testing.T) {
	savedNumber, savedCommit := Number, Commit
	defer func() { Number, Commit = savedNumber, savedCommit }()

	Number, Commit = "0.1.0", ""
	if got := String(); got != "v0.1.0" {
		t.Errorf("String() = %q", got)
	}
	Number, Commit = "0.1.0", "a1b2c3d"
	if got := String(); got != "v0.1.0 (a1b2c3d)" {
		t.Errorf("String() = %q", got)
	}
	Number, Commit = "", ""
	if got := String(); got != "v"+Default {
		t.Errorf("an unset Number reported %q, want the default", got)
	}
}
