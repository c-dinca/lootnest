//go:build windows

package win

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

// RunHiddenOutput runs a console program without a window and returns what
// it printed, stopping it after limit.
func RunHiddenOutput(exe, args string, limit time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe)
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: syscall.EscapeArg(exe) + " " + args, HideWindow: true, CreationFlags: 0x08000000 /*CREATE_NO_WINDOW*/}
	out, err := cmd.Output()
	return string(out), err
}
