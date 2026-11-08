//go:build windows

package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// scanPC finds games from Steam, Epic, GOG, common game folders on every
// drive, and desktop / Start menu shortcuts. It also says, per source, what
// it found there and why it found nothing.
func (a *App) scanPC() ([]core.FoundGame, []core.SourceCheck) {
	env := a.env
	var all []core.FoundGame
	all = append(all, env.ScanSteam()...)

	// The other launchers: from Windows' installed programs, the Xbox app's
	// folders, the itch app's and Hydra's library.
	var entries []core.UninstallEntry
	for _, e := range win.InstalledPrograms() {
		entries = append(entries, core.UninstallEntry(e))
	}
	installed := core.InstalledLaunchers(entries)
	errs := map[string]error{}
	there := func(src, dir string) {
		if dirExists(dir) {
			installed[src] = dir
		}
	}
	if env.SteamPath != "" {
		installed["steam"] = env.SteamPath
	}
	if pd := win.ProgramData(); pd != "" {
		manifests := filepath.Join(pd, "Epic", "EpicGamesLauncher", "Data", "Manifests")
		there("epic", manifests)
		there("riot", filepath.Join(pd, "Riot Games"))
		all = append(all, core.EpicGames(manifests)...)
	}
	for _, g := range win.GOGGames() {
		exe := g.Exe
		if !filepath.IsAbs(exe) {
			exe = filepath.Join(g.Path, exe)
		}
		if _, err := os.Stat(exe); err == nil {
			all = append(all, core.FoundGame{Key: "dir:" + strings.ToLower(filepath.Clean(g.Path)), Name: g.Name, Source: "gog", Exe: exe, Dir: filepath.Clean(g.Path), StoreID: g.ID})
		}
	}

	all = append(all, core.LauncherGames(entries)...)
	xbox := win.XboxGameFolders()
	if len(xbox) > 0 {
		installed["xbox"] = xbox[0]
	}
	all = append(all, core.XboxGames(xbox)...)
	roots, containers := gameFolders(env)
	if ad := os.Getenv("APPDATA"); ad != "" {
		itch, hydraDB := filepath.Join(ad, "itch", "apps"), filepath.Join(ad, "hydralauncher", "hydra-db")
		there("itch", itch)
		there("hydra", hydraDB)
		all = append(all, core.ItchGames(itch)...)
		hydra, err := core.HydraGames(hydraDB, filepath.Join(DataDir(), "hydra-tmp"), containers)
		if err != nil {
			errs["hydra"] = err
			a.logf("ERROR reading Hydra's library: %v", err)
		}
		all = append(all, hydra...)
	}

	all = append(all, core.ScanFolders(roots, containers)...)

	// Shortcuts the user made (common for games installed by hand).
	windir := strings.ToLower(win.WindowsDir())
	for lnk, target := range win.ResolveShortcuts(win.ShortcutFiles()) {
		if !strings.EqualFold(filepath.Ext(target), ".exe") || (windir != "" && strings.HasPrefix(strings.ToLower(target), windir)) {
			continue
		}
		if _, err := os.Stat(target); err != nil {
			continue
		}
		base := strings.ToLower(filepath.Base(target))
		if base == "steam.exe" || strings.Contains(base, "launcher") || strings.Contains(base, "unins") || strings.Contains(base, "setup") ||
			strings.Contains(base, "install") || base == "lootnest.exe" {
			continue
		}
		root := core.GameRoot(target, containers)
		skip := false
		for _, c := range containers {
			if strings.EqualFold(filepath.Clean(root), filepath.Clean(c)) {
				skip = true
				break
			}
		}
		if skip || core.SkipShortcut(target, root) || !core.LooksLikeGame(root) {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(lnk), filepath.Ext(lnk))
		all = append(all, core.FoundGame{Key: "dir:" + strings.ToLower(filepath.Clean(root)), Name: name, Source: "shortcut", Exe: target, Dir: root})
	}
	list := env.FinishFound(all)
	return list, core.SourceChecks(list, installed, errs)
}

// gameFolders lists the folders the scan looks inside (roots) and the
// folders that hold games but are not games themselves (containers).
func gameFolders(env *core.Env) (roots, containers []string) {
	home := os.Getenv("USERPROFILE")
	containers = []string{home, win.Desktop(), win.Documents(), win.ProgramFiles(), win.ProgramFilesX86(), win.WindowsDir()}
	gameDirNames := []string{"Games", "Game", "Jocuri", "Gaming", "Juegos", "Jeux", "Spiele", "Epic Games", "GOG Games",
		`GOG Galaxy\Games`, "PC Games"}
	for _, d := range win.FixedDrives() {
		containers = append(containers, d)
		roots = append(roots, d)
		for _, n := range gameDirNames {
			p := filepath.Join(d, n)
			if st, err := os.Stat(p); err == nil && st.IsDir() {
				roots = append(roots, p)
				containers = append(containers, p)
			}
		}
	}
	for _, pf := range []string{win.ProgramFiles(), win.ProgramFilesX86()} {
		if pf == "" {
			continue
		}
		for _, n := range []string{"Epic Games", "Rockstar Games", "EA Games", "Electronic Arts", "Ubisoft", `Ubisoft\Ubisoft Game Launcher\games`,
			`GOG Galaxy\Games`, "Riot Games", "Bethesda.net Launcher\\games", "Square Enix", "Games"} {
			p := filepath.Join(pf, n)
			if st, err := os.Stat(p); err == nil && st.IsDir() {
				roots = append(roots, p)
				containers = append(containers, p)
			}
		}
	}
	if home != "" {
		p := filepath.Join(home, "Games")
		if _, err := os.Stat(p); err == nil {
			roots = append(roots, p)
			containers = append(containers, p)
		}
	}
	for _, lib := range core.SteamLibraries(env.SteamPath) {
		containers = append(containers, lib, filepath.Join(lib, "steamapps"), filepath.Join(lib, "steamapps", "common"))
	}
	return roots, containers
}

// libraryGames is every game in the Library without profiles: what the scan
// found plus what the user added by hand, with their names and launch
// options applied. a.mu must not be held.
func (a *App) libraryGames() []core.FoundGame {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.found == nil {
		a.found = a.store.LoadLibrary()
	}
	return core.ApplyLibraryPrefs(core.MergeAdded(a.found, a.cfg.AddedGames), a.cfg.LibraryPrefs)
}

// visibleFound hides games already in Lootnest and ones the user dismissed.
func (a *App) visibleFound(list []core.FoundGame) []core.FoundGame {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	ignored := map[string]bool{}
	for _, k := range cfg.IgnoredFound {
		ignored[k] = true
	}
	out := []core.FoundGame{}
	for _, f := range list {
		if ignored[f.Key] || a.isAdded(cfg, f) {
			continue
		}
		out = append(out, f)
	}
	return out
}

func (a *App) isAdded(cfg *core.Config, f core.FoundGame) bool {
	for i := range cfg.Games {
		if a.isFoundGame(&cfg.Games[i], f) {
			return true
		}
	}
	return false
}

// isFoundGame reports whether a game with profiles is this scan result.
func (a *App) isFoundGame(g *core.Game, f core.FoundGame) bool {
	if f.AppID != 0 && g.Launch.Type == "steam" && g.Launch.AppID == f.AppID {
		return true
	}
	if g.Launch.Type == "uri" && f.URI != "" && g.Launch.URI == f.URI {
		return true
	}
	if (g.Launch.Type == "exe" || g.Launch.Type == "uri") && g.Launch.Exe != "" && f.Dir != "" {
		if exe, err := a.env.ResolveExe(g.Launch.Exe); err == nil {
			if rel, err := filepath.Rel(strings.ToLower(f.Dir), strings.ToLower(exe)); err == nil && !strings.HasPrefix(rel, "..") {
				return true
			}
		}
	}
	return false
}

// gameSources says where each game with profiles came from (epic, gog, ...),
// from the scan result it was added from. Games added by hand have none.
func (a *App) gameSources(cfg *core.Config) map[string]string {
	found := a.libraryGames()
	out := map[string]string{}
	for i := range cfg.Games {
		g := &cfg.Games[i]
		best := -1
		for _, f := range found {
			if len(f.Dir) > best && f.Source != "" && a.isFoundGame(g, f) {
				out[g.ID], best = f.Source, len(f.Dir)
			}
		}
	}
	return out
}

// startScan refreshes the library in the background. The first time, the
// library saved by the last scan is shown at once, so games don't vanish and
// reappear each time Lootnest opens.
func (a *App) startScan() {
	a.mu.Lock()
	if a.scanning {
		a.mu.Unlock()
		return
	}
	a.scanning = true
	if a.found == nil {
		a.found = a.store.LoadLibrary()
	}
	saved := append([]core.FoundGame{}, a.found...)
	a.mu.Unlock()
	a.emit("found", map[string]any{"scanning": true})
	go func() {
		start := time.Now()
		var list []core.FoundGame
		var checks []core.SourceCheck
		func() {
			defer func() {
				if r := recover(); r != nil {
					a.logf("ERROR scan crashed: %v", r)
				}
			}()
			list, checks = a.scanPC()
		}()
		fresh := len(list)
		list = core.MergeFound(saved, list, dirExists)
		a.mu.Lock()
		a.found = list
		a.scanning = false
		if checks != nil {
			a.sources, a.sourcesAt = checks, time.Now()
		}
		a.mu.Unlock()
		if err := a.store.SaveLibrary(list); err != nil {
			a.logf("ERROR saving the library: %v", err)
		}
		a.logf("Scan found %d games in %s (%d in the library)", fresh, time.Since(start).Round(100*time.Millisecond), len(list))
		a.emit("found", map[string]any{"scanning": false})
		go a.noteCopies()
		a.scanCapturesSoon()
	}()
}

type FoundState struct {
	Scanning   bool                `json:"scanning"`   // first scan, nothing to show yet
	Refreshing bool                `json:"refreshing"` // a scan is updating the saved library
	Games      []core.FoundGame    `json:"games"`
	Hidden     map[string]string   `json:"hidden"` // names of hidden games the library still has, by key
	Groups     []core.CopyGroup    `json:"groups"` // the same game from more than one store, shown once
	Acts       map[string][]string `json:"acts"`   // what each game's store can do for it (core.StoreActions), by key
}

func (a *App) apiFound() FoundState {
	list := a.libraryGames()
	a.mu.Lock()
	scanning := a.scanning || a.found == nil // nil: never scanned on this PC
	ignored := map[string]bool{}
	for _, k := range a.cfg.IgnoredFound {
		ignored[k] = true
	}
	separate, primary := a.cfg.Separate, a.cfg.Primary
	a.mu.Unlock()
	hidden := map[string]string{}
	for _, f := range list {
		if ignored[f.Key] {
			hidden[f.Key] = f.Name
		}
	}
	games := a.visibleFound(list)
	groups := core.GroupCopies(games, separate, primary, a.store.LoadRecent())
	if groups == nil {
		groups = []core.CopyGroup{}
	}
	acts := map[string][]string{}
	for _, f := range games {
		for _, act := range core.StoreActions(f.Source, storeIDOf(f.Source, f.AppID, f.StoreID)) {
			if act != "install" { // it's installed
				acts[f.Key] = append(acts[f.Key], act)
			}
		}
	}
	return FoundState{Scanning: scanning && len(list) == 0, Refreshing: scanning, Games: games, Hidden: hidden, Groups: groups, Acts: acts}
}

// apiSetPrimaryCopy chooses the copy Start with uses for a game from more
// than one store.
func (a *App) apiSetPrimaryCopy(group, key string) error {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	if cfg.Primary == nil {
		cfg.Primary = map[string]string{}
	}
	cfg.Primary[group] = key
	return a.saveConfig(cfg)
}

// apiSeparateCopies shows a game's copies from each store on their own
// (apart), or as one game again.
func (a *App) apiSeparateCopies(group string, apart bool) error {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	cfg.Separate = withoutKey(cfg.Separate, group)
	if apart {
		cfg.Separate = append(cfg.Separate, group)
	}
	return a.saveConfig(cfg)
}

func (a *App) apiScan() { a.startScan() }

// Sources is what the last scan since Lootnest started saw of each place
// games come from. At is zero before the first scan ends.
type Sources struct {
	At      time.Time          `json:"at"`
	Sources []core.SourceCheck `json:"sources"`
}

func (a *App) apiSources() Sources {
	a.mu.Lock()
	defer a.mu.Unlock()
	return Sources{At: a.sourcesAt, Sources: append([]core.SourceCheck{}, a.sources...)}
}

func (a *App) apiIgnoreFound(key string) error {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	cfg.IgnoredFound = append(cfg.IgnoredFound, key)
	cfg.AddedGames = withoutAdded(cfg.AddedGames, key)
	return a.saveConfig(cfg)
}

func withoutAdded(games []core.FoundGame, key string) []core.FoundGame {
	out := []core.FoundGame{}
	for _, g := range games {
		if g.Key != key {
			out = append(out, g)
		}
	}
	return out
}

// apiUnhideFound shows a hidden game in the Library again. The library
// still holds it, so no scan is needed.
func (a *App) apiUnhideFound(key string) error {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	cfg.IgnoredFound = withoutKey(cfg.IgnoredFound, key)
	return a.saveConfig(cfg)
}

func withoutKey(keys []string, key string) []string {
	out := []string{}
	for _, k := range keys {
		if k != key {
			out = append(out, k)
		}
	}
	return out
}

type AddResult struct {
	Added  []string `json:"added"`
	Failed []string `json:"failed"`
	Steam  int      `json:"steam"`
}

// gameFromFound turns a scan result into a game definition.
func gameFromFound(f core.FoundGame) core.Game {
	g := core.Game{Name: f.Name, ProcessName: f.ProcessName, PlayerArgs: map[string]string{}}
	if f.Source == "steam" {
		g.Launch = core.Launch{Type: "steam", AppID: f.AppID, Args: f.Args}
	} else if f.URI != "" && core.SafeLaunchURI(f.URI) {
		g.Launch = core.Launch{Type: "uri", URI: f.URI, Exe: f.Exe, Args: f.Args}
	} else {
		g.Launch = core.Launch{Type: "exe", Exe: f.Exe, Args: f.Args}
	}
	for _, p := range f.Paths {
		g.Paths = append(g.Paths, core.PathEntry{Path: p})
	}
	return g
}

func (a *App) apiFoundGame(key string) (core.FoundGame, error) {
	for _, f := range a.libraryGames() {
		if f.Key == key {
			return f, nil
		}
	}
	return core.FoundGame{}, fmt.Errorf("That game isn't in the scan results anymore; scan again.")
}

// apiAddFound adds the given scan results that have everything they need.
func (a *App) apiAddFound(keys []string) (AddResult, error) {
	res := AddResult{Added: []string{}, Failed: []string{}}
	want := map[string]bool{}
	for _, k := range keys {
		want[k] = true
	}
	found := a.libraryGames() // names and launch options carry over
	for _, f := range found {
		if !want[f.Key] {
			continue
		}
		if !f.Ready {
			res.Failed = append(res.Failed, f.Name+": needs a save location")
			continue
		}
		g, err := a.apiSaveGameQuiet(gameFromFound(f))
		if err != nil {
			res.Failed = append(res.Failed, f.Name+": "+err.Error())
			continue
		}
		res.Added = append(res.Added, g.Name)
		if g.Launch.Type == "steam" {
			res.Steam++
		}
	}
	if res.Steam > 0 {
		a.autoHooks()
	}
	return res, nil
}

type AddedGame struct {
	Key    string `json:"key,omitempty"`    // the Library game to open
	GameID string `json:"gameId,omitempty"` // or the game with profiles it already is
	Name   string `json:"name"`
}

// apiAddLibraryGame adds a game to the Library by hand, without profiles:
// an installed Steam game (kind "steam", appID) or any game by its exe
// (kind "exe", path). A game already there is just shown again.
func (a *App) apiAddLibraryGame(kind string, appID int, path string) (AddedGame, error) {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	var f core.FoundGame
	switch kind {
	case "steam":
		for _, g := range cfg.Games {
			if g.Launch.Type == "steam" && g.Launch.AppID == appID {
				return AddedGame{GameID: g.ID, Name: g.Name}, nil
			}
		}
		var sg *core.SteamGame
		for _, g := range core.InstalledSteamGames(a.env.SteamPath) {
			if g.AppID == appID {
				g := g
				sg = &g
				break
			}
		}
		if sg == nil {
			return AddedGame{}, errors.New("That Steam game isn't installed.")
		}
		f = core.FoundGame{Key: fmt.Sprintf("steam:%d", appID), Name: sg.Name, Source: "steam", AppID: appID, Dir: sg.InstallDir,
			Suggestions: []core.Suggestion{}, Paths: []string{}}
		if c := core.ProcessCandidates(sg.InstallDir); len(c) > 0 {
			f.ProcessName = c[0].Name
		}
	case "exe":
		exe, err := a.env.ResolveExe(path)
		if err != nil {
			return AddedGame{}, err
		}
		_, containers := gameFolders(a.env)
		f = a.env.ExeGame(exe, containers)
		for i := range cfg.Games {
			if a.isFoundGame(&cfg.Games[i], f) {
				return AddedGame{GameID: cfg.Games[i].ID, Name: cfg.Games[i].Name}, nil
			}
		}
	default:
		return AddedGame{}, errors.New("unknown kind of game")
	}

	// Show it again if it was hidden, and remember it so scans keep it.
	cfg.IgnoredFound = withoutKey(cfg.IgnoredFound, f.Key)
	known := false
	for _, g := range a.libraryGames() {
		if g.Key == f.Key && !g.Added && (f.Source == "steam" || g.Exe == f.Exe) {
			known = true // the scan already has it, exactly like this
			f.Name = g.Name
		}
	}
	if !known {
		replaced := false
		for i := range cfg.AddedGames {
			if cfg.AddedGames[i].Key == f.Key {
				cfg.AddedGames[i], replaced = f, true
			}
		}
		if !replaced {
			cfg.AddedGames = append(cfg.AddedGames, f)
		}
	}
	if err := a.saveConfig(cfg); err != nil {
		return AddedGame{}, err
	}
	a.logf("Added %s to the Library (%s)", f.Name, f.Key)
	for _, g := range a.libraryGames() {
		if g.Key == f.Key {
			f.Name = g.Name // with the user's own name for it, if any
		}
	}
	return AddedGame{Key: f.Key, Name: f.Name}, nil
}
