//go:build windows

package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Storage (core/sizes.go, core/movegame.go): what each Library game takes on
// this PC, counted in the background after a scan, one game at a time, and
// moving a folder game to another drive. Steam games move with Game Mover
// (mover.go); other stores' games with their own launchers.
// ---------------------------------------------------------------------------

// GameSize is one Library game's sizes, all approximate.
type GameSize struct {
	Key      string `json:"key"` // "game:<id>" or the Library's key
	Dir      string `json:"dir,omitempty"`
	Install  int64  `json:"install"`
	Saves    int64  `json:"saves"`
	Profiles int64  `json:"profiles"`
	// Copies are Lootnest's own: copies before swaps, the safety net, save
	// points and the files kept from updates.
	Copies int64 `json:"copies"`
}

// Drive is one of the PC's drives.
type Drive struct {
	Root  string `json:"root"` // C:\
	Free  int64  `json:"free"`
	Size  int64  `json:"size"`
	Games string `json:"games"` // where a moved game goes by default
}

const (
	sizeEvery = 30 * time.Minute       // a pass after a scan at most this often
	sizePause = 150 * time.Millisecond // between games, so a pass stays in the background
)

var (
	sizeMu   sync.Mutex
	sizesNow = map[string]GameSize{}
	sizesAt  time.Time
	sizing   bool
)

// sizesSoon counts the sizes after a scan, unless a pass ran lately.
// Never in safe mode.
func (a *App) sizesSoon() {
	sizeMu.Lock()
	due := !sizing && time.Since(sizesAt) >= sizeEvery
	sizeMu.Unlock()
	if due && !a.safe {
		go a.sizePass()
	}
}

func (a *App) sizePass() {
	sizeMu.Lock()
	if sizing {
		sizeMu.Unlock()
		return
	}
	sizing = true
	sizeMu.Unlock()
	a.emit("sizes", a.apiSizes())
	cache, now := a.store.LoadSizes(), time.Now()
	out := map[string]GameSize{}
	for _, it := range a.sizeItems() {
		out[it.Key] = a.sizeOf(it, &cache, now)
		time.Sleep(sizePause)
	}
	if err := a.store.SaveSizes(cache, now); err != nil {
		a.logf("ERROR saving sizes: %v", err)
	}
	sizeMu.Lock()
	sizesNow, sizesAt, sizing = out, time.Now(), false
	sizeMu.Unlock()
	a.emit("sizes", a.apiSizes())
}

// sizeItem is one Library game to count: its folder, and what it is.
type sizeItem struct {
	GameSize
	found *core.FoundGame
	game  *core.Game
}

func (a *App) sizeItems() []sizeItem {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	all := a.libraryGames()
	_, containers := gameFolders(a.env)
	var out []sizeItem
	for _, f := range a.visibleFound(all) {
		f := f
		out = append(out, sizeItem{GameSize: GameSize{Key: f.Key, Dir: f.Dir}, found: &f})
	}
	for i := range cfg.Games {
		g := cfg.Games[i]
		it := sizeItem{GameSize: GameSize{Key: "game:" + g.ID}, game: &g}
		if g.Launch.Type == "steam" {
			it.Dir = a.steamInstallDir(g.Launch.AppID)
		} else if exe, err := a.env.ResolveExe(g.Launch.Exe); err == nil {
			for _, f := range all {
				if a.isFoundGame(&g, f) && len(f.Dir) > len(it.Dir) {
					it.Dir = f.Dir
				}
			}
			if it.Dir == "" {
				it.Dir = core.GameFolderOf(exe, containers)
			}
		}
		out = append(out, it)
	}
	return out
}

func (a *App) sizeOf(it sizeItem, cache *core.SizeCache, now time.Time) GameSize {
	gs := it.GameSize
	count := func(p string) int64 {
		if p == "" {
			return 0
		}
		d, err := cache.Size(p, now)
		if err != nil {
			return 0
		}
		return d.Bytes
	}
	gs.Install = count(gs.Dir)
	name := ""
	if f := it.found; f != nil {
		name = f.Name
		for _, l := range a.vaultLocs(*f) {
			gs.Saves += count(l.Resolved)
		}
		if gameSavesOn(*f) {
			gs.Copies += count(netStore().SavesDir(gameSavesKey(*f)))
		}
	}
	if g := it.game; g != nil {
		name = g.Name
		for _, pe := range g.Paths {
			if r, err := a.env.Resolve(pe.Path); err == nil {
				gs.Saves += count(r)
			}
		}
		a.mu.Lock()
		players := append([]core.Player{}, a.cfg.Players...)
		a.mu.Unlock()
		for _, p := range players {
			gs.Profiles += a.store.ProfileSize(p.ID, g.ID)
		}
		gs.Copies += count(filepath.Join(a.store.BackupsDir(), g.ID))
	}
	gs.Copies += count(a.store.SavePointsDir(gs.Key)) + count(a.updatesDir(updTarget{Key: updKey(gs.Key), Name: name}))
	return gs
}

// updKey is the Updates page's key for a Library key.
func updKey(key string) string {
	if strings.HasPrefix(key, "game:") {
		return key
	}
	return "found:" + key
}

func drives() []Drive {
	out := []Drive{}
	for _, d := range win.FixedDrives() {
		free, size := win.DiskSpace(d)
		out = append(out, Drive{Root: d, Free: free, Size: size, Games: filepath.Join(d, "Games")})
	}
	return out
}

// apiSizes is what the Storage page shows: each game's sizes from the last
// pass and the drives. The first call starts a pass, outside safe mode.
func (a *App) apiSizes() map[string]any {
	sizeMu.Lock()
	games := make([]GameSize, 0, len(sizesNow))
	for _, g := range sizesNow {
		games = append(games, g)
	}
	first, counting := sizesAt.IsZero() && !sizing, sizing
	sizeMu.Unlock()
	if first && !a.safe {
		go a.sizePass()
		counting = true
	}
	return map[string]any{"games": games, "drives": drives(), "counting": counting, "safe": a.safe}
}

// ---------------------------------------------------------------- moving --

var moveStop atomic.Bool

// apiMoveFolderGame moves a folder game into the folder dst: copied and
// checked first, then the Library points at the new folder, and only then
// does the old one go to the Recycle Bin. Progress comes as
// "folderMoveProgress" events and the outcome as "folderMoveDone".
func (a *App) apiMoveFolderGame(key, dst string) error {
	if a.safe {
		return errors.New("Lootnest is in safe mode, which leaves work like this off. Restart it normally, then move the game.")
	}
	t, err := a.updTarget(updKey(key))
	if err != nil {
		return errors.New("Only a game in a folder of its own moves from here.")
	}
	moveMu.Lock()
	if movingApp != 0 || movingGame != "" {
		moveMu.Unlock()
		return errors.New("Lootnest is already moving a game. Wait for it to finish, then move this one.")
	}
	movingGame, movingDir = t.Name, t.Dir
	moveStop.Store(false)
	moveMu.Unlock()
	go func() {
		out := map[string]any{"key": key, "name": t.Name}
		func() {
			defer func() {
				if r := recover(); r != nil {
					a.logf("ERROR move crashed: %v", r)
					out["error"] = "The move stopped unexpectedly. Details are in lootnest.log."
				}
			}()
			to, leftover, err := a.moveFolderGame(t, key, dst)
			switch {
			case errors.Is(err, core.ErrMoveCancelled):
				out["cancelled"] = true
				a.logf("Move of %s cancelled; it stays in %s", t.Name, t.Dir)
			case err != nil:
				out["error"] = err.Error()
				a.logf("Move of %s to %s failed: %v", t.Name, dst, err)
			default:
				out["to"], out["leftover"], out["newKey"] = to, leftover, movedKey(to)
				a.logf("Moved %s from %s to %s", t.Name, t.Dir, to)
				sizeMu.Lock()
				sizesAt = time.Time{} // counted again after the scan, under its new key
				sizeMu.Unlock()
			}
		}()
		moveMu.Lock()
		movingGame, movingDir = "", ""
		moveMu.Unlock()
		a.emit("folderMoveDone", out)
		a.startScan()
	}()
	return nil
}

func (a *App) apiMoveFolderCancel() { moveStop.Store(true) }

// inUse says the game, or a game with profiles that starts from its
// folder, is running or being played.
func (a *App) inUse(t updTarget) bool {
	if t.running() {
		return true
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, g := range a.cfg.Games {
		if a.sessions[g.ID] == nil {
			continue
		}
		if t.game != nil && g.ID == t.game.ID {
			return true
		}
		if exe, err := a.env.ResolveExe(g.Launch.Exe); err == nil && g.Launch.Type != "steam" && isInside(exe, t.Dir) {
			return true
		}
	}
	return false
}

// errMoving is why a game that starts from p (its exe or folder) can't
// start now: its folder is being moved to another drive. Else nil.
func (a *App) errMoving(p string) error {
	moveMu.Lock()
	dir, name := movingDir, movingGame
	moveMu.Unlock()
	if dir == "" || p == "" {
		return nil
	}
	if r, err := a.env.ResolveExe(p); err == nil {
		p = r
	}
	if !isInside(p, dir) {
		return nil
	}
	return fmt.Errorf("Lootnest is moving %s to another drive. Start it once the move is done.", name)
}

// moveFolderGame does the move, and says where the game is now and, if the
// old folder couldn't go to the Recycle Bin, where it still is.
func (a *App) moveFolderGame(t updTarget, key, dst string) (string, string, error) {
	updMu.Lock() // no update goes into the folder meanwhile
	defer updMu.Unlock()
	a.mu.Lock()
	offline := len(a.cfg.Offline[key]) > 0
	a.mu.Unlock()
	if offline {
		return "", "", fmt.Errorf("Windows Firewall keeps %s offline by its programs' places on this PC. Let it go online again first (Manage), move it, then keep it offline again.", t.Name)
	}
	vol := filepath.VolumeName(filepath.Clean(dst)) + `\`
	m, err := a.env.MoveGamePlan(t.Name, t.Source, t.Dir, dst, a.inUse(t), win.DiskFree(vol))
	if err != nil {
		return "", "", err
	}
	a.logf("Moving %s (%s) from %s to %s", m.Name, gb(m.Size), m.From, m.To)
	var last time.Time
	progress := func(n int64) {
		if time.Since(last) > 250*time.Millisecond {
			last = time.Now()
			a.emit("folderMoveProgress", map[string]any{"key": key, "done": n, "total": m.Size})
		}
	}
	a.emit("folderMoveProgress", map[string]any{"key": key, "done": 0, "total": m.Size})
	if err := core.MoveGame(m, progress, moveStop.Load); err != nil {
		if errors.Is(err, core.ErrMoveCancelled) {
			return "", "", err
		}
		return "", "", fmt.Errorf("copying it failed (%v). It stays where it was.", err)
	}
	a.emit("folderMoveProgress", map[string]any{"key": key, "done": m.Size, "total": m.Size, "step": "check"})
	undo := func(why error) (string, string, error) {
		if err := os.RemoveAll(m.To); err != nil {
			a.logf("ERROR removing the copy in %s: %v", m.To, err)
		}
		return "", "", fmt.Errorf("%v. It stays where it was.", why)
	}
	if a.inUse(t) {
		return undo(fmt.Errorf("%s started while it was copied", t.Name))
	}
	if exe, err := a.env.ResolveExe(t.Exe); t.Exe != "" && err == nil {
		if rel, err := filepath.Rel(m.From, exe); err == nil && isInside(exe, m.From) && !fileExists(filepath.Join(m.To, rel)) {
			return undo(fmt.Errorf("its program isn't in the copy (%s)", rel))
		}
	}
	if err := a.retargetMoved(m, dst); err != nil {
		return undo(fmt.Errorf("Lootnest couldn't point the Library at the new folder (%v)", err))
	}
	if err := win.Recycle(a.hwnd, m.From); err != nil {
		a.logf("The old folder of %s stays in %s: %v", m.Name, m.From, err)
		return m.To, m.From, nil
	}
	return m.To, "", nil
}

// movedKey is a moved folder game's Library key.
func movedKey(to string) string { return "dir:" + strings.ToLower(to) }

// retargetMoved points everything Lootnest keeps about the game at its new
// folder: the Library entry under its new key with what's recorded for it,
// and games with profiles that start from it. Only the config must be
// saved for the move to go on; the rest is logged if it fails.
func (a *App) retargetMoved(m core.GameMove, dst string) error {
	movedAbs := func(r string) (string, bool) {
		rel, err := filepath.Rel(m.From, r)
		if err != nil || !isInside(r, m.From) {
			return r, false
		}
		return filepath.Join(m.To, rel), true
	}
	moved := func(p string) string {
		if r, err := a.env.Resolve(p); err == nil {
			if to, ok := movedAbs(r); ok {
				return a.env.Tokenize(to)
			}
		}
		return p
	}
	movedAll := func(ps []string) []string {
		out := make([]string, len(ps))
		for i, p := range ps {
			out[i] = moved(p)
		}
		return out
	}
	var f *core.FoundGame
	for _, x := range a.libraryGames() {
		if x.Dir != "" && strings.EqualFold(filepath.Clean(x.Dir), m.From) {
			x := x
			f = &x
		}
	}
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	for i := range cfg.Games {
		g := &cfg.Games[i]
		if exe, err := a.env.ResolveExe(g.Launch.Exe); err != nil || g.Launch.Type == "steam" || !isInside(exe, m.From) {
			continue
		}
		g.Launch.Exe = moved(g.Launch.Exe)
		a.env.RetargetPaths(g, m.From, m.To)
		if o, ok := cfg.LaunchOpts["game:"+g.ID]; ok {
			cfg.LaunchOpts["game:"+g.ID] = o.Moved(m.From, m.To)
		}
	}
	oldKey, newKey := "", movedKey(m.To)
	var nf core.FoundGame
	if f != nil {
		oldKey, nf = f.Key, *f
		nf.Key, nf.Dir, nf.Exe, nf.Paths = newKey, m.To, moved(f.Exe), movedAll(f.Paths)
		nf.Suggestions = append([]core.Suggestion{}, f.Suggestions...)
		for i := range nf.Suggestions {
			nf.Suggestions[i].Path = moved(nf.Suggestions[i].Path)
			nf.Suggestions[i].Resolved, _ = movedAbs(nf.Suggestions[i].Resolved)
		}
		if f.ScanName != "" {
			nf.Name = f.ScanName // the user's name stays in LibraryPrefs
		}
		nf.ScanName, nf.Args, nf.Saves = "", "", nil
		roots, _ := gameFolders(a.env)
		inRoot := false
		for _, r := range roots {
			inRoot = inRoot || strings.EqualFold(filepath.Clean(r), filepath.Clean(dst))
		}
		cfg.AddedGames = withoutAdded(cfg.AddedGames, oldKey)
		if f.Added || !inRoot { // else the scan finds it in its new folder
			nf.Added = true
			cfg.AddedGames = append(cfg.AddedGames, nf)
		}
		core.RekeyConfig(cfg, oldKey, newKey)
		if p, ok := cfg.LibraryPrefs[newKey]; ok {
			p.Saves = movedAll(p.Saves)
			cfg.LibraryPrefs[newKey] = p
		}
		if o, ok := cfg.LaunchOpts[newKey]; ok {
			cfg.LaunchOpts[newKey] = o.Moved(m.From, m.To)
		}
	}
	if err := a.saveConfig(cfg); err != nil {
		return err
	}
	if f == nil {
		return nil
	}
	if err := a.store.RekeyGame(oldKey, newKey); err != nil && err.Error() != "nothing to move" {
		a.logf("ERROR moving %s's history to its new key: %v", m.Name, err)
	}
	if err := a.store.UpdateCopies(func(l *core.CopyLedger) { l.NoteMoved(oldKey, newKey, m.To) }); err != nil {
		a.logf("ERROR noting that %s moved: %v", m.Name, err)
	}
	for _, d := range [][2]string{
		{a.store.SavePointsDir(oldKey), a.store.SavePointsDir(newKey)},
		{a.updatesDir(updTarget{Key: updKey(oldKey), Name: f.Name}), a.updatesDir(updTarget{Key: updKey(newKey), Name: f.Name})},
	} {
		if dirExists(d[0]) && !dirExists(d[1]) {
			if err := os.Rename(d[0], d[1]); err != nil {
				a.logf("ERROR moving %s to %s: %v", d[0], d[1], err)
			}
		}
	}
	a.mu.Lock()
	for i := range a.found {
		if a.found[i].Key == oldKey {
			a.found[i] = nf
		}
	}
	a.mu.Unlock()
	return nil
}
