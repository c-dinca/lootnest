//go:build !windows && !linux

package app

import (
	"errors"

	"lootnest/internal/plat"
)

func (a *App) steamPlay(appID int, args string) error {
	return errors.New("Starting Steam games isn't done on this system")
}

func (a *App) runProgram(key, exe, args, dir string, admin bool) error {
	return plat.StartDetached(exe, args, dir)
}
