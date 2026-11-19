package core

import (
	"encoding/json"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// The launchers of a Linux PC besides Steam (docs/ROADMAP.md, section 9,
// L1b): Heroic (Epic, GOG, Amazon and its own added games), Lutris, Bottles
// and the itch app, from the files each keeps, native or Flatpak. A Windows
// game gets its Wine prefix (FoundGame.Prefix), where its saves are.
// Everything is only read.

// ScanLinuxLaunchers finds the games every Linux launcher knows, with
// their prefixes and paths stored with this Env's tokens.
func (e *Env) ScanLinuxLaunchers() []FoundGame {
	home := e.Home
	data, conf := XDGDataHome(home, e.Getenv), XDGConfigHome(home, e.Getenv)
	flat := func(app, kind string) string { return filepath.Join(home, ".var", "app", app, kind) }
	var out []FoundGame
	for _, dir := range []string{filepath.Join(conf, "heroic"), filepath.Join(flat("com.heroicgameslauncher.hgl", "config"), "heroic")} {
		out = append(out, HeroicGames(dir, home)...)
	}
	for _, l := range [][2]string{
		{filepath.Join(data, "lutris"), filepath.Join(conf, "lutris")},
		{filepath.Join(flat("net.lutris.Lutris", "data"), "lutris"), filepath.Join(flat("net.lutris.Lutris", "config"), "lutris")},
	} {
		out = append(out, LutrisGames(l[0], l[1])...)
	}
	for _, dir := range []string{filepath.Join(data, "bottles", "bottles"), filepath.Join(flat("com.usebottles.bottles", "data"), "bottles", "bottles")} {
		out = append(out, BottlesGames(dir)...)
	}
	for _, dir := range []string{filepath.Join(conf, "itch", "apps"), filepath.Join(flat("io.itch.itch", "config"), "itch", "apps")} {
		out = append(out, itchGames(dir, mainAnyExe)...)
	}
	for i := range out {
		if out[i].Prefix != "" {
			out[i].Prefix = e.Tokenize(out[i].Prefix)
		}
	}
	return out
}

// prefixAt is the Wine prefix a launcher names: the folder itself, or its
// pfx folder when the game runs through Proton (Heroic, Lutris), or "".
func prefixAt(p string) string {
	if p == "" || !filepath.IsAbs(p) {
		return ""
	}
	p = filepath.Clean(p)
	switch {
	case IsPrefix(p):
		return p
	case IsPrefix(filepath.Join(p, "pfx")):
		return filepath.Join(p, "pfx")
	}
	return ""
}

func exeName(exe string) string {
	base := filepath.Base(exe)
	for _, ext := range []string{".exe", ".x86_64", ".x86", ".sh", ".AppImage"} {
		if strings.HasSuffix(strings.ToLower(base), strings.ToLower(ext)) {
			return base[:len(base)-len(ext)]
		}
	}
	return base
}

// mainAnyExe is a folder's game program: a Windows .exe, else a Linux one.
func mainAnyExe(dir string) (string, string) {
	if exe, process := MainExe(dir); exe != "" {
		return exe, process
	}
	return MainLinuxExe(dir)
}

// MainLinuxExe finds a native Linux game's program in its folder: the
// largest executable ELF file (or .x86_64, .sh) up to three folders down,
// leaving out libraries and helpers.
func MainLinuxExe(dir string) (exe, process string) {
	type cand struct {
		path string
		size int64
		elf  bool
	}
	var cands []cand
	base := strings.Count(filepath.Clean(dir), string(filepath.Separator))
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != dir && (strings.HasPrefix(d.Name(), ".") || strings.Count(p, string(filepath.Separator))-base >= 3) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		n := strings.ToLower(d.Name())
		if strings.Contains(n, ".so") || strings.HasSuffix(n, ".exe") || strings.HasSuffix(n, ".dll") {
			return nil
		}
		for _, h := range []string{"crash", "unins", "setup", "install", "update", "report", "helper", "launcher-helper"} {
			if strings.Contains(n, h) {
				return nil
			}
		}
		info, err := d.Info()
		if err != nil || info.Mode().Perm()&0o111 == 0 {
			return nil
		}
		elf := false
		if f, err := os.Open(p); err == nil {
			head := make([]byte, 4)
			if n, _ := f.Read(head); n == 4 && string(head) == "\x7fELF" {
				elf = true
			}
			f.Close()
		}
		if elf || strings.HasSuffix(n, ".x86_64") || strings.HasSuffix(n, ".sh") {
			cands = append(cands, cand{p, info.Size(), elf})
		}
		return nil
	})
	if len(cands) == 0 {
		return "", ""
	}
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].elf != cands[j].elf {
			return cands[i].elf
		}
		return cands[i].size > cands[j].size
	})
	return cands[0].path, exeName(cands[0].path)
}

// --------------------------------------------------------------- Heroic ---

// heroicURI starts a game through Heroic (protocol.ts: heroic://launch with
// appName and runner).
func heroicURI(runner, app string) string {
	return "heroic://launch?appName=" + url.QueryEscape(app) + "&runner=" + runner
}

// HeroicGames reads the games Heroic installed (dir is its config folder,
// ~/.config/heroic): Epic through legendary, GOG, Amazon through nile, and
// the ones added to it by hand. A Windows game's prefix is the one its
// GamesConfig file names, else Heroic's default.
func HeroicGames(dir, home string) []FoundGame {
	if !isDir(dir) {
		return nil
	}
	tilde := func(p string) string {
		if strings.HasPrefix(p, "~") && home != "" {
			return filepath.Join(home, p[1:])
		}
		return p
	}
	var global struct {
		Default struct {
			WinePrefix    string `json:"winePrefix"`
			WinePrefixDir string `json:"defaultWinePrefixDir"`
		} `json:"defaultSettings"`
	}
	readJSON(filepath.Join(dir, "config.json"), &global)
	prefixOf := func(app, title string) string {
		var per map[string]struct {
			WinePrefix string `json:"winePrefix"`
		}
		readJSON(filepath.Join(dir, "GamesConfig", app+".json"), &per)
		cands := []string{per[app].WinePrefix, global.Default.WinePrefix}
		for _, d := range []string{global.Default.WinePrefixDir, filepath.Join(home, "Games", "Heroic", "Prefixes", "default")} {
			if d != "" && title != "" {
				cands = append(cands, filepath.Join(tilde(d), title))
			}
		}
		cands = append(cands, filepath.Join(home, "Games", "Heroic", "Prefixes", "default"))
		for _, c := range cands {
			if p := prefixAt(tilde(c)); p != "" {
				return p
			}
		}
		return ""
	}
	var out []FoundGame
	add := func(runner, app, title, installDir, exe, platform string) {
		if app == "" || installDir == "" {
			return
		}
		installDir = filepath.Clean(tilde(installDir))
		windows := platform == "" || strings.EqualFold(platform, "windows") || strings.EqualFold(platform, "win32")
		if !strings.EqualFold(platform, "linux") && !windows {
			return // a Mac build
		}
		if exe != "" && !filepath.IsAbs(exe) {
			exe = filepath.Join(installDir, filepath.FromSlash(strings.ReplaceAll(exe, `\`, "/")))
		}
		if exe == "" || !isFile(exe) {
			if windows {
				exe, _ = MainExe(installDir)
			} else {
				exe, _ = MainLinuxExe(installDir)
			}
		}
		if exe == "" {
			return
		}
		if title == "" {
			title = filepath.Base(installDir)
		}
		f := FoundGame{Key: "heroic:" + runner + ":" + app, Name: title, Source: "heroic", Exe: exe, Dir: installDir,
			ProcessName: exeName(exe), StoreID: runner + ":" + app}
		if u := heroicURI(runner, app); SafeLaunchURI(u) {
			f.URI = u
		}
		if windows {
			f.Prefix = prefixOf(app, title)
		}
		out = append(out, f)
	}

	// Epic, through legendary.
	var legendary map[string]struct {
		App      string `json:"app_name"`
		Title    string `json:"title"`
		Path     string `json:"install_path"`
		Exe      string `json:"executable"`
		Platform string `json:"platform"`
		DLC      bool   `json:"is_dlc"`
	}
	readJSON(filepath.Join(dir, "legendaryConfig", "legendary", "installed.json"), &legendary)
	for app, g := range legendary {
		if !g.DLC {
			add("legendary", app, g.Title, g.Path, g.Exe, g.Platform)
		}
	}

	// GOG: titles from Heroic's library cache, else the game's own info file.
	var gogLib struct {
		Games []struct {
			App   string `json:"app_name"`
			Title string `json:"title"`
		} `json:"games"`
	}
	readJSON(filepath.Join(dir, "store_cache", "gog_library.json"), &gogLib)
	gogTitles := map[string]string{}
	for _, g := range gogLib.Games {
		gogTitles[g.App] = g.Title
	}
	var gog struct {
		Installed []struct {
			App      string `json:"appName"`
			Path     string `json:"install_path"`
			Exe      string `json:"executable"`
			Platform string `json:"platform"`
			DLC      bool   `json:"is_dlc"`
		} `json:"installed"`
	}
	readJSON(filepath.Join(dir, "gog_store", "installed.json"), &gog)
	for _, g := range gog.Installed {
		if g.DLC {
			continue
		}
		title, exe := gogTitles[g.App], g.Exe
		if info := gogInfo(tilde(g.Path), g.App); info.name != "" {
			if title == "" {
				title = info.name
			}
			if exe == "" {
				exe = info.exe
			}
		}
		add("gog", g.App, title, g.Path, exe, g.Platform)
	}

	// Amazon, through nile.
	var nileLib []struct {
		ID      string `json:"id"`
		Product struct {
			Title string `json:"title"`
		} `json:"product"`
	}
	readJSON(filepath.Join(dir, "nile_config", "nile", "library.json"), &nileLib)
	nileTitles := map[string]string{}
	for _, g := range nileLib {
		nileTitles[g.ID] = g.Product.Title
	}
	var nile []struct {
		ID   string `json:"id"`
		Path string `json:"path"`
	}
	readJSON(filepath.Join(dir, "nile_config", "nile", "installed.json"), &nile)
	for _, g := range nile {
		add("nile", g.ID, nileTitles[g.ID], g.Path, "", "windows")
	}

	// Games added to Heroic by hand.
	var side struct {
		Games []struct {
			App       string `json:"app_name"`
			Title     string `json:"title"`
			Installed bool   `json:"is_installed"`
			Install   struct {
				Exe      string `json:"executable"`
				Platform string `json:"platform"`
			} `json:"install"`
		} `json:"games"`
	}
	readJSON(filepath.Join(dir, "sideload_apps", "library.json"), &side)
	for _, g := range side.Games {
		if g.Installed && g.Install.Exe != "" && filepath.IsAbs(g.Install.Exe) {
			add("sideload", g.App, g.Title, filepath.Dir(g.Install.Exe), g.Install.Exe, g.Install.Platform)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

type gogGameInfo struct{ name, exe string }

// gogInfo reads a GOG game's goggame-<id>.info: its name and the program
// its primary play task starts.
func gogInfo(dir, id string) gogGameInfo {
	var info struct {
		Name  string `json:"name"`
		Tasks []struct {
			Primary bool   `json:"isPrimary"`
			Path    string `json:"path"`
			Type    string `json:"type"`
		} `json:"playTasks"`
	}
	if !readJSON(filepath.Join(dir, "goggame-"+id+".info"), &info) {
		return gogGameInfo{}
	}
	out := gogGameInfo{name: info.Name}
	for _, t := range info.Tasks {
		if t.Primary && t.Path != "" && (t.Type == "" || t.Type == "FileTask") {
			out.exe = t.Path
			break
		}
	}
	return out
}

// readJSON reads a small JSON file into v; false when it's missing or
// can't be read.
func readJSON(path string, v any) bool {
	st, err := os.Stat(path)
	if err != nil || st.IsDir() || st.Size() > 64<<20 {
		return false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return json.Unmarshal(b, v) == nil
}

// --------------------------------------------------------------- Lutris ---

// LutrisGames reads Lutris's library (pga.db in dataDir) and each game's
// file in games/ (in configDir, or dataDir since Lutris 0.5.13 when there's
// no config folder): Windows games through Wine with their prefix, and
// native Linux games. Games of other runners (Steam's, emulators') are left
// to their own scans.
func LutrisGames(dataDir, configDir string) []FoundGame {
	rows, err := sqliteTable(filepath.Join(dataDir, "pga.db"), "games")
	if err != nil {
		return nil
	}
	gameDirs := []string{filepath.Join(configDir, "games"), filepath.Join(dataDir, "games")}
	var out []FoundGame
	for _, r := range rows {
		str := func(k string) string { s, _ := r[k].(string); return s }
		num := func(k string) int64 { n, _ := r[k].(int64); return n }
		runner := str("runner")
		if num("installed") != 1 || (runner != "wine" && runner != "linux") {
			continue
		}
		id := num("id")
		dir := str("directory")
		var game map[string]any
		if cp := str("configpath"); cp != "" && !strings.ContainsAny(cp, `/\`) {
			for _, d := range gameDirs {
				if b, err := os.ReadFile(filepath.Join(d, cp+".yml")); err == nil && len(b) < 4<<20 {
					game = readYAMLMap(string(b))
					break
				}
			}
		}
		exe := yamlStr(game, "game", "exe")
		if exe == "" {
			exe = str("executable")
		}
		if exe != "" && !filepath.IsAbs(exe) && dir != "" {
			exe = filepath.Join(dir, exe)
		}
		prefix := ""
		if runner == "wine" {
			prefix = prefixAt(yamlStr(game, "game", "prefix"))
			if prefix == "" {
				prefix = prefixAt(dir)
			}
		}
		if dir == "" && exe != "" {
			dir = filepath.Dir(exe)
		}
		if exe == "" || !isFile(exe) || dir == "" {
			continue
		}
		f := FoundGame{Key: "lutris:" + strconv.FormatInt(id, 10), Name: str("name"), Source: "lutris", Exe: exe,
			Dir: filepath.Clean(dir), ProcessName: exeName(exe), StoreID: str("slug"), Prefix: prefix}
		if f.Name == "" {
			f.Name = filepath.Base(f.Dir)
		}
		if u := "lutris:rungameid/" + strconv.FormatInt(id, 10); SafeLaunchURI(u) {
			f.URI = u
		}
		out = append(out, f)
	}
	return out
}

// -------------------------------------------------------------- Bottles ---

// BottlesGames reads the programs added to each of Bottles' bottles (dir is
// its bottles folder): every bottle is a Wine prefix, and its bottle.yml
// lists the programs under External_Programs.
func BottlesGames(dir string) []FoundGame {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []FoundGame
	for _, d := range ents {
		root := filepath.Join(dir, d.Name())
		if real, err := filepath.EvalSymlinks(root); err == nil {
			root = real // a bottle kept in a folder of the user's choosing
		}
		b, err := os.ReadFile(filepath.Join(root, "bottle.yml"))
		if err != nil || len(b) > 4<<20 || !IsPrefix(root) {
			continue
		}
		cfg := readYAMLMap(string(b))
		bottle := yamlStr(cfg, "Name")
		if bottle == "" {
			bottle = d.Name()
		}
		progs, _ := cfg["External_Programs"].(map[string]any)
		keys := make([]string, 0, len(progs))
		for k := range progs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		drive := filepath.Join(root, "drive_c")
		for _, k := range keys {
			p, ok := progs[k].(map[string]any)
			if !ok || yamlStr(p, "removed") == "true" {
				continue
			}
			exe := bottlesPath(root, yamlStr(p, "path"))
			if exe == "" || !isFile(exe) {
				continue
			}
			name := yamlStr(p, "name")
			if name == "" {
				name = exeName(exe)
			}
			gameDir := GameRoot(exe, []string{filepath.Join(drive, "Program Files"), filepath.Join(drive, "Program Files (x86)"), drive, filepath.Join(drive, "Games")})
			if gameDir == "" {
				gameDir = filepath.Dir(exe)
			}
			out = append(out, FoundGame{Key: "bottles:" + bottle + ":" + k, Name: name, Source: "bottles", Exe: exe,
				Dir: gameDir, ProcessName: exeName(exe), StoreID: bottle, Prefix: root})
		}
	}
	return out
}

// bottlesPath turns a program's path in bottle.yml into one on this
// system: a Windows path (C:\Games\x.exe) is inside the bottle's drives.
func bottlesPath(root, p string) string {
	if len(p) >= 3 && p[1] == ':' && (p[2] == '\\' || p[2] == '/') {
		letter := strings.ToLower(p[:1])
		rest := filepath.FromSlash(strings.ReplaceAll(p[3:], `\`, "/"))
		if letter == "c" {
			return filepath.Join(root, "drive_c", rest)
		}
		return filepath.Join(root, "dosdevices", letter+":", rest)
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return ""
}
