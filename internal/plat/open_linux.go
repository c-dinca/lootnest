package plat

import (
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// ShellOpen opens a link, file or folder with the desktop's choice
// (xdg-open), without waiting for it.
func ShellOpen(target string) error {
	if strings.TrimSpace(target) == "" {
		return errors.New("nothing to open")
	}
	return startQuiet("xdg-open", target)
}

// RevealInExplorer shows a file or folder in the file manager, selected
// when the file manager can (org.freedesktop.FileManager1), else opens its
// folder.
func RevealInExplorer(p string) error {
	abs, err := filepath.Abs(p)
	if err != nil {
		return err
	}
	u := (&url.URL{Scheme: "file", Path: abs}).String()
	cmd := exec.Command("dbus-send", "--session", "--print-reply", "--dest=org.freedesktop.FileManager1",
		"--type=method_call", "/org/freedesktop/FileManager1", "org.freedesktop.FileManager1.ShowItems",
		"array:string:"+u, "string:")
	if cmd.Run() == nil {
		return nil
	}
	dir := abs
	if st, err := os.Stat(abs); err == nil && !st.IsDir() {
		dir = filepath.Dir(abs)
	}
	return ShellOpen(dir)
}

// StartDetached starts a program that outlives Lootnest, in its own
// session. args is split on spaces (Lootnest's own arguments have none).
func StartDetached(exe, args, dir string) error {
	cmd := exec.Command(exe, strings.Fields(args)...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// startQuiet starts a helper program without its output, and reaps it.
func startQuiet(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
