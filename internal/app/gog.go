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
// GOG (core/gog.go): the games in the user's GOG library, installed and
// updated from GOG's offline installers once they sign in, in Settings,
// Online. Only their own library. The installer's files are downloaded into
// Lootnest's downloads\gog folder and deleted once the game is in. An
// update runs through the update panel, so the saves and every file it
// changes are copied first and it can be undone. Games GOG Galaxy
// installed have no record here: Galaxy updates them.
// ---------------------------------------------------------------------------

var (
	gogMu      sync.Mutex
	gogOwned   []core.GogGame                    // the library last read; nil until read
	gogIns     = map[int64][]core.GogInstaller{} // the installers last listed, per game: the only ones Install takes
	gogNewer   = map[string]core.GogInstaller{}  // Library key -> the installer of the update the last check found
	gogBad     = map[string]string{}             // Library key -> why its last check failed
	gogPending = map[string]gogUpdate{}          // a downloaded update's installer -> its record once applied
	gogStop    context.CancelFunc                // the download running, if any
	gogReading bool
	gogErr     string
	gogFailed  time.Time

	gogTokMu sync.Mutex
	gogTok   core.GogTokens // the access token, only ever in memory
)

type gogUpdate struct {
	key    string // Library key
	target string // the update panel's key for the game
	rec    core.GogRecord
}

var errGogOff = errors.New("Sign in to GOG first, in Settings, Online.")

func gogDownloads() string { return filepath.Join(DataDir(), "downloads", "gog") }

func (a *App) gogUser() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cfg.Settings.GogToken == "" {
		return ""
	}
	return a.cfg.Settings.GogUser
}

func gogForget() {
	gogMu.Lock()
	defer gogMu.Unlock()
	gogOwned, gogIns, gogNewer, gogPending, gogBad = nil, map[int64][]core.GogInstaller{}, map[string]core.GogInstaller{}, map[string]gogUpdate{}, map[string]string{}
	gogErr, gogFailed = "", time.Time{}
}

// gogAccess is the access token, refreshed when it has run out (or with
// force). When GOG refuses the refresh token, Lootnest signs out.
func (a *App) gogAccess(ctx context.Context, force bool) (string, error) {
	gogTokMu.Lock()
	defer gogTokMu.Unlock()
	if !force && gogTok.Access != "" && time.Until(gogTok.Expires) > time.Minute {
		return gogTok.Access, nil
	}
	a.mu.Lock()
	saved := a.cfg.Settings.GogToken
	a.mu.Unlock()
	refresh := revealSecret(saved)
	if refresh == "" {
		return "", errGogOff
	}
	t, err := core.GogRefresh(ctx, refresh)
	if errors.Is(err, core.ErrGogSignedOut) {
		gogTok = core.GogTokens{}
		a.gogSignedOut(saved)
		return "", err
	}
	if err != nil {
		return "", err
	}
	gogTok = t
	if t.Refresh != refresh {
		enc, err := protectSecret(t.Refresh)
		a.mu.Lock()
		cfg := a.cfg.Clone()
		a.mu.Unlock()
		if err == nil && cfg.Settings.GogToken == saved {
			cfg.Settings.GogToken = enc
			if err := a.saveConfig(cfg); err != nil {
				a.logf("ERROR keeping GOG's new sign-in: %v", err)
			}
		}
	}
	return t.Access, nil
}

// gogSignedOut forgets a sign-in GOG no longer takes, unless another one
// replaced it meanwhile.
func (a *App) gogSignedOut(saved string) {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	if cfg.Settings.GogToken != saved {
		return
	}
	cfg.Settings.GogToken, cfg.Settings.GogUser = "", ""
	if err := a.saveConfig(cfg); err != nil {
		a.logf("ERROR %v", err)
	}
	a.logf("GOG signed Lootnest out; the games installed from it stay")
	go func() {
		gogForget()
		a.emit("gog", map[string]any{"signedOut": true, "error": core.ErrGogSignedOut.Error()})
		a.apiOwnedGames()
	}()
}

// gogCall runs f with the access token, once more with a fresh one if GOG
// refused it.
func (a *App) gogCall(ctx context.Context, f func(access string) error) error {
	tok, err := a.gogAccess(ctx, false)
	if err != nil {
		return err
	}
	if err = f(tok); errors.Is(err, core.ErrGogAuth) {
		if tok, err = a.gogAccess(ctx, true); err == nil {
			err = f(tok)
		}
	}
	return err
}

// apiGogSignIn finishes signing in with what the user pasted from the
// browser: the address GOG's page ended on, or its code. The answer is a
// "gog" event.
func (a *App) apiGogSignIn(pasted string) error {
	code, err := core.GogCode(pasted)
	if err != nil {
		return err
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		fail := func(err error) { a.emit("gog", map[string]any{"error": sentence(err)}) }
		t, err := core.GogToken(ctx, code)
		if err != nil {
			fail(err)
			return
		}
		user, err := core.GogUser(ctx, t.Access)
		if err != nil {
			fail(err)
			return
		}
		enc, err := protectSecret(t.Refresh)
		if err != nil {
			fail(errors.New("Lootnest couldn't keep the sign-in safely (" + err.Error() + ")"))
			return
		}
		a.mu.Lock()
		cfg := a.cfg.Clone()
		a.mu.Unlock()
		cfg.Settings.GogToken, cfg.Settings.GogUser = enc, user
		if err := a.saveConfig(cfg); err != nil {
			fail(err)
			return
		}
		gogTokMu.Lock()
		gogTok = t
		gogTokMu.Unlock()
		gogForget()
		a.logf("Signed in to GOG as %s", user)
		a.emit("gog", map[string]any{"signedIn": true, "user": user})
		a.apiOwnedGames()
	}()
	return nil
}

// apiGogSignOut forgets the sign-in. Games installed from GOG stay, as do
// their records.
func (a *App) apiGogSignOut() error {
	gogMu.Lock()
	busy := gogStop != nil
	gogMu.Unlock()
	if busy {
		return errors.New("Wait for the GOG download to finish, or cancel it first.")
	}
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	cfg.Settings.GogToken, cfg.Settings.GogUser = "", ""
	if err := a.saveConfig(cfg); err != nil {
		return err
	}
	gogTokMu.Lock()
	gogTok = core.GogTokens{}
	gogTokMu.Unlock()
	gogForget()
	_ = os.RemoveAll(gogDownloads()) // Lootnest's own downloads only
	a.logf("Signed out of GOG; the games installed from it stay")
	a.apiOwnedGames()
	return nil
}

// apiGogRefresh reads the GOG library again, answered by "owned" events.
func (a *App) apiGogRefresh() {
	gogMu.Lock()
	if !gogReading {
		gogOwned, gogIns, gogErr, gogFailed = nil, map[int64][]core.GogInstaller{}, "", time.Time{}
	}
	gogMu.Unlock()
	a.apiOwnedGames()
}

// gogLibrary is the account's games as last read, as itchLibrary is.
func (a *App) gogLibrary() (list []core.GogGame, errText string, reading bool) {
	user := a.gogUser()
	if user == "" {
		return nil, "", false
	}
	gogMu.Lock()
	defer gogMu.Unlock()
	if gogOwned != nil || gogReading || time.Since(gogFailed) < 5*time.Minute {
		return gogOwned, gogErr, gogReading
	}
	gogReading = true
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		var list []core.GogGame
		err := a.gogCall(ctx, func(tok string) (err error) { list, err = core.GogOwned(ctx, tok); return err })
		gogMu.Lock()
		gogReading = false
		switch {
		case a.gogUser() != user: // signed out or in as someone else meanwhile
		case err != nil:
			gogErr, gogFailed = sentence(err), time.Now()
			a.logf("Couldn't read the GOG library: %v", err)
		default:
			gogOwned, gogErr = append([]core.GogGame{}, list...), ""
		}
		gogMu.Unlock()
		a.apiOwnedGames()
	}()
	return nil, "", true
}

// gogOwnedGames is the GOG games to list as not installed: the ones no
// Library game is, by GOG's id (installed) or by the record of what
// Lootnest installed. Names are matched by apiOwnedGames.
func (a *App) gogOwnedGames(lib, gogIDs map[string]bool) ([]core.OwnedGame, string, bool) {
	games, errText, reading := a.gogLibrary()
	in := map[int64]bool{}
	for k, r := range a.store.GogRecords() {
		in[r.ProductID] = in[r.ProductID] || lib[k]
	}
	var out []core.OwnedGame
	for _, g := range games {
		id := strconv.FormatInt(g.ID, 10)
		if in[g.ID] || gogIDs[id] {
			continue
		}
		out = append(out, core.OwnedGame{Key: "owned:gog:" + id, Name: g.Title, Source: "gog", StoreID: id, Hero: g.Image})
	}
	return out, errText, reading
}

func gogGameOf(id int64) (core.GogGame, bool) {
	gogMu.Lock()
	defer gogMu.Unlock()
	for _, g := range gogOwned {
		if g.ID == id {
			return g, true
		}
	}
	return core.GogGame{}, false
}

// gogParent is the folder games from GOG go in.
func (a *App) gogParent(p string) string {
	if p == "" {
		a.mu.Lock()
		p = a.cfg.Settings.GogDir
		a.mu.Unlock()
	}
	if p == "" {
		p = filepath.Join(os.Getenv("USERPROFILE"), "Games")
	}
	return filepath.Clean(p)
}

// gogKept is how much of a game's installer an earlier try downloaded.
func gogKept(id int64) int64 {
	return core.Size(filepath.Join(gogDownloads(), strconv.FormatInt(id, 10)))
}

// apiGogInstallers lists a game's Windows installers, one per language,
// with the one in Windows' language first picked, the folder the game
// would go in and the room there, as a "gogInstallers" event.
func (a *App) apiGogInstallers(id int64, parent string) {
	go func() {
		out := map[string]any{"id": id}
		defer func() { a.emit("gogInstallers", out) }()
		g, ok := gogGameOf(id)
		if a.gogUser() == "" || !ok {
			out["error"] = "That game isn't in your GOG library here. Refresh the library in Settings, Online."
			return
		}
		gogMu.Lock()
		ins, have := gogIns[id]
		gogMu.Unlock()
		if !have {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			err := a.gogCall(ctx, func(tok string) (err error) { ins, err = core.GogInstallers(ctx, tok, id); return err })
			if err != nil {
				out["error"] = sentence(err)
				return
			}
			gogMu.Lock()
			gogIns[id] = ins
			gogMu.Unlock()
		}
		dir := a.gogParent(parent)
		out["installers"], out["pick"], out["dir"], out["target"] = ins, core.GogDefault(ins, core.GogLang(win.UILanguage())), dir, filepath.Join(dir, core.GameFolderName(g.Title))
		out["free"], out["dataDrive"], out["dataFree"], out["kept"] = driveFree(dir), strings.ToUpper(filepath.VolumeName(DataDir())), driveFree(DataDir()), gogKept(id)
	}()
}

type GogInstallReq struct {
	ID          int64  `json:"id"`
	InstallerID string `json:"installerId"`
	Parent      string `json:"parent"`
}

func (a *App) gogStart() (context.Context, error) {
	gogMu.Lock()
	defer gogMu.Unlock()
	if gogStop != nil {
		return nil, errors.New("A GOG download is already running. Wait for it, or cancel it first.")
	}
	ctx, cancel := context.WithCancel(context.Background())
	gogStop = cancel
	return ctx, nil
}

func gogEnd() {
	gogMu.Lock()
	defer gogMu.Unlock()
	if gogStop != nil {
		gogStop()
		gogStop = nil
	}
}

// apiGogCancel stops the download running; what it has so far is kept, so
// the next try goes on from there.
func (a *App) apiGogCancel() {
	gogMu.Lock()
	defer gogMu.Unlock()
	if gogStop != nil {
		gogStop()
	}
}

func (a *App) gogProgress(key, stage string, done, total int64, file, files int) {
	a.emit("gogProgress", map[string]any{"key": key, "stage": stage, "done": done, "total": total, "file": file, "files": files})
}

// gogFetch downloads every file of an installer, one after another, into
// Lootnest's downloads\gog\<id> folder, with "download" progress events for
// key across all of them, and returns their paths, the .exe first among
// them as GOG lists it. A file an earlier try finished isn't downloaded
// again, and one it started goes on from where it stopped.
func (a *App) gogFetch(ctx context.Context, key string, id int64, in core.GogInstaller) ([]string, error) {
	var total int64
	for _, f := range in.Files {
		total += f.Size
	}
	if total <= 0 {
		total = in.Size
	}
	var done int64
	var last time.Time
	paths, keep := []string{}, map[string]bool{}
	a.gogProgress(key, "download", 0, total, 1, len(in.Files))
	for i, f := range in.Files {
		var l core.GogLink
		err := a.gogCall(ctx, func(tok string) (err error) { l, err = core.GogFileLink(ctx, tok, f); return err })
		if err != nil {
			return nil, err
		}
		p := core.GogDownloadPath(DataDir(), id, l.Name)
		paths, keep[filepath.Base(p)] = append(paths, p), true
		if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() && (l.Size <= 0 || st.Size() == l.Size) {
			done += st.Size()
			continue
		}
		base := done
		a.logf("Downloading %s from GOG (%s)", filepath.Base(p), gb(l.Size))
		err = core.GogDownload(ctx, l, p, func(d, _ int64) {
			if time.Since(last) >= 250*time.Millisecond {
				last = time.Now()
				a.gogProgress(key, "download", base+d, total, i+1, len(in.Files))
			}
		})
		if err != nil {
			return nil, err
		}
		st, _ := os.Stat(p)
		if done = base + l.Size; st != nil {
			done = base + st.Size()
		}
	}
	a.gogProgress(key, "download", total, total, len(in.Files), len(in.Files))
	// What an earlier download of another version left, Lootnest's own.
	es, _ := os.ReadDir(filepath.Join(gogDownloads(), strconv.FormatInt(id, 10)))
	for _, e := range es {
		if !keep[strings.TrimSuffix(e.Name(), ".part")] {
			_ = os.RemoveAll(filepath.Join(gogDownloads(), strconv.FormatInt(id, 10), e.Name()))
		}
	}
	return paths, nil
}

// gogStopped reports a GOG download or install that was cancelled or failed.
func (a *App) gogStopped(out map[string]any, name string, err error) {
	if errors.Is(err, context.Canceled) {
		out["cancelled"] = true
		a.logf("Stopped the GOG download of %s; what it had is kept for the next try", name)
		return
	}
	out["error"] = sentence(err)
	a.logf("ERROR with %s from GOG: %v", name, err)
}

// apiGogInstall downloads the installer of a game the user owns, runs it
// into a new folder of its own under parent without questions, and finds
// the game it installed. The end is a "gogDone" event.
func (a *App) apiGogInstall(r GogInstallReq) error {
	if a.safe {
		return errors.New("Lootnest is in safe mode, so it doesn't install games. Restart it normally first.")
	}
	if a.gogUser() == "" {
		return errGogOff
	}
	g, ok := gogGameOf(r.ID)
	if !ok {
		return errors.New("That game isn't in your GOG library here. Refresh the library in Settings, Online.")
	}
	var in core.GogInstaller
	gogMu.Lock()
	for _, x := range gogIns[r.ID] {
		if x.ID == r.InstallerID {
			in = x
		}
	}
	gogMu.Unlock()
	if in.ID == "" {
		return errors.New("Choose one of the game's installers.")
	}
	parent := a.gogParent(r.Parent)
	if !filepath.IsAbs(parent) || filepath.VolumeName(parent) == "" {
		return errors.New("Choose a folder on this PC for the game.")
	}
	dir := filepath.Join(parent, core.GameFolderName(g.Title))
	if st, err := os.Stat(dir); err == nil {
		if es, _ := os.ReadDir(dir); !st.IsDir() || len(es) > 0 {
			return fmt.Errorf("%s already has files in it. Choose another folder for the game.", dir)
		}
	}
	// The game takes about 2.5 times its installer on its drive, and the
	// installer what's still to download on Lootnest's.
	need := map[string]int64{}
	need[strings.ToUpper(filepath.VolumeName(parent))] += in.Size * 5 / 2
	need[strings.ToUpper(filepath.VolumeName(DataDir()))] += max(0, in.Size-gogKept(r.ID))
	for vol, n := range need {
		if free := driveFree(vol); free >= 0 && free < n+(64<<20) {
			return fmt.Errorf("There isn't enough room on %s: it needs about %s.", vol, gb(n))
		}
	}
	ctx, err := a.gogStart()
	if err != nil {
		return err
	}
	if r.Parent != "" {
		a.mu.Lock()
		cfg := a.cfg.Clone()
		a.mu.Unlock()
		if cfg.Settings.GogDir != parent {
			cfg.Settings.GogDir = parent
			if err := a.saveConfig(cfg); err != nil {
				a.logf("ERROR remembering the folder for GOG games: %v", err)
			}
		}
	}
	evKey := "owned:gog:" + strconv.FormatInt(g.ID, 10)
	go func() {
		defer gogEnd()
		out := map[string]any{"key": evKey, "name": g.Title}
		defer func() { a.emit("gogDone", out) }()
		paths, err := a.gogFetch(ctx, evKey, g.ID, in)
		if err != nil {
			a.gogStopped(out, g.Title, err)
			return
		}
		libKey, err := a.gogRunInstaller(evKey, g, core.GogSetup(paths), dir)
		if err != nil {
			a.gogStopped(out, g.Title, err)
			return
		}
		if err := a.store.SetGogRecord(libKey, core.GogRecordOf(g.ID, in, time.Now())); err != nil {
			a.logf("ERROR saving gog.json: %v", err)
		}
		_ = os.RemoveAll(filepath.Join(gogDownloads(), strconv.FormatInt(g.ID, 10))) // the installer, Lootnest's own
		a.logf("Installed %s from GOG (%s, %s) into %s", g.Title, in.ID, in.Version, dir)
		out["libKey"] = libKey
		a.apiOwnedGames()
	}()
	return nil
}

// gogRunInstaller runs a downloaded installer into dir, then scans and
// returns the Library key of the game it installed: by GOG's id, then by
// its folder, then by name. The files stay for another try if it fails.
func (a *App) gogRunInstaller(evKey string, g core.GogGame, setup, dir string) (string, error) {
	if setup == "" {
		return "", errors.New("GOG's download has no installer program in it")
	}
	before := map[string]bool{}
	for _, f := range a.libraryGames() {
		before[f.Key] = true
	}
	a.gogProgress(evKey, "install", 0, 0, 0, 0)
	a.logf("Running %s, the installer of %s from GOG, into %s", filepath.Base(setup), g.Title, dir)
	code, err := win.RunAndWait(a.hwnd, setup, core.GogInnoArgs(dir), filepath.Dir(setup))
	if err != nil {
		return "", fmt.Errorf("the installer didn't run (%w). The download is kept, so Install doesn't download it again", err)
	}
	if code != 0 {
		return "", fmt.Errorf("the installer stopped before it finished (code %d). The download is kept, so Install doesn't download it again", code)
	}
	a.gogProgress(evKey, "check", 0, 0, 0, 0)
	a.scanAndWait()
	id, lib := strconv.FormatInt(g.ID, 10), a.libraryGames()
	for _, same := range []func(core.FoundGame) bool{
		func(f core.FoundGame) bool { return f.Source == "gog" && f.StoreID == id },
		func(f core.FoundGame) bool {
			return f.Dir != "" && strings.EqualFold(filepath.Clean(f.Dir), filepath.Clean(dir))
		},
		func(f core.FoundGame) bool { return !before[f.Key] && core.CoverKey(f.Name) == core.CoverKey(g.Title) },
	} {
		for _, f := range lib {
			if same(f) {
				return f.Key, nil
			}
		}
	}
	return "", fmt.Errorf("the installer finished, but Lootnest didn't find %s among your games. If it's in %s, add it with Add a game", g.Title, dir)
}

// apiGogCheck asks GOG whether the games Lootnest installed from it (only,
// or every one) have an installer of another version, at most once in six
// hours for each unless now. The answer is a "gogUpdates" event.
func (a *App) apiGogCheck(only string, now bool) {
	go func() {
		on := a.gogUser() != ""
		lib := map[string]bool{}
		for _, f := range a.libraryGames() {
			lib[f.Key] = true
		}
		errText, asked := "", 0
		for k, r := range a.store.GogRecords() {
			if !on || !lib[k] || (only != "" && k != only) || (!now && !storeDue("gog:"+k, r.CheckedAt)) {
				continue
			}
			if asked++; asked > 1 {
				time.Sleep(400 * time.Millisecond)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			var ins []core.GogInstaller
			err := a.gogCall(ctx, func(tok string) (err error) { ins, err = core.GogInstallers(ctx, tok, r.ProductID); return err })
			cancel()
			if err != nil {
				errText = sentence(err)
				a.logf("Couldn't check GOG for an update of %s: %v", k, err)
				gogMu.Lock()
				gogBad[k] = errText
				gogMu.Unlock()
				continue
			}
			storeChecked("gog:" + k)
			gogMu.Lock()
			delete(gogNewer, k)
			delete(gogBad, k)
			if in, newer := core.GogUpdate(r, ins); newer {
				gogNewer[k] = in
			}
			cur := a.store.GogRecords()[k]
			cur.CheckedAt = time.Now().UTC()
			if err := a.store.SetGogRecord(k, cur); err != nil {
				a.logf("ERROR saving gog.json: %v", err)
			}
			gogMu.Unlock()
		}
		a.emitGogUpdates(errText)
	}()
}

// apiGogRecords sends the records as a "gogUpdates" event, asking GOG nothing.
func (a *App) apiGogRecords() { go a.emitGogUpdates("") }

func (a *App) emitGogUpdates(errText string) {
	recs := map[string]any{}
	list := []map[string]any{}
	all := a.store.GogRecords()
	games := map[string]string{}
	for k := range all {
		games[k] = a.profilesGameOf(k)
	}
	gogMu.Lock()
	for k, r := range all {
		recs[k] = map[string]any{"version": r.Version, "checkedAt": r.CheckedAt, "game": games[k], "failed": gogBad[k]}
		if in, ok := gogNewer[k]; ok {
			list = append(list, map[string]any{"key": k, "version": in.Version, "size": in.Size})
		}
	}
	gogMu.Unlock()
	a.emit("gogUpdates", map[string]any{"list": list, "records": recs, "error": errText})
}

// gogUpdTarget is a game Lootnest installed from GOG as the update panel
// takes it. Its own installer updates it, so updTargets leaves it out; the
// Updates page lists it from its GOG record.
func (a *App) gogUpdTarget(key string) (updTarget, bool) {
	recs := a.store.GogRecords()
	for _, f := range a.libraryGames() {
		f := f
		if _, ok := recs[f.Key]; !ok || f.Dir == "" || a.libUpdKey(f.Key) != key {
			continue
		}
		exe, _ := a.env.ResolveExe(f.Exe)
		t := updTarget{Key: key, Name: f.Name, Source: f.Source, Dir: f.Dir, Exe: exe, Process: f.ProcessName, SteamID: f.SteamID, found: &f}
		if id := strings.TrimPrefix(key, "game:"); id != key {
			a.mu.Lock()
			cfg := a.cfg.Clone()
			a.mu.Unlock()
			for i := range cfg.Games {
				if g := cfg.Games[i]; g.ID == id {
					t.Name, t.Process, t.found, t.game = g.Name, g.ProcessName, nil, &g
				}
			}
		}
		return t, true
	}
	return updTarget{}, false
}

// apiGogUpdate downloads the installer of the update the last check found.
// The "gogDone" event names it, and the page opens it in the update panel:
// nothing in the game's folder changes before the user runs it there.
func (a *App) apiGogUpdate(libKey string) error {
	if a.safe {
		return errors.New("Lootnest is in safe mode, so it doesn't update games. Restart it normally first.")
	}
	if a.gogUser() == "" {
		return errGogOff
	}
	r, ok := a.store.GogRecords()[libKey]
	gogMu.Lock()
	in, newer := gogNewer[libKey]
	gogMu.Unlock()
	if !ok || !newer {
		return errors.New("Check for an update first.")
	}
	t, err := a.updTarget(a.libUpdKey(libKey))
	if err != nil {
		return err
	}
	if t.running() {
		return fmt.Errorf("%s is running; close it first.", t.Name)
	}
	if free := driveFree(DataDir()); free >= 0 && free < in.Size-gogKept(r.ProductID)+(64<<20) {
		return fmt.Errorf("There isn't enough room on %s for the download: it needs %s.", strings.ToUpper(filepath.VolumeName(DataDir())), gb(in.Size))
	}
	ctx, err := a.gogStart()
	if err != nil {
		return err
	}
	go func() {
		defer gogEnd()
		out := map[string]any{"key": libKey, "name": t.Name}
		defer func() { a.emit("gogDone", out) }()
		paths, err := a.gogFetch(ctx, libKey, r.ProductID, in)
		if err != nil {
			a.gogStopped(out, t.Name, err)
			return
		}
		setup := core.GogSetup(paths)
		if setup == "" {
			out["error"] = "GOG's download has no installer program in it."
			return
		}
		gogMu.Lock()
		gogPending[setup] = gogUpdate{key: libKey, target: t.Key, rec: core.GogRecordOf(r.ProductID, in, time.Now())}
		gogMu.Unlock()
		a.logf("Downloaded the GOG update of %s (%s); it goes in from the update panel", t.Name, in.Version)
		out["path"], out["updKey"] = setup, t.Key
	}()
	return nil
}

// gogArgs is how the update panel runs a GOG update's installer: into the
// game's folder, without questions.
func gogArgs(updKey, path, dir string) string {
	gogMu.Lock()
	defer gogMu.Unlock()
	if p, ok := gogPending[path]; ok && p.target == updKey {
		return core.GogInnoArgs(dir)
	}
	return ""
}

// gogApplied records an update from GOG once the update panel has applied
// it as update id, keeping the record before it for Undo, and deletes the
// downloaded installer.
func (a *App) gogApplied(updKey, path, id string) {
	gogMu.Lock()
	defer gogMu.Unlock()
	p, ok := gogPending[path]
	if !ok || p.target != updKey {
		return
	}
	delete(gogPending, path)
	delete(gogNewer, p.key)
	rec := p.rec
	rec.UpdateID = id
	if old, had := a.store.GogRecords()[p.key]; had {
		rec.Prev = &old
	}
	if err := a.store.SetGogRecord(p.key, rec); err != nil {
		a.logf("ERROR saving gog.json: %v", err)
	}
	_ = os.RemoveAll(filepath.Dir(path))
}

// gogUndone puts back the record from before a GOG update the update panel
// has undone.
func (a *App) gogUndone(updKey, id string) {
	gogMu.Lock()
	defer gogMu.Unlock()
	for k, r := range a.store.GogRecords() {
		if r.UpdateID == id && r.Prev != nil && a.libUpdKey(k) == updKey {
			delete(gogNewer, k)
			if err := a.store.SetGogRecord(k, *r.Prev); err != nil {
				a.logf("ERROR saving gog.json: %v", err)
			}
		}
	}
}
