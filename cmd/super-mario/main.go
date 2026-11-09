// Command super-mario runs the game in the current terminal.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"

	"super-mario/pkg/app"
)

func main() {
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
