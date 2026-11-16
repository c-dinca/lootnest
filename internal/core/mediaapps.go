package core

import (
	"math"
	"net/url"
	"path/filepath"
	"strings"
)

// Apps TV mode offers beside the games: YouTube's TV interface and Spotify's
// web player, each a window of Lootnest's own (WebApp), and Spotify's own
// program and Stremio when they're on the PC. Nothing is ever installed or
// added to them.

// MediaApp is one app TV mode can start.
type MediaApp struct {
	ID   string `json:"id"`   // "youtube", "spotify", "discord", "spotify-app" or "stremio"
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
	// Nav is how a controller drives it: "keys" for a site made for a
	// remote, which reads arrows, Enter and Escape itself (YouTube's TV
	// interface); "spatial" for an ordinary site, where the D-pad moves a
	// focus of Lootnest's own to the nearest thing that way (Spotify).
	Nav string
	// Width is how many of the site's pixels the screen is across, so an
	// ordinary site is large enough to read from a sofa; 0 leaves it.
	Width int
	// Mic lets the site's own pages use the microphone without asking each
	// time: a voice chat's site (Discord).
	Mic bool
}

// WebAppZoom is how much a site is magnified to be width of its pixels
// across a screen of screenPx real ones at this DPI (96 is 100%), which
// Windows has magnified once already. Between 1 and 4.
func WebAppZoom(width, screenPx, dpi int) float64 {
	if width <= 0 || screenPx <= 0 {
		return 1
	}
	if dpi <= 0 {
		dpi = 96
	}
	z := float64(screenPx) / (float64(width) * float64(dpi) / 96)
	return math.Max(1, math.Min(4, z))
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
	{ID: "youtube", Name: "YouTube", URL: "https://www.youtube.com/tv", UserAgent: tvUserAgent, Host: "www.youtube.com", Nav: "keys"},
	// Spotify has no interface for TVs on the web (it serves the same page
	// to a TV's or a console's browser), so this is its web player,
	// magnified, with the focus Lootnest adds.
	{ID: "spotify", Name: "Spotify", URL: "https://open.spotify.com/", Host: "open.spotify.com", Nav: "spatial", Width: 1280},
	// Discord in a browser: servers, text and voice. Signing in can be the
	// QR code its page shows for the phone's app. What friends say comes
	// out wherever the PC's sound goes (the TV, in a stream); what you say
	// needs a microphone the PC can hear, since a stream carries none back.
	{ID: "discord", Name: "Discord", URL: "https://discord.com/app", Host: "discord.com", Nav: "spatial", Width: 1280, Mic: true},
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
	if p.Self != "" {
		for _, w := range webApps {
			out = append(out, MediaApp{ID: w.ID, Name: w.Name, Exe: p.Self, Args: WebAppArgs(w.ID), ByPID: true, Own: true})
		}
	}
	// Spotify's own program too, when it's installed, beside the web one.
	if exe := first(j(p.AppData, "Spotify", "Spotify.exe"), j(p.LocalAppData, "Microsoft", "WindowsApps", "Spotify.exe")); exe != "" {
		out = append(out, MediaApp{ID: "spotify-app", Name: "Spotify on this PC", Exe: exe, Proc: "spotify"})
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
