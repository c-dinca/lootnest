package core

import (
	"archive/zip"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// pluginZip writes a .zip of name -> content.
func pluginZip(t *testing.T, files map[string]string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "plugin.zip")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return p
}

func manifest(id, version string, provides ...string) string {
	b, _ := json.Marshal(PluginManifest{ID: id, Name: "Night " + id, Version: version, Author: "Someone", Provides: provides})
	return string(b)
}

const goodTheme = `{"name": "Deep sea", "tokens": {"--bg-ground": "#0d1520", "--accent": "oklch(0.70 0.15 220)", "--fg-on-accent": "#0d1520"}}`

func TestInstallPlugin(t *testing.T) {
	data := t.TempDir()
	m, err := InstallPlugin(data, pluginZip(t, map[string]string{"plugin.json": manifest("deep-sea", "1.0.0", "theme"), "theme.json": goodTheme, "readme.txt": "hi"}))
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "deep-sea" || !isFile(filepath.Join(data, "plugins", "deep-sea", "theme.json")) {
		t.Fatalf("installed %+v", m)
	}
	list := ListPlugins(data)
	if len(list) != 1 || !list[0].On || list[0].Problem != "" || !list[0].Has("theme") || list[0].Older != "" {
		t.Fatalf("listed %+v", list)
	}
	if ents, _ := os.ReadDir(PluginsDir(data)); len(ents) != 1 {
		t.Fatalf("left behind %v", ents)
	}
	if err := SetPluginOn(data, "deep-sea", false); err != nil || len(ActivePlugins(data)) != 0 {
		t.Fatalf("off: %v %+v", err, ActivePlugins(data))
	}

	// A folder installs the same way, while a plugin is being made.
	dir := filepath.Join(t.TempDir(), "dev")
	write(t, filepath.Join(dir, "plugin.json"), manifest("dev-theme", "0.1.0", "theme"))
	write(t, filepath.Join(dir, "theme.json"), goodTheme)
	if _, err := InstallPlugin(data, dir); err != nil {
		t.Fatal(err)
	}
	if err := RemovePlugin(data, "dev-theme"); err != nil || isDir(filepath.Join(data, "plugins", "dev-theme")) || !isDir(filepath.Join(data, "plugins", "deep-sea")) {
		t.Fatalf("remove: %v", err)
	}
}

func TestInstallPluginRefusals(t *testing.T) {
	big := strings.Repeat("x", pluginFileMax+1)
	cases := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"bad id", map[string]string{"plugin.json": manifest("Bad_ID", "1.0.0", "theme"), "theme.json": goodTheme}, "its id"},
		{"lootnest's id", map[string]string{"plugin.json": manifest("ln-arcade", "1.0.0", "theme"), "theme.json": goodTheme}, `starting "ln-"`},
		{"exe inside", map[string]string{"plugin.json": manifest("has-exe", "1.0.0", "theme"), "theme.json": goodTheme, "bin/tool.exe": "MZ"}, "isn't data (bin/tool.exe)"},
		{"script inside", map[string]string{"plugin.json": manifest("has-js", "1.0.0", "theme"), "theme.json": goodTheme, "panel.js": "alert(1)"}, "isn't data"},
		{"traversal", map[string]string{"plugin.json": manifest("escape", "1.0.0", "theme"), "theme.json": goodTheme, "../evil.json": "{}"}, "outside its own folder"},
		{"too big", map[string]string{"plugin.json": manifest("too-big", "1.0.0", "theme"), "theme.json": goodTheme, "big.txt": big}, "over 8 MB"},
		{"no manifest", map[string]string{"theme.json": goodTheme}, "plugin.json isn't at its top"},
		{"bad version", map[string]string{"plugin.json": manifest("bad-ver", "1.0", "theme"), "theme.json": goodTheme}, "its version"},
		{"unknown kind", map[string]string{"plugin.json": manifest("scripts", "1.0.0", "scripts")}, `provides "scripts"`},
		{"missing kind", map[string]string{"plugin.json": manifest("no-theme", "1.0.0", "theme")}, "no theme.json"},
	}
	for _, c := range cases {
		data := t.TempDir()
		_, err := InstallPlugin(data, pluginZip(t, c.files))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
		if ents, _ := os.ReadDir(PluginsDir(data)); len(ents) != 0 {
			t.Errorf("%s: left %v", c.name, ents)
		}
	}
	many := map[string]string{"plugin.json": manifest("many", "1.0.0", "theme"), "theme.json": goodTheme}
	for i := 0; i < pluginFilesMax; i++ {
		many[filepath.ToSlash(filepath.Join("pics", strings.Repeat("a", 1+i%7)+string(rune('a'+i%26))+itoa(i)+".txt"))] = "x"
	}
	if _, err := InstallPlugin(t.TempDir(), pluginZip(t, many)); err == nil || !strings.Contains(err.Error(), "more than 500 files") {
		t.Errorf("many files: %v", err)
	}
}

func TestReplacePluginAndRollback(t *testing.T) {
	data := t.TempDir()
	v1 := map[string]string{"plugin.json": manifest("deep-sea", "1.0.0", "theme"), "theme.json": goodTheme}
	if _, err := InstallPlugin(data, pluginZip(t, v1)); err != nil {
		t.Fatal(err)
	}
	_ = SetPluginOn(data, "deep-sea", false)
	// A new version that doesn't check out leaves the old one as it was.
	bad := map[string]string{"plugin.json": manifest("deep-sea", "2.0.0", "theme"), "theme.json": `{"name": "x", "tokens": {"--fs-14": "40px"}}`}
	if _, err := InstallPlugin(data, pluginZip(t, bad)); err == nil {
		t.Fatal("a bad theme replaced the plugin")
	}
	if l := ListPlugins(data); l[0].Version != "1.0.0" || l[0].Older != "" {
		t.Fatalf("after a refused replace %+v", l)
	}
	v2 := map[string]string{"plugin.json": manifest("deep-sea", "1.1.0", "theme"), "theme.json": goodTheme}
	if _, err := InstallPlugin(data, pluginZip(t, v2)); err != nil {
		t.Fatal(err)
	}
	l := ListPlugins(data)
	if l[0].Version != "1.1.0" || l[0].Older != "1.0.0" || l[0].On || !isDir(filepath.Join(data, "plugins", ".old", "deep-sea", "1.0.0")) {
		t.Fatalf("after replace %+v (it was off, and stays off)", l)
	}
	m, err := RollbackPlugin(data, "deep-sea")
	if err != nil || m.Version != "1.0.0" {
		t.Fatalf("rollback %+v %v", m, err)
	}
	if l := ListPlugins(data); l[0].Version != "1.0.0" || l[0].Older != "" || !isDir(filepath.Join(data, "plugins", ".old", "deep-sea", "1.1.0")) {
		t.Fatalf("after rollback %+v (1.1.0 is kept)", l)
	}
	if _, err := RollbackPlugin(data, "deep-sea"); err == nil {
		t.Fatal("rolled back with nothing kept")
	}
}

// pluginDir writes a plugin's files into a folder of its own.
func pluginDir(t *testing.T, files map[string]string) Plugin {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "p")
	for name, body := range files {
		write(t, filepath.Join(dir, filepath.FromSlash(name)), body)
	}
	m, err := CheckPluginDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return Plugin{PluginManifest: *m, On: true, Dir: dir}
}

func jsonStr(s string) string { b, _ := json.Marshal(s); return string(b) }

func TestPluginSources(t *testing.T) {
	root := t.TempDir()
	appdata := filepath.Join(root, "AppData")
	getenv := func(k string) string {
		if k == "APPDATA" {
			return appdata
		}
		return ""
	}
	game := func(name string) string {
		dir := filepath.Join(root, "Emu", name)
		write(t, filepath.Join(dir, name+".exe"), "MZ")
		return dir
	}
	snes, gba, vdfGame, regGame := game("Snes"), game("Gba"), game("Tale"), game("Racer")

	// A RetroArch-style JSON playlist.
	write(t, filepath.Join(appdata, "Retro", "playlists", "a.lpl"), `{"version": "1.5", "items": [
		{"label": "Super Game", "path": `+jsonStr(filepath.Join(snes, "Snes.exe"))+`, "db_name": "x"},
		{"label": "Hand Game", "dir": `+jsonStr(gba)+`, "steam": 4242},
		{"label": "Gone", "path": `+jsonStr(filepath.Join(root, "nowhere.exe"))+`}]}`)
	// Another launcher's VDF.
	write(t, filepath.Join(appdata, "Other", "games.vdf"), `"Library" { "Games" { "0" { "Title" "Night Tale" "Folder" `+jsonStr(vdfGame)+` "Exe" "Tale.exe" } } }`)

	p := pluginDir(t, map[string]string{
		"plugin.json": manifest("retro", "1.0.0", "sources"),
		"sources/retroarch.json": `{"name": "RetroArch", "kind": "json", "path": "%APPDATA%\\Retro\\playlists\\*.lpl", "list": "items",
			"fields": {"name": "label", "exe": "path", "dir": "dir", "steamId": "steam"}}`,
		"sources/other.json": `{"name": "Other launcher", "kind": "vdf", "path": "%APPDATA%/Other/games.vdf", "list": "Library.Games",
			"fields": {"name": "Title", "exe": "Exe", "dir": "Folder"}}`,
		"sources/reg.json": `{"name": "Registry games", "kind": "registry", "path": "HKCU\\Software\\Racers\\Games",
			"fields": {"name": "DisplayName", "dir": "InstallDir"}}`,
	})
	fake := func(key string) ([]string, map[string]string, error) {
		switch key {
		case `HKCU\Software\Racers\Games`:
			return []string{"1", "2"}, nil, nil
		case `HKCU\Software\Racers\Games\1`:
			return nil, map[string]string{"displayname": "Fast Racer", "InstallDir": regGame}, nil
		}
		return nil, nil, errors.New("not found")
	}
	got := map[string]FoundGame{}
	for _, f := range PluginGames(p, getenv, fake) {
		got[f.Name] = f
	}
	if len(got) != 4 {
		t.Fatalf("found %+v", got)
	}
	if f := got["Super Game"]; f.Exe != filepath.Join(snes, "Snes.exe") || f.Dir != snes || f.Source != "plugin:retro" || f.SourceName != "RetroArch" || f.Key != "dir:"+strings.ToLower(snes) {
		t.Errorf("json %+v", f)
	}
	if f := got["Hand Game"]; f.Exe != filepath.Join(gba, "Gba.exe") || f.SteamID != 4242 {
		t.Errorf("json, exe picked in its folder %+v", f)
	}
	if f := got["Night Tale"]; f.Exe != filepath.Join(vdfGame, "Tale.exe") || f.SourceName != "Other launcher" {
		t.Errorf("vdf %+v", f)
	}
	if f := got["Fast Racer"]; f.Exe != filepath.Join(regGame, "Racer.exe") || f.ProcessName != "Racer" {
		t.Errorf("registry %+v", f)
	}

	// A game the folder scan also finds keeps the folder as its source.
	folder := FoundGame{Key: got["Night Tale"].Key, Name: "Night Tale", Source: "folder", Dir: vdfGame, Exe: got["Night Tale"].Exe}
	if out := newFixture(t).env.FinishFound([]FoundGame{got["Night Tale"], folder}); len(out) != 1 || out[0].Source != "folder" {
		t.Errorf("a folder game taken by a plugin %+v", out)
	}

	list := []FoundGame{got["Super Game"], {Key: "steam:1", Source: "steam"}}
	if out := DropPluginGames(list, map[string]bool{}); len(out) != 1 || out[0].Source != "steam" {
		t.Errorf("off plugin kept %+v", out)
	}
	if out := DropPluginGames(list, map[string]bool{"retro": true}); len(out) != 2 {
		t.Errorf("on plugin dropped %+v", out)
	}

	for _, bad := range []string{
		`{"name": "x", "kind": "json", "path": "C:\\Games\\list.json", "fields": {"name": "n", "exe": "e"}}`,
		`{"name": "x", "kind": "json", "path": "%WINDIR%\\list.json", "fields": {"name": "n", "exe": "e"}}`,
		`{"name": "x", "kind": "json", "path": "%APPDATA%\\..\\..\\list.json", "fields": {"name": "n", "exe": "e"}}`,
		`{"name": "x", "kind": "registry", "path": "HKLM\\SYSTEM\\Games", "fields": {"name": "n", "exe": "e"}}`,
		`{"name": "x", "kind": "script", "path": "%APPDATA%\\a", "fields": {"name": "n", "exe": "e"}}`,
		`{"name": "x", "kind": "json", "path": "%APPDATA%\\a.json", "fields": {"name": "n"}}`,
	} {
		dir := filepath.Join(t.TempDir(), "p")
		write(t, filepath.Join(dir, "plugin.json"), manifest("bad-src", "1.0.0", "sources"))
		write(t, filepath.Join(dir, "sources", "s.json"), bad)
		if _, err := CheckPluginDir(dir); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}

func TestPluginSavesAreSuggestionsOnly(t *testing.T) {
	f := newFixture(t)
	p := pluginDir(t, map[string]string{
		"plugin.json": manifest("night-saves", "1.0.0", "saves"),
		"saves/games.yaml": `"Night Tale":
  files:
    "<winAppData>/NightStudio/Tale":
      tags:
        - save
`,
	})
	db, err := PluginSaves(p.Dir)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(f.appdata, "NightStudio", "Tale", "slot1.sav"), "1")
	f.env.PluginSaves = func() []PluginSaveDB { return []PluginSaveDB{{Name: p.Name, DB: db}} }
	sug := f.env.SuggestPaths("Night Tale", filepath.Join(f.root, "Games", "Night Tale"), 0)
	var from *Suggestion
	for i := range sug {
		if sug[i].Resolved == filepath.Join(f.appdata, "NightStudio", "Tale") {
			from = &sug[i]
		}
	}
	if from == nil || from.Plugin != p.Name || from.Reason != "From "+p.Name {
		t.Fatalf("suggested %+v", sug)
	}
	for _, got := range ProposePaths(sug, nil) {
		if got == from.Path {
			t.Fatalf("a plugin's location was proposed by itself: %v", got)
		}
	}

	dir := filepath.Join(t.TempDir(), "p")
	write(t, filepath.Join(dir, "plugin.json"), manifest("bad-saves", "1.0.0", "saves"))
	write(t, filepath.Join(dir, "saves", "x.yaml"), "not: [a manifest")
	if _, err := CheckPluginDir(dir); err == nil || !strings.Contains(err.Error(), "Ludusavi") {
		t.Fatalf("bad saves: %v", err)
	}
}

func TestPluginMetadataLayers(t *testing.T) {
	p := pluginDir(t, map[string]string{
		"plugin.json": manifest("retro-info", "1.0.0", "metadata"),
		"metadata/games.json": `{
			"1145360": {"summary": "From the plugin", "genres": ["Roguelike"], "developer": "Supergiant", "released": "2020"},
			"Super Game": {"summary": "  A   game  ", "publisher": "Nin"}}`,
	})
	meta, err := PluginMetadata(p.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if d := meta.Find(0, "SUPER game"); d == nil || d.Summary != "A game" || d.Publishers[0] != "Nin" {
		t.Fatalf("by name %+v", d)
	}
	under := meta.Find(1145360, "Hades")
	if meta.Find(0, "Unknown") != nil || under == nil {
		t.Fatal("find")
	}
	// Steam's details win where it has them; the plugin fills the rest.
	steam := &GameDetails{Summary: "From Steam", StoreURL: "https://store.steampowered.com/app/1145360"}
	d := LayerDetails(under, p.Name, steam)
	if d.Summary != "From Steam" || d.Developers[0] != "Supergiant" || d.Released != "2020" || d.From != p.Name || d.StoreURL == "" {
		t.Fatalf("layered %+v", d)
	}
	// Without Steam's, the plugin's alone; the user's edits win over both.
	dev := "Me"
	d = MergeDetails(LayerDetails(under, p.Name, nil), DetailsEdit{Developer: &dev})
	if d.Summary != "From the plugin" || d.Developers[0] != "Me" {
		t.Fatalf("edited %+v", d)
	}
	if LayerDetails(nil, p.Name, steam) != steam {
		t.Fatal("no plugin details changed Steam's")
	}
}

func TestPluginTheme(t *testing.T) {
	if err := CheckTheme(map[string]string{"--bg-ground": "#0d1520", "--fg-2": "oklch(80% 0.02 240)", "--edge-1": "oklch(1 0 0 / 0.1)"}); err != nil {
		t.Fatalf("a good theme: %v", err)
	}
	cases := map[string]struct {
		tokens map[string]string
		want   string
	}{
		"low contrast":   {map[string]string{"--fg-3": "#555a60"}, "dates and counts (--fg-3) on wells and inputs is"},
		"light theme":    {map[string]string{"--bg-ground": "#f4f4f4", "--bg-well": "#ffffff", "--bg-surface": "#eeeeee", "--bg-raised": "#e6e6e6", "--bg-active": "#dddddd", "--fg-1": "#111111", "--fg-2": "#222222", "--fg-3": "#333333"}, "on the play dock"},
		"mid grey":       {map[string]string{"--bg-ground": "#6c6c6c", "--bg-well": "#6a6a6a", "--bg-surface": "#6d6d6d", "--bg-raised": "#6e6e6e", "--bg-active": "#6e6e6e", "--fg-1": "#ffffff", "--fg-2": "#ffffff", "--fg-3": "#ffffff", "--accent": "#000000", "--accent-hover": "#000000", "--fg-on-accent": "#ffffff"}, "error text (--danger) on"},
		"red text":       {map[string]string{"--fg-on-accent": "#ffffff"}, "text on the accent (--fg-on-accent) on the accent"},
		"size token":     {map[string]string{"--fs-14": "40px"}, "--fs-14, which a theme can't change"},
		"font token":     {map[string]string{"--font-ui": "Comic Sans"}, "can't change"},
		"not a colour":   {map[string]string{"--bg-ground": "url(x.png)"}, "isn't a hex, rgb() or oklch() colour"},
		"named colour":   {map[string]string{"--accent": "red"}, "isn't a hex, rgb() or oklch() colour"},
		"see-through bg": {map[string]string{"--bg-ground": "oklch(0.1 0 0 / 0.5)"}, "must be a solid colour"},
		"nothing":        {map[string]string{}, "sets no colours"},
	}
	for name, c := range cases {
		if err := CheckTheme(c.tokens); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
	}
	// WCAG's own numbers for the brand colours (DESIGN.md).
	a, _ := parseColor("#DFE5EB")
	b, _ := parseColor("#111217")
	if r := contrast(a, b); r < 14.6 || r > 14.8 {
		t.Errorf("Ice Grey on Obsidian %.2f", r)
	}
}

// Lootnest's own values for the theme tokens are system.css's.
func TestThemeTokensMatchSystemCSS(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "app", "ui", "system.css"))
	if err != nil {
		t.Fatal(err)
	}
	for _, set := range []map[string]string{ThemeTokens, themeFixed} {
		for k, v := range set {
			m := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(k) + `:\s*([^;]+);`).FindSubmatch(b)
			if m == nil || strings.TrimSpace(string(m[1])) != v {
				t.Errorf("%s is %q in system.css, %q here", k, m, v)
			}
		}
	}
}

// signedZip is a plugin's files as a .zip, signed with priv.
func signedZip(t *testing.T, files map[string]string, priv ed25519.PrivateKey) string {
	t.Helper()
	dir := pluginDir(t, files).Dir
	if err := SignPlugin(dir, priv); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "signature.json"))
	if err != nil {
		t.Fatal(err)
	}
	all := map[string]string{"signature.json": string(b)}
	for k, v := range files {
		all[k] = v
	}
	return pluginZip(t, all)
}

func newKey(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	return priv
}

func TestPluginSignature(t *testing.T) {
	priv := newKey(t)
	files := map[string]string{"plugin.json": manifest("deep-sea", "1.0.0", "theme"), "theme.json": goodTheme, "art/readme.txt": "hi"}
	dir := pluginDir(t, files).Dir
	if fp, signed, err := VerifyPlugin(dir); err != nil || signed || fp != "" {
		t.Fatalf("unsigned: %q %v %v", fp, signed, err)
	}
	if err := SignPlugin(dir, priv); err != nil {
		t.Fatal(err)
	}
	fp, signed, err := VerifyPlugin(dir)
	want := KeyFingerprint(priv.Public().(ed25519.PublicKey))
	if err != nil || !signed || fp != want || !regexp.MustCompile(`^[0-9A-F]{4}( [0-9A-F]{4}){3}$`).MatchString(fp) {
		t.Fatalf("signed: %q %v %v", fp, signed, err)
	}
	d, _ := PluginDigest(dir)
	if !strings.HasPrefix(string(d), "art/readme.txt\t") || strings.Contains(string(d), "signature.json") || strings.Count(string(d), "\n") != 3 {
		t.Fatalf("digest %q", d)
	}

	data := t.TempDir()
	m, err := InstallPlugin(data, dir)
	if err != nil || m.Key != want || m.PinnedNow {
		t.Fatalf("install %+v %v", m, err)
	}
	if l := ListPlugins(data); l[0].Key != want || l[0].Problem != "" {
		t.Fatalf("listed %+v", l)
	}

	// A file changed after signing: refused, and nothing is left.
	write(t, filepath.Join(dir, "art", "readme.txt"), "changed")
	if _, _, err := VerifyPlugin(dir); !errors.Is(err, ErrPluginSignature) {
		t.Fatalf("tampered: %v", err)
	}
	other := t.TempDir()
	if _, err := InstallPlugin(other, dir); !errors.Is(err, ErrPluginSignature) {
		t.Fatalf("tampered install: %v", err)
	}
	if ents, _ := os.ReadDir(PluginsDir(other)); len(ents) != 0 {
		t.Fatalf("left %v", ents)
	}
	// Changed on disk once installed: it's off, with the reason.
	write(t, filepath.Join(data, "plugins", "deep-sea", "theme.json"), strings.Replace(goodTheme, "Deep sea", "Deeper", 1))
	if l := ListPlugins(data); l[0].Problem == "" || len(ActivePlugins(data)) != 0 {
		t.Fatalf("tampered on disk %+v", l)
	}
	// Its signature deleted: off, and the reason says so.
	if err := os.Remove(filepath.Join(data, "plugins", "deep-sea", "signature.json")); err != nil {
		t.Fatal(err)
	}
	if l := ListPlugins(data); !strings.Contains(l[0].Problem, "its signature is gone") || !strings.Contains(l[0].Problem, want) {
		t.Fatalf("signature gone %+v", l)
	}
}

func TestPluginKeyPinning(t *testing.T) {
	a, b := newKey(t), newKey(t)
	ver := func(v string) map[string]string {
		return map[string]string{"plugin.json": manifest("deep-sea", v, "theme"), "theme.json": goodTheme}
	}
	data := t.TempDir()
	if _, err := InstallPlugin(data, signedZip(t, ver("1.0.0"), a)); err != nil {
		t.Fatal(err)
	}
	// Another key: refused, naming both, and the installed one stays.
	_, err := InstallPlugin(data, signedZip(t, ver("1.1.0"), b))
	var kc *PluginKeyChange
	if !errors.As(err, &kc) || kc.Pinned != KeyFingerprint(a.Public().(ed25519.PublicKey)) || kc.Key != KeyFingerprint(b.Public().(ed25519.PublicKey)) || kc.Name != "Night deep-sea" {
		t.Fatalf("key change: %v", err)
	}
	if !strings.Contains(err.Error(), kc.Pinned) || !strings.Contains(err.Error(), kc.Key) {
		t.Fatalf("reason %q", err)
	}
	if l := ListPlugins(data); l[0].Version != "1.0.0" {
		t.Fatalf("replaced by a refused key: %+v", l)
	}
	// Unsigned can't replace signed, even when the user accepts a new key.
	if _, err := InstallPlugin(data, pluginZip(t, ver("1.2.0"))); err == nil || !strings.Contains(err.Error(), "this version is not signed") {
		t.Fatalf("unsigned over signed: %v", err)
	}
	if _, err := InstallPluginNewKey(data, pluginZip(t, ver("1.2.0")), kc.Key); err == nil || !strings.Contains(err.Error(), "this version is not signed") {
		t.Fatalf("unsigned over signed, new key accepted: %v", err)
	}
	// Only the key the user accepted installs: another one is asked again.
	c := newKey(t)
	var again *PluginKeyChange
	if _, err := InstallPluginNewKey(data, signedZip(t, ver("1.1.0"), c), kc.Key); !errors.As(err, &again) || again.Key != KeyFingerprint(c.Public().(ed25519.PublicKey)) {
		t.Fatalf("a key nobody accepted: %v", err)
	}
	// Accepted: installed, and the new key is pinned.
	m, err := InstallPluginNewKey(data, signedZip(t, ver("1.1.0"), b), kc.Key)
	if err != nil || m.Key != kc.Key {
		t.Fatalf("new key %+v %v", m, err)
	}
	// 1.0.0 has the old key, so Roll back doesn't offer it.
	if l := ListPlugins(data); l[0].Version != "1.1.0" || l[0].Key != kc.Key || l[0].Problem != "" || l[0].Older != "" {
		t.Fatalf("after new key %+v", l)
	}
	if _, err := InstallPlugin(data, signedZip(t, ver("1.3.0"), a)); !errors.As(err, &kc) {
		t.Fatalf("old key after accepting the new one: %v", err)
	}
	if _, err := RollbackPlugin(data, "deep-sea"); err == nil {
		t.Fatal("rolled back to another key")
	}
	if l := ListPlugins(data); l[0].Version != "1.1.0" {
		t.Fatalf("after a refused rollback %+v", l)
	}
	// Removed: the pin goes, so another key installs as a first one.
	if err := RemovePlugin(data, "deep-sea"); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallPlugin(data, signedZip(t, ver("2.0.0"), a)); err != nil {
		t.Fatalf("after remove: %v", err)
	}
}

func TestSignedReplacesUnsigned(t *testing.T) {
	priv := newKey(t)
	ver := func(v string) map[string]string {
		return map[string]string{"plugin.json": manifest("deep-sea", v, "theme"), "theme.json": goodTheme}
	}
	data := t.TempDir()
	if m, err := InstallPlugin(data, pluginZip(t, ver("1.0.0"))); err != nil || m.Key != "" {
		t.Fatalf("unsigned %+v %v", m, err)
	}
	if l := ListPlugins(data); l[0].Key != "" || l[0].Problem != "" {
		t.Fatalf("unsigned listed %+v", l)
	}
	m, err := InstallPlugin(data, signedZip(t, ver("1.1.0"), priv))
	if err != nil || !m.PinnedNow || m.Key == "" {
		t.Fatalf("signed over unsigned %+v %v", m, err)
	}
	if _, err := InstallPlugin(data, signedZip(t, ver("1.2.0"), newKey(t))); err == nil {
		t.Fatal("the key wasn't pinned")
	}
	// Rolling back to the unsigned version would drop the signature.
	if l := ListPlugins(data); l[0].Older != "" {
		t.Fatalf("offers the unsigned version %+v", l)
	}
	if _, err := RollbackPlugin(data, "deep-sea"); err == nil {
		t.Fatal("rolled back to unsigned")
	}
	if l := ListPlugins(data); l[0].Version != "1.1.0" || l[0].Problem != "" {
		t.Fatalf("after a refused rollback %+v", l)
	}
	// A kept version with the pinned key rolls back.
	if _, err := InstallPlugin(data, signedZip(t, ver("1.2.0"), priv)); err != nil {
		t.Fatal(err)
	}
	if l := ListPlugins(data); l[0].Older != "1.1.0" {
		t.Fatalf("older %+v", l)
	}
	if m, err := RollbackPlugin(data, "deep-sea"); err != nil || m.Version != "1.1.0" {
		t.Fatalf("rollback %+v %v", m, err)
	}
}
