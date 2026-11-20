.PHONY: build test run fmt vet clean gen-level fmt-levels wasm e2e

BIN := build/super-mario.exe
WEB := pkg/editor/web
GOROOT := $(shell go env GOROOT)

# The version comes from the VERSION file, the one thing a release edits; see
# VERSIONING.md. The commit is what git says, when git is available. Both are
# compiled in with -ldflags, so the binary knows what it is; a plain `go build`
# without the Makefile reports pkg/version.Default instead.
VERSION := $(shell cat VERSION 2>/dev/null)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null)
LDFLAGS := -X super-mario/pkg/version.Number=$(VERSION) -X super-mario/pkg/version.Commit=$(COMMIT)

# The editor runs the game's own Go code in the browser. Its WebAssembly build
# and the matching wasm_exec.js are generated, not committed, and embedded into
# the game binary, so build the wasm first.
build: wasm
	go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/super-mario

wasm:
	GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o $(WEB)/editor.wasm ./cmd/wasm
	cp "$(GOROOT)/lib/wasm/wasm_exec.js" $(WEB)/wasm_exec.js

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
	go run ./cmd/super-mario fmt-levels

# Rewrite every built-in level in the canonical layout.
fmt-levels:
	go run ./cmd/super-mario fmt-levels

# Drive the level editor in headless Chrome (needs Chrome and Node, no npm).
e2e: build
	sh tools/editor-e2e/run.sh
