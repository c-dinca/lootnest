package app

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"lootnest/internal/core"
)

// Stremio's web version plays most things through Stremio's streaming
// server, which the Stremio program brings and runs beside its own window.
// Lootnest's Stremio window starts that same server from the installed
// program when it isn't running (as the program does: its runtime with
// server.js, no window), and ends it when the window closes. With no
// Stremio installed nothing is started, and the page says the server is
// missing; Lootnest never installs it.

// startStremioServer starts the installed Stremio's streaming server when
// nothing answers on its port. It returns what ends it again, or nil when
// there was nothing to start.
func startStremioServer() (stop func()) {
	addr := net.JoinHostPort("127.0.0.1", core.StremioServerPort)
	if c, err := net.DialTimeout("tcp", addr, 400*time.Millisecond); err == nil {
		c.Close()
		appLog("Stremio: its streaming server is already running")
		return nil // Stremio or its service is already running
	}
	exists := func(p string) bool { _, err := os.Stat(p); return err == nil }
	runtime, script, ok := core.StremioServer(core.MediaPaths{
		LocalAppData: os.Getenv("LOCALAPPDATA"), ProgramFiles: os.Getenv("ProgramFiles"), ProgramFilesX86: os.Getenv("ProgramFiles(x86)"),
	}, exists)
	if !ok {
		appLog("Stremio: no Stremio program on this PC, so no streaming server; videos need it (stremio.com/downloads)")
		return nil
	}
	cmd := exec.Command(runtime, script)
	cmd.Dir = filepath.Dir(runtime)
	noConsole(cmd)
	if err := cmd.Start(); err != nil {
		appLog("Stremio: its streaming server didn't start: %v", err)
		return nil
	}
	go func() {
		// Say whether it answers, for a report.
		for i := 0; i < 20; i++ {
			time.Sleep(500 * time.Millisecond)
			if c, err := net.DialTimeout("tcp", addr, 400*time.Millisecond); err == nil {
				c.Close()
				appLog("Stremio: started its streaming server (%s)", runtime)
				return
			}
		}
		appLog("Stremio: started %s, but nothing answers on %s after 10 s", runtime, addr)
	}()
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	return func() {
		_ = cmd.Process.Kill()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}
}

// appLog adds a line to Lootnest's log from a site's own process, which
// has no App: the same file, opened for each line.
func appLog(format string, args ...any) {
	f, err := os.OpenFile(filepath.Join(DataDir(), "lootnest.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] %s\r\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
}
