package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var (
	// ErrNeedOwner: Lootnest has never managed this game, so it must be told
	// whose files are currently on the PC before it can swap.
	ErrNeedOwner = errors.New("need-owner")
	// ErrNeedNewMode: the chosen player has no profile yet for this game; the
	// caller must say whether to start from a copy or fresh.
	ErrNeedNewMode = errors.New("need-new-mode")
)

const (
	OwnerNobody = "none"
	NewCopy     = "copy"
	NewFresh    = "fresh"
)

const initMarker = ".initialized"

type Engine struct {
	Store *Store
	Env   *Env
	Log   func(format string, args ...any)
	Now   func() time.Time
	// Name turns a player ID into a display name for the log.
	Name func(playerID string) string
}

func (e *Engine) name(id string) string {
	if e.Name != nil {
		return e.Name(id)
	}
	return id
}

func (e *Engine) logf(format string, args ...any) {
	if e.Log != nil {
		e.Log(format, args...)
	}
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

type resolved struct {
	entry PathEntry
	live  string
}

// resolveAll resolves every path of a game up front so a bad path aborts
// before anything on disk is touched.
func (e *Engine) resolveAll(g *Game) ([]resolved, error) {
	out := make([]resolved, 0, len(g.Paths))
	for _, p := range g.Paths {
		live, err := e.Env.Resolve(p.Path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p.Path, err)
		}
		out = append(out, resolved{p, live})
	}
	out = append(out, e.achievementEntries(g, out)...)
	// Two entries must not overlap, or mirroring one would wipe the other.
	for i := range out {
		for j := range out {
			if i != j && isWithin(out[i].live, out[j].live) {
				return nil, fmt.Errorf("%s is inside %s; keep only the outer one", out[i].entry.Path, out[j].entry.Path)
			}
		}
	}
	return out, nil
}

// CheckPaths validates a game's paths without touching anything.
func (e *Engine) CheckPaths(g *Game) error {
	_, err := e.resolveAll(g)
	return err
}

func (e *Engine) HasProfile(playerID, gameID string) bool {
	return exists(filepath.Join(e.Store.ProfileDir(playerID, gameID), initMarker))
}

// SaveLiveTo copies the game's live files into a player's profile. It
// refuses while a swap of the game is unfinished: the files in place may
// then be a mix of two profiles.
func (e *Engine) SaveLiveTo(g *Game, playerID string) error {
	rs, err := e.resolveAll(g)
	if err != nil {
		return err
	}
	if _, pending := e.Store.PendingSwaps()[g.ID]; pending {
		return fmt.Errorf("the last swap of %s didn't finish; Lootnest finishes it before the game next starts, or when Lootnest starts again", g.Name)
	}
	return e.saveLive(g, playerID, rs)
}

func (e *Engine) saveLive(g *Game, playerID string, rs []resolved) error {
	dir := e.Store.ProfileDir(playerID, g.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, r := range rs {
		if err := Mirror(r.live, filepath.Join(dir, r.entry.ID)); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(dir, initMarker), []byte(e.now().Format(time.RFC3339)), 0o644); err != nil {
		return err
	}
	e.logf("Saved %s's files for %s", e.name(playerID), g.Name)
	return nil
}

// copyProfile starts to's profile for a game as a copy of from's.
func (e *Engine) copyProfile(g *Game, from, to string, rs []resolved) error {
	src, dst := e.Store.ProfileDir(from, g.ID), e.Store.ProfileDir(to, g.ID)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, r := range rs {
		if err := Mirror(filepath.Join(src, r.entry.ID), filepath.Join(dst, r.entry.ID)); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(dst, initMarker), []byte(e.now().Format(time.RFC3339)), 0o644); err != nil {
		return err
	}
	e.logf("Started %s's files for %s from a copy of %s's", e.name(to), g.Name, e.name(from))
	return nil
}

// restore copies a player's profile into the live locations.
func (e *Engine) restore(g *Game, playerID string, rs []resolved) error {
	dir := e.Store.ProfileDir(playerID, g.ID)
	for _, r := range rs {
		if err := Mirror(filepath.Join(dir, r.entry.ID), r.live); err != nil {
			return err
		}
	}
	e.logf("Loaded %s's files for %s", e.name(playerID), g.Name)
	return nil
}

// Backup copies the current live files to backups/<game>/<timestamp> and
// prunes old backups.
func (e *Engine) Backup(g *Game, keep int) (string, error) {
	rs, err := e.resolveAll(g)
	if err != nil {
		return "", err
	}
	root := filepath.Join(e.Store.BackupsDir(), g.ID)
	stamp := e.now().Format("20060102-150405")
	dir := filepath.Join(root, stamp)
	for n := 2; exists(dir); n++ {
		dir = filepath.Join(root, fmt.Sprintf("%s-%d", stamp, n))
	}
	any := false
	for _, r := range rs {
		if exists(r.live) {
			if err := Mirror(r.live, filepath.Join(dir, r.entry.ID)); err != nil {
				return "", err
			}
			any = true
		}
	}
	if any {
		st := e.Store.LoadState()
		_ = os.WriteFile(filepath.Join(dir, "owner.txt"), []byte(st[g.ID]), 0o644)
	}
	e.pruneBackups(root, keep)
	if !any {
		return "", nil
	}
	return dir, nil
}

func (e *Engine) pruneBackups(root string, keep int) {
	if keep <= 0 {
		keep = 10
	}
	ents, err := os.ReadDir(root)
	if err != nil {
		return
	}
	var dirs []string
	for _, d := range ents {
		if d.IsDir() {
			dirs = append(dirs, d.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dirs)))
	for i, d := range dirs {
		if i >= keep {
			_ = removeAll(filepath.Join(root, d))
		}
	}
}

type ActivateOptions struct {
	// FirstOwner answers ErrNeedOwner: a player ID, or OwnerNobody.
	FirstOwner string
	// NewProfile answers ErrNeedNewMode: NewCopy or NewFresh.
	NewProfile string
	// CopyFrom, with NewCopy, starts the new profile from another profile's
	// files instead of the ones on the PC: "Cristian, hardcore" from
	// Cristian's saves while Mihai's are loaded.
	CopyFrom string
	Keep     int
}

// Activate puts playerID's files in place for the game. The files currently
// in place are first saved back to whoever they belong to.
func (e *Engine) Activate(cfg *Config, gameID, playerID string, opt ActivateOptions) error {
	g := cfg.Game(gameID)
	if g == nil {
		return fmt.Errorf("unknown game %q", gameID)
	}
	if cfg.Player(playerID) == nil {
		return fmt.Errorf("unknown player %q", playerID)
	}
	rs, err := e.resolveAll(g)
	if err != nil {
		return err
	}
	if _, err := e.finishPending(cfg, g, rs, opt.Keep); err != nil {
		return err
	}

	active := e.Store.LoadState()[gameID]
	if active != "" && cfg.Player(active) == nil {
		active = "" // that player was deleted
	}
	if active == "" {
		switch {
		case opt.FirstOwner == "":
			return ErrNeedOwner
		case opt.FirstOwner == OwnerNobody:
			// Files on disk belong to nobody we know; they get backed up below.
		case cfg.Player(opt.FirstOwner) != nil:
			if err := e.SaveLiveTo(g, opt.FirstOwner); err != nil {
				return err
			}
			if err := e.Store.SetActive(gameID, opt.FirstOwner); err != nil {
				return err
			}
			active = opt.FirstOwner
		default:
			return fmt.Errorf("unknown player %q", opt.FirstOwner)
		}
	} else {
		if err := e.SaveLiveTo(g, active); err != nil {
			return err
		}
	}

	if active == playerID {
		return nil
	}

	if e.HasProfile(playerID, gameID) {
		if _, err := e.Backup(g, opt.Keep); err != nil {
			return err
		}
		return e.swapIn(g, playerID, rs, false)
	} else {
		switch opt.NewProfile {
		case NewCopy:
			if opt.CopyFrom != "" && opt.CopyFrom != active && opt.CopyFrom != playerID && e.HasProfile(opt.CopyFrom, gameID) {
				if err := e.copyProfile(g, opt.CopyFrom, playerID, rs); err != nil {
					return err
				}
				if _, err := e.Backup(g, opt.Keep); err != nil {
					return err
				}
				return e.swapIn(g, playerID, rs, false)
			}
			if err := e.SaveLiveTo(g, playerID); err != nil {
				return err
			}
		case NewFresh:
			if _, err := e.Backup(g, opt.Keep); err != nil {
				return err
			}
			return e.swapIn(g, playerID, rs, true)
		default:
			return ErrNeedNewMode
		}
	}
	return e.Store.SetActive(gameID, playerID)
}

// FinishSession saves the live files back to the player who just played.
func (e *Engine) FinishSession(cfg *Config, gameID, playerID string) error {
	g := cfg.Game(gameID)
	if g == nil {
		return nil // game was removed meanwhile
	}
	if active := e.Store.LoadState()[gameID]; active != playerID {
		return fmt.Errorf("%s's files are no longer the ones loaded for %s; not saving", cfg.PlayerName(playerID), g.Name)
	}
	return e.SaveLiveTo(g, playerID)
}

// DeleteGameData removes all profiles, backups and state for a game.
func (e *Engine) DeleteGameData(cfg *Config, gameID string) {
	for _, p := range cfg.Players {
		_ = removeAll(e.Store.ProfileDir(p.ID, gameID))
	}
	_ = removeAll(filepath.Join(e.Store.BackupsDir(), gameID))
	_ = e.Store.SetActive(gameID, "")
}

// DeletePlayerData removes a player's profiles.
func (e *Engine) DeletePlayerData(playerID string) {
	if strings.TrimSpace(playerID) == "" {
		return
	}
	_ = removeAll(filepath.Join(e.Store.ProfilesDir(), playerID))
}

type BackupInfo struct {
	Dir   string `json:"dir"`
	When  string `json:"when"`
	Owner string `json:"owner"`
}

func (e *Engine) Backups(gameID string) []BackupInfo {
	root := filepath.Join(e.Store.BackupsDir(), gameID)
	ents, _ := os.ReadDir(root)
	var out []BackupInfo
	for i := len(ents) - 1; i >= 0; i-- {
		d := ents[i]
		if !d.IsDir() {
			continue
		}
		bi := BackupInfo{Dir: filepath.Join(root, d.Name())}
		if t, err := time.ParseInLocation("20060102-150405", d.Name()[:min(15, len(d.Name()))], time.Local); err == nil {
			bi.When = t.Format(time.RFC3339)
		}
		if b, err := os.ReadFile(filepath.Join(bi.Dir, "owner.txt")); err == nil {
			bi.Owner = strings.TrimSpace(string(b))
		}
		out = append(out, bi)
	}
	return out
}

// ProfileInfo is when a player's profile for a game was last written by the
// game (its newest file: copies keep the game's own times) and its size.
type ProfileInfo struct {
	Saved time.Time `json:"saved"`
	Size  int64     `json:"size"`
}

// ProfileInfo reads a player's profile for a game; ok is false without one.
func (e *Engine) ProfileInfo(playerID, gameID string) (ProfileInfo, bool) {
	if !e.HasProfile(playerID, gameID) {
		return ProfileInfo{}, false
	}
	dir := e.Store.ProfileDir(playerID, gameID)
	var info ProfileInfo
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable: count what can be read
		}
		name := d.Name()
		if d.IsDir() {
			if p != dir && (strings.HasSuffix(name, tmpSuffix) || strings.HasSuffix(name, oldSuffix)) {
				return filepath.SkipDir // a copy in progress or left over
			}
			return nil
		}
		if filepath.Dir(p) == dir && name == initMarker {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		info.Size += fi.Size()
		if fi.ModTime().After(info.Saved) {
			info.Saved = fi.ModTime()
		}
		return nil
	})
	return info, true
}
