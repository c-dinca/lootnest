//go:build !windows

package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// linuxFixture is a Linux home with Steam in ~/.local/share/Steam (and the
// ~/.steam/steam link to it) and one Windows game run through Proton.
type linuxFixture struct {
	root, home, steam, pfx string
	env                    *Env
}

const protonID = 1245620

func newLinuxFixture(t *testing.T) *linuxFixture {
	t.Helper()
	root := t.TempDir()
	f := &linuxFixture{root: root, home: filepath.Join(root, "home", "alex")}
	f.steam = filepath.Join(f.home, ".local", "share", "Steam")
	apps := filepath.Join(f.steam, "steamapps")
	write(t, filepath.Join(apps, "appmanifest_1245620.acf"), `"AppState" { "appid" "1245620" "name" "Ashen Ring" "installdir" "Ashen Ring" }`)
	write(t, filepath.Join(apps, "common", "Ashen Ring", "Game", "ashenring.exe"), "MZ")
	f.pfx = filepath.Join(apps, "compatdata", "1245620", "pfx")
	write(t, filepath.Join(f.pfx, "system.reg"), "WINE REGISTRY Version 2")
	_ = os.MkdirAll(filepath.Join(f.pfx, "drive_c", "windows"), 0o755)
	_ = os.MkdirAll(filepath.Join(f.pfx, "drive_c", "users", "Public"), 0o755)
	write(t, filepath.Join(f.pfx, "drive_c", "users", "steamuser", "AppData", "Roaming", "AshenRing", "slot0.sav"), "save")
	_ = os.MkdirAll(filepath.Join(f.steam, "userdata", "4242"), 0o755)
	if err := os.MkdirAll(filepath.Join(f.home, ".steam"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(f.steam, filepath.Join(f.home, ".steam", "steam")); err != nil {
		t.Fatal(err)
	}
	vars := map[string]string{"HOME": f.home, "USER": "alex"}
	f.env = NewLinuxEnv(f.home, func(k string) string { return vars[k] }, filepath.Join(f.home, ".local", "share", "lootnest"))
	return f
}

func (f *linuxFixture) roaming() string {
	return filepath.Join(f.pfx, "drive_c", "users", "steamuser", "AppData", "Roaming")
}

func TestLinuxEnv(t *testing.T) {
	f := newLinuxFixture(t)
	if f.env.SteamPath != f.steam {
		t.Fatalf("Steam is found in ~/.local/share: %q", f.env.SteamPath)
	}
	if f.env.Documents != filepath.Join(f.home, "Documents") {
		t.Fatalf("Documents by default: %q", f.env.Documents)
	}
	if su, err := f.env.SteamUser(); err != nil || su != filepath.Join(f.steam, "userdata", "4242") {
		t.Fatalf("Steam's account folder: %q %v", su, err)
	}
	// A native game's folder is kept with %HOME%, read back with $HOME.
	native := filepath.Join(f.home, ".local", "share", "native-game", "saves")
	tok := f.env.Tokenize(native)
	if tok != "%HOME%/.local/share/native-game/saves" {
		t.Fatalf("tokenized: %q", tok)
	}
	if p, err := f.env.Resolve(tok); err != nil || p != native {
		t.Fatalf("resolved: %q %v", p, err)
	}
	for _, bad := range []string{f.home, filepath.Join(f.home, ".local", "share"), filepath.Join(f.home, ".config"), f.steam, "/usr/share"} {
		if f.env.CheckSafe(bad) == nil {
			t.Errorf("%s is too broad to be a game path", bad)
		}
	}

	// A desktop in another language names its folders in user-dirs.dirs.
	write(t, filepath.Join(f.home, ".config", "user-dirs.dirs"), "# made by xdg-user-dirs-update\nXDG_DOCUMENTS_DIR=\"$HOME/Documente\"\nXDG_DOWNLOAD_DIR=\"$HOME/\"\n")
	if d := XDGUserDir(f.home, nil, "DOCUMENTS"); d != filepath.Join(f.home, "Documente") {
		t.Fatalf("Documents from user-dirs.dirs: %q", d)
	}
	if d := XDGUserDir(f.home, nil, "DOWNLOAD"); d != "" {
		t.Fatalf("a folder pointed at home is off: %q", d)
	}
	if d := XDGDataHome(f.home, func(string) string { return "relative/path" }); d != filepath.Join(f.home, ".local", "share") {
		t.Fatalf("a relative XDG_DATA_HOME is ignored: %q", d)
	}

	// The Flatpak's folder when there's no other.
	other := filepath.Join(f.root, "home", "sam")
	flat := filepath.Join(other, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam")
	_ = os.MkdirAll(filepath.Join(flat, "steamapps"), 0o755)
	if got := FindSteamLinux(other); got != flat {
		t.Fatalf("Flatpak Steam: %q", got)
	}
	if FindSteamLinux(filepath.Join(f.root, "home", "nobody")) != "" {
		t.Fatal("no Steam, no folder")
	}
}

func TestPrefix(t *testing.T) {
	f := newLinuxFixture(t)
	if !IsPrefix(f.pfx) || IsPrefix(filepath.Dir(f.pfx)) || IsPrefix("") {
		t.Fatal("a prefix is drive_c with Windows or Wine's registry beside it")
	}
	if got := ProtonPrefix(filepath.Join(f.steam, "steamapps", "common", "Ashen Ring"), protonID); got != f.pfx {
		t.Fatalf("Proton's prefix in the game's library: %q", got)
	}
	if ProtonPrefix(filepath.Join(f.steam, "steamapps", "common", "Native"), 99) != "" {
		t.Fatal("a game Proton never ran has no prefix")
	}

	p := OpenPrefix(f.pfx, "alex")
	if p.User != ProtonUser {
		t.Fatalf("Proton's user: %q", p.User)
	}
	in := f.env.InPrefix(p)
	if in.Getenv("APPDATA") != f.roaming() || in.Getenv("appdata") != f.roaming() {
		t.Fatalf("%%APPDATA%% inside the prefix: %q", in.Getenv("APPDATA"))
	}
	if in.Getenv("HOME") != f.home {
		t.Fatal("other variables are still this system's")
	}
	if in.OS != "" || in.Prefix == nil {
		t.Fatal("the game sees Windows")
	}

	// A path stored on Windows: backslashes, and a case the game didn't use.
	got, err := in.Resolve(`%APPDATA%\ASHENRING\Slot0.sav`)
	want := filepath.Join(f.roaming(), "AshenRing", "slot0.sav")
	if err != nil || got != want {
		t.Fatalf("resolved %q %v, want %q", got, err, want)
	}
	// A folder that isn't there yet keeps the spelling given.
	newDir := filepath.Join(f.pfx, "drive_c", "users", "steamuser", "AppData", "Local", "NewGame", "Saved")
	if got, err := in.Resolve(`%LOCALAPPDATA%\NewGame\Saved`); err != nil || got != newDir {
		t.Fatalf("new folder: %q %v", got, err)
	}
	if tok := in.Tokenize(want); tok != "%APPDATA%/AshenRing/slot0.sav" {
		t.Fatalf("tokenized inside the prefix: %q", tok)
	}
	for _, bad := range []string{f.pfx, filepath.Join(f.pfx, "drive_c"), filepath.Join(f.pfx, "drive_c", "users"), f.roaming(), filepath.Dir(f.roaming()), f.home} {
		if in.CheckSafe(bad) == nil {
			t.Errorf("%s is too broad inside the prefix", bad)
		}
	}
	if err := in.CheckSafe(filepath.Join(f.roaming(), "AshenRing")); err != nil {
		t.Fatalf("the game's own folder: %v", err)
	}

	// A plain Wine prefix: the one user folder there.
	wine := filepath.Join(f.home, "Games", "wine-game")
	_ = os.MkdirAll(filepath.Join(wine, "drive_c", "windows"), 0o755)
	_ = os.MkdirAll(filepath.Join(wine, "drive_c", "users", "Public"), 0o755)
	_ = os.MkdirAll(filepath.Join(wine, "drive_c", "users", "alexw", "Documents"), 0o755)
	if u := OpenPrefix(wine, "alex").User; u != "alexw" {
		t.Fatalf("Wine's user: %q", u)
	}
	_ = os.MkdirAll(filepath.Join(wine, "drive_c", "users", "other"), 0o755)
	if u := OpenPrefix(wine, "alex").User; u != "alex" {
		t.Fatalf("two users: the Linux user's name: %q", u)
	}

	// For: a stored prefix, read with this system's tokens.
	env, err := f.env.For(f.env.Tokenize(f.pfx))
	if err != nil || env.Prefix == nil || env.Prefix.Root != f.pfx {
		t.Fatalf("For: %+v %v", env, err)
	}
	if same, err := f.env.For(""); err != nil || same != f.env {
		t.Fatal("no prefix: the same Env")
	}
	if _, err := f.env.For(filepath.Join(f.home, "gone")); err == nil {
		t.Fatal("a prefix that isn't there")
	}
}

func TestScanSteamLinux(t *testing.T) {
	f := newLinuxFixture(t)
	games := f.env.ScanSteam()
	if len(games) != 1 {
		t.Fatalf("one game: %+v", games)
	}
	g := games[0]
	if g.Prefix != "{STEAM}/steamapps/compatdata/1245620/pfx" {
		t.Fatalf("its prefix, with the Steam token: %q", g.Prefix)
	}
	found := false
	for _, s := range g.Suggestions {
		found = found || s.Resolved == filepath.Join(f.roaming(), "AshenRing")
	}
	if !found || len(g.Paths) == 0 || g.Paths[0] != "%APPDATA%/AshenRing" {
		t.Fatalf("its saves, found inside the prefix: %+v %v", g.Suggestions, g.Paths)
	}
}

func TestSwapInPrefix(t *testing.T) {
	f := newLinuxFixture(t)
	save := filepath.Join(f.roaming(), "AshenRing", "slot0.sav")
	gse := filepath.Join(f.roaming(), "GSE Saves", "1245620", "achievements.json")
	write(t, gse, `{"ACH_1":{"earned":true,"earned_time":1}}`)
	eng := &Engine{Store: &Store{Dir: f.env.DataDir}, Env: f.env}
	cfg := DefaultConfig()
	cfg.Players = []Player{{ID: "p1", Name: "Player 1"}, {ID: "p2", Name: "Player 2"}}
	cfg.Games = []Game{{
		ID: "ashen", Name: "Ashen Ring", ProcessName: "ashenring",
		Launch: Launch{Type: "steam", AppID: protonID},
		Prefix: f.env.Tokenize(f.pfx),
		// Set up on Windows and brought over: backslashes, other case.
		Paths: []PathEntry{{ID: "save", Path: `%APPDATA%\ashenring`}},
	}}
	if err := eng.CheckPaths(&cfg.Games[0]); err != nil {
		t.Fatal(err)
	}
	if got := eng.AchLive(&cfg.Games[0], achEntryID(achFiles[1])); got != filepath.Dir(gse) {
		t.Fatalf("the tool's achievements inside the prefix: %q", got)
	}
	if err := eng.Activate(cfg, "ashen", "p1", ActivateOptions{FirstOwner: "p1"}); err != nil {
		t.Fatal(err)
	}
	if err := eng.Activate(cfg, "ashen", "p2", ActivateOptions{NewProfile: NewFresh}); err != nil {
		t.Fatal(err)
	}
	if read(t, save) != "<missing>" || read(t, gse) != "<missing>" {
		t.Fatal("a fresh profile starts with no saves and no achievements in the prefix")
	}
	write(t, save, "p2 save")
	if err := eng.Activate(cfg, "ashen", "p1", ActivateOptions{}); err != nil {
		t.Fatal(err)
	}
	if read(t, save) != "save" || !strings.Contains(read(t, gse), "ACH_1") {
		t.Fatalf("p1's save and achievements are back: %q %q", read(t, save), read(t, gse))
	}

	// The prefix gone (the game uninstalled): nothing is touched.
	cfg.Games[0].Prefix = "%HOME%/gone/pfx"
	if err := eng.CheckPaths(&cfg.Games[0]); err == nil || !strings.Contains(err.Error(), "prefix") {
		t.Fatalf("a missing prefix is refused: %v", err)
	}
}

func TestSaveDBLinux(t *testing.T) {
	const manifest = `---
Native Game:
  files:
    <xdgData>/native-game/saves:
      tags:
        - save
      when:
        - os: linux
    <winAppData>/NativeGame:
      tags:
        - save
      when:
        - os: windows
    <base>/settings.cfg:
      tags:
        - config
  steam:
    id: 555
Windows Only:
  files:
    <winAppData>/WinOnly:
      tags:
        - save
`
	db, err := ParseSaveDB(strings.NewReader(manifest))
	if err != nil {
		t.Fatal(err)
	}
	ng := db.Find(555)
	if ng == nil || len(ng.Files) != 2 || len(ng.Linux) != 2 {
		t.Fatalf("Windows and Linux locations kept apart: %+v", ng)
	}
	if wo := db.Find(0, "Windows Only"); wo == nil || len(wo.Linux) != 0 {
		t.Fatalf("Windows only: %+v", wo)
	}

	f := newLinuxFixture(t)
	saves := filepath.Join(f.home, ".local", "share", "native-game", "saves")
	_ = os.MkdirAll(saves, 0o755)
	_ = os.MkdirAll(filepath.Join(f.roaming(), "NativeGame"), 0o755)
	base := filepath.Join(f.home, "Games", "NativeGame")
	write(t, filepath.Join(base, "settings.cfg"), "x")

	locs := f.env.FindSaves(ng, base)
	got := map[string]string{}
	for _, l := range locs {
		got[l.Resolved] = l.Path
	}
	if len(locs) != 2 || got[saves] != "%HOME%/.local/share/native-game/saves" || got[filepath.Join(base, "settings.cfg")] == "" {
		t.Fatalf("native: the XDG folder and the game's own file: %+v", locs)
	}
	in := f.env.InPrefix(OpenPrefix(f.pfx, "alex"))
	locs = in.FindSaves(ng, "")
	if len(locs) != 1 || locs[0].Resolved != filepath.Join(f.roaming(), "NativeGame") {
		t.Fatalf("in the prefix: the Windows location: %+v", locs)
	}
}
