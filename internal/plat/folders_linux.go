package plat

import (
	"os"

	"lootnest/internal/core"
)

// The user's folders on Linux, from xdg-user-dirs (core.XDGUserDir).

func home() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return os.Getenv("HOME")
}

func userDir(name string) string { return core.XDGUserDir(home(), os.Getenv, name) }

func Desktop() string   { return userDir("DESKTOP") }
func Documents() string { return userDir("DOCUMENTS") }
func Downloads() string { return userDir("DOWNLOAD") }
func Pictures() string  { return userDir("PICTURES") }
func Videos() string    { return userDir("VIDEOS") }

// SavedGames and LocalLow are Windows' folders: a Windows game's are in
// its Wine prefix (core.Env.InPrefix).
func SavedGames() string { return "" }
func LocalLow() string   { return "" }

// SteamInfo is Steam's folder; Linux has no active-user value to read, so
// the account comes from loginusers.vdf (core.SteamUserDir).
func SteamInfo() (string, uint32) { return core.FindSteamLinux(home()), 0 }
