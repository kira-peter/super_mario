.PHONY: build test run fmt vet clean gen-level

BIN := build/super-mario.exe

build:
	go build -o $(BIN) ./cmd/super-mario

test:
	go test ./...

run: build
	./$(BIN)

fmt:
	gofmt -l -w .

vet:
	go vet ./...

clean:
	rm -rf build

# Regenerate the level JSON from the authoring script, then refresh the
# copy that gets compiled into the binary.
gen-level:
	python tools/genlevel.py
	cp levels/1-1.json pkg/level/data/1-1.json
