//go:build windows

package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Cloud saves for Library games outside Steam (core/gamesaves.go): their
// saves are found (Ludusavi's list from PCGamingWiki when "Find save
// locations online" is on, else by the game's name), and a new version goes
// to the cloud folder whenever they changed and the game isn't running.
// ---------------------------------------------------------------------------

const (
	saveDBEvery    = 7 * 24 * time.Hour
	gameSavesEvery = 10 * time.Minute
	gameSavesKeep  = 10
)

var (
	saveDBMu   sync.Mutex
	saveDB     *core.SaveDB
	saveDBTime time.Time
	gameSaveMu sync.Mutex // one game-saves job at a time
)

func saveDBPath() string { return filepath.Join(DataDir(), "savedb.json.gz") }

// loadSaveDB is the save locations list, or nil without one.
func loadSaveDB() *core.SaveDB {
	saveDBMu.Lock()
	defer saveDBMu.Unlock()
	if saveDB == nil {
		if db, err := core.LoadSaveDB(saveDBPath()); err == nil {
			saveDB = db
			if st, err := os.Stat(saveDBPath()); err == nil {
				saveDBTime = st.ModTime()
			}
		}
	}
	return saveDB
}

// refreshSaveDB downloads the list when it's over a week old (or forced).
func (a *App) refreshSaveDB(force bool) error {
	a.mu.Lock()
	on := a.cfg.Settings.SaveDB
	a.mu.Unlock()
	if !on {
		return nil
	}
	loadSaveDB()
	saveDBMu.Lock()
	fresh := !saveDBTime.IsZero() && time.Since(saveDBTime) < saveDBEvery
	saveDBMu.Unlock()
	if fresh && !force {
		return nil
	}
	req, err := http.NewRequest("GET", core.SaveDBURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", Name+"/"+Version)
	client := &http.Client{Timeout: 3 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("couldn't download the save locations list: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("the save locations list answered %s", resp.Status)
	}
	db, err := core.ParseSaveDB(io.LimitReader(resp.Body, 100<<20))
	if err != nil {
		return err
	}
	if err := core.SaveDBFile(saveDBPath(), db); err != nil {
		return err
	}
	saveDBMu.Lock()
	saveDB, saveDBTime = db, time.Now()
	saveDBMu.Unlock()
	a.logf("Save locations list: %d games", len(db.Games))
	return nil
}

// gameSaveSource says where a game's save locations came from.
type gameLocs struct {
	Locs   []core.SaveLocation
	Source string // "list" (PCGamingWiki), "name" (folders named like the game), ""
}

// savesOf finds a Library game's saves on this PC.
func (a *App) savesOf(f core.FoundGame) gameLocs {
	base := ""
	if f.Dir != "" {
		base = f.Dir
	}
	// The ones the user chose come first.
	if len(f.Saves) > 0 {
		var locs []core.SaveLocation
		for _, p := range f.Saves {
			if r, err := a.env.Resolve(p); err == nil {
				if _, err := os.Stat(r); err == nil {
					locs = append(locs, core.SaveLocation{Path: p, Resolved: r, Size: core.Size(r)})
				}
			}
		}
		return gameLocs{locs, "you"}
	}
	if db := loadSaveDB(); db != nil {
		names := []string{f.Name, f.ScanName}
		if base != "" {
			names = append(names, filepath.Base(base))
		}
		if dg := db.Find(f.AppID, names...); dg != nil {
			if locs := a.env.FindSaves(dg, base); len(locs) > 0 {
				return gameLocs{locs, "list"}
			}
		}
	}
	var locs []core.SaveLocation
	for _, p := range f.Paths {
		r, err := a.env.Resolve(p)
		if _, serr := os.Stat(r); err != nil || serr != nil {
			continue
		}
		locs = append(locs, core.SaveLocation{Path: p, Resolved: r, Size: core.Size(r)})
	}
	if len(locs) > 0 {
		return gameLocs{locs, "name"}
	}
	return gameLocs{}
}

// gameSavesOn says whether a Library game gets cloud saves: outside Steam
// (Steam keeps its own), and without profiles (those sync per player).
func gameSavesOn(f core.FoundGame) bool { return f.Source != "steam" && f.AppID == 0 }

func gameSavesKey(f core.FoundGame) string { return core.GameSavesKey(f.Name, f.URI) }

// gamesavesState is gamesaves.json: what each game's saves were at its last
// version, so an unchanged game isn't read again.
type gamesavesState map[string]struct {
	Quick string    `json:"quick"` // sizes and times of the files
	Hash  string    `json:"hash"`
	At    time.Time `json:"at"`
}

func gamesavesPath() string { return filepath.Join(DataDir(), "gamesaves.json") }

func loadGamesaves() gamesavesState {
	st := gamesavesState{}
	if b, err := os.ReadFile(gamesavesPath()); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	return st
}

func saveGamesaves(st gamesavesState) {
	b, _ := json.Marshal(st)
	_ = os.WriteFile(gamesavesPath()+".tmp", b, 0o644)
	_ = os.Rename(gamesavesPath()+".tmp", gamesavesPath())
}

// quickPrint is a cheap fingerprint of the files: names, sizes and times.
func quickPrint(locs []core.SaveLocation) string {
	var parts []string
	for _, l := range locs {
		_ = filepath.Walk(l.Resolved, func(p string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				parts = append(parts, fmt.Sprintf("%s|%d|%d", strings.ToLower(p), info.Size(), info.ModTime().UnixNano()))
			}
			return nil
		})
	}
	sort.Strings(parts)
	return strings.Join(parts, "\n")
}

// backUpGame makes a new version of a game's saves when they changed.
func (a *App) backUpGame(c *core.Cloud, f core.FoundGame, force bool) (*core.SaveVersion, error) {
	gl := a.savesOf(f)
	if len(gl.Locs) == 0 {
		return nil, errors.New("Lootnest doesn't know where this game keeps its saves.")
	}
	key := gameSavesKey(f)
	st := loadGamesaves()
	quick := quickPrint(gl.Locs)
	if !force && st[key].Quick == quick {
		return nil, nil
	}
	hash, err := core.SavesHash(gl.Locs)
	if err != nil {
		return nil, err
	}
	e := st[key]
	e.Quick = quick
	if !force && e.Hash == hash {
		st[key] = e
		saveGamesaves(st)
		return nil, nil
	}
	if vs := c.GameVersions(key); !force && len(vs) > 0 && vs[0].Hash == hash {
		e.Hash, e.At = hash, vs[0].Created
		st[key] = e
		saveGamesaves(st)
		return nil, nil // another PC made the same version
	}
	v, err := c.SaveGameVersion(key, f.Name, gl.Locs, hash, gameSavesKeep)
	if err != nil {
		return nil, err
	}
	e.Hash, e.At = hash, v.Created
	st[key] = e
	saveGamesaves(st)
	a.logf("Cloud saves: new version of %s (%d bytes)", f.Name, v.Size)
	return v, nil
}

// watchGameSaves backs up the saves of games outside Steam that changed,
// every few minutes, while the game isn't running.
func (a *App) watchGameSaves() {
	time.Sleep(90 * time.Second)
	for {
		if err := a.refreshSaveDB(false); err != nil {
			a.logf("Save locations list: %v", err)
		}
		a.backUpChangedGames()
		time.Sleep(gameSavesEvery)
	}
}

func (a *App) backUpChangedGames() {
	a.mu.Lock()
	s := a.cfg.Settings
	a.mu.Unlock()
	if !s.GameSaves {
		return
	}
	c, err := a.cloud(core.Settings{CloudSync: true, BackupDir: s.BackupDir, BackupKey: s.BackupKey, BackupSalt: s.BackupSalt})
	if c == nil || err != nil {
		return
	}
	gameSaveMu.Lock()
	defer gameSaveMu.Unlock()
	procs, _ := win.ProcessNames()
	made := 0
	for _, f := range a.visibleFound(a.libraryGames()) {
		if !gameSavesOn(f) || f.ProcessName != "" && procs[strings.ToLower(strings.TrimSuffix(f.ProcessName, ".exe"))] {
			continue
		}
		if v, err := a.backUpGame(c, f, false); err == nil && v != nil {
			made++
		}
	}
	if made > 0 {
		a.emit("gameSaves", map[string]any{"changed": true})
	}
}

// ------------------------------------------------------------------ API ---

type GameSavesInfo struct {
	Key       string              `json:"key"`
	Locations []core.SaveLocation `json:"locations"`
	Source    string              `json:"source"`
	Versions  []core.SaveVersion  `json:"versions"`
	Error     string              `json:"error,omitempty"`
	Off       string              `json:"off,omitempty"` // why there are none: "steam", "nocloud", "disabled", "locked"
}

// apiGameSaves sends a Library game's cloud saves as a "gameSaves" event.
func (a *App) apiGameSaves(key string) {
	go func() {
		info := GameSavesInfo{Key: key, Locations: []core.SaveLocation{}, Versions: []core.SaveVersion{}}
		defer func() { a.emit("gameSaves", map[string]any{"info": info}) }()
		f, ok := a.foundByKey(key)
		if !ok {
			info.Error = errFoundGone.Error()
			return
		}
		if !gameSavesOn(f) {
			info.Off = "steam"
			return
		}
		gl := a.savesOf(f)
		info.Locations, info.Source = append(info.Locations, gl.Locs...), gl.Source
		a.mu.Lock()
		s := a.cfg.Settings
		a.mu.Unlock()
		switch {
		case s.BackupDir == "":
			info.Off = "nocloud"
			return
		case !s.GameSaves:
			info.Off = "disabled"
		}
		c, err := a.cloud(core.Settings{CloudSync: true, BackupDir: s.BackupDir, BackupKey: s.BackupKey, BackupSalt: s.BackupSalt})
		if errors.Is(err, errCloudPassword) {
			info.Off = "locked"
			return
		}
		if err != nil {
			info.Error = err.Error()
			return
		}
		info.Versions = append(info.Versions, c.GameVersions(gameSavesKey(f))...)
	}()
}

func (a *App) gameCloud() (*core.Cloud, error) {
	a.mu.Lock()
	s := a.cfg.Settings
	a.mu.Unlock()
	if s.BackupDir == "" {
		return nil, errors.New("Choose a cloud folder on Cloud saves first.")
	}
	c, err := a.cloud(core.Settings{CloudSync: true, BackupDir: s.BackupDir, BackupKey: s.BackupKey, BackupSalt: s.BackupSalt})
	if err != nil {
		return nil, err
	}
	return c, nil
}

// apiGameSaveNow makes a new version now, changed or not.
func (a *App) apiGameSaveNow(key string) error {
	f, ok := a.foundByKey(key)
	if !ok {
		return errFoundGone
	}
	c, err := a.gameCloud()
	if err != nil {
		return err
	}
	go func() {
		out := map[string]any{"key": key, "saved": true}
		defer func() { a.emit("gameSaves", out); a.apiGameSaves(key) }()
		gameSaveMu.Lock()
		defer gameSaveMu.Unlock()
		if _, err := a.backUpGame(c, f, true); err != nil {
			out["error"] = err.Error()
		}
	}()
	return nil
}

// apiGameSaveRestore puts a version in place. The saves here are first
// kept in backups\<game>\..., and the game must not be running.
func (a *App) apiGameSaveRestore(key, id string) error {
	f, ok := a.foundByKey(key)
	if !ok {
		return errFoundGone
	}
	if procs, _ := win.ProcessNames(); f.ProcessName != "" && procs[strings.ToLower(strings.TrimSuffix(f.ProcessName, ".exe"))] {
		return fmt.Errorf("%s is running; close it first.", f.Name)
	}
	c, err := a.gameCloud()
	if err != nil {
		return err
	}
	go func() {
		out := map[string]any{"key": key, "restored": true}
		defer func() { a.emit("gameSaves", out); a.apiGameSaves(key) }()
		gameSaveMu.Lock()
		defer gameSaveMu.Unlock()
		gl := a.savesOf(f)
		keep := filepath.Join(a.store.BackupsDir(), "saves-"+gameSavesKey(f), time.Now().Format("20060102-150405")+"-before-restore")
		for i, l := range gl.Locs {
			if err := core.Mirror(l.Resolved, filepath.Join(keep, fmt.Sprint(i))); err != nil {
				out["error"] = "Nothing was restored: Lootnest couldn't keep the saves here first (" + err.Error() + ")."
				return
			}
		}
		tmp := filepath.Join(DataDir(), "saves-tmp-"+time.Now().Format("150405.000"))
		skipped, err := c.RestoreGameVersion(gameSavesKey(f), id, gl.Locs, a.env.Resolve, tmp)
		if err != nil {
			out["error"] = err.Error()
			return
		}
		st := loadGamesaves()
		delete(st, gameSavesKey(f)) // read them again next time
		saveGamesaves(st)
		a.logf("Cloud saves: restored %s from version %s (the saves before are in %s)", f.Name, id, keep)
		out["skipped"] = skipped
	}()
	return nil
}
