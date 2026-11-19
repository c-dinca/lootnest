//go:build !windows && !linux

package app

import "lootnest/internal/plat"

// openWindow has no window to make on this system.
func (a *App) openWindow(opt Options, dpi int, scale func(int) int) error {
	return plat.ErrUnsupported
}

func (a *App) present()     {}
func (a *App) closeWindow() {}

type platWindow struct{}

func (a *App) trayReady() {}
