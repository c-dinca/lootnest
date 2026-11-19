package app

import (
	"os"
	"path/filepath"

	"lootnest/internal/core"
)

// systemEnv is where things are on this Linux PC (core.NewLinuxEnv):
// Windows games' paths are read in their Wine prefixes.
func systemEnv(a *App) *core.Env {
	e := core.NewLinuxEnv(homeDir(), os.Getenv, DataDir())
	steam := e.SteamPath
	e.SteamUser = func() (string, error) { return core.SteamUserDir(steam, a.steamAccount(), 0) }
	return e
}

func homeDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return os.Getenv("HOME")
}

// systemDataDir is Lootnest's data folder on Linux, $XDG_DATA_HOME/lootnest
// (~/.local/share/lootnest).
func systemDataDir() string {
	return filepath.Join(core.XDGDataHome(homeDir(), os.Getenv), "lootnest")
}

// webkitDir is where WebKit keeps the window's own storage and cache.
func webkitDir() string { return filepath.Join(DataDir(), "WebKit") }
