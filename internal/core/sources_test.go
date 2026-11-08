package core

import (
	"errors"
	"testing"
)

func TestSourceChecks(t *testing.T) {
	found := []FoundGame{{Source: "steam"}, {Source: "steam"}, {Source: "itch"}, {Source: "added"}}
	installed := map[string]string{"steam": `C:\Steam`, "gog": ""}
	errs := map[string]error{"hydra": errors.New("leveldb: manifest corrupted\nmore detail")}
	got := map[string]SourceCheck{}
	for _, c := range SourceChecks(found, installed, errs) {
		got[c.Source] = c
	}
	if len(got) != len(ScanSources) {
		t.Fatalf("got %d sources, want %d", len(got), len(ScanSources))
	}
	for src, want := range map[string][2]string{
		"steam":    {"ok", ""},
		"itch":     {"ok", ""},
		"gog":      {"info", "Installed, no games yet"},
		"epic":     {"info", "Not installed"},
		"hydra":    {"warn", "Couldn't read its library: leveldb: manifest corrupted"},
		"folder":   {"info", "No folders that look like games on this PC's drives"},
		"shortcut": {"info", "No shortcuts to games on the desktop or in the Start menu"},
	} {
		if c := got[src]; c.State != want[0] || c.Reason != want[1] {
			t.Errorf("%s: %s %q, want %s %q", src, c.State, c.Reason, want[0], want[1])
		}
	}
	if c := got["steam"]; !c.Installed || c.Path != `C:\Steam` || c.Found != 2 {
		t.Errorf("steam = %+v", c)
	}
	if !got["itch"].Installed || !got["folder"].Installed || got["epic"].Installed {
		t.Error("a source with games, and game folders, are on this PC; Epic isn't")
	}
}

func TestInstalledLaunchers(t *testing.T) {
	got := InstalledLaunchers([]UninstallEntry{
		{DisplayName: "Ubisoft Connect", InstallLocation: `"C:\Ubisoft"`},
		{DisplayName: "Battle.net (Beta)", InstallLocation: `C:\Battle.net`},
		{DisplayName: "EA app"},
		{DisplayName: "Riot Vanguard", InstallLocation: `C:\Riot Vanguard`},
		{DisplayName: "Diablo IV", InstallLocation: `C:\Diablo IV`},
	})
	want := map[string]string{"ubisoft": `C:\Ubisoft`, "battlenet": `C:\Battle.net`, "ea": ""}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for k, v := range want {
		if p, ok := got[k]; !ok || p != v {
			t.Errorf("%s = %q, %v", k, p, ok)
		}
	}
}
