package core

import (
	"io/fs"
	"syscall"
	"time"
)

// created is when a file or folder was made on this PC. Extracting an
// archive keeps its files' times but not this.
func created(info fs.FileInfo) time.Time {
	if d, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		return time.Unix(0, d.CreationTime.Nanoseconds())
	}
	return time.Time{}
}
