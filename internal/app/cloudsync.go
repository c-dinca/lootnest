//go:build windows

package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
)

// ---------------------------------------------------------------------------
// Cloud saves (core/cloudsync.go has the format): each player's saves go up
// to the cloud folder after every session, and newer ones come down before a
// game starts, so every PC plays from the newest. When a player's saves
// changed on two PCs since they last synced, they choose which to keep;
// the other copy is kept as a backup.
// ---------------------------------------------------------------------------

var syncMu sync.Mutex // one sync at a time

// errCloudPassword: the cloud folder is encrypted and this PC has no key.
var errCloudPassword = errors.New("The cloud folder has a password; enter it on Cloud saves to sync.")

// cloud is the cloud folder to sync with, or nil when sync is off.
func (a *App) cloud(s core.Settings) (*core.Cloud, error) {
	if !s.CloudSync || s.BackupDir == "" {
		return nil, nil
	}
	if st, err := os.Stat(s.BackupDir); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("The cloud folder (%s) isn't there. Is the drive connected, and the cloud app running?", s.BackupDir)
	}
	// The folder decides: with a key.json its files are encrypted, and this
	// PC needs the same password; without, they aren't.
	if !core.CloudHasKey(s.BackupDir) {
		return &core.Cloud{Root: s.BackupDir, Computer: computerName()}, nil
	}
	key, _, err := backupKey(s)
	if err != nil || key == nil || !core.CloudKeyMatches(s.BackupDir, key) {
		return nil, errCloudPassword
	}
	return &core.Cloud{Root: s.BackupDir, Key: key, Computer: computerName()}, nil
}

func (a *App) profileDirs(pid string, g *core.Game) func(core.PathEntry) string {
	dir := a.store.ProfileDir(pid, g.ID)
	return func(pe core.PathEntry) string { return filepath.Join(dir, pe.ID) }
}

// syncGame is g with the locations cloud saves carries: its own paths and
// each profile's achievements (all: every place they could go, to bring
// them down to a PC that has none yet).
func (a *App) syncGame(g *core.Game, all bool) *core.Game {
	c := *g
	c.Paths = a.eng.SyncPaths(g, all)
	return &c
}

func (a *App) liveDirs(g *core.Game) func(core.PathEntry) string {
	return func(pe core.PathEntry) string {
		if strings.HasPrefix(pe.Path, core.AchToken) {
			if p := a.eng.AchLive(g, pe.ID); p != "" {
				return p
			}
			return filepath.Join(os.TempDir(), "lootnest-unresolved", pe.ID) // never exists
		}
		p, err := a.env.Resolve(pe.Path)
		if err != nil {
			return filepath.Join(os.TempDir(), "lootnest-unresolved", pe.ID) // never exists
		}
		return p
	}
}

// syncPlan is what syncing a player's saves for a game would do.
type syncPlan struct {
	Action   core.SyncAction
	Meta     *core.SyncMeta
	Local    string
	Active   bool // their files are the ones in place
	CloudPID string
	Key      string
	Rec      core.SyncRecord
	Here     time.Time // when they last changed on this PC, for a conflict
}

// planSync compares a player's saves here with the cloud's. With live, the
// files in place count when they're the player's (they may have played
// without Lootnest); else their profile.
func (a *App) planSync(c *core.Cloud, cfg *core.Config, g *core.Game, pid string, live bool) (*syncPlan, error) {
	st := a.store.LoadSync()
	if cps := c.Players(); core.LinkPlayers(&st, cfg.Players, cps) {
		links := st.Links
		_ = a.store.UpdateSync(func(s *core.SyncState) { s.Links = links })
	}
	p := &syncPlan{CloudPID: st.CloudPlayerID(pid), Key: core.GameSyncKey(g)}
	p.Active = a.store.LoadState()[g.ID] == pid
	dirs := a.profileDirs(pid, g)
	if live && p.Active {
		dirs = a.liveDirs(g)
	}
	var err error
	sg := a.syncGame(g, false)
	if p.Active || a.eng.HasProfile(pid, g.ID) {
		if p.Local, err = core.SyncHash(sg, dirs); err != nil {
			return nil, err
		}
	}
	p.Meta = c.Meta(p.CloudPID, p.Key)
	rec, seen := st.Records[pid][p.Key]
	p.Rec = rec
	p.Action = core.DecideSync(p.Local, p.Meta, rec, seen)
	if p.Action == core.SyncConflict {
		p.Here = core.SavesChanged(sg, dirs)
	}
	return p, nil
}

func (a *App) saveRecord(pid, key, local, cloud string) {
	_ = a.store.UpdateSync(func(s *core.SyncState) {
		if s.Records[pid] == nil {
			s.Records[pid] = map[string]core.SyncRecord{}
		}
		s.Records[pid][key] = core.SyncRecord{Local: local, Cloud: cloud, At: time.Now().UTC()}
	})
}

// syncUp uploads a player's saves for a game when only this PC changed
// them (after a session, or after their files were swapped out).
func (a *App) syncUp(cfg *core.Config, g *core.Game, pid string) {
	syncMu.Lock()
	defer syncMu.Unlock()
	c, err := a.cloud(cfg.Settings)
	if c == nil {
		if err != nil {
			a.logf("Cloud saves: %v", err)
		}
		return
	}
	p, err := a.planSync(c, cfg, g, pid, false)
	if err != nil {
		a.logf("ERROR cloud saves for %s: %v", g.Name, err)
		return
	}
	switch p.Action {
	case core.SyncUp:
		a.upload(c, cfg, g, pid, p)
	case core.SyncNothing:
		if p.Meta != nil {
			a.saveRecord(pid, p.Key, p.Local, p.Meta.Hash)
		}
	case core.SyncConflict:
		a.logf("Cloud saves: %s's %s changed here and on %s; Lootnest asks before the next start", cfg.PlayerName(pid), g.Name, p.Meta.Computer)
	}
	a.emit("savesChanged", map[string]any{})
}

func (a *App) upload(c *core.Cloud, cfg *core.Config, g *core.Game, pid string, p *syncPlan) {
	pl := cfg.Player(pid)
	if pl == nil {
		return
	}
	version := 1
	if p.Meta != nil {
		version = p.Meta.Version + 1
	}
	meta, err := c.Upload(p.CloudPID, pl, a.syncGame(g, false), a.profileDirs(pid, g), p.Local, version)
	if err != nil {
		a.logf("ERROR uploading %s's %s saves: %v", pl.Name, g.Name, err)
		return
	}
	a.saveRecord(pid, p.Key, p.Local, meta.Hash)
	a.logf("Cloud saves: uploaded %s's %s (%d bytes)", pl.Name, g.Name, meta.Size)
}

// syncDown brings a player's saves for a game from the cloud: into the
// files in place when they're the player's (after a backup of those), else
// into their profile. With keepHere, their saves here are kept first as a
// backup (they chose the cloud's in a conflict).
func (a *App) syncDown(c *core.Cloud, cfg *core.Config, g *core.Game, pid string, p *syncPlan, keepHere bool) error {
	tmp := filepath.Join(DataDir(), "sync-tmp-"+time.Now().Format("150405.000"))
	defer os.RemoveAll(tmp)
	dirs := a.profileDirs(pid, g)
	if p.Active {
		if _, err := a.eng.Backup(g, cfg.Settings.BackupsToKeep); err != nil {
			return fmt.Errorf("couldn't back up the files in place first: %w", err)
		}
		dirs = a.liveDirs(g)
	} else if keepHere && a.eng.HasProfile(pid, g.ID) {
		keep := filepath.Join(a.store.BackupsDir(), g.ID, time.Now().Format("20060102-150405")+"-"+pid+"-before-cloud")
		if err := core.Mirror(a.store.ProfileDir(pid, g.ID), keep); err != nil {
			return fmt.Errorf("couldn't keep the saves here first: %w", err)
		}
	}
	skipped, err := c.Download(p.CloudPID, a.syncGame(g, true), dirs, tmp)
	if err != nil {
		return err
	}
	if !p.Active {
		_ = os.MkdirAll(a.store.ProfileDir(pid, g.ID), 0o755)
		_ = os.WriteFile(filepath.Join(a.store.ProfileDir(pid, g.ID), ".initialized"), []byte(time.Now().Format(time.RFC3339)), 0o644)
	}
	local, _ := core.SyncHash(a.syncGame(g, false), dirs)
	a.saveRecord(pid, p.Key, local, p.Meta.Hash)
	a.logf("Cloud saves: brought %s's %s from %s%s", cfg.PlayerName(pid), g.Name, p.Meta.Computer,
		map[bool]string{true: fmt.Sprintf(" (%d save locations this PC's game doesn't have were left out)", skipped), false: ""}[skipped > 0])
	return nil
}

// keepCloudAside saves the cloud's copy into backups before this PC's is
// uploaded over it (they chose this PC's in a conflict).
func (a *App) keepCloudAside(c *core.Cloud, g *core.Game, p *syncPlan) {
	dir := filepath.Join(a.store.BackupsDir(), g.ID, time.Now().Format("20060102-150405")+"-cloud-"+p.Meta.Computer)
	tmp := filepath.Join(DataDir(), "sync-tmp-"+time.Now().Format("150405.000"))
	defer os.RemoveAll(tmp)
	if _, err := c.Download(p.CloudPID, a.syncGame(g, true), func(pe core.PathEntry) string { return filepath.Join(dir, pe.ID) }, tmp); err != nil {
		a.logf("ERROR keeping the cloud's %s saves: %v", g.Name, err)
	}
}

// CloudConflict is what the picker shows when a player's saves changed on
// two PCs.
type CloudConflict struct {
	Computer string    `json:"computer"`
	Changed  time.Time `json:"changed"`
}

// beforePlay decides what syncing needs before a player's files load: a
// question (a conflict without an answer), or a plan for runSwap.
func (a *App) beforePlay(cfg *core.Config, g *core.Game, pid, answer string) (*syncPlan, *CloudConflict) {
	c, err := a.cloud(cfg.Settings)
	if c == nil {
		if err != nil {
			a.logf("Cloud saves: %v", err)
		}
		return nil, nil
	}
	// This runs while the window waits for the game to start: never longer
	// than a few seconds behind an upload.
	locked := false
	for i := 0; i < 50 && !locked; i++ {
		if locked = syncMu.TryLock(); !locked {
			time.Sleep(100 * time.Millisecond)
		}
	}
	if !locked {
		a.logf("Cloud saves: still uploading; %s starts with the saves here", g.Name)
		return nil, nil
	}
	defer syncMu.Unlock()
	p, err := a.planSync(c, cfg, g, pid, true)
	if err != nil {
		a.logf("ERROR cloud saves for %s: %v", g.Name, err)
		return nil, nil
	}
	if p.Action == core.SyncConflict && answer == "" {
		return nil, &CloudConflict{Computer: p.Meta.Computer, Changed: p.Meta.Changed}
	}
	return p, nil
}

// applyBeforePlay runs the plan beforePlay made, before the swap.
func (a *App) applyBeforePlay(cfg *core.Config, g *core.Game, pid, answer string, p *syncPlan) error {
	if p == nil {
		return nil
	}
	c, err := a.cloud(cfg.Settings)
	if c == nil {
		return err
	}
	syncMu.Lock()
	defer syncMu.Unlock()
	switch {
	case p.Action == core.SyncDown:
		return a.syncDown(c, cfg, g, pid, p, false)
	case p.Action == core.SyncConflict && answer == "cloud":
		return a.syncDown(c, cfg, g, pid, p, true)
	case p.Action == core.SyncConflict && answer == "here":
		a.keepCloudAside(c, g, p)
		// This PC's saves win: they go up after the session (or now, when
		// they're already in the profile).
		a.saveRecord(pid, p.Key, "", p.Meta.Hash)
	}
	return nil
}

// ----------------------------------------------------------------- status ---

type SavesGame struct {
	GameID   string    `json:"gameId"`
	PlayerID string    `json:"playerId"`
	State    string    `json:"state"` // same, up, down, conflict, none
	Computer string    `json:"computer,omitempty"`
	Changed  time.Time `json:"changed,omitempty"`
	Here     time.Time `json:"here,omitzero"` // changed on this PC, for a conflict
	Size     int64     `json:"size,omitempty"`
}

type SavesStatus struct {
	Error   string      `json:"error,omitempty"`
	Locked  bool        `json:"locked,omitempty"` // needs the folder's password
	Games   []SavesGame `json:"games"`
	Players []string    `json:"players"` // players in the cloud not on this PC
}

// apiSavesStatus works out every player's state for every game in the
// background (it reads all the saves) and sends a "savesStatus" event.
func (a *App) apiSavesStatus() {
	go func() {
		a.mu.Lock()
		cfg := a.cfg.Clone()
		a.mu.Unlock()
		st := SavesStatus{Games: []SavesGame{}, Players: []string{}}
		defer func() { a.emit("savesStatus", map[string]any{"status": st}) }()
		c, err := a.cloud(cfg.Settings)
		if errors.Is(err, errCloudPassword) {
			st.Locked = true
		}
		if err != nil {
			st.Error = err.Error()
			return
		}
		if c == nil {
			return
		}
		syncMu.Lock()
		defer syncMu.Unlock()
		for i := range cfg.Games {
			g := &cfg.Games[i]
			for _, pl := range cfg.Players {
				p, err := a.planSync(c, cfg, g, pl.ID, true)
				if err != nil {
					continue
				}
				if p.Local == "" && p.Meta == nil {
					continue
				}
				cg := SavesGame{GameID: g.ID, PlayerID: pl.ID}
				switch {
				case p.Action == core.SyncUp && p.Meta == nil:
					cg.State = "none"
				case p.Action == core.SyncUp:
					cg.State = "up"
				case p.Action == core.SyncDown:
					cg.State = "down"
				case p.Action == core.SyncConflict:
					cg.State, cg.Here = "conflict", p.Here
				default:
					cg.State = "same"
				}
				if p.Meta != nil {
					cg.Computer, cg.Changed, cg.Size = p.Meta.Computer, p.Meta.Changed, p.Meta.Size
				}
				st.Games = append(st.Games, cg)
			}
		}
		sst := a.store.LoadSync()
		for _, cp := range c.Players() {
			here := false
			for _, pl := range cfg.Players {
				here = here || sst.CloudPlayerID(pl.ID) == cp.ID
			}
			if !here {
				st.Players = append(st.Players, cp.Name)
			}
		}
	}()
}

// apiSyncNow uploads what changed here and brings what changed elsewhere
// (for games that aren't running), then sends "savesChanged".
func (a *App) apiSyncNow() error {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	busy := map[string]bool{}
	for id := range a.sessions {
		busy[id] = true
	}
	a.mu.Unlock()
	c, err := a.cloud(cfg.Settings)
	if c == nil {
		if err == nil {
			err = errors.New("Cloud saves are off.")
		}
		return err
	}
	go func() {
		out := map[string]any{"done": true}
		defer a.emit("savesChanged", out)
		syncMu.Lock()
		defer syncMu.Unlock()
		up, down := 0, 0
		for i := range cfg.Games {
			g := &cfg.Games[i]
			if busy[g.ID] || isRunning(g) {
				continue
			}
			for _, pl := range cfg.Players {
				p, err := a.planSync(c, cfg, g, pl.ID, true)
				if err != nil {
					continue
				}
				switch p.Action {
				case core.SyncUp:
					if p.Active {
						// The files in place are newer than the profile.
						if err := a.eng.SaveLiveTo(g, pl.ID); err != nil {
							continue
						}
					}
					a.upload(c, cfg, g, pl.ID, p)
					up++
				case core.SyncDown:
					if err := a.syncDown(c, cfg, g, pl.ID, p, false); err != nil {
						a.logf("ERROR cloud saves: %v", err)
						continue
					}
					down++
				}
			}
		}
		out["up"], out["down"] = up, down
	}()
	return nil
}

// apiSyncResolve settles a conflict from the Cloud saves page: "here" keeps
// this PC's saves (uploaded now), "cloud" takes the cloud's. The other copy
// is kept in backups.
func (a *App) apiSyncResolve(gameID, pid, answer string) error {
	if answer != "here" && answer != "cloud" {
		return errors.New("Choose which saves to keep.")
	}
	a.mu.Lock()
	cfg := a.cfg.Clone()
	busy := a.sessions[gameID] != nil
	a.mu.Unlock()
	g := cfg.Game(gameID)
	if g == nil || cfg.Player(pid) == nil {
		return errors.New("That game or profile no longer exists.")
	}
	if busy || isRunning(g) {
		return fmt.Errorf("%s is running; choose after it closes.", g.Name)
	}
	c, err := a.cloud(cfg.Settings)
	if c == nil {
		return err
	}
	go func() {
		out := map[string]any{"done": true}
		defer a.emit("savesChanged", out)
		syncMu.Lock()
		defer syncMu.Unlock()
		p, err := a.planSync(c, cfg, g, pid, true)
		if err != nil {
			out["error"] = err.Error()
			return
		}
		if p.Meta == nil {
			return
		}
		if answer == "cloud" {
			if err := a.syncDown(c, cfg, g, pid, p, true); err != nil {
				out["error"] = err.Error()
			}
			return
		}
		a.keepCloudAside(c, g, p)
		if p.Active {
			if err := a.eng.SaveLiveTo(g, pid); err != nil {
				out["error"] = err.Error()
				return
			}
		}
		local, _ := core.SyncHash(a.syncGame(g, false), a.profileDirs(pid, g))
		p.Local = local
		a.upload(c, cfg, g, pid, p)
	}()
	return nil
}
