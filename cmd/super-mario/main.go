// Command super-mario runs the game in the current terminal.
//
// Subcommands:
//
//	super-mario edit [--port N] [--dir D] [level]   open the level editor in a browser
//	super-mario fmt-levels [-check] [dir]           rewrite level files in the canonical layout
//	super-mario version                             print the version and exit
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/app"
	"super-mario/pkg/catalog"
	"super-mario/pkg/editor"
	"super-mario/pkg/version"
)

// builtinLevelDir is where the compiled-in levels live, relative to the
// repository root.
const builtinLevelDir = catalog.RepoLevelDir

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "-version", "--version":
			fmt.Println(version.String())
			return
		case "fmt-levels":
			os.Exit(runFmtLevels(os.Args[2:]))
		case "edit":
			os.Exit(runEdit(os.Args[2:]))
		}
	}
	runGame()
}

func runEdit(args []string) int {
	fs := flag.NewFlagSet("edit", flag.ExitOnError)
	port := fs.Int("port", 0, "监听端口（默认随机挑一个空闲端口）")
	dir := fs.String("dir", "", "内置关卡目录（默认在仓库里自动找 pkg/level/data）")
	local := fs.String("local", "", "本地关卡目录（默认是程序旁的 levels/）")
	drafts := fs.String("draft-dir", "", "草稿目录（默认使用应用配置目录）")
	noBrowser := fs.Bool("no-browser", false, "启动编辑器服务，不自动打开浏览器")
	fs.Parse(args)

	cat := catalog.Default()
	if *dir != "" {
		if st, err := os.Stat(*dir); err != nil || !st.IsDir() {
			fmt.Fprintf(os.Stderr, "关卡目录不存在：%s\n", *dir)
			return 1
		}
		cat.BuiltinDir = *dir
	}
	if *local != "" {
		cat.LocalDir = *local
	}

	open := ""
	if fs.NArg() > 0 {
		open = strings.TrimSuffix(filepath.Base(fs.Arg(0)), ".json")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	err := editor.Run(ctx, editor.Config{
		Port:      *port,
		Catalog:   cat,
		Open:      open,
		Out:       os.Stdout,
		DraftDir:  *drafts,
		NoBrowser: *noBrowser,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func runFmtLevels(args []string) int {
	fs := flag.NewFlagSet("fmt-levels", flag.ExitOnError)
	check := fs.Bool("check", false, "只检查不写入；有未规整的文件时以非零状态退出")
	fs.Parse(args)
	dir := builtinLevelDir
	if fs.NArg() > 0 {
		dir = fs.Arg(0)
	}
	n, err := fmtLevels(dir, *check)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if *check && n > 0 {
		fmt.Fprintf(os.Stderr, "%d 个关卡文件排版不一致，运行 `go run ./cmd/super-mario fmt-levels` 修正\n", n)
		return 1
	}
	return 0
}

func runGame() {
	var (
		levelPath = flag.String("level", "", "关卡文件路径（默认使用内置 1-1）")
		themeName = flag.String("theme", "ink", "配色主题：ink 或 paper")
		scale     = flag.String("scale", "auto", "缩放：auto、wide（8 像素/瓦片）、mid（12）、native（16）")
		fps       = flag.Int("fps", 30, "渲染帧率（模拟固定为 60Hz）")
		noAirJump = flag.Bool("noairjump", false, "关闭空中二次跳跃（原版没有这个机制，本作默认开启）")
	)
	flag.Parse()

	opts := app.Options{
		LevelPath: *levelPath,
		ThemeName: *themeName,
		Scale:     *scale,
		FPS:       *fps,
		AirJump:   !*noAirJump,
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "无法初始化终端：%v\n", err)
		os.Exit(1)
	}
	if err := screen.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "无法初始化终端：%v\n", err)
		os.Exit(1)
	}
	screen.SetStyle(tcell.StyleDefault)

	a, err := app.New(screen, opts)
	if err != nil {
		screen.Fini()
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if err := a.Run(); err != nil {
		screen.Fini()
		fmt.Fprintf(os.Stderr, "运行出错：%v\n", err)
		os.Exit(1)
	}
}
