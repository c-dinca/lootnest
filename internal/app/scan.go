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
// drive, and desktop / Start menu shortcuts.
func (a *App) scanPC() []core.FoundGame {
	env := a.env
	var all []core.FoundGame
	all = append(all, env.ScanSteam()...)

	if pd := win.ProgramData(); pd != "" {
		all = append(all, core.EpicGames(filepath.Join(pd, "Epic", "EpicGamesLauncher", "Data", "Manifests"))...)
	}
	for _, g := range win.GOGGames() {
		exe := g.Exe
		if !filepath.IsAbs(exe) {
			exe = filepath.Join(g.Path, exe)
		}
		if _, err := os.Stat(exe); err == nil {
			all = append(all, core.FoundGame{Key: "dir:" + strings.ToLower(filepath.Clean(g.Path)), Name: g.Name, Source: "gog", Exe: exe, Dir: filepath.Clean(g.Path)})
		}
	}

	roots, containers := gameFolders(env)
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
	return env.FinishFound(all)
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
	if g.Launch.Type == "exe" && f.Dir != "" {
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
		func() {
			defer func() {
				if r := recover(); r != nil {
					a.logf("ERROR scan crashed: %v", r)
				}
			}()
			list = a.scanPC()
		}()
		fresh := len(list)
		list = core.MergeFound(saved, list, dirExists)
		a.mu.Lock()
		a.found = list
		a.scanning = false
		a.mu.Unlock()
		if err := a.store.SaveLibrary(list); err != nil {
			a.logf("ERROR saving the library: %v", err)
		}
		a.logf("Scan found %d games in %s (%d in the library)", fresh, time.Since(start).Round(100*time.Millisecond), len(list))
		a.emit("found", map[string]any{"scanning": false})
	}()
}

type FoundState struct {
	Scanning   bool             `json:"scanning"`   // first scan, nothing to show yet
	Refreshing bool             `json:"refreshing"` // a scan is updating the saved library
	Games      []core.FoundGame `json:"games"`
}

func (a *App) apiFound() FoundState {
	list := a.libraryGames()
	a.mu.Lock()
	scanning := a.scanning || a.found == nil // nil: never scanned on this PC
	a.mu.Unlock()
	return FoundState{Scanning: scanning && len(list) == 0, Refreshing: scanning, Games: a.visibleFound(list)}
}

func (a *App) apiScan() { a.startScan() }

func (a *App) apiIgnoreFound(key string) error {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	cfg.IgnoredFound = append(cfg.IgnoredFound, key)
	kept := cfg.AddedGames[:0]
	for _, g := range cfg.AddedGames {
		if g.Key != key {
			kept = append(kept, g)
		}
	}
	cfg.AddedGames = kept
	return a.saveConfig(cfg)
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
	ignored := cfg.IgnoredFound[:0]
	for _, k := range cfg.IgnoredFound {
		if k != f.Key {
			ignored = append(ignored, k)
		}
	}
	cfg.IgnoredFound = ignored
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
