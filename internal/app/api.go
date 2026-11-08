//go:build windows

package app

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// bind exposes the API to the page as window.ps_<name>(...) promises.
func (a *App) bind() {
	must := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	must(a.w.Bind("ps_ready", a.apiReady))
	must(a.w.Bind("ps_state", a.apiState))
	must(a.w.Bind("ps_savePlayers", a.apiSavePlayers))
	must(a.w.Bind("ps_saveGame", a.apiSaveGame))
	must(a.w.Bind("ps_deleteGame", a.apiDeleteGame))
	must(a.w.Bind("ps_saveSettings", a.apiSaveSettings))
	must(a.w.Bind("ps_setSeat", a.apiSetSeat))
	must(a.w.Bind("ps_testToast", a.apiTestToast))
	must(a.w.Bind("ps_setOffline", a.apiSetOffline))
	must(a.w.Bind("ps_savePoints", a.apiSavePoints))
	must(a.w.Bind("ps_saveCheck", a.apiSaveCheck))
	must(a.w.Bind("ps_bigPicture", a.apiBigPicture))
	must(a.w.Bind("ps_gameBackups", a.apiGameBackups))
	must(a.w.Bind("ps_restoreBackup", a.apiRestoreBackup))
	must(a.w.Bind("ps_dismissConflict", a.apiDismissConflict))
	must(a.w.Bind("ps_gameFacts", a.apiGameFacts))
	must(a.w.Bind("ps_reportProblem", a.apiReportProblem))
	must(a.w.Bind("ps_makeSavePoint", a.apiMakeSavePoint))
	must(a.w.Bind("ps_restoreSavePoint", a.apiRestoreSavePoint))
	must(a.w.Bind("ps_renameSavePoint", a.apiRenameSavePoint))
	must(a.w.Bind("ps_deleteSavePoint", a.apiDeleteSavePoint))
	must(a.w.Bind("ps_branchSavePoint", a.apiBranchSavePoint))
	must(a.w.Bind("ps_trySave", a.apiTrySave))
	must(a.w.Bind("ps_saveRecap", a.apiSaveRecap))
	must(a.w.Bind("ps_achDoctor", a.apiAchDoctor))
	must(a.w.Bind("ps_setNameInGame", a.apiSetNameInGame))
	must(a.w.Bind("ps_steamGames", a.apiSteamGames))
	must(a.w.Bind("ps_inspectSteam", a.apiInspectSteam))
	must(a.w.Bind("ps_inspectExe", a.apiInspectExe))
	must(a.w.Bind("ps_checkPath", a.apiCheckPath))
	must(a.w.Bind("ps_browse", a.apiBrowse))
	must(a.w.Bind("ps_play", a.apiPlay))
	must(a.w.Bind("ps_load", a.apiLoad))
	must(a.w.Bind("ps_closeGame", a.apiCloseGame))
	must(a.w.Bind("ps_profileInfo", a.apiProfileInfo))
	must(a.w.Bind("ps_achievements", a.apiAchievements))
	must(a.w.Bind("ps_setSharedAchievements", a.apiSetSharedAchievements))
	must(a.w.Bind("ps_allAchievements", a.apiAllAchievements))
	must(a.w.Bind("ps_shortcut", a.apiShortcut))
	must(a.w.Bind("ps_open", a.apiOpen))
	must(a.w.Bind("ps_log", a.apiLog))
	must(a.w.Bind("ps_steamSetup", a.apiSteamSetup))
	must(a.w.Bind("ps_copy", a.apiCopy))
	must(a.w.Bind("ps_steamPick", a.apiSteamPick))
	must(a.w.Bind("ps_cancelRequest", a.apiCancelRequest))
	must(a.w.Bind("ps_found", a.apiFound))
	must(a.w.Bind("ps_scan", a.apiScan))
	must(a.w.Bind("ps_sources", a.apiSources))
	must(a.w.Bind("ps_ignoreFound", a.apiIgnoreFound))
	must(a.w.Bind("ps_unhideFound", a.apiUnhideFound))
	must(a.w.Bind("ps_setPrimaryCopy", a.apiSetPrimaryCopy))
	must(a.w.Bind("ps_separateCopies", a.apiSeparateCopies))
	must(a.w.Bind("ps_addFound", a.apiAddFound))
	must(a.w.Bind("ps_foundGame", a.apiFoundGame))
	must(a.w.Bind("ps_applyHooks", a.apiApplyHooks))
	must(a.w.Bind("ps_fixBlocked", a.apiFixBlocked))
	must(a.w.Bind("ps_cover", a.apiCover))
	must(a.w.Bind("ps_launchFound", a.apiLaunchFound))
	must(a.w.Bind("ps_checkUpdate", a.apiCheckUpdate))
	must(a.w.Bind("ps_cloudStatus", a.apiCloudStatus))
	must(a.w.Bind("ps_installUpdate", a.apiInstallUpdate))
	must(a.w.Bind("ps_moveInfo", a.apiMoveInfo))
	must(a.w.Bind("ps_moveGame", a.apiMoveGame))
	must(a.w.Bind("ps_saveLibraryPref", a.apiSaveLibraryPref))
	must(a.w.Bind("ps_addLibraryGame", a.apiAddLibraryGame))
	must(a.w.Bind("ps_gameIcon", a.apiGameIcon))
	must(a.w.Bind("ps_art", a.apiArt))
	must(a.w.Bind("ps_setAvatar", a.apiSetAvatar))
	must(a.w.Bind("ps_removeAvatar", a.apiRemoveAvatar))
	must(a.w.Bind("ps_setBanner", a.apiSetBanner))
	must(a.w.Bind("ps_removeBanner", a.apiRemoveBanner))
	must(a.w.Bind("ps_setShelf", a.apiSetShelf))
	must(a.w.Bind("ps_setTags", a.apiSetTags))
	must(a.w.Bind("ps_backupInfo", a.apiBackupInfo))
	must(a.w.Bind("ps_setBackup", a.apiSetBackup))
	must(a.w.Bind("ps_setBackupPassword", a.apiSetBackupPassword))
	must(a.w.Bind("ps_backupNow", a.apiBackupNow))
	must(a.w.Bind("ps_restoreOpen", a.apiRestoreOpen))
	must(a.w.Bind("ps_restoreApply", a.apiRestoreApply))
	must(a.w.Bind("ps_restoreCancel", a.apiRestoreCancel))
	must(a.w.Bind("ps_importApply", a.apiImportApply))
	must(a.w.Bind("ps_exportPlayer", a.apiExportPlayer))
	must(a.w.Bind("ps_savesStatus", a.apiSavesStatus))
	must(a.w.Bind("ps_syncNow", a.apiSyncNow))
	must(a.w.Bind("ps_syncResolve", a.apiSyncResolve))
	must(a.w.Bind("ps_gameSaves", a.apiGameSaves))
	must(a.w.Bind("ps_gameSaveNow", a.apiGameSaveNow))
	must(a.w.Bind("ps_gameSaveRestore", a.apiGameSaveRestore))
	must(a.w.Bind("ps_gameSaveRestoreLocal", a.apiGameSaveRestoreLocal))
	must(a.w.Bind("ps_findGames", a.apiFindGames))
	must(a.w.Bind("ps_gameOffer", a.apiGameOffer))
	must(a.w.Bind("ps_storeFront", a.apiStoreFront))
	must(a.w.Bind("ps_updates", a.apiUpdates))
	must(a.w.Bind("ps_updatePlan", a.apiUpdatePlan))
	must(a.w.Bind("ps_updateApply", a.apiUpdateApply))
	must(a.w.Bind("ps_updateUndo", a.apiUpdateUndo))
	must(a.w.Bind("ps_updateInstaller", a.apiUpdateInstaller))
	must(a.w.Bind("ps_downloads", a.apiDownloads))
	must(a.w.Bind("ps_ignoreDownload", a.apiIgnoreDownload))
	must(a.w.Bind("ps_recycleDownload", a.apiRecycleDownload))
	must(a.w.Bind("ps_newCopies", a.apiNewCopies))
	must(a.w.Bind("ps_carryOver", a.apiCarryOver))
	must(a.w.Bind("ps_dismissCopy", a.apiDismissCopy))
	must(a.w.Bind("ps_deleteOldCopy", a.apiDeleteOldCopy))
	must(a.w.Bind("ps_keepSavesNow", a.apiKeepSavesNow))
	must(a.w.Bind("ps_setGameSaves", a.apiSetGameSaves))
	must(a.w.Bind("ps_gameNews", a.apiGameNews))
	must(a.w.Bind("ps_openGameFolder", a.apiOpenGameFolder))
	must(a.w.Bind("ps_foundShortcut", a.apiFoundShortcut))
	must(a.w.Bind("ps_steamVerify", a.apiSteamVerify))
	must(a.w.Bind("ps_hide", func() { win.ShowWindow(a.hwnd, win.SW_HIDE) }))
	must(a.w.Bind("ps_quit", func() { a.ui(func() { a.quit(false) }) }))
}

func (a *App) apiReady() map[string]any {
	a.mu.Lock()
	defer a.mu.Unlock()
	first := !a.ready
	a.ready = true
	p, r, bp := a.pending, a.pendReq, a.openBP
	a.pending, a.pendReq, a.openBP = "", "", false
	if first {
		go a.logf("Interface loaded")
	}
	return map[string]any{"launch": p, "reqId": r, "bigPicture": bp}
}

type SteamStatus struct {
	Found        bool     `json:"found"`
	Path         string   `json:"path"`
	Accounts     []string `json:"accounts"`
	AccountInUse string   `json:"accountInUse"`
	AccountError string   `json:"accountError,omitempty"`
}

type AppState struct {
	Version   string                                `json:"version"`
	Stage     string                                `json:"stage,omitempty"` // "beta"
	Config    *core.Config                          `json:"config"`
	FirstRun  bool                                  `json:"firstRun"`
	LoadError string                                `json:"loadError,omitempty"`
	Active    map[string]string                     `json:"active"`
	Sessions  []Session                             `json:"sessions"`
	Profiles  map[string][]string                   `json:"profiles"`
	Steam     SteamStatus                           `json:"steam"`
	Autostart bool                                  `json:"autostart"`
	DataDir   string                                `json:"dataDir"`
	Colors    []string                              `json:"colors"`
	Sources   map[string]string                     `json:"sources"`   // game id -> where the scan found it
	Recent    map[string]core.Played                `json:"recent"`    // Library key -> last played
	Playtime  map[string]map[string]int64           `json:"playtime"`  // Library key -> player -> seconds, with each player's correction
	SteamTime map[string]int64                      `json:"steamTime"` // Library key -> seconds Steam counts for the account
	Avatars   map[string]string                     `json:"avatars"`   // player id -> profile picture as a data: URL
	Banners   map[string]string                     `json:"banners"`   // player id -> banner picture as a data: URL
	Shelf     map[string]map[string]core.ShelfEntry `json:"shelf"`     // player id -> Library key -> their take
	Activity  core.Activity                         `json:"activity"`  // time played per day
	Conflicts map[string]SteamCloudNote             `json:"conflicts"` // game id -> what Steam Cloud replaced, until dismissed
}

func (a *App) apiState() AppState {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	st := AppState{Version: Version, Stage: Stage, Config: cfg, FirstRun: a.firstRun, LoadError: a.loadErr, DataDir: DataDir(), Colors: core.PlayerColors}
	a.mu.Unlock()
	st.Active = a.store.LoadState()
	st.Sessions = a.sessionsSnapshot()
	conflictMu.Lock()
	st.Conflicts = loadConflicts()
	conflictMu.Unlock()
	st.Sources = a.gameSources(cfg)
	st.Recent = a.store.LoadRecent()
	st.Shelf = a.store.LoadShelf()
	st.Playtime = core.WithOffsets(a.store.LoadPlaytime(), st.Shelf)
	st.Avatars, st.Banners = map[string]string{}, map[string]string{}
	for _, p := range cfg.Players {
		if p.Avatar {
			if u := dataURL(avatarPath(p.ID)); u != "" {
				st.Avatars[p.ID] = u
			}
		}
		if p.Banner == "image" {
			if u := dataURL(bannerPath(p.ID)); u != "" {
				st.Banners[p.ID] = u
			}
		}
	}
	st.Activity = a.store.LoadActivity()
	st.SteamTime = map[string]int64{}
	// Steam's own record of what was played and when, so the Library's
	// shelves fill in for games started from Steam too.
	for id, p := range a.steamActivity() {
		keys := []string{fmt.Sprintf("steam:%d", id)}
		for _, g := range cfg.Games {
			if g.Launch.Type == "steam" && g.Launch.AppID == id {
				keys = append(keys, "game:"+g.ID)
			}
		}
		for _, k := range keys {
			if p.Minutes > 0 {
				st.SteamTime[k] = int64(p.Minutes) * 60
			}
			if rec := st.Recent[k]; !p.LastPlayed.IsZero() && p.LastPlayed.After(rec.At) {
				who := rec.Player // Lootnest started it: Steam's time is a few minutes later
				if p.LastPlayed.After(rec.At.Add(30 * time.Minute)) {
					who = "" // started from Steam since: nobody knows who played
				}
				st.Recent[k] = core.Played{At: p.LastPlayed, Player: who, Players: rec.Players}
			}
		}
	}
	st.Profiles = map[string][]string{}
	for _, g := range cfg.Games {
		list := []string{}
		for _, p := range cfg.Players {
			if a.eng.HasProfile(p.ID, g.ID) {
				list = append(list, p.ID)
			}
		}
		st.Profiles[g.ID] = list
	}
	st.Steam.Path = a.env.SteamPath
	st.Steam.Found = a.env.SteamPath != ""
	if st.Steam.Found {
		st.Steam.Accounts = core.SteamAccounts(a.env.SteamPath)
		if d, err := a.env.SteamUser(); err == nil {
			st.Steam.AccountInUse = filepath.Base(d)
		} else {
			st.Steam.AccountError = err.Error()
		}
	}
	_, st.Autostart = win.Autostart(Name)
	return st
}

func (a *App) saveConfig(cfg *core.Config) error {
	if err := a.store.SaveConfig(cfg); err != nil {
		return fmt.Errorf("couldn't save settings: %w", err)
	}
	a.mu.Lock()
	a.cfg = cfg
	a.firstRun = false
	a.mu.Unlock()
	return nil
}

func (a *App) activeSessionFor(gameID string) *Session {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.sessions[gameID]
}

// ---------------------------------------------------------------- players --

func (a *App) apiSavePlayers(players []core.Player) error {
	if err := core.ValidatePlayers(players); err != nil {
		return err
	}
	a.mu.Lock()
	cfg := a.cfg.Clone()
	keep := map[string]bool{}
	for _, p := range players {
		keep[p.ID] = true
	}
	var removed []string
	for _, p := range cfg.Players {
		if !keep[p.ID] {
			removed = append(removed, p.ID)
		}
	}
	for _, s := range a.sessions {
		for _, r := range removed {
			if s.PlayerID == r {
				a.mu.Unlock()
				return fmt.Errorf("%s is playing right now; remove them after the game closes.", cfg.PlayerName(r))
			}
		}
	}
	a.mu.Unlock()

	cfg.Players = players
	for i := range cfg.Games {
		for _, r := range removed {
			delete(cfg.Games[i].PlayerArgs, r)
		}
	}
	if err := a.saveConfig(cfg); err != nil {
		return err
	}
	state := a.store.LoadState()
	for _, r := range removed {
		a.archive(filepath.Join(a.store.ProfilesDir(), r), "player-"+r)
		for g, p := range state {
			if p == r {
				_ = a.store.SetActive(g, "")
			}
		}
		a.logf("Removed a player (%s)", r)
	}
	return nil
}

// archive moves data aside instead of deleting it, so a misclick is recoverable.
func (a *App) archive(src, label string) {
	if _, err := os.Stat(src); err != nil {
		return
	}
	dst := filepath.Join(a.store.Dir, "removed", label+"-"+time.Now().Format("20060102-150405"))
	_ = os.MkdirAll(filepath.Dir(dst), 0o755)
	if err := os.Rename(src, dst); err != nil {
		a.logf("ERROR archiving %s: %v", src, err)
	}
}

// ------------------------------------------------------------------ games --

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func (a *App) apiSaveGame(g core.Game) (core.Game, error) {
	isNew := g.ID == ""
	wasDirect := false
	a.mu.Lock()
	if old := a.cfg.Game(g.ID); old != nil {
		wasDirect = old.SteamDirect
	}
	a.mu.Unlock()
	saved, err := a.apiSaveGameQuiet(g)
	if err != nil || saved.Launch.Type != "steam" {
		return saved, err
	}
	if saved.SteamDirect && (isNew || !wasDirect) {
		a.directShortcut(saved.ID)
	}
	if !isNew && saved.SteamDirect != wasDirect {
		// Taking Lootnest out of Steam isn't optional; putting it back follows
		// the automatic setting.
		a.autoHooksAlways()
	} else if isNew {
		a.autoHooks()
	}
	return saved, err
}

func (a *App) apiSaveGameQuiet(g core.Game) (core.Game, error) {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	isNew := g.ID == ""
	if isNew {
		slug := strings.Trim(slugRe.ReplaceAllString(strings.ToLower(g.Name), "-"), "-")
		if len(slug) > 24 {
			slug = slug[:24]
		}
		if slug == "" {
			slug = "game"
		}
		g.ID = slug
		for i := 2; cfg.Game(g.ID) != nil; i++ {
			g.ID = fmt.Sprintf("%s-%d", slug, i)
		}
	} else if cfg.Game(g.ID) == nil {
		return g, errors.New("That game no longer exists.")
	}
	if err := core.ValidateGame(&g); err != nil {
		return g, err
	}
	for pid := range g.PlayerArgs {
		if cfg.Player(pid) == nil || strings.TrimSpace(g.PlayerArgs[pid]) == "" {
			delete(g.PlayerArgs, pid)
		}
	}
	if err := a.eng.CheckPaths(&g); err != nil {
		return g, err
	}
	if g.Launch.Type == "exe" {
		if _, err := a.env.ResolveExe(g.Launch.Exe); err != nil {
			return g, err
		}
	}
	if !isNew && a.activeSessionFor(g.ID) != nil {
		return g, fmt.Errorf("%s is in use right now; edit it after the game closes.", g.Name)
	}
	if isNew {
		cfg.Games = append(cfg.Games, g)
	} else {
		*cfg.Game(g.ID) = g
	}
	if err := a.saveConfig(cfg); err != nil {
		return g, err
	}
	a.logf("Saved game %s (%s)", g.Name, g.ID)
	return g, nil
}

func (a *App) apiDeleteGame(id string) error {
	if a.activeSessionFor(id) != nil {
		return errors.New("This game is in use right now; remove it after it closes.")
	}
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	out := cfg.Games[:0]
	var name string
	steamRemoved := false
	for _, g := range cfg.Games {
		if g.ID == id {
			name = g.Name
			if g.Launch.Type == "steam" {
				// put the game's own Steam launch options back
				if cfg.SteamRestores == nil {
					cfg.SteamRestores = map[string]string{}
				}
				cfg.SteamRestores[strconv.Itoa(g.Launch.AppID)] = g.SteamPrevOptions
				steamRemoved = true
			}
			continue
		}
		out = append(out, g)
	}
	cfg.Games = out
	if err := a.saveConfig(cfg); err != nil {
		return err
	}
	for _, p := range cfg.Players {
		a.archive(a.store.ProfileDir(p.ID, id), "game-"+id+"-"+p.ID)
	}
	_ = a.store.SetActive(id, "")
	a.logf("Removed game %s", name)
	if steamRemoved {
		a.autoHooksAlways()
	}
	return nil
}

// --------------------------------------------------------------- settings --

type SettingsPayload struct {
	Settings  core.Settings `json:"settings"`
	Autostart bool          `json:"autostart"`
}

func (a *App) apiSaveSettings(p SettingsPayload) error {
	s := p.Settings
	if s.BackupsToKeep < 1 || s.BackupsToKeep > 100 {
		return errors.New("Keep between 1 and 100 backups per game.")
	}
	if s.StartTimeoutSeconds < 30 || s.StartTimeoutSeconds > 1800 {
		return errors.New("The start wait must be between 30 and 1800 seconds.")
	}
	s.SteamAccountID = strings.TrimSpace(s.SteamAccountID)
	key, err := protectSecret(s.GridDBKey)
	if err != nil {
		return fmt.Errorf("couldn't save the SteamGridDB key safely: %w", err)
	}
	s.GridDBKey = key
	if s.SteamAPIKey, err = protectSecret(s.SteamAPIKey); err != nil {
		return fmt.Errorf("couldn't save the Steam Web API key safely: %w", err)
	}
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	checkNow := s.CheckUpdates && !cfg.Settings.CheckUpdates
	// Backups have their own calls (cloudbackup.go); what the page sends
	// back for them is ignored.
	old := cfg.Settings
	s.BackupDir, s.BackupAuto, s.BackupKeep, s.BackupSalt, s.BackupKey = old.BackupDir, old.BackupAuto, old.BackupKeep, old.BackupSalt, old.BackupKey
	s.CloudSync = old.CloudSync
	saveDBOn := s.SaveDB && !old.SaveDB
	cfg.Settings = s
	if err := a.saveConfig(cfg); err != nil {
		return err
	}
	if checkNow {
		go a.checkUpdate(false)
	}
	if saveDBOn {
		go func() {
			if err := a.refreshSaveDB(true); err != nil {
				a.emit("toast", map[string]any{"kind": "error", "text": err.Error()})
				return
			}
			a.emit("gameSaves", map[string]any{"changed": true})
			a.startScan() // the games found so far get the list's locations
			a.backUpChangedGames()
		}()
	}
	if err := win.SetAutostart(Name, fmt.Sprintf(`"%s" --tray`, a.exePath), p.Autostart); err != nil {
		return fmt.Errorf("couldn't change the startup setting: %w", err)
	}
	if s.AutoSteamHooks {
		a.autoHooks()
	}
	return nil
}

// ------------------------------------------------------------ discovery ---

func (a *App) apiSteamGames() ([]core.SteamGame, error) {
	if a.env.SteamPath == "" {
		return nil, errors.New("Steam wasn't found on this PC.")
	}
	return core.InstalledSteamGames(a.env.SteamPath), nil
}

type Inspect struct {
	ProcessName string              `json:"processName"`
	Name        string              `json:"name"`
	Preset      *core.Preset        `json:"preset,omitempty"`
	Candidates  []core.ExeCandidate `json:"candidates"`
	Suggestions []core.Suggestion   `json:"suggestions"`
	// Proposed are the locations to fill in without asking (the preset, else
	// the best suggestions), the same choice the scan makes.
	Proposed []string `json:"proposed"`
}

func (a *App) apiInspectSteam(appID int) (Inspect, error) {
	var game *core.SteamGame
	for _, g := range core.InstalledSteamGames(a.env.SteamPath) {
		if g.AppID == appID {
			g := g
			game = &g
			break
		}
	}
	if game == nil {
		return Inspect{}, errors.New("That Steam game isn't installed.")
	}
	out := Inspect{Name: game.Name}
	out.Candidates = core.ProcessCandidates(game.InstallDir)
	if p, ok := core.KnownSteamGames[appID]; ok {
		p := p
		out.Preset = &p
		out.ProcessName = p.ProcessName
	} else if len(out.Candidates) > 0 {
		out.ProcessName = out.Candidates[0].Name
	}
	out.Suggestions = a.env.SuggestPaths(game.Name, game.InstallDir, appID)
	if out.Candidates == nil {
		out.Candidates = []core.ExeCandidate{}
	}
	if out.Suggestions == nil {
		out.Suggestions = []core.Suggestion{}
	}
	out.Proposed = core.ProposePaths(out.Suggestions, out.Preset)
	if out.Proposed == nil {
		out.Proposed = []string{}
	}
	return out, nil
}

func (a *App) apiInspectExe(path string) (Inspect, error) {
	exe, err := a.env.ResolveExe(path)
	if err != nil {
		return Inspect{}, err
	}
	_, containers := gameFolders(a.env)
	f := a.env.ExeGame(exe, containers) // its name, from the game's folder rather than bin, steam or Win64
	base, name := f.ProcessName, f.Name
	out := Inspect{ProcessName: base, Name: name, Candidates: []core.ExeCandidate{}}
	dir := f.Dir
	if dir == "" {
		dir = base
	}
	out.Suggestions = a.env.SuggestPaths(name, dir, 0)
	if out.Suggestions == nil {
		out.Suggestions = []core.Suggestion{}
	}
	out.Proposed = core.ProposePaths(out.Suggestions, nil)
	if out.Proposed == nil {
		out.Proposed = []string{}
	}
	return out, nil
}

type PathCheck struct {
	Resolved string `json:"resolved"`
	Exists   bool   `json:"exists"`
	IsDir    bool   `json:"isDir"`
	Size     int64  `json:"size"`
	Error    string `json:"error,omitempty"`
}

func (a *App) apiCheckPath(raw string) PathCheck {
	p, err := a.env.Resolve(raw)
	if err != nil {
		return PathCheck{Error: err.Error()}
	}
	out := PathCheck{Resolved: p}
	if st, err := os.Stat(p); err == nil {
		out.Exists, out.IsDir = true, st.IsDir()
		out.Size = core.Size(p)
	}
	return out
}

// apiBrowse opens a native picker after the current call returns; the result
// arrives as a "browse" event.
func (a *App) apiBrowse(reqID, kind, start string) {
	a.ui(func() {
		opt := win.BrowseOptions{Owner: a.hwnd}
		if start != "" {
			if p, err := a.env.Resolve(start); err == nil {
				for p != "" {
					if st, err := os.Stat(p); err == nil && st.IsDir() {
						break
					}
					parent := filepath.Dir(p)
					if parent == p {
						p = ""
						break
					}
					p = parent
				}
				opt.Start = p
			}
		}
		switch kind {
		case "exe":
			opt.Title = "Choose the game's .exe"
			opt.Filters = []win.FileFilter{{Name: "Programs (*.exe)", Spec: "*.exe"}}
		case "image":
			opt.Title = "Choose a picture"
			opt.Filters = []win.FileFilter{{Name: "Pictures (*.png, *.jpg, *.gif)", Spec: "*.png;*.jpg;*.jpeg;*.gif"}}
		case "backup":
			opt.Title = "Choose a Lootnest backup or profile"
			opt.Filters = []win.FileFilter{{Name: "Lootnest backups (*.zip, *.lnbak)", Spec: "*.zip;*.lnbak"}}
		case "backupDir":
			opt.Title = "Choose where backups go"
			opt.Folder = true
		case "exportDir":
			opt.Title = "Choose where to save the profile"
			opt.Folder = true
		case "update":
			opt.Title = "Choose the update"
			opt.Filters = []win.FileFilter{{Name: "Archives (*.zip, *.7z, *.rar)", Spec: "*.zip;*.7z;*.rar"}}
		case "installer":
			opt.Title = "Choose the update's installer"
			opt.Filters = []win.FileFilter{{Name: "Installers (*.exe)", Spec: "*.exe"}}
		case "updateDir":
			opt.Title = "Choose the update's folder"
			opt.Folder = true
		case "trySaveDir":
			opt.Title = "Choose the folder with the save to try"
			opt.Folder = true
		case "trySaveFile":
			opt.Title = "Choose the save to try"
			opt.Filters = []win.FileFilter{{Name: "Saves and archives", Spec: "*.*"}, {Name: "Archives (*.zip, *.7z, *.rar)", Spec: "*.zip;*.7z;*.rar"}}
		case "file":
			opt.Title = "Choose a settings or save file"
			opt.Filters = []win.FileFilter{{Name: "All files", Spec: "*.*"}}
		default:
			opt.Title = "Choose a settings or save folder"
			opt.Folder = true
		}
		path, err := win.Browse(opt)
		res := map[string]any{"reqId": reqID}
		switch {
		case errors.Is(err, win.ErrCancelled):
			res["cancelled"] = true
		case err != nil:
			res["error"] = err.Error()
		default:
			res["path"] = path
			res["tokenized"] = a.env.Tokenize(path)
		}
		a.emit("browse", res)
	})
}

// -------------------------------------------------------------- actions ---

func (a *App) apiPlay(r swapRequest) (Need, error) {
	r.Launch = true
	return a.startSwap(r)
}

func (a *App) apiLoad(r swapRequest) (Need, error) {
	r.Launch = false
	r.ReqID = ""
	return a.startSwap(r)
}

// apiSteamPick answers a Steam request: swap the files, then let the
// waiting wrapper start the game.
func (a *App) apiSteamPick(r swapRequest) (Need, error) {
	r.Launch = false
	a.mu.Lock()
	gid, ok := a.requests[r.ReqID]
	a.mu.Unlock()
	if !ok || gid != r.GameID {
		return Need{}, errors.New("Steam stopped waiting for this game. Press Play in Steam again.")
	}
	return a.startSwap(r)
}

var badFileChars = regexp.MustCompile(`[\\/:*?"<>|]+`)

func (a *App) apiShortcut(gameID string) (string, error) {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	g := cfg.Game(gameID)
	if g == nil {
		return "", errors.New("That game no longer exists.")
	}
	desktop := win.Desktop()
	if desktop == "" {
		return "", errors.New("Couldn't find your Desktop folder.")
	}
	lnk := filepath.Join(desktop, strings.TrimSpace(badFileChars.ReplaceAllString(g.Name, " "))+" (Lootnest).lnk")
	icon, iconIdx := a.exePath, 0
	switch g.Launch.Type {
	case "exe", "uri":
		if exe, err := a.env.ResolveExe(g.Launch.Exe); err == nil {
			icon = exe
		}
	case "steam":
		for _, sg := range core.InstalledSteamGames(a.env.SteamPath) {
			if sg.AppID == g.Launch.AppID {
				if exe := core.FindExe(sg.InstallDir, g.ProcessName); exe != "" {
					icon = exe
				}
				break
			}
		}
	}
	err := win.CreateShortcut(win.Shortcut{
		Path: lnk, Target: a.exePath, Args: "--launch " + g.ID, WorkDir: filepath.Dir(a.exePath),
		Icon: icon, IconIndex: iconIdx, Description: "Pick who's playing, then start " + g.Name,
	})
	if err != nil {
		return "", fmt.Errorf("couldn't create the shortcut: %w", err)
	}
	found := false
	for _, s := range cfg.Shortcuts {
		if strings.EqualFold(s, lnk) {
			found = true
		}
	}
	if !found {
		cfg.Shortcuts = append(cfg.Shortcuts, lnk)
		_ = a.saveConfig(cfg)
	}
	a.logf("Created shortcut %s", lnk)
	return lnk, nil
}

func (a *App) apiOpen(kind, arg string) error {
	var target string
	switch kind {
	case "data":
		target = DataDir()
	case "backupFolder":
		return a.openBackupFolder()
	case "gameSaves":
		return a.openGameSaves(arg)
	case "backups":
		target = filepath.Join(a.store.BackupsDir(), filepath.Base(arg))
	case "profile":
		target = filepath.Join(a.store.ProfilesDir(), filepath.Base(arg))
	case "path":
		p, err := a.env.Resolve(arg)
		if err != nil {
			return err
		}
		for {
			if _, err := os.Stat(p); err == nil {
				break
			}
			parent := filepath.Dir(p)
			if parent == p {
				return errors.New("That folder doesn't exist yet. Start the game once so it creates it.")
			}
			p = parent
		}
		target = p
	case "pcgw":
		target = "https://www.pcgamingwiki.com/w/index.php?search=" + url.QueryEscape(arg)
		return win.ShellOpen(target)
	case "link": // a link inside an announcement: web pages only, in the browser
		u := core.SafeLink(arg)
		if u == "" {
			return errors.New("That link can't be opened.")
		}
		return win.ShellOpen(u)
	case "store": // a game's page in a store, from Find games
		return openStore(arg)
	case "news": // a game's patch notes or announcement, on Steam's own site only
		if !core.SafeNewsURL(arg) {
			return errors.New("That isn't a Steam news page.")
		}
		return win.ShellOpen(arg)
	case "release": // the newer release's page ("What's new"); only Lootnest's own
		updateMu.Lock()
		page := updateFound.Page
		updateMu.Unlock()
		if !strings.HasPrefix(page, "https://github.com/c-dinca/lootnest/releases/") {
			return errors.New("Check for updates first.")
		}
		return win.ShellOpen(page)
	case "log":
		target = a.store.LogPath()
	default:
		return errors.New("unknown location")
	}
	if _, err := os.Stat(target); err != nil {
		if kind == "log" {
			return errors.New("Nothing has been logged yet.")
		}
		_ = os.MkdirAll(target, 0o755)
	}
	return win.ShellOpen(target)
}

func (a *App) apiLog() string {
	b, err := os.ReadFile(a.store.LogPath())
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimRight(string(b), "\r\n"), "\n")
	if len(lines) > 300 {
		lines = lines[len(lines)-300:]
	}
	return strings.Join(lines, "\n")
}
