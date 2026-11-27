// Package editor serves the level editor: a small web server on the loopback
// interface that hands out the editor page and reads and writes level files
// (spec level-editor, design decision 1).
package editor

import (
	"embed"
	"io/fs"
)

// web holds the editor's page, scripts and styles, plus the generated
// editor.wasm and wasm_exec.js when `make build` has made them. The .keep file
// is committed so the pattern always matches and a plain `go build` works on a
// fresh clone; HasWasm tells the server whether the real module is there.
//
//go:embed all:web
var web embed.FS

// Web is the editor's static files, rooted at the web directory.
func Web() fs.FS {
	sub, err := fs.Sub(web, "web")
	if err != nil {
		panic(err) // the directory is embedded; this cannot fail
	}
	return sub
}

// HasWasm reports whether this binary was built with the editor's WebAssembly
// module. A binary from a plain `go build`, without `make build`, lacks it.
func HasWasm() bool {
	for _, name := range []string{"editor.wasm", "wasm_exec.js"} {
		if st, err := fs.Stat(Web(), name); err != nil || st.Size() == 0 {
			return false
		}
	}
	return true
}
