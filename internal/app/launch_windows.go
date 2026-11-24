package app

import (
	"fmt"
	"strings"

	"lootnest/internal/plat"
)

// steamPlay starts a Steam game through steam.exe, with its arguments.
func (a *App) steamPlay(appID int, args string) error {
	return plat.StartDetached(a.env.SteamPath+`\steam.exe`, strings.TrimSpace(fmt.Sprintf("-applaunch %d %s", appID, args)), a.env.SteamPath)
}

// runProgram starts a program for the game key (its exe, a Start with
// program, or one of its launch options' programs), as administrator when
// asked.
func (a *App) runProgram(key, exe, args, dir string, admin bool) error {
	if admin {
		return plat.StartElevated(a.hwnd, exe, args, dir)
	}
	return plat.StartDetached(exe, args, dir)
}
