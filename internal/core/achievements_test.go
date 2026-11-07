package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseUnlocks(t *testing.T) {
	at := time.Unix(1762500000, 0).UTC()
	cases := []struct {
		name, format, body string
		want               map[string]time.Time
	}{
		{"json object", "json", `{"ACH_WIN":{"earned":true,"earned_time":1762500000},"ACH_LOSE":{"earned":false,"earned_time":0}}`, map[string]time.Time{"ACH_WIN": at}},
		{"json list", "json", `[{"name":"ACH_WIN","earned":true,"earned_time":1762500000}]`, map[string]time.Time{"ACH_WIN": at}},
		{"ini", "ini", "[SteamAchievements]\nCount=2\n\n[ACH_WIN]\nAchieved=1\nUnlockTime=1762500000\n[ACH_LOSE]\nAchieved=0\n", map[string]time.Time{"ACH_WIN": at}},
		{"ini true", "ini", "[ACH_WIN]\nachieved=true\ntimestamp=1762500000\n", map[string]time.Time{"ACH_WIN": at}},
		{"hex", "hex", "[Steam]\nACHCount=1\n[ACH_WIN]\nState = 01000000\nTime = A0C40D69\n", map[string]time.Time{"ACH_WIN": time.Unix(0x690DC4A0, 0).UTC()}},
		{"at", "at", "[Achievements]\nACH_WIN=1@0@0@1762500000\nACH_LOSE=0@0@0@0\n", map[string]time.Time{"ACH_WIN": at}},
		{"no time", "ini", "[ACH_WIN]\nAchieved=1\n", map[string]time.Time{"ACH_WIN": {}}},
	}
	for _, c := range cases {
		got := ParseUnlocks([]byte(c.body), c.format)
		if len(got) != len(c.want) {
			t.Errorf("%s: got %v", c.name, got)
			continue
		}
		for k, v := range c.want {
			if g, ok := got[k]; !ok || !g.Equal(v) {
				t.Errorf("%s: %s = %v, want %v", c.name, k, g, v)
			}
		}
	}
}

func TestPCUnlocksAndFolders(t *testing.T) {
	appdata := t.TempDir()
	env := &Env{Getenv: func(k string) string {
		if k == "APPDATA" {
			return appdata
		}
		return ""
	}}
	if u, _ := env.PCUnlocks(570); u != nil {
		t.Fatal("nothing saved yet")
	}
	dir := filepath.Join(appdata, "GSE Saves", "570")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "achievements.json"), []byte(`{"A":{"earned":true,"earned_time":1762500000}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	u, file := env.PCUnlocks(570)
	if len(u) != 1 || file != filepath.Join(dir, "achievements.json") {
		t.Fatalf("got %v %q", u, file)
	}
	found := false
	for _, f := range env.AchievementFolders(570) {
		found = found || f == dir
	}
	if !found {
		t.Fatalf("folders: %v", env.AchievementFolders(570))
	}
}

func TestFolderAppIDAndSchema(t *testing.T) {
	dir := t.TempDir()
	if FolderAppID(dir) != 0 || LocalSchema(dir) != nil {
		t.Fatal("empty folder")
	}
	ss := filepath.Join(dir, "bin", "steam_settings")
	if err := os.MkdirAll(ss, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(ss, "steam_appid.txt"), []byte("1091500\n"), 0o644)
	_ = os.WriteFile(filepath.Join(ss, "achievements.json"), []byte(`[{"name":"ACH_A","displayName":{"english":"The Fool"},"description":"Meet Jackie","hidden":"0"},{"name":"ACH_B","displayName":"Hidden one","hidden":1}]`), 0o644)
	if n := FolderAppID(dir); n != 1091500 {
		t.Fatalf("app id %d", n)
	}
	s := LocalSchema(dir)
	if len(s) != 2 || s[0].Name != "The Fool" || s[0].Description != "Meet Jackie" || s[0].Hidden || !s[1].Hidden {
		t.Fatalf("schema %+v", s)
	}
	dir2 := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir2, "steam_emu.ini"), []byte("[Settings]\nAppId=413150\n"), 0o644)
	if n := FolderAppID(dir2); n != 413150 {
		t.Fatalf("ini app id %d", n)
	}
}

func TestSteamCacheAndMerge(t *testing.T) {
	b := []byte(`[["achievements",{"data":{"nTotal":3,"nAchieved":1,
		"vecHighlight":[{"strID":"A","strName":"First","strDescription":"d","strImage":"abc.jpg","bAchieved":true,"rtUnlocked":1762500000,"flAchieved":12.5}],
		"vecUnachieved":[{"strID":"B","strName":"Second","bAchieved":false}]},"version":1}],["badge",{"data":{}}]]`)
	list, total, ok := ParseSteamCache(b, 730)
	if !ok || total != 3 || len(list) != 2 || list[0].Icon != "https://cdn.akamai.steamstatic.com/steamcommunity/public/images/apps/730/abc.jpg" || list[0].Percent != 12.5 {
		t.Fatalf("%+v %d %v", list, total, ok)
	}
	// A file on this PC decides what's unlocked; ones it has that the list
	// doesn't are added with a readable name.
	at := time.Unix(1762600000, 0).UTC()
	m, total, n := MergeAchievements(list, total, map[string]time.Time{"B": at, "ACH_KILL_TEN": {}})
	if total != 3 || n != 2 || m[0].ID != "B" || m[1].Name != "Kill ten" || m[2].Unlocked {
		t.Fatalf("%+v %d %d", m, total, n)
	}
}

// Each player keeps their own achievements: the folder a Steam-compatible
// tool keeps them in goes with their profile, like their saves.
func TestAchievementsSwapPerPlayer(t *testing.T) {
	f := newFixture(t)
	ach := filepath.Join(f.appdata, "GSE Saves", "730", "achievements.json")
	stats := filepath.Join(f.appdata, "GSE Saves", "730", "stats", "kills")
	write(t, f.optFile(), "sens=1.2")
	write(t, ach, `{"A":{"earned":true,"earned_time":1762500000}}`)
	write(t, stats, "41")
	// Cristian's files and achievements are on the PC; Mihai starts fresh.
	if err := f.eng.Activate(f.cfg, "cs2", "p2", ActivateOptions{FirstOwner: "p1", NewProfile: NewFresh}); err != nil {
		t.Fatal(err)
	}
	if read(t, ach) != "<missing>" || read(t, stats) != "<missing>" {
		t.Fatal("Mihai should start without Cristian's achievements")
	}
	g := f.cfg.Game("cs2")
	if u, ok := f.eng.ProfileUnlocks("p1", g); !ok || len(u) != 1 {
		t.Fatalf("Cristian's achievements are in his profile: %v %v", u, ok)
	}
	// Mihai unlocks B; going back to Cristian brings his own back.
	write(t, ach, `{"B":{"earned":true,"earned_time":1762600000}}`)
	if err := f.eng.Activate(f.cfg, "cs2", "p1", ActivateOptions{}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, ach); got != `{"A":{"earned":true,"earned_time":1762500000}}` || read(t, stats) != "41" {
		t.Fatalf("Cristian's achievements back: %s", got)
	}
	if u, ok := f.eng.ProfileUnlocks("p2", g); !ok || len(u) != 1 {
		t.Fatalf("Mihai's achievements saved: %v", u)
	}
	// Shared: the folder isn't touched.
	g.SharedAchievements = true
	if err := f.eng.Activate(f.cfg, "cs2", "p2", ActivateOptions{}); err != nil {
		t.Fatal(err)
	}
	if read(t, stats) != "41" {
		t.Fatal("shared achievements stay put")
	}
}

// When the game already swaps a folder inside the achievements folder (its
// saves), only the achievements file goes with the profile.
func TestAchievementsSwapFileWhenFolderOverlaps(t *testing.T) {
	f := newFixture(t)
	base := filepath.Join(f.appdata, "GSE Saves", "730")
	f.cfg.Games[0].Paths = []PathEntry{{ID: "saves", Path: "%APPDATA%/GSE Saves/730/remote"}}
	write(t, filepath.Join(base, "remote", "save.dat"), "c")
	write(t, filepath.Join(base, "achievements.json"), `{"A":{"earned":true}}`)
	if err := f.eng.Activate(f.cfg, "cs2", "p2", ActivateOptions{FirstOwner: "p1", NewProfile: NewFresh}); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(base, "achievements.json")) != "<missing>" {
		t.Fatal("the file goes with Cristian's profile")
	}
	if u, ok := f.eng.ProfileUnlocks("p1", f.cfg.Game("cs2")); !ok || len(u) != 1 {
		t.Fatalf("read from the profile: %v %v", u, ok)
	}
}

func TestSteamSchemaComplete(t *testing.T) {
	b := []byte(`{"game":{"gameName":"x","availableGameStats":{"achievements":[
		{"name":"A","displayName":"First","description":"d","hidden":0,"icon":"https://steamcdn-a.akamaihd.net/steamcommunity/public/images/apps/730/a.jpg"},
		{"name":"B","displayName":"Second","hidden":1,"icon":"https://evil.example/b.jpg"}]}}}`)
	s := ParseSteamSchema(b, 730)
	if len(s) != 2 || s[0].Name != "First" || s[0].Icon == "" || !s[1].Hidden || s[1].Icon != "" {
		t.Fatalf("%+v", s)
	}
	full := CompleteList([]Achievement{{ID: "B", Unlocked: true}}, s)
	if len(full) != 2 || full[0].ID != "A" || full[1].Name != "Second" || !full[1].Unlocked {
		t.Fatalf("%+v", full)
	}
}
