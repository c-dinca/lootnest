//go:build !windows

package app

import "os/exec"

// noConsole: a program started here opens no console window anyway.
func noConsole(cmd *exec.Cmd) {}
