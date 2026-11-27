.PHONY: build test run fmt vet clean gen-level fmt-levels wasm e2e

BIN := build/super-mario.exe
WEB := pkg/editor/web

# The editor runs the game's own Go code in the browser. Its WebAssembly build
# and the matching wasm_exec.js are generated, not committed, and embedded into
# the game binary, so build the wasm first.
build:
	go run ./tools/build

wasm:
	go run ./tools/build -wasm-only

test:
	go test ./...
	node --test $(WEB)/model.test.mjs

run: build
	./$(BIN)

fmt:
	gofmt -l -w .

vet:
	go vet ./...

clean:
	rm -rf build $(WEB)/editor.wasm $(WEB)/wasm_exec.js

# Regenerate 1-1 from the authoring script. It writes straight into the one
# copy the game compiles in, and fmt-levels puts it in the canonical layout,
# so running this on an unchanged script leaves git with nothing to show.
gen-level:
	python tools/genlevel.py
	python tools/genworlds.py
	go run ./cmd/super-mario fmt-levels

# Rewrite every built-in level in the canonical layout.
fmt-levels:
	go run ./cmd/super-mario fmt-levels

# Drive the level editor in headless Chrome (needs Chrome and Node, no npm).
e2e: build
	node tools/editor-e2e/run.mjs
