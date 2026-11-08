//go:build !windows

package core

import (
	"io/fs"
	"time"
)

func created(fs.FileInfo) time.Time { return time.Time{} }
