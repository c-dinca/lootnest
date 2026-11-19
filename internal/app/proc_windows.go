package app

import (
	"os/exec"
	"syscall"
)

// noConsole starts cmd with no console window.
func noConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000 /*CREATE_NO_WINDOW*/}
}
