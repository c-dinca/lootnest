package core

import (
	"net/url"
	"path/filepath"
	"strings"
)

// Apps TV mode offers beside the games: Spotify, YouTube's TV interface and
// Stremio. Spotify and Stremio are started when they're on the PC, and
// nothing is ever installed or added to them; YouTube is a window of
// Lootnest's own (WebApp).

// MediaApp is one app TV mode can start.
type MediaApp struct {
	ID   string `json:"id"`   // "spotify", "youtube" or "stremio"
	Name string `json:"name"` // as shown
	Exe  string `json:"-"`
	Args string `json:"-"`
	// Proc is the process name it's followed by (lower case, no .exe); an
	// app Lootnest shows in a window of its own (Own) is followed by the
	// process started for it instead (ByPID).
	Proc  string `json:"-"`
	ByPID bool   `json:"-"`
	Own   bool   `json:"-"`
}

// MediaPaths are the folders the apps are looked for in.
type MediaPaths struct {
	AppData, LocalAppData, ProgramFiles, ProgramFilesX86 string
	Self                                                 string // Lootnest's own program, which shows YouTube
}

// WebApp is a site Lootnest shows full screen in a window of its own, as
// an app: Lootnest's program started with --app <id>.
type WebApp struct {
	ID, Name, URL string
	// UserAgent is what the window says it is. YouTube gives its TV
	// interface (Leanback), which arrows, OK and Back drive and where
	// signing in is a code typed on a phone, only to a TV or a console;
	// anything else is sent to the desktop site.
	UserAgent string
	// Host is the one host whose pages may ask for a key press (the
	// controller's, see app/webapp.go).
	Host string
}

// Owns says whether a page at this address is the site's own: https, on
// its host exactly. Anything else (another site it went to, a frame, a
// look-alike host) may not ask Lootnest for anything.
func (w WebApp) Owns(source string) bool {
	u, err := url.Parse(source)
	return err == nil && u.Scheme == "https" && w.Host != "" && u.Host == w.Host
}

const tvUserAgent = "Mozilla/5.0 (PS4; Leanback Shell) Gecko/20100101 Firefox/65.0 LeanbackShell/01.00.01.75 Sony PS4/ (PS4, , no, CH)"

var webApps = []WebApp{
	{ID: "youtube", Name: "YouTube", URL: "https://www.youtube.com/tv", UserAgent: tvUserAgent, Host: "www.youtube.com"},
}

// WebAppByID is the site shown for --app <id>.
func WebAppByID(id string) (WebApp, bool) {
	for _, w := range webApps {
		if w.ID == id {
			return w, true
		}
	}
	return WebApp{}, false
}

// WebAppArgs are the arguments Lootnest's program shows that site with.
func WebAppArgs(id string) string { return "--app " + id }

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
	if p.Self != "" {
		out = append(out, MediaApp{ID: "youtube", Name: "YouTube", Exe: p.Self, Args: WebAppArgs("youtube"), ByPID: true, Own: true})
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
