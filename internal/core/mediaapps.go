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

// tvUserAgent asks YouTube for its TV interface (Leanback), which a
// controller drives and where signing in is a code typed on a phone.
// YouTube also reads client hints, which would still say Windows, so
// they're turned off for that browser profile (youTubeArgs).
const tvUserAgent = "Mozilla/5.0 (PS4; Leanback Shell) Gecko/20100101 Firefox/65.0 LeanbackShell/01.00.01.75 Sony PS4/ (PS4, , no, CH)"

// youTubeArgs start the browser as a TV app: full screen with no browser
// window around it (kiosk), in Lootnest's own profile for it, with that
// profile's extensions, sync and implicit sign-in off so nothing of the
// user's browser opens on the TV. The user's own browser isn't touched.
func youTubeArgs(browser, profile string) string {
	args := []string{
		"--kiosk", "https://www.youtube.com/tv",
		`--user-data-dir="` + profile + `"`,
		`--user-agent="` + tvUserAgent + `"`,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-extensions",
		"--disable-sync",
		"--disable-component-extensions-with-background-pages",
		"--autoplay-policy=no-user-gesture-required",
		"--disable-features=UserAgentClientHint,msImplicitSignin,msEdgeSidebarV2,msSyncPromo,Translate",
	}
	if strings.EqualFold(filepath.Base(browser), "msedge.exe") {
		args = append(args, "--edge-kiosk-type=fullscreen")
	}
	return strings.Join(args, " ")
}

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
		args := youTubeArgs(browser, filepath.Join(p.DataDir, "youtube-tv"))
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
