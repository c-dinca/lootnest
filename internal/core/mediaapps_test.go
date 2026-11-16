package core

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestFindMediaApps(t *testing.T) {
	p := MediaPaths{AppData: "/u/roaming", LocalAppData: "/u/local", ProgramFiles: "/pf", ProgramFilesX86: "/pf86", Self: "/pf/Lootnest/Lootnest.exe"}
	have := map[string]bool{
		filepath.Join("/u/roaming", "Spotify", "Spotify.exe"):                    true,
		filepath.Join("/u/local", "Programs", "LNV", "Stremio-4", "stremio.exe"): true,
		filepath.Join("/u/local", "Programs", "Stremio", "stremio-shell-ng.exe"): true,
		filepath.Join("/u/local", "Programs", "Stremio", "stremio-runtime.exe"):  true,
		filepath.Join("/u/local", "Programs", "Stremio", "server.js"):            true,
		filepath.Join("/pf", "Stremio", "stremio-runtime.exe"):                   true, // no server.js beside it
	}
	apps := FindMediaApps(p, func(x string) bool { return have[x] })
	ids := ""
	for _, a := range apps {
		ids += a.ID + " "
	}
	if ids != "youtube spotify discord stremio " {
		t.Fatalf("apps: %s", ids)
	}
	if !apps[3].Own || apps[3].Args != "--app stremio" || !apps[0].ByPID || !apps[0].Own || !apps[1].Own || !apps[2].Own {
		t.Fatalf("following: %+v", apps)
	}
	// YouTube and Spotify are Lootnest's own windows, not a browser on the PC.
	if apps[0].Exe != p.Self || apps[0].Args != "--app youtube" || apps[1].Args != "--app spotify" {
		t.Fatalf("own apps %+v %+v", apps[0], apps[1])
	}
	w, ok := WebAppByID("youtube")
	if !ok || w.URL != "https://www.youtube.com/tv" || !strings.Contains(w.UserAgent, "Leanback") || !w.Owns(w.URL) {
		t.Fatalf("youtube site %+v", w)
	}
	// The pages that may ask for a key press: the site's own, as its TV
	// interface names them, and nothing that only looks like it.
	for src, want := range map[string]bool{
		"https://www.youtube.com/tv":                      true,
		"https://www.youtube.com/tv#/":                    true,
		"https://www.youtube.com/tv?env=x#/browse":        true,
		"http://www.youtube.com/tv":                       false,
		"https://www.youtube.com.evil.example/tv":         false,
		"https://evil.example/https://www.youtube.com/tv": false,
		"https://accounts.google.com/":                    false,
		"":                                                false,
		"about:blank":                                     false,
	} {
		if got := w.Owns(src); got != want {
			t.Errorf("Owns(%q) = %v, want %v", src, got, want)
		}
	}
	sp, ok := WebAppByID("spotify")
	if !ok || !sp.Owns("https://open.spotify.com/playlist/x") || sp.Owns("https://accounts.spotify.com/login") || sp.Nav != "spatial" || w.Nav != "keys" {
		t.Fatalf("spotify site %+v", sp)
	}
	// Only Discord gets the microphone, and only its own pages.
	dc, ok := WebAppByID("discord")
	if !ok || !dc.Mic || sp.Mic || w.Mic || !dc.Owns("https://discord.com/channels/@me") || dc.Owns("https://discord.com.evil.example/app") || dc.Owns("https://cdn.discordapp.com/x") {
		t.Fatalf("discord site %+v", dc)
	}
	// Stremio's web version, and only its own pages press keys.
	st, ok := WebAppByID("stremio")
	if !ok || st.Nav != "spatial" || st.Mic || !st.Owns("https://web.stremio.com/#/detail/movie/tt1") || st.Owns("https://www.stremio.com/") || st.Owns("http://web.stremio.com/") {
		t.Fatalf("stremio site %+v", st)
	}
	// The program's streaming server: its runtime and script, together.
	exists := func(x string) bool { return have[x] }
	if r, sc, ok := StremioServer(p, exists); !ok || r != filepath.Join("/u/local", "Programs", "Stremio", "stremio-runtime.exe") || sc != filepath.Join("/u/local", "Programs", "Stremio", "server.js") {
		t.Fatalf("server %q %q %v", r, sc, ok)
	}
	if _, _, ok := StremioServer(MediaPaths{ProgramFiles: "/pf"}, exists); ok {
		t.Fatal("a runtime without its script counts as a server")
	}
	if _, _, ok := StremioServer(MediaPaths{}, exists); ok {
		t.Fatal("a server on a PC with no folders")
	}
	for _, c := range []struct {
		width, px, dpi int
		want           float64
	}{{1280, 1920, 96, 1.5}, {1280, 3840, 192, 1.5}, {1280, 3840, 96, 3}, {1280, 1280, 96, 1}, {1280, 1024, 96, 1}, {0, 1920, 96, 1}, {1280, 7680, 96, 4}} {
		if got := WebAppZoom(c.width, c.px, c.dpi); got != c.want {
			t.Errorf("WebAppZoom(%d, %d, %d) = %v, want %v", c.width, c.px, c.dpi, got, c.want)
		}
	}
	if got := FindMediaApps(MediaPaths{AppData: "/u/roaming", LocalAppData: "/u/local"}, func(string) bool { return false }); len(got) != 0 {
		t.Fatalf("found %+v on an empty PC", got)
	}
	// Spotify's own program isn't an app here, however it was installed:
	// Apps has one Spotify, the web one.
	alias := filepath.Join("/u/local", "Microsoft", "WindowsApps", "Spotify.exe")
	if got := FindMediaApps(MediaPaths{LocalAppData: "/u/local"}, func(x string) bool { return x == alias }); len(got) != 0 {
		t.Fatalf("the Spotify program is listed: %+v", got)
	}
}
