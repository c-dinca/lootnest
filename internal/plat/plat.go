// Package plat is Lootnest's desktop layer: what the app asks of the system
// it runs on (windows, the tray, files, processes, controllers, the
// network's settings).
//
// On Windows every name forwards to internal/win (win_windows.go). Other
// systems get the same names from stub_other.go, which do nothing, until a
// hand-written file here gives one its real code; tools/platgen writes both
// generated files and leaves out what such a file declares. The plan is
// docs/ROADMAP.md, section 9.
package plat

import (
	"errors"
	"fmt"
	"runtime"
)

// ErrUnsupported is what a function answers on a system where Lootnest
// can't do it yet. errors.Is matches it with errors.ErrUnsupported too.
var ErrUnsupported = fmt.Errorf("not available on %s yet: %w", runtime.GOOS, errors.ErrUnsupported)
