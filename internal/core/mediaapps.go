package core

import (
	"path/filepath"
	"strings"
)

// Apps TV mode offers beside the games: Spotify, YouTube's TV interface and
// Stremio, when they're on the PC. Lootnest only starts what's installed;
// it never installs or adds anything to them.

// MediaApp is one app TV mode can start.
type MediaApp struct {
	ID   string `json:"id"`   // "spotify", "youtube" or "stremio"
	Name string `json:"name"` // as shown
	Exe  string `json:"-"`
	Args string `json:"-"`
	// Proc is the process name it's followed by (lower case, no .exe); an
	// app sharing its program with other windows (YouTube in Edge) is
	// followed by the process Lootnest started instead (ByPID).
	Proc  string `json:"-"`
	ByPID bool   `json:"-"`
}

// MediaPaths are the folders the apps are looked for in.
type MediaPaths struct {
	AppData, LocalAppData, ProgramFiles, ProgramFilesX86 string
	DataDir                                              string // Lootnest's own, for YouTube's browser profile
}

// tvUserAgent asks YouTube for its TV interface, which a controller drives.
const tvUserAgent = "Mozilla/5.0 (SMART-TV; Linux; Tizen 6.5) AppleWebKit/537.36 (KHTML, like Gecko) 85.0.4183.93/6.5 TV Safari/537.36"

// FindMediaApps lists the apps found, in the order they're shown. exists
// says whether a file is there.
func FindMediaApps(p MediaPaths, exists func(string) bool) []MediaApp {
	first := func(paths ...string) string {
		for _, x := range paths {
			if x != "" && exists(x) {
				return x
			}
		}
		return ""
	}
	j := func(base string, parts ...string) string {
		if base == "" {
			return ""
		}
		return filepath.Join(append([]string{base}, parts...)...)
	}
	var out []MediaApp
	if exe := first(j(p.AppData, "Spotify", "Spotify.exe"), j(p.LocalAppData, "Microsoft", "WindowsApps", "Spotify.exe")); exe != "" {
		out = append(out, MediaApp{ID: "spotify", Name: "Spotify", Exe: exe, Proc: "spotify"})
	}
	if browser := first(j(p.ProgramFilesX86, "Microsoft", "Edge", "Application", "msedge.exe"), j(p.ProgramFiles, "Microsoft", "Edge", "Application", "msedge.exe"),
		j(p.ProgramFiles, "Google", "Chrome", "Application", "chrome.exe"), j(p.LocalAppData, "Google", "Chrome", "Application", "chrome.exe")); browser != "" && p.DataDir != "" {
		args := strings.Join([]string{
			"--app=https://www.youtube.com/tv",
			"--start-fullscreen",
			"--no-first-run",
			`--user-data-dir="` + filepath.Join(p.DataDir, "youtube-tv") + `"`,
			`--user-agent="` + tvUserAgent + `"`,
		}, " ")
		out = append(out, MediaApp{ID: "youtube", Name: "YouTube", Exe: browser, Args: args, ByPID: true})
	}
	if exe := first(j(p.LocalAppData, "Programs", "LNV", "Stremio-4", "stremio.exe"), j(p.LocalAppData, "Programs", "LNV", "Stremio", "stremio.exe"),
		j(p.LocalAppData, "Programs", "Stremio", "stremio.exe"), j(p.LocalAppData, "Programs", "stremio-shell-ng", "stremio-shell-ng.exe"),
		j(p.ProgramFiles, "Stremio", "stremio.exe")); exe != "" {
		out = append(out, MediaApp{ID: "stremio", Name: "Stremio", Exe: exe, Proc: strings.ToLower(strings.TrimSuffix(filepath.Base(exe), ".exe"))})
	}
	return out
}

// MediaAppKey is an app's key among TV mode's running things.
func MediaAppKey(id string) string { return "app:" + id }
