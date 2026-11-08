//go:build windows

package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// itch.io (core/itchio.go): the games in the user's library there,
// installed and updated from itch.io with the API key they made, once they
// connect it in Settings, Online. Only their own library. Files are downloaded into
// Lootnest's downloads\itch folder and deleted once the game is in. An
// update goes through the update panel, so the saves and every file it
// replaces are copied first and it can be undone.
// ---------------------------------------------------------------------------

var (
	itchMu      sync.Mutex
	itchOwned   []core.ItchGame                 // the library last read; nil until read
	itchUps     = map[int64][]core.ItchUpload{} // the uploads last listed, per game: the only ones Install takes
	itchNewer   = map[string]core.ItchUpload{}  // Library key -> the newer upload the last check found
	itchPending = map[string]itchUpdate{}       // a downloaded update's path -> its record once applied
	itchStop    context.CancelFunc              // the download running, if any
	itchReading bool                            // the library is being read
	itchErr     string                          // why the last read failed
	itchFailed  time.Time                       // when; no new read for a few minutes unless asked
)

type itchUpdate struct {
	key    string // Library key
	target string // the update panel's key for the game
	rec    core.ItchRecord
}

const itchCheckEvery = 6 * time.Hour

var errItchOff = errors.New("Connect itch.io first, in Settings, Online.")

func (a *App) itchKey() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return revealSecret(a.cfg.Settings.ItchKey)
}

func itchForget() {
	itchMu.Lock()
	defer itchMu.Unlock()
	itchOwned, itchUps, itchNewer, itchPending = nil, map[int64][]core.ItchUpload{}, map[string]core.ItchUpload{}, map[string]itchUpdate{}
	itchErr, itchFailed = "", time.Time{}
}

func itchDownloads() string { return filepath.Join(DataDir(), "downloads", "itch") }

// apiItchConnect checks a key with itch.io, then keeps it, encrypted, with
// the account's name. The answer is an "itch" event.
func (a *App) apiItchConnect(key string) error {
	key = strings.TrimSpace(key)
	if key == "" || len(key) > 200 || strings.ContainsAny(key, " \t\r\n\"'<>") {
		return errors.New("Paste the API key you made on itch.io.")
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		user, err := core.ItchProfile(ctx, key)
		if err != nil {
			a.emit("itch", map[string]any{"error": sentence(err)})
			return
		}
		enc, err := protectSecret(key)
		if err != nil {
			a.emit("itch", map[string]any{"error": "Lootnest couldn't keep the key safely (" + err.Error() + ")."})
			return
		}
		a.mu.Lock()
		cfg := a.cfg.Clone()
		a.mu.Unlock()
		cfg.Settings.ItchKey, cfg.Settings.ItchUser = enc, user
		if err := a.saveConfig(cfg); err != nil {
			a.emit("itch", map[string]any{"error": sentence(err)})
			return
		}
		a.logf("Connected itch.io as %s", user)
		itchForget()
		a.emit("itch", map[string]any{"connected": true, "user": user})
		a.apiOwnedGames()
	}()
	return nil
}

// apiItchDisconnect forgets the key. Games installed from itch.io stay, as
// do their records.
func (a *App) apiItchDisconnect() error {
	itchMu.Lock()
	busy := itchStop != nil
	itchMu.Unlock()
	if busy {
		return errors.New("Wait for the itch.io download to finish, or cancel it first.")
	}
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	cfg.Settings.ItchKey, cfg.Settings.ItchUser = "", ""
	if err := a.saveConfig(cfg); err != nil {
		return err
	}
	itchForget()
	_ = os.RemoveAll(itchDownloads()) // Lootnest's own downloads only
	a.logf("Disconnected itch.io; the games installed from it stay")
	a.apiOwnedGames()
	return nil
}

// apiItchRefresh reads the itch.io library again, answered by "owned" events.
func (a *App) apiItchRefresh() {
	itchMu.Lock()
	if !itchReading {
		itchOwned, itchUps, itchErr, itchFailed = nil, map[int64][]core.ItchUpload{}, "", time.Time{}
	}
	itchMu.Unlock()
	a.apiOwnedGames()
}

// itchLibrary is the account's games as last read, once per start and on
// demand. Reading them takes a while, so it happens on its own, one read
// at a time, and sends "owned" again when it's done; after a failure there
// is no new read for five minutes unless the user refreshes.
func (a *App) itchLibrary() (list []core.ItchGame, errText string, reading bool) {
	key := a.itchKey()
	if key == "" {
		return nil, "", false
	}
	itchMu.Lock()
	defer itchMu.Unlock()
	if itchOwned != nil || itchReading || time.Since(itchFailed) < 5*time.Minute {
		return itchOwned, itchErr, itchReading
	}
	itchReading = true
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		list, err := core.ItchOwned(ctx, key)
		itchMu.Lock()
		itchReading = false
		switch {
		case a.itchKey() != key: // disconnected or another account meanwhile
		case err != nil:
			itchErr, itchFailed = sentence(err), time.Now()
			a.logf("Couldn't read the itch.io library: %v", err)
		default:
			itchOwned, itchErr = append([]core.ItchGame{}, list...), ""
		}
		itchMu.Unlock()
		a.apiOwnedGames()
	}()
	return nil, "", true
}

// itchOwnedGames is the itch.io games to list as not installed: the ones
// with no Library game Lootnest installed them as.
func (a *App) itchOwnedGames(lib map[string]bool) ([]core.OwnedGame, string, bool) {
	games, errText, reading := a.itchLibrary()
	in := map[int64]bool{}
	for k, r := range a.store.ItchRecords() {
		in[r.GameID] = in[r.GameID] || lib[k]
	}
	var out []core.OwnedGame
	for _, g := range games {
		if in[g.GameID] || g.Title == "" {
			continue
		}
		cover := g.CoverURL
		if !strings.HasPrefix(cover, "https://img.itch.zone/") {
			cover = ""
		}
		id := strconv.FormatInt(g.GameID, 10)
		out = append(out, core.OwnedGame{Key: "owned:itch:" + id, Name: g.Title, Source: "itch", StoreID: id, KeyID: g.KeyID, Cover: cover})
	}
	return out, errText, reading
}

func itchGameOf(id int64) (core.ItchGame, bool) {
	itchMu.Lock()
	defer itchMu.Unlock()
	for _, g := range itchOwned {
		if g.GameID == id {
			return g, true
		}
	}
	return core.ItchGame{}, false
}

// itchParent is the folder games from itch.io go in.
func (a *App) itchParent(p string) string {
	if p == "" {
		a.mu.Lock()
		p = a.cfg.Settings.ItchDir
		a.mu.Unlock()
	}
	if p == "" {
		p = filepath.Join(os.Getenv("USERPROFILE"), "Games")
	}
	return filepath.Clean(p)
}

func driveFree(dir string) int64 { return win.DiskFree(filepath.VolumeName(dir) + `\`) }

// apiItchUploads lists the files of a game the user owns that Lootnest can
// install, best first, with the folder it would go in and the room there,
// as an "itchUploads" event.
func (a *App) apiItchUploads(gameID int64, parent string) {
	go func() {
		out := map[string]any{"gameId": gameID}
		defer func() { a.emit("itchUploads", out) }()
		key := a.itchKey()
		g, ok := itchGameOf(gameID)
		if key == "" || !ok {
			out["error"] = "That game isn't in your itch.io library here. Refresh the library in Settings, Online."
			return
		}
		itchMu.Lock()
		ups, have := itchUps[gameID]
		itchMu.Unlock()
		if !have {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			all, err := core.ItchUploads(ctx, key, g.GameID, g.KeyID)
			if err != nil {
				out["error"] = sentence(err)
				return
			}
			ups = core.ItchPickUpload(all)
			itchMu.Lock()
			itchUps[gameID] = ups
			itchMu.Unlock()
		}
		dir := a.itchParent(parent)
		out["uploads"], out["dir"], out["target"], out["free"] = ups, dir, filepath.Join(dir, core.ItchFolderName(g.Title)), driveFree(dir)
		out["dataDrive"], out["dataFree"] = strings.ToUpper(filepath.VolumeName(DataDir())), driveFree(DataDir())
	}()
}

type ItchInstallReq struct {
	GameID   int64  `json:"gameId"`
	UploadID int64  `json:"uploadId"`
	Parent   string `json:"parent"`
}

func (a *App) itchStart() (context.Context, error) {
	itchMu.Lock()
	defer itchMu.Unlock()
	if itchStop != nil {
		return nil, errors.New("An itch.io download is already running. Wait for it, or cancel it first.")
	}
	ctx, cancel := context.WithCancel(context.Background())
	itchStop = cancel
	return ctx, nil
}

func itchEnd() {
	itchMu.Lock()
	defer itchMu.Unlock()
	if itchStop != nil {
		itchStop()
		itchStop = nil
	}
}

// apiItchCancel stops the download running; what it had downloaded is deleted.
func (a *App) apiItchCancel() {
	itchMu.Lock()
	defer itchMu.Unlock()
	if itchStop != nil {
		itchStop()
	}
}

func (a *App) itchProgress(key, stage string, done, total int64) {
	a.emit("itchProgress", map[string]any{"key": key, "stage": stage, "done": done, "total": total})
}

// itchFetch downloads an upload to path, with "download" progress events
// for key. An earlier download of the game, Lootnest's own, goes first.
func (a *App) itchFetch(ctx context.Context, apiKey, key string, g core.ItchGame, up core.ItchUpload, path string) error {
	_ = os.RemoveAll(filepath.Dir(path))
	a.logf("Downloading %s from itch.io (%s)", up.Filename, gb(up.Size))
	var last time.Time
	a.itchProgress(key, "download", 0, up.Size)
	return core.ItchDownload(ctx, core.ItchDownloadURL(apiKey, up.ID, g.KeyID), path, up.Size, up.MD5, func(done, total int64) {
		if time.Since(last) >= 250*time.Millisecond || done == total {
			last = time.Now()
			a.itchProgress(key, "download", done, total)
		}
	})
}

func (a *App) itchFailed(out map[string]any, name string, err error) {
	if errors.Is(err, context.Canceled) {
		out["cancelled"] = true
		a.logf("Stopped the itch.io download of %s; what it had downloaded is deleted", name)
		return
	}
	out["error"] = sentence(err)
	a.logf("ERROR with %s from itch.io: %v", name, err)
}

// apiItchInstall downloads a game the user owns and installs it: an
// archive is unpacked (a game that is one .exe copied) into a new folder of
// its own under parent and joins the Library like a game added by its
// .exe; an installer runs, and the game it installs is found by a scan. The end is an "itchDone" event.
func (a *App) apiItchInstall(r ItchInstallReq) error {
	if a.safe {
		return errors.New("Lootnest is in safe mode, so it doesn't install games. Restart it normally first.")
	}
	key := a.itchKey()
	if key == "" {
		return errItchOff
	}
	g, ok := itchGameOf(r.GameID)
	if !ok {
		return errors.New("That game isn't in your itch.io library here. Refresh the library in Settings, Online.")
	}
	var up core.ItchUpload
	itchMu.Lock()
	for _, u := range itchUps[r.GameID] {
		if u.ID == r.UploadID {
			up = u
		}
	}
	itchMu.Unlock()
	if up.ID == 0 {
		return errors.New("Choose one of the game's downloads.")
	}
	parent := a.itchParent(r.Parent)
	if !filepath.IsAbs(parent) || filepath.VolumeName(parent) == "" {
		return errors.New("Choose a folder on this PC for the game.")
	}
	kind := core.ItchKind(up)
	dir := filepath.Join(parent, core.ItchFolderName(g.Title))
	if kind != "installer" {
		if es, err := os.ReadDir(dir); err == nil && len(es) > 0 {
			return fmt.Errorf("%s already has files in it. Choose another folder for the game.", dir)
		}
		if free := driveFree(parent); free >= 0 && free < up.Size*3+(64<<20) {
			return fmt.Errorf("There isn't enough room on %s: downloading and unpacking the game needs about %s.", strings.ToUpper(filepath.VolumeName(parent)), gb(up.Size*3))
		}
	}
	if free := driveFree(DataDir()); free >= 0 && free < up.Size+(64<<20) {
		return fmt.Errorf("There isn't enough room on %s for the download: it needs %s.", strings.ToUpper(filepath.VolumeName(DataDir())), gb(up.Size))
	}
	ctx, err := a.itchStart()
	if err != nil {
		return err
	}
	if r.Parent != "" {
		a.mu.Lock()
		cfg := a.cfg.Clone()
		a.mu.Unlock()
		if cfg.Settings.ItchDir != parent {
			cfg.Settings.ItchDir = parent
			if err := a.saveConfig(cfg); err != nil {
				a.logf("ERROR remembering the folder for itch.io games: %v", err)
			}
		}
	}
	evKey := "owned:itch:" + strconv.FormatInt(g.GameID, 10)
	go func() {
		defer itchEnd()
		out := map[string]any{"key": evKey, "name": g.Title}
		defer func() { a.emit("itchDone", out) }()
		path := core.ItchDownloadPath(DataDir(), g.GameID, up.Filename)
		defer os.RemoveAll(filepath.Dir(path))
		if err := a.itchFetch(ctx, key, evKey, g, up, path); err != nil {
			a.itchFailed(out, g.Title, err)
			return
		}
		libKey, err := "", error(nil)
		if kind == "installer" {
			if libKey, err = a.itchRunInstaller(evKey, g, path); err != nil {
				a.itchFailed(out, g.Title, err)
				return
			}
		} else {
			a.itchProgress(evKey, "extract", 0, 0)
			if dir, err = core.ItchInstallArchive(path, parent, g.Title); err != nil {
				a.itchFailed(out, g.Title, err)
				return
			}
			exe, _ := core.MainExe(dir)
			if exe == "" {
				out["error"] = fmt.Sprintf("%s is in %s, but Lootnest found no program in it to start. Add it with Add a game.", g.Title, dir)
				a.logf("Installed %s from itch.io into %s, but found no .exe in it", g.Title, dir)
				return
			}
			_, containers := gameFolders(a.env)
			f := a.env.ExeGame(exe, append(containers, parent))
			f.Name = g.Title
			added, err := a.addToLibrary(f)
			if err != nil {
				a.itchFailed(out, g.Title, err)
				return
			}
			libKey = added.Key
		}
		itchMu.Lock()
		if err := a.store.SetItchRecord(libKey, core.ItchRecordOf(g, up, time.Now())); err != nil {
			a.logf("ERROR saving itch.json: %v", err)
		}
		itchMu.Unlock()
		a.logf("Installed %s from itch.io (%s)", g.Title, up.Filename)
		out["libKey"] = libKey
		a.apiOwnedGames()
	}()
	return nil
}

// itchRunInstaller runs a downloaded installer, then scans and returns the
// Library key of the new game named like it.
func (a *App) itchRunInstaller(evKey string, g core.ItchGame, path string) (string, error) {
	before := map[string]bool{}
	for _, f := range a.libraryGames() {
		before[f.Key] = true
	}
	a.itchProgress(evKey, "install", 0, 0)
	a.logf("Running %s, the installer of %s from itch.io", filepath.Base(path), g.Title)
	if _, err := win.RunAndWait(a.hwnd, path, filepath.Dir(path)); err != nil {
		return "", err
	}
	a.itchProgress(evKey, "check", 0, 0)
	a.scanAndWait()
	want := core.CoverKey(g.Title)
	for _, f := range a.libraryGames() {
		if !before[f.Key] && core.CoverKey(f.Name) == want {
			return f.Key, nil
		}
	}
	return "", fmt.Errorf("the installer finished, but Lootnest didn't find %s among your games. If it's installed, add it with Add a game", g.Title)
}

// scanAndWait waits for a scan that's running, then runs one and waits
// for it.
func (a *App) scanAndWait() {
	wait := func() {
		for i := 0; i < 1200; i++ {
			a.mu.Lock()
			busy := a.scanning
			a.mu.Unlock()
			if !busy {
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
	wait()
	a.startScan()
	wait()
}

// apiItchCheck asks itch.io whether the games Lootnest installed from it
// (only, or every one) have a newer upload, at most once in six hours for
// each unless now. The answer is an "itchUpdates" event.
func (a *App) apiItchCheck(only string, now bool) {
	go func() {
		key := a.itchKey()
		lib := map[string]bool{}
		for _, f := range a.libraryGames() {
			lib[f.Key] = true
		}
		errText, asked := "", 0
		for k, r := range a.store.ItchRecords() {
			if key == "" || !lib[k] || (only != "" && k != only) || (!now && time.Since(r.CheckedAt) < itchCheckEvery) {
				continue
			}
			if asked++; asked > 1 {
				time.Sleep(400 * time.Millisecond)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			all, err := core.ItchUploads(ctx, key, r.GameID, r.KeyID)
			cancel()
			if err != nil {
				errText = sentence(err)
				a.logf("Couldn't check itch.io for an update of %s: %v", k, err)
				continue
			}
			best, found := core.ItchUpload{}, false
			for _, u := range core.ItchPickUpload(all) {
				if !found || u.ID == r.UploadID { // the upload installed, or else the best one now
					best, found = u, true
				}
			}
			itchMu.Lock()
			delete(itchNewer, k)
			if found && core.ItchNewer(r, best) {
				itchNewer[k] = best
			}
			cur := a.store.ItchRecords()[k]
			cur.CheckedAt = time.Now().UTC()
			if err := a.store.SetItchRecord(k, cur); err != nil {
				a.logf("ERROR saving itch.json: %v", err)
			}
			itchMu.Unlock()
		}
		a.emitItchUpdates(errText)
	}()
}

// apiItchRecords sends the records as an "itchUpdates" event, asking itch.io nothing.
func (a *App) apiItchRecords() { go a.emitItchUpdates("") }

// itchProfilesGame is the id of the game with profiles a Library game
// became, or "".
func (a *App) itchProfilesGame(libKey string) string {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	for _, f := range a.libraryGames() {
		if f.Key != libKey {
			continue
		}
		for i := range cfg.Games {
			if a.isFoundGame(&cfg.Games[i], f) {
				return cfg.Games[i].ID
			}
		}
	}
	return ""
}

// itchTarget is the update panel's key for a Library game, with profiles or not.
func (a *App) itchTarget(libKey string) string {
	if id := a.itchProfilesGame(libKey); id != "" {
		return "game:" + id
	}
	return "found:" + libKey
}

// emitItchUpdates sends the records, with the game with profiles each
// became, and the updates found. Versions are the ones the developer
// wrote, or none.
func (a *App) emitItchUpdates(errText string) {
	recs := map[string]any{}
	list := []map[string]any{}
	all := a.store.ItchRecords()
	games := map[string]string{}
	for k := range all {
		games[k] = a.itchProfilesGame(k)
	}
	itchMu.Lock()
	for k, r := range all {
		recs[k] = map[string]any{"version": r.UserVersion, "checkedAt": r.CheckedAt, "game": games[k]}
		if u, ok := itchNewer[k]; ok {
			list = append(list, map[string]any{"key": k, "version": u.UserVersion, "size": u.Size, "uploadId": u.ID})
		}
	}
	itchMu.Unlock()
	a.emit("itchUpdates", map[string]any{"list": list, "records": recs, "error": errText})
}

// apiItchUpdate downloads the newer upload of a game Lootnest installed
// from itch.io. The "itchDone" event names the file, which the page then
// opens in the update panel: nothing in the game's folder changes before
// the user applies it there.
func (a *App) apiItchUpdate(libKey string) error {
	if a.safe {
		return errors.New("Lootnest is in safe mode, so it doesn't update games. Restart it normally first.")
	}
	key := a.itchKey()
	if key == "" {
		return errItchOff
	}
	r, ok := a.store.ItchRecords()[libKey]
	itchMu.Lock()
	up, newer := itchNewer[libKey]
	itchMu.Unlock()
	if !ok || !newer {
		return errors.New("Check for an update first.")
	}
	t, err := a.updTarget(a.itchTarget(libKey))
	if err != nil {
		return err
	}
	if t.running() {
		return fmt.Errorf("%s is running; close it first.", t.Name)
	}
	if free := driveFree(DataDir()); free >= 0 && free < up.Size+(64<<20) {
		return fmt.Errorf("There isn't enough room on %s for the download: it needs %s.", strings.ToUpper(filepath.VolumeName(DataDir())), gb(up.Size))
	}
	ctx, err := a.itchStart()
	if err != nil {
		return err
	}
	go func() {
		defer itchEnd()
		out := map[string]any{"key": libKey, "name": t.Name}
		defer func() { a.emit("itchDone", out) }()
		g := core.ItchGame{GameID: r.GameID, KeyID: r.KeyID}
		path := core.ItchDownloadPath(DataDir(), r.GameID, up.Filename)
		if core.ItchKind(up) == "exe" { // the game's own .exe: an update folder holding it, not an installer to run
			name := filepath.Base(path)
			path = filepath.Join(filepath.Dir(path), strings.TrimSuffix(name, filepath.Ext(name)), name)
		}
		if err := a.itchFetch(ctx, key, libKey, g, up, path); err != nil {
			a.itchFailed(out, t.Name, err)
			return
		}
		if core.ItchKind(up) == "exe" {
			path = filepath.Dir(path)
		}
		itchMu.Lock()
		itchPending[path] = itchUpdate{key: libKey, target: t.Key, rec: core.ItchRecordOf(g, up, time.Now())}
		itchMu.Unlock()
		a.logf("Downloaded the itch.io update of %s (%s); it goes in from the update panel", t.Name, up.Filename)
		out["path"], out["updKey"] = path, t.Key
	}()
	return nil
}

// itchApplied records an update from itch.io once the update panel has
// applied it as update id, keeping the record before it for Undo, and
// deletes the downloaded file.
func (a *App) itchApplied(updKey, path, id string) {
	itchMu.Lock()
	defer itchMu.Unlock()
	p, ok := itchPending[path]
	if !ok || p.target != updKey {
		return
	}
	delete(itchPending, path)
	delete(itchNewer, p.key)
	rec := p.rec
	rec.UpdateID = id
	if old, had := a.store.ItchRecords()[p.key]; had {
		rec.Prev = &old
	}
	if err := a.store.SetItchRecord(p.key, rec); err != nil {
		a.logf("ERROR saving itch.json: %v", err)
	}
	_ = os.RemoveAll(filepath.Dir(path))
}

// itchUndone puts back the record from before an itch.io update the update
// panel has undone.
func (a *App) itchUndone(updKey, id string) {
	itchMu.Lock()
	defer itchMu.Unlock()
	for k, r := range a.store.ItchRecords() {
		if r.UpdateID == id && r.Prev != nil && a.itchTarget(k) == updKey {
			delete(itchNewer, k)
			if err := a.store.SetItchRecord(k, *r.Prev); err != nil {
				a.logf("ERROR saving itch.json: %v", err)
			}
		}
	}
}
