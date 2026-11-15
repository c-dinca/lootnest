package core

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestFindMediaApps(t *testing.T) {
	p := MediaPaths{AppData: "/u/roaming", LocalAppData: "/u/local", ProgramFiles: "/pf", ProgramFilesX86: "/pf86", DataDir: "/u/lootnest"}
	have := map[string]bool{
		filepath.Join("/u/roaming", "Spotify", "Spotify.exe"):                    true,
		filepath.Join("/pf86", "Microsoft", "Edge", "Application", "msedge.exe"): true,
		filepath.Join("/u/local", "Programs", "LNV", "Stremio-4", "stremio.exe"): true,
	}
	apps := FindMediaApps(p, func(x string) bool { return have[x] })
	if len(apps) != 3 || apps[0].ID != "spotify" || apps[1].ID != "youtube" || apps[2].ID != "stremio" {
		t.Fatalf("%+v", apps)
	}
	if apps[0].Proc != "spotify" || apps[2].Proc != "stremio" || !apps[1].ByPID {
		t.Fatalf("following: %+v", apps)
	}
	if !strings.Contains(apps[1].Args, "youtube.com/tv") || !strings.Contains(apps[1].Args, filepath.Join("/u/lootnest", "youtube-tv")) {
		t.Fatalf("youtube args %q", apps[1].Args)
	}
	if got := FindMediaApps(p, func(string) bool { return false }); len(got) != 0 {
		t.Fatalf("found %+v on an empty PC", got)
	}
	// The Store's Spotify, through its app alias.
	alias := filepath.Join("/u/local", "Microsoft", "WindowsApps", "Spotify.exe")
	if got := FindMediaApps(p, func(x string) bool { return x == alias }); len(got) != 1 || got[0].Exe != alias {
		t.Fatalf("store spotify %+v", got)
	}
}
