package editor

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"super-mario/pkg/catalog"
)

// TestListenIsLoopbackOnly: the editor's port is on 127.0.0.1 and nothing else.
func TestListenIsLoopbackOnly(t *testing.T) {
	ln, err := Listen(0)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	host, _, _ := net.SplitHostPort(ln.Addr().String())
	if host != "127.0.0.1" {
		t.Errorf("listening on %s, want 127.0.0.1", host)
	}
	// Another address of this machine does not reach it.
	addrs, _ := net.InterfaceAddrs()
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	for _, a := range addrs {
		ipn, ok := a.(*net.IPNet)
		if !ok || ipn.IP.IsLoopback() || ipn.IP.To4() == nil {
			continue
		}
		c, err := net.DialTimeout("tcp", net.JoinHostPort(ipn.IP.String(), port), 300*time.Millisecond)
		if err == nil {
			c.Close()
			t.Errorf("the editor answered on %s", ipn.IP)
		}
	}
}

// TestRunPrintsTheAddressAndServes starts the real server on a free port.
func TestRunPrintsTheAddressAndServes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var out bytes.Buffer
	opened := make(chan string, 1)
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, Config{Out: &out, Catalog: &catalog.Catalog{LocalDir: t.TempDir()}, Browser: func(u string) error { opened <- u; return nil }})
	}()
	var url string
	select {
	case url = <-opened:
	case <-time.After(5 * time.Second):
		t.Fatal("the browser was never opened")
	}
	if !strings.HasPrefix(url, "http://127.0.0.1:") {
		t.Errorf("url = %s", url)
	}
	if !strings.Contains(out.String(), url) {
		t.Errorf("the address was not printed:\n%s", out.String())
	}
	resp, err := http.Get(url + "api/info")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("info: %d", resp.StatusCode)
	}
	cancel()
	if err := <-done; err != nil {
		t.Errorf("shutdown: %v", err)
	}
}
