package editor

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"time"

	"super-mario/pkg/catalog"
)

// Config is how the editor is started.
type Config struct {
	// Port is the port to listen on; 0 picks a free one.
	Port int
	// Catalog is where levels are read and written.
	Catalog *catalog.Catalog
	// Open is a level to open first, as a catalog key.
	Open string
	// Browser opens the page; nil uses the system's default browser.
	Browser func(url string) error
	// Out is where the address and messages are printed; nil prints nothing.
	Out io.Writer
}

// Listen binds the editor's address. It is always the loopback interface:
// the editor writes into the repository and must not be reachable from other
// machines (spec level-editor / 启动编辑器).
func Listen(port int) (net.Listener, error) {
	return net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
}

// Service is a running editor.
type Service struct {
	URL  string
	srv  *http.Server
	done chan error
	once sync.Once
}

// Start begins serving the editor in the background and returns at once.
func Start(c Config) (*Service, error) {
	ln, err := Listen(c.Port)
	if err != nil {
		return nil, err
	}
	addr := ln.Addr().String()
	s := &Server{Catalog: c.Catalog, Open: c.Open}
	svc := &Service{
		URL:  "http://" + addr + "/",
		srv:  &http.Server{Handler: s.Handler(addr), ReadHeaderTimeout: 10 * time.Second},
		done: make(chan error, 1),
	}
	go func() {
		err := svc.srv.Serve(ln)
		if err == http.ErrServerClosed {
			err = nil
		}
		svc.done <- err
	}()
	return svc, nil
}

// PageURL is the address to open, with the level to show first.
func (s *Service) PageURL(open string) string {
	if open == "" {
		return s.URL
	}
	return s.URL + "#" + open
}

// Stop shuts the editor down.
func (s *Service) Stop() error {
	var err error
	s.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		err = s.srv.Shutdown(ctx)
	})
	return err
}

// Run serves the editor until ctx is cancelled: the `edit` command.
func Run(ctx context.Context, c Config) error {
	if c.Out == nil {
		c.Out = io.Discard
	}
	svc, err := Start(c)
	if err != nil {
		return fmt.Errorf("无法监听端口 %d：%w", c.Port, err)
	}
	url := svc.PageURL(c.Open)

	fmt.Fprintf(c.Out, "关卡编辑器已启动：%s\n", url)
	if c.Catalog.InRepo() {
		fmt.Fprintf(c.Out, "内置关卡：%s\n", c.Catalog.BuiltinDir)
	}
	fmt.Fprintf(c.Out, "本地关卡：%s\n", c.Catalog.LocalDir)
	if !HasWasm() {
		fmt.Fprintln(c.Out, "注意：这个程序是用 go build 编出来的，缺少编辑器的 WebAssembly 模块。请运行 make build 重新编译。")
	}
	fmt.Fprintln(c.Out, "按 Ctrl+C 退出。")

	open := c.Browser
	if open == nil {
		open = OpenBrowser
	}
	if err := open(url); err != nil {
		fmt.Fprintf(c.Out, "没能自动打开浏览器（%v），请手动访问上面的地址。\n", err)
	}

	select {
	case <-ctx.Done():
		return svc.Stop()
	case err := <-svc.done:
		return err
	}
}

// OpenBrowser opens a URL in the system's default browser.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
