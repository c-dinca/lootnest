//go:build windows

package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Game settings that have nothing to do with profiles: the name a Library
// game is shown with, launch options, its folder, a desktop shortcut that
// starts it directly, and Steam's file check.
// ---------------------------------------------------------------------------

func (a *App) foundByKey(key string) (core.FoundGame, bool) {
	for _, f := range a.libraryGames() {
		if f.Key == key {
			return f, true
		}
	}
	return core.FoundGame{}, false
}

var errFoundGone = errors.New("That game isn't on the list anymore. Scan again from the Library.")

// apiSaveLibraryPref saves a Library game's name and launch options. An
// empty name puts back the name the scan found.
func (a *App) apiSaveLibraryPref(key, name, args string) error {
	f, ok := a.foundByKey(key)
	if !ok {
		return errFoundGone
	}
	p, err := core.CleanLibraryPref(core.LibraryPref{Name: name, Args: args})
	if err != nil {
		return err
	}
	scanName := f.Name
	if f.ScanName != "" {
		scanName = f.ScanName
	}
	if p.Name == scanName {
		p.Name = ""
	}
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	if cfg.LibraryPrefs == nil {
		cfg.LibraryPrefs = map[string]core.LibraryPref{}
	}
	p.Saves = cfg.LibraryPrefs[key].Saves // set on the game's page, not here
	if p.Empty() {
		delete(cfg.LibraryPrefs, key)
	} else {
		cfg.LibraryPrefs[key] = p
	}
	return a.saveConfig(cfg)
}

// apiSetGameSaves sets the save folders and files of a Library game for its
// cloud saves (none: Lootnest finds them again).
func (a *App) apiSetGameSaves(key string, paths []string) error {
	f, ok := a.foundByKey(key)
	if !ok {
		return errFoundGone
	}
	var saves []string
	for _, raw := range paths {
		r, err := a.env.Resolve(raw) // refuses drives, the user folder, AppData itself, ...
		if err != nil {
			return err
		}
		if _, err := os.Stat(r); err != nil {
			return fmt.Errorf("%s doesn't exist.", r)
		}
		if isInside(r, DataDir()) {
			return errors.New("Lootnest's own data folder can't be a game's save folder.")
		}
		t := a.env.Tokenize(r)
		dup := false
		for _, s := range saves {
			dup = dup || strings.EqualFold(s, t)
		}
		if !dup {
			saves = append(saves, t)
		}
	}
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	if cfg.LibraryPrefs == nil {
		cfg.LibraryPrefs = map[string]core.LibraryPref{}
	}
	p := cfg.LibraryPrefs[key]
	p.Saves = saves
	p, err := core.CleanLibraryPref(p)
	if err != nil {
		return err
	}
	if p.Empty() {
		delete(cfg.LibraryPrefs, key)
	} else {
		cfg.LibraryPrefs[key] = p
	}
	if err := a.saveConfig(cfg); err != nil {
		return err
	}
	st := loadGamesaves()
	delete(st, gameSavesKey(f)) // read them again
	saveGamesaves(st)
	return nil
}

// steamInstallDir is where Steam installed a game, or "".
func (a *App) steamInstallDir(appID int) string {
	for _, sg := range core.InstalledSteamGames(a.env.SteamPath) {
		if sg.AppID == appID {
			return sg.InstallDir
		}
	}
	return ""
}

// apiOpenGameFolder opens the folder a game is installed in. kind is
// "found" (id is a Library key) or "game" (id is a game with profiles).
func (a *App) apiOpenGameFolder(kind, id string) error {
	dir := ""
	switch kind {
	case "found":
		f, ok := a.foundByKey(id)
		if !ok {
			return errFoundGone
		}
		dir = f.Dir
	case "game":
		a.mu.Lock()
		g := a.cfg.Clone().Game(id)
		a.mu.Unlock()
		if g == nil {
			return errors.New("That game no longer exists.")
		}
		if g.Launch.Type == "steam" {
			dir = a.steamInstallDir(g.Launch.AppID)
		} else if exe, err := a.env.ResolveExe(g.Launch.Exe); err == nil {
			dir = filepath.Dir(exe)
		}
	}
	if dir == "" {
		return errors.New("Lootnest doesn't know where this game is installed.")
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return fmt.Errorf("%s isn't there anymore. Scan again from the Library.", dir)
	}
	return win.ShellOpen(dir)
}

// apiFoundShortcut puts a shortcut on the desktop that starts a Library
// game directly, with its launch options.
func (a *App) apiFoundShortcut(key string) (string, error) {
	f, ok := a.foundByKey(key)
	if !ok {
		return "", errFoundGone
	}
	desktop := win.Desktop()
	if desktop == "" {
		return "", errors.New("Couldn't find your Desktop folder.")
	}
	lnk := filepath.Join(desktop, strings.TrimSpace(badFileChars.ReplaceAllString(f.Name, " "))+".lnk")
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	known := false
	for _, s := range cfg.Shortcuts {
		if strings.EqualFold(s, lnk) {
			known = true
		}
	}
	if _, err := os.Stat(lnk); err == nil && !known {
		return "", fmt.Errorf("There's already a shortcut named %s on your desktop.", filepath.Base(lnk))
	}
	sc := win.Shortcut{Path: lnk, Description: "Start " + f.Name}
	if f.AppID > 0 {
		if a.env.SteamPath == "" {
			return "", errors.New("Steam wasn't found on this PC")
		}
		sc.Target = filepath.Join(a.env.SteamPath, "steam.exe")
		sc.Args = strings.TrimSpace(fmt.Sprintf("-applaunch %d %s", f.AppID, f.Args))
		sc.WorkDir = a.env.SteamPath
		sc.Icon = sc.Target
		if exe := core.FindExe(f.Dir, f.ProcessName); exe != "" {
			sc.Icon = exe
		}
	} else {
		exe, err := a.env.ResolveExe(f.Exe)
		if err != nil {
			return "", err
		}
		sc.Target, sc.Args, sc.WorkDir, sc.Icon = exe, f.Args, filepath.Dir(exe), exe
	}
	if err := win.CreateShortcut(sc); err != nil {
		return "", fmt.Errorf("couldn't create the shortcut: %w", err)
	}
	if !known {
		cfg.Shortcuts = append(cfg.Shortcuts, lnk)
		_ = a.saveConfig(cfg)
	}
	a.logf("Created shortcut %s", lnk)
	return lnk, nil
}

// apiSteamVerify asks Steam to check a game's files and download any that
// are missing or damaged.
func (a *App) apiSteamVerify(appID int) error {
	if appID <= 0 {
		return errors.New("That isn't a Steam game.")
	}
	if a.env.SteamPath == "" {
		return errors.New("Steam wasn't found on this PC")
	}
	a.logf("Asked Steam to check the files of app %d", appID)
	return win.ShellOpen("steam://validate/" + strconv.Itoa(appID))
}
