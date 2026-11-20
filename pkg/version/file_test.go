package version

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVERSIONFile: the VERSION file at the repository root must hold a version
// the parser accepts, and it must be the one the default build reports. The
// Makefile injects it, so a plain `go build` and a `make build` agree only if
// these two stay in step.
func TestVERSIONFile(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "VERSION"))
	if err != nil {
		t.Skip("VERSION file not found")
	}
	got := strings.TrimSpace(string(data))
	if _, err := Parse(got); err != nil {
		t.Fatalf("VERSION holds %q: %v", got, err)
	}
	if got != Default {
		t.Errorf("VERSION holds %q but pkg/version.Default is %q; update both together", got, Default)
	}
}
