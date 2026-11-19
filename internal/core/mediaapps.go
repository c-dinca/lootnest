package core

import (
	"math"
	"net/url"
	"path/filepath"
)

// Apps TV mode offers beside the games: YouTube's TV interface and Spotify's
// web player, each a window of Lootnest's own (WebApp), and Spotify's own
// program and Stremio when they're on the PC. Nothing is ever installed or
// added to them.

// MediaApp is one app TV mode can start.
type MediaApp struct {
	ID   string `json:"id"`   // "youtube", "spotify", "discord" or "stremio"
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
	// Loopback is for a site whose page plays through a server on this PC
	// (Stremio's streaming server on 127.0.0.1): the window lets the page
	// reach it without asking (WebAppBrowserArgs).
	Loopback bool
}

// WebAppBrowserArgs are the browser's arguments for a Loopback site's own
// window. Chromium's Local Network Access asks before a site on the
// internet may reach 127.0.0.1, in a prompt a controller can't answer, and
// Stremio's page then says the streaming server isn't there. The checks
// are turned off in that window only (its own process and data folder),
// which shows nothing but that site.
const WebAppBrowserArgs = "--disable-features=LocalNetworkAccessChecks,PrivateNetworkAccessSendPreflights,PrivateNetworkAccessRespectPreflightResults,BlockInsecurePrivateNetworkRequests"

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
	// Stremio's own web version, which is laid out for a TV already, with
	// the streaming server of the Stremio program on this PC behind it
	// (StremioServer), so what the program plays, this plays.
	{ID: "stremio", Name: "Stremio", URL: "https://web.stremio.com/", Host: "web.stremio.com", Nav: "spatial", Width: 1280, Loopback: true},
}

// StremioServerPort is where Stremio's streaming server listens, and where
// its web version looks for it.
const StremioServerPort = "11470"

// StremioServer finds the streaming server inside a Stremio installed on
// this PC: its runtime and the script it runs, which is how Stremio's own
// program starts it (stremio-shell-ng, server.rs). ok is false when no
// Stremio with a server is installed. Lootnest never installs it.
func StremioServer(p MediaPaths, exists func(string) bool) (runtime, script string, ok bool) {
	for _, dir := range stremioDirs(p) {
		r, s := filepath.Join(dir, "stremio-runtime.exe"), filepath.Join(dir, "server.js")
		if exists(r) && exists(s) {
			return r, s, true
		}
	}
	return "", "", false
}

// stremioDirs are the folders Stremio installs into: Stremio 5's
// installer (for one user, or for everyone), then Stremio 4's.
func stremioDirs(p MediaPaths) []string {
	var out []string
	add := func(base string, parts ...string) {
		if base != "" {
			out = append(out, filepath.Join(append([]string{base}, parts...)...))
		}
	}
	add(p.LocalAppData, "Programs", "Stremio")
	add(p.ProgramFiles, "Stremio")
	add(p.LocalAppData, "Programs", "LNV", "Stremio-5")
	add(p.LocalAppData, "Programs", "LNV", "Stremio-4")
	add(p.LocalAppData, "Programs", "LNV", "Stremio")
	add(p.LocalAppData, "Programs", "stremio-shell-ng")
	add(p.ProgramFilesX86, "Stremio")
	return out
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

// FindMediaApps lists the apps, in the order they're shown: each is a
// site in a window of Lootnest's own. One tile each (the owner, 16 Nov
// 2026): Spotify's and Stremio's own programs aren't listed beside their
// web versions; Stremio's program still does its part, as the streaming
// server behind the web one (StremioServer). exists is kept for programs
// that may be listed again.
func FindMediaApps(p MediaPaths, exists func(string) bool) []MediaApp {
	_ = exists
	var out []MediaApp
	if p.Self != "" {
		for _, w := range webApps {
			out = append(out, MediaApp{ID: w.ID, Name: w.Name, Exe: p.Self, Args: WebAppArgs(w.ID), ByPID: true, Own: true})
		}
	}
	return out
}

// MediaAppKey is an app's key among TV mode's running things.
func MediaAppKey(id string) string { return "app:" + id }
