package app

import (
	"os"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// systemEnv is where things are on this Windows PC.
func systemEnv(a *App) *core.Env {
	steamPath, _ := plat.SteamInfo()
	home := os.Getenv("USERPROFILE")
	extra := []string{plat.ProgramFiles(), plat.ProgramFilesX86(), plat.WindowsDir(), plat.Desktop()}
	if sd := os.Getenv("SystemDrive"); sd != "" {
		extra = append(extra, sd+`\Users`, sd+`\ProgramData`)
	}
	return &core.Env{
		Getenv:     os.Getenv,
		Home:       home,
		Documents:  plat.Documents(),
		SavedGames: plat.SavedGames(),
		LocalLow:   plat.LocalLow(),
		SteamPath:  steamPath,
		SteamUser: func() (string, error) {
			_, active := plat.SteamInfo()
			return core.SteamUserDir(steamPath, a.steamAccount(), active)
		},
		DataDir:        DataDir(),
		ExtraProtected: extra,
	}
}

// systemDataDir: on Windows DataDir is under %LOCALAPPDATA%.
func systemDataDir() string { return "" }
