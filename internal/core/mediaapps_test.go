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
	}
	apps := FindMediaApps(p, func(x string) bool { return have[x] })
	if len(apps) != 3 || apps[0].ID != "spotify" || apps[1].ID != "youtube" || apps[2].ID != "stremio" {
		t.Fatalf("%+v", apps)
	}
	if apps[0].Proc != "spotify" || apps[2].Proc != "stremio" || !apps[1].ByPID || !apps[1].Own {
		t.Fatalf("following: %+v", apps)
	}
	// YouTube is Lootnest's own window, not a browser on the PC.
	if apps[1].Exe != p.Self || apps[1].Args != "--app youtube" {
		t.Fatalf("youtube %+v", apps[1])
	}
	w, ok := WebAppByID("youtube")
	if !ok || w.URL != "https://www.youtube.com/tv" || !strings.Contains(w.UserAgent, "Leanback") || !strings.HasPrefix(w.URL, w.Origin) {
		t.Fatalf("youtube site %+v", w)
	}
	if _, ok := WebAppByID("spotify"); ok {
		t.Fatal("spotify isn't a site")
	}
	if got := FindMediaApps(MediaPaths{AppData: "/u/roaming", LocalAppData: "/u/local"}, func(string) bool { return false }); len(got) != 0 {
		t.Fatalf("found %+v on an empty PC", got)
	}
	// The Store's Spotify, through its app alias.
	alias := filepath.Join("/u/local", "Microsoft", "WindowsApps", "Spotify.exe")
	if got := FindMediaApps(MediaPaths{LocalAppData: "/u/local"}, func(x string) bool { return x == alias }); len(got) != 1 || got[0].Exe != alias {
		t.Fatalf("store spotify %+v", got)
	}
}
