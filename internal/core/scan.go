package core

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// FoundGame is a game discovered on the PC that Lootnest could manage.
type FoundGame struct {
	Key    string `json:"key"` // stable id: "steam:730", "dir:d:\games\x"
	Name   string `json:"name"`
	Source string `json:"source"` // steam, epic, gog, folder, shortcut
	AppID  int    `json:"appId,omitempty"`
	Exe    string `json:"exe,omitempty"` // launch target for non-Steam games
	// URI starts the game through its launcher (see SafeLaunchURI), when
	// the launcher must start it.
	URI         string       `json:"uri,omitempty"`
	Dir         string       `json:"dir"`
	ProcessName string       `json:"processName"`
	Paths       []string     `json:"paths"` // proposed save/settings locations
	Suggestions []Suggestion `json:"suggestions"`
	Ready       bool         `json:"ready"` // has a confident set of paths
	Note        string       `json:"note,omitempty"`
	// Args are launch options the user set in the game's settings (from
	// Config.LibraryPrefs; never saved in library.json).
	Args string `json:"args,omitempty"`
	// Saves are the save locations the user chose (Config.LibraryPrefs).
	Saves []string `json:"saves,omitempty"`
	// ScanName is the name the scan found, when the user renamed the game.
	ScanName string `json:"scanName,omitempty"`
	// Added marks a game the user added to the Library by hand.
	Added bool `json:"added,omitempty"`
}

// MergeAdded puts the games the user added by hand into the scan results.
// A game the scan also found keeps the scan's details (such as its store),
// but the exe the user picked.
func MergeAdded(found, added []FoundGame) []FoundGame {
	out := append([]FoundGame{}, found...)
	for _, a := range added {
		matched := false
		for i := range out {
			if out[i].Key == a.Key {
				if a.Exe != "" {
					out[i].Exe, out[i].ProcessName = a.Exe, a.ProcessName
				}
				out[i].Added = true
				matched = true
				break
			}
		}
		if !matched {
			a.Added = true
			out = append(out, a)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out
}

// genericDirs are folder names inside a game's folder that never name the
// game: an exe in D:\Games\Hades\steam belongs to "Hades", not "steam".
var genericDirs = map[string]bool{
	"bin": true, "bin32": true, "bin64": true, "binaries": true, "win32": true, "win64": true, "x86": true, "x64": true,
	"game": true, "retail": true, "shipping": true, "release": true, "client": true, "app": true, "windows": true,
	"windowsnoeditor": true, "steam": true, "epic": true, "gog": true, "exe": true,
}

// ExeGame describes a game the user added to the Library by its exe: its
// folder, a name for it, and the save locations profiles would use.
func (e *Env) ExeGame(exe string, containers []string) FoundGame {
	exe = filepath.Clean(exe)
	root := GameRoot(exe, containers)
	for i := 0; i < 4 && genericDirs[strings.ToLower(filepath.Base(root))]; i++ {
		parent := filepath.Dir(root)
		if parent == root || isContainer(parent, containers) {
			break
		}
		root = parent
	}
	process := strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))
	name := prettyName(filepath.Base(root))
	if isContainer(root, containers) || genericDirs[strings.ToLower(filepath.Base(root))] {
		root = filepath.Dir(exe) // an exe straight in D:\Games: the exe names the game
		name = prettyName(process)
	}
	f := FoundGame{Key: "dir:" + strings.ToLower(root), Name: name, Source: "added", Exe: e.Tokenize(exe), Dir: root, ProcessName: process}
	f.Suggestions = e.SuggestPaths(f.Name, filepath.Base(root), 0)
	if f.Suggestions == nil {
		f.Suggestions = []Suggestion{}
	}
	f.Paths = ProposePaths(f.Suggestions, nil)
	if f.Paths == nil {
		f.Paths = []string{}
	}
	f.Ready = len(f.Paths) > 0
	return f
}

// ApplyLibraryPrefs returns the list with the user's names and launch
// options filled in.
func ApplyLibraryPrefs(list []FoundGame, prefs map[string]LibraryPref) []FoundGame {
	out := make([]FoundGame, len(list))
	for i, f := range list {
		if p, ok := prefs[f.Key]; ok {
			if p.Name != "" && p.Name != f.Name {
				f.ScanName, f.Name = f.Name, p.Name
			}
			f.Args = p.Args
			f.Saves = p.Saves
		}
		out[i] = f
	}
	return out
}

var gameMarkerFiles = map[string]bool{
	"steam_api.dll": true, "steam_api64.dll": true, "unityplayer.dll": true, "gameassembly.dll": true,
	"eossdk-win64-shipping.dll": true, "eossdk-win32-shipping.dll": true, "bink2w64.dll": true, "binkw64.dll": true,
	"binkw32.dll": true, "bink2w32.dll": true, "galaxy64.dll": true, "galaxy.dll": true, "fmod.dll": true,
	"fmodex.dll": true, "fmodex64.dll": true, "fmodstudio.dll": true, "fmodstudiol.dll": true, "phonon.dll": true,
	"steam_appid.txt": true,
}

var gameMarkerExt = map[string]bool{
	".pak": true, ".vpk": true, ".bik": true, ".bk2": true, ".bsa": true, ".ba2": true, ".forge": true,
	".rpf": true, ".wad": true, ".pck": true, ".assets": true, ".big": true, ".gpk": true, ".upk": true, ".ucas": true,
}

// hasChromium reports a folder that holds a Chromium, Electron, CEF or Qt
// WebEngine build (icudtl.dat or libcef.dll) in it or up to two levels
// below. Apps built on them (browsers, Discord, launchers, hardware tools)
// ship .pak files too, so in such a folder .pak files don't count as a game
// marker.
func hasChromium(dir string) bool {
	var walk func(d string, depth int) bool
	walk = func(d string, depth int) bool {
		ents, err := os.ReadDir(d)
		if err != nil || len(ents) > 3000 {
			return false
		}
		for _, e := range ents {
			if n := strings.ToLower(e.Name()); !e.IsDir() && (n == "icudtl.dat" || n == "libcef.dll") {
				return true
			}
		}
		if depth >= 2 {
			return false
		}
		for _, e := range ents {
			if e.IsDir() && walk(filepath.Join(d, e.Name()), depth+1) {
				return true
			}
		}
		return false
	}
	return walk(dir, 0)
}

// LooksLikeGame checks a folder (and two levels below) for files that game
// engines and game stores leave behind.
func LooksLikeGame(dir string) bool {
	chromium := hasChromium(dir)
	var walk func(d string, depth int) bool
	walk = func(d string, depth int) bool {
		ents, err := os.ReadDir(d)
		if err != nil || len(ents) > 3000 {
			return false
		}
		hasExe, hasEngine := false, false
		for _, e := range ents {
			n := strings.ToLower(e.Name())
			if e.IsDir() {
				if n == "engine" {
					hasEngine = true
				}
				if strings.HasSuffix(n, "_data") && exists(filepath.Join(d, e.Name(), "Managed")) {
					return true
				}
				continue
			}
			if gameMarkerFiles[n] || (gameMarkerExt[filepath.Ext(n)] && !(chromium && filepath.Ext(n) == ".pak")) {
				return true
			}
			if strings.HasSuffix(n, ".exe") {
				hasExe = true
			}
		}
		if hasEngine && (hasExe || depth == 0) && exists(filepath.Join(d, "Engine", "Binaries")) {
			return true // Unreal Engine layout
		}
		if depth >= 2 {
			return false
		}
		for _, e := range ents {
			if e.IsDir() {
				n := strings.ToLower(e.Name())
				if n == "redist" || n == "_commonredist" || n == "directx" || n == "locales" || n == "swiftshader" || strings.HasPrefix(n, "$") {
					continue
				}
				if walk(filepath.Join(d, e.Name()), depth+1) {
					return true
				}
			}
		}
		return false
	}
	return walk(dir, 0)
}

func isContainer(dir string, containers []string) bool {
	if filepath.Dir(dir) == dir {
		return true
	}
	for _, c := range containers {
		if samePath(dir, c) {
			return true
		}
	}
	return false
}

// GameRoot finds the game's top folder from its exe (walking up through
// Binaries\Win64 and similar, never into a container folder).
func GameRoot(exe string, containers []string) string {
	dir := filepath.Dir(exe)
	for i := 0; i < 3; i++ {
		parent := filepath.Dir(dir)
		if isContainer(parent, containers) || parent == dir {
			break
		}
		base := strings.ToLower(filepath.Base(dir))
		if base == "bin" || base == "binaries" || base == "win64" || base == "x64" || base == "x86" || base == "win32" ||
			base == "bin64" || base == "game" || base == "retail" || base == "shipping" ||
			exists(filepath.Join(parent, "Engine", "Binaries")) || LooksLikeGame(parent) && !LooksLikeGame(dir) {
			dir = parent
			continue
		}
		break
	}
	return dir
}

// MainExe picks the exe to launch in a game folder: one named like the
// folder if there is one at the top, else the biggest plausible exe.
func MainExe(dir string) (launch, process string) {
	cands := ProcessCandidates(dir)
	if len(cands) == 0 {
		return "", ""
	}
	process = cands[0].Name
	launch = filepath.Join(dir, cands[0].RelPath)
	key := norm(filepath.Base(dir))
	for _, c := range cands {
		if !strings.ContainsAny(c.RelPath, `\/`) && (norm(c.Name) == key || strings.Contains(key, norm(c.Name)) && len(norm(c.Name)) >= 4) {
			launch = filepath.Join(dir, c.RelPath)
			break
		}
	}
	return launch, process
}

// ProposePaths picks the save/settings locations to use without asking:
// the preset if known, else name-matched folders under 1 GB that don't overlap.
func ProposePaths(sugg []Suggestion, preset *Preset) []string {
	if preset != nil {
		return append([]string{}, preset.Paths...)
	}
	var out []string
	var picked []string
	for _, s := range sugg {
		if s.Size > 1<<30 {
			continue
		}
		overlap := false
		for _, p := range picked {
			if isWithin(s.Resolved, p) || isWithin(p, s.Resolved) {
				overlap = true
				break
			}
		}
		if overlap {
			continue
		}
		picked = append(picked, s.Resolved)
		out = append(out, s.Path)
		if len(out) == 3 {
			break
		}
	}
	return out
}

var skipSteamScan = map[int]bool{
	228980: true, 431960: true, 250820: true, 1070560: true, 1391110: true, 1628350: true, 1493710: true,
	2180100: true, 1887720: true, 961940: true, 1580130: true, 2348590: true, 3658110: true,
}

func skipSteamName(n string) bool {
	l := strings.ToLower(n)
	for _, w := range []string{"redistributable", "soundtrack", "dedicated server", "sdk", "proton", "steam linux runtime",
		"steamvr", "wallpaper engine", "benchmark", "steamworks"} {
		if strings.Contains(l, w) {
			return true
		}
	}
	return false
}

// ScanSteam lists installed Steam games as FoundGames.
func (e *Env) ScanSteam() []FoundGame {
	var out []FoundGame
	for _, g := range InstalledSteamGames(e.SteamPath) {
		if skipSteamScan[g.AppID] || skipSteamName(g.Name) {
			continue
		}
		if st, err := os.Stat(g.InstallDir); err != nil || !st.IsDir() {
			continue
		}
		f := FoundGame{Key: "steam:" + itoa(g.AppID), Name: g.Name, Source: "steam", AppID: g.AppID, Dir: g.InstallDir}
		var preset *Preset
		if p, ok := KnownSteamGames[g.AppID]; ok {
			preset = &p
			f.ProcessName = p.ProcessName
			f.Note = p.Note
		} else if c := ProcessCandidates(g.InstallDir); len(c) > 0 {
			f.ProcessName = c[0].Name
		}
		f.Suggestions = e.SuggestPaths(g.Name, filepath.Base(g.InstallDir), g.AppID)
		f.Paths = ProposePaths(f.Suggestions, preset)
		f.Ready = f.ProcessName != "" && len(f.Paths) > 0
		out = append(out, f)
	}
	return out
}

type epicManifest struct {
	DisplayName          string   `json:"DisplayName"`
	InstallLocation      string   `json:"InstallLocation"`
	LaunchExecutable     string   `json:"LaunchExecutable"`
	AppName              string   `json:"AppName"`
	CatalogNamespace     string   `json:"CatalogNamespace"`
	CatalogItemID        string   `json:"CatalogItemId"`
	BIsIncompleteInstall bool     `json:"bIsIncompleteInstall"`
	AppCategories        []string `json:"AppCategories"`
}

// EpicGames reads Epic Games Launcher manifests from
// %ProgramData%\Epic\EpicGamesLauncher\Data\Manifests.
func EpicGames(manifestDir string) []FoundGame {
	files, _ := filepath.Glob(filepath.Join(manifestDir, "*.item"))
	var out []FoundGame
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var m epicManifest
		if json.Unmarshal(b, &m) != nil || m.BIsIncompleteInstall || m.DisplayName == "" || m.InstallLocation == "" || m.LaunchExecutable == "" {
			continue
		}
		isGame := len(m.AppCategories) == 0
		for _, c := range m.AppCategories {
			if strings.EqualFold(c, "games") {
				isGame = true
			}
		}
		if !isGame {
			continue
		}
		exe := filepath.Join(m.InstallLocation, filepath.FromSlash(m.LaunchExecutable))
		if !exists(exe) {
			continue
		}
		f := FoundGame{Key: "dir:" + strings.ToLower(filepath.Clean(m.InstallLocation)), Name: m.DisplayName, Source: "epic",
			Exe: exe, Dir: filepath.Clean(m.InstallLocation)}
		// Through the Epic launcher, so the game is signed in like when it
		// starts from Epic.
		if m.CatalogNamespace != "" && m.CatalogItemID != "" && m.AppName != "" {
			if u := "com.epicgames.launcher://apps/" + url.PathEscape(m.CatalogNamespace) + "%3A" + url.PathEscape(m.CatalogItemID) + "%3A" +
				url.PathEscape(m.AppName) + "?action=launch&silent=true"; SafeLaunchURI(u) {
				f.URI = u
			}
		}
		out = append(out, f)
	}
	return out
}

// versionedDir matches app install folders named after a version, such as
// "Battle.net.17709" or Discord's "app-1.0.9187".
var versionedDir = regexp.MustCompile(`(^app-\d)|([.-]\d{3,}$)`)

// notGameDir reports folder names that hold launchers, runtimes or
// installers rather than a game.
func notGameDir(name string) bool {
	n := strings.ToLower(name)
	if versionedDir.MatchString(n) {
		return true
	}
	for _, w := range []string{"launcher", "online services", "social club", "redist", "installer", "setup"} {
		if strings.Contains(n, w) {
			return true
		}
	}
	return false
}

// vendorDirs are hardware and driver makers' install folders: their tools
// (AMD Software, G HUB, iCUE...) are never games.
var vendorDirs = map[string]bool{
	"amd": true, "ati technologies": true, "nvidia": true, "nvidia corporation": true, "intel": true, "realtek": true,
	"logitech": true, "logishrd": true, "lghub": true, "razer": true, "corsair": true, "steelseries": true,
	"asus": true, "msi": true, "gigabyte": true, "nzxt": true,
}

// SkipShortcut reports shortcuts that point at a launcher, installer or
// hardware tool rather than a game: root is the game folder GameRoot found
// for the target exe.
func SkipShortcut(target, root string) bool {
	if notGameDir(filepath.Base(root)) {
		return true // e.g. Ubisoft Game Launcher, which has the games inside it
	}
	for _, part := range strings.Split(filepath.ToSlash(filepath.Dir(target)), "/") {
		if vendorDirs[strings.ToLower(part)] {
			return true
		}
	}
	return false
}

// ScanFolders looks one level inside each root (and inside known game
// folders like D:\Games) for folders that look like games.
func ScanFolders(roots []string, containers []string) []FoundGame {
	var out []FoundGame
	seen := map[string]bool{}
	for _, root := range roots {
		ents, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if !e.IsDir() {
				continue
			}
			n := strings.ToLower(e.Name())
			if strings.HasPrefix(n, "$") || strings.HasPrefix(n, ".") || n == "windows" || n == "users" || n == "programdata" ||
				n == "system volume information" || n == "recovery" || n == "perflogs" || n == "steamlibrary" || n == "steam" ||
				n == "msocache" || n == "intel" || n == "amd" || n == "nvidia" || n == "drivers" || n == "windowsapps" {
				continue
			}
			dir := filepath.Join(root, e.Name())
			if notGameDir(e.Name()) || isContainer(dir, containers) || seen[strings.ToLower(dir)] {
				continue
			}
			seen[strings.ToLower(dir)] = true
			if !LooksLikeGame(dir) {
				continue
			}
			launch, proc := MainExe(dir)
			if launch == "" {
				continue
			}
			out = append(out, FoundGame{Key: "dir:" + strings.ToLower(filepath.Clean(dir)), Name: prettyName(e.Name()), Source: "folder",
				Exe: launch, Dir: dir, ProcessName: proc})
		}
	}
	return out
}

// prettyName turns "Hollow.Knight.v1.5-GOG" style folder names into a title.
func prettyName(s string) string {
	// "Words.Joined.By.Dots-Tag": what follows the last hyphen isn't part of
	// the title (a version, store or packager tag). Names with spaces, or
	// with a hyphen inside the title ("Spider-Man.2"), are left alone.
	if !strings.Contains(s, " ") && strings.Count(s, ".") >= 2 {
		if i := strings.LastIndex(s, "-"); i > 0 && i > strings.LastIndex(s, ".") {
			s = s[:i]
		}
	}
	s = strings.NewReplacer(".", " ", "_", " ").Replace(s)
	f := strings.Fields(s)
	var keep []string
	for _, w := range f {
		l := strings.ToLower(w)
		if strings.HasPrefix(l, "v1") || strings.HasPrefix(l, "v2") || strings.HasPrefix(l, "build") || strings.HasSuffix(l, "-gog") {
			break
		}
		keep = append(keep, w)
	}
	if len(keep) == 0 {
		return strings.TrimSpace(s)
	}
	return strings.Join(keep, " ")
}

// FinishFound fills in exe/process/paths for games found by folder, Epic,
// GOG or shortcut, and drops duplicates and folders inside other games.
func (e *Env) FinishFound(list []FoundGame) []FoundGame {
	sort.SliceStable(list, func(i, j int) bool { return sourceRank(list[i].Source) < sourceRank(list[j].Source) })
	var out []FoundGame
	for _, f := range list {
		if f.Source != "added" && f.Source != "steam" && IsLauncherApp(f.Name) {
			continue
		}
		dup := false
		for _, o := range out {
			if o.Key == f.Key || (f.Dir != "" && o.Dir != "" && (isWithin(f.Dir, o.Dir) || isWithin(o.Dir, f.Dir))) {
				dup = true
				break
			}
		}
		if dup {
			continue
		}
		if f.Source != "steam" {
			if f.ProcessName == "" {
				_, f.ProcessName = MainExe(f.Dir)
				if f.ProcessName == "" && f.Exe != "" {
					f.ProcessName = strings.TrimSuffix(filepath.Base(f.Exe), filepath.Ext(f.Exe))
				}
			}
			f.Suggestions = e.SuggestPaths(f.Name, filepath.Base(f.Dir), 0)
			f.Paths = ProposePaths(f.Suggestions, nil)
			f.Ready = f.ProcessName != "" && f.Exe != "" && len(f.Paths) > 0
			if f.Exe != "" {
				f.Exe = e.Tokenize(f.Exe)
			}
		}
		if f.Suggestions == nil {
			f.Suggestions = []Suggestion{}
		}
		if f.Paths == nil {
			f.Paths = []string{}
		}
		out = append(out, f)
	}
	sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out
}

func sourceRank(s string) int {
	switch s {
	case "steam":
		return 0
	case "epic", "gog", "xbox", "ubisoft", "ea", "battlenet", "riot", "amazon", "rockstar", "itch":
		return 1
	case "shortcut":
		return 2
	}
	return 3
}

// ---------------------------------------------------------------------------
// The library is saved, so it shows at once when Lootnest opens; the scan
// then refreshes it in the background.
// ---------------------------------------------------------------------------

func (s *Store) LibraryPath() string { return filepath.Join(s.Dir, "library.json") }

// LoadLibrary returns the games found by the last scan, or nil.
func (s *Store) LoadLibrary() []FoundGame {
	b, err := os.ReadFile(s.LibraryPath())
	if err != nil {
		return nil
	}
	var list []FoundGame
	if json.Unmarshal(b, &list) != nil {
		return nil
	}
	return list
}

func (s *Store) SaveLibrary(list []FoundGame) error {
	if list == nil {
		list = []FoundGame{}
	}
	return writeJSONAtomic(s.LibraryPath(), list)
}

// MergeFound combines a fresh scan with the saved library. Everything the
// scan found is kept, as found. A saved game the scan missed stays while its
// folder still exists (a slow drive, a scan rule that changed), and is
// dropped once its folder is gone (the game was uninstalled).
func MergeFound(saved, fresh []FoundGame, stillThere func(dir string) bool) []FoundGame {
	out := append([]FoundGame{}, fresh...)
	have := map[string]bool{}
	for _, f := range fresh {
		have[f.Key] = true
	}
	for _, f := range saved {
		// A launcher an earlier scan took for a game is left out now.
		if have[f.Key] || f.Dir == "" || !stillThere(f.Dir) || f.Source != "added" && IsLauncherApp(f.Name) {
			continue
		}
		out = append(out, f)
		have[f.Key] = true
	}
	sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out
}
