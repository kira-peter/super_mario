// Command build compiles the browser module and embeds it in the game binary.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"super-mario/pkg/version"
)

func main() {
	wasmOnly := flag.Bool("wasm-only", false, "build only the editor's browser assets")
	flag.Parse()
	if err := build(*wasmOnly); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func build(wasmOnly bool) error {
	web := filepath.Join("pkg", "editor", "web")
	runtimeScript, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "lib", "wasm", "wasm_exec.js"))
	if err != nil {
		return err
	}
	wasm := exec.Command("go", "build", "-ldflags=-s -w", "-o", filepath.Join(web, "editor.wasm"), "./cmd/wasm")
	wasm.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	if err := run(wasm); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(web, "wasm_exec.js"), runtimeScript, 0o644); err != nil {
		return err
	}
	if wasmOnly {
		return nil
	}

	rawVersion, err := os.ReadFile("VERSION")
	if err != nil {
		return err
	}
	number := strings.TrimSpace(string(rawVersion))
	if _, err := version.Parse(number); err != nil {
		return fmt.Errorf("VERSION: %w", err)
	}
	commit, _ := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	ldflags := "-X super-mario/pkg/version.Number=" + number +
		" -X super-mario/pkg/version.Commit=" + strings.TrimSpace(string(commit))
	return run(exec.Command("go", "build", "-ldflags", ldflags, "-o", filepath.Join("build", "super-mario.exe"), "./cmd/super-mario"))
}

func run(cmd *exec.Cmd) error {
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", cmd, err)
	}
	return nil
}
