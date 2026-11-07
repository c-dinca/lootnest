//go:build windows

package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"lootnest/internal/core"
)

// ---------------------------------------------------------------------------
// Safety you can see: the copies made before each swap, put back in one
// click; what Steam Cloud replaced, kept on the game's page until it's read;
// and which games ship with an anti-cheat.
// ---------------------------------------------------------------------------

// GameBackup is one copy made before a swap, for the Manage panel.
type GameBackup struct {
	ID    string `json:"id"`
	When  string `json:"when"`
	Owner string `json:"owner"`
	Size  int64  `json:"size"`
}

func (a *App) apiGameBackups(gameID string) []GameBackup {
	out := []GameBackup{}
	for _, b := range a.eng.Backups(gameID) {
		out = append(out, GameBackup{ID: core.BackupID(b), When: b.When, Owner: b.Owner, Size: core.Size(b.Dir)})
	}
	return out
}

// apiRestoreBackup puts a copy back for the profile it belongs to.
func (a *App) apiRestoreBackup(gameID, id string) (map[string]string, error) {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	_, active := a.sessions[gameID]
	a.mu.Unlock()
	g := cfg.Game(gameID)
	if g == nil {
		return nil, errors.New("That game no longer exists.")
	}
	if active || isRunning(g) {
		return nil, errors.New(g.Name + " is running. Close it first.")
	}
	owner, where, err := a.eng.RestoreBackup(cfg, gameID, id)
	if err != nil {
		return nil, errors.New(sentence(err))
	}
	a.logf("Put back the copy of %s from %s for %s (%s)", g.Name, id, cfg.PlayerName(owner), where)
	a.emitSessions()
	return map[string]string{"owner": owner, "where": where}, nil
}

// ------------------------------------------------------------ conflicts --

// SteamCloudNote is what Steam Cloud replaced in a session, kept for the
// game's page until it's dismissed.
type SteamCloudNote struct {
	At     time.Time `json:"at"`
	Player string    `json:"player"`
	Files  []string  `json:"files"`
}

var conflictMu sync.Mutex

func conflictsPath() string { return filepath.Join(DataDir(), "conflicts.json") }

func loadConflicts() map[string]SteamCloudNote {
	out := map[string]SteamCloudNote{}
	if b, err := os.ReadFile(conflictsPath()); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	return out
}

func (a *App) noteConflict(gameID, playerID string, files []string) {
	conflictMu.Lock()
	defer conflictMu.Unlock()
	c := loadConflicts()
	if len(files) > 12 {
		files = files[:12]
	}
	c[gameID] = SteamCloudNote{At: time.Now().UTC(), Player: playerID, Files: files}
	if b, err := json.Marshal(c); err == nil {
		_ = os.WriteFile(conflictsPath(), b, 0o644)
	}
}

func (a *App) apiDismissConflict(gameID string) error {
	conflictMu.Lock()
	defer conflictMu.Unlock()
	c := loadConflicts()
	delete(c, gameID)
	b, _ := json.Marshal(c)
	return os.WriteFile(conflictsPath(), b, 0o644)
}

// ------------------------------------------------------------ anti-cheat --

// apiGameFacts says what Lootnest can tell about a game from its folder: the
// anti-cheat it ships with, if any.
func (a *App) apiGameFacts(dir string) map[string]string {
	if filepath.Ext(dir) != "" {
		dir = filepath.Dir(dir)
	}
	if d, err := a.env.ResolveExe(dir); err == nil && d != "" {
		dir = d
	}
	return map[string]string{"antiCheat": core.AntiCheat(dir)}
}
