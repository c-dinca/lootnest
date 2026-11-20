//go:build windows

package win

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// ServeOutboundPipe makes a named pipe this process writes and one other
// process reads (the frame reader's, app/framereader.go). It refuses when
// the name is already taken, so nobody can sit on it first unnoticed, and
// takes no one from another computer. The default security lets the same
// account and administrators in, and everyone read.
func ServeOutboundPipe(name string) (*os.File, error) {
	n, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	const (
		pipeAccessOutbound     = 0x00000002
		fileFlagFirstInstance  = 0x00080000
		pipeTypeByte           = 0x00000000
		pipeWait               = 0x00000000
		pipeRejectRemoteClient = 0x00000008
	)
	h, err := windows.CreateNamedPipe(n, pipeAccessOutbound|fileFlagFirstInstance,
		pipeTypeByte|pipeWait|pipeRejectRemoteClient, 1, 64*1024, 0, 0, nil)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(h), name), nil
}

// WaitPipeReader blocks until a reader opens the pipe.
func WaitPipeReader(f *os.File) error {
	err := windows.ConnectNamedPipe(windows.Handle(f.Fd()), nil)
	if err == nil || errors.Is(err, windows.ERROR_PIPE_CONNECTED) {
		return nil
	}
	return err
}
