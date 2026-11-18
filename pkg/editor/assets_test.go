package editor

import (
	"io/fs"
	"testing"
)

func TestWebAssetsAreEmbedded(t *testing.T) {
	if _, err := fs.Stat(Web(), ".keep"); err != nil {
		t.Fatalf("the web directory is not embedded: %v", err)
	}
	// HasWasm must answer without panicking whether or not make build ran.
	_ = HasWasm()
}
