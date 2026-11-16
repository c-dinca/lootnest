//go:build windows

package app

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
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
		return nil // Stremio or its service is already running
	}
	exists := func(p string) bool { _, err := os.Stat(p); return err == nil }
	runtime, script, ok := core.StremioServer(core.MediaPaths{
		LocalAppData: os.Getenv("LOCALAPPDATA"), ProgramFiles: os.Getenv("ProgramFiles"), ProgramFilesX86: os.Getenv("ProgramFiles(x86)"),
	}, exists)
	if !ok {
		return nil
	}
	cmd := exec.Command(runtime, script)
	cmd.Dir = filepath.Dir(runtime)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000 /*CREATE_NO_WINDOW*/}
	if err := cmd.Start(); err != nil {
		return nil
	}
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
