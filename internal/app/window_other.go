//go:build !windows

package app

import "lootnest/internal/plat"

// openWindow has no window to make yet on this system: docs/ROADMAP.md,
// section 9, phase L2.
func (a *App) openWindow(opt Options, dpi int, scale func(int) int) error {
	return plat.ErrUnsupported
}
