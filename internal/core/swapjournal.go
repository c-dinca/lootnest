package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ---------------------------------------------------------------------------
// The swap journal. Loading a profile's files takes several steps: the files
// in place are saved to whoever they belong to, copied to backups, then the
// profile's files replace them one location at a time, and only then does
// state.json say whose they are. If Lootnest closes or the PC turns off in
// the middle, state.json still names the previous owner while the files in
// place may already be someone else's, partly or fully. The next swap would
// then save that mix over the previous owner's profile.
//
// So before the files in place change, swaps.json records whose files are
// going in. Both profiles are intact on disk at that point (the previous
// owner's was just saved, the new one is only read), so an interrupted swap
// is simply done again before anything else touches the game's files.
// ---------------------------------------------------------------------------

// PendingSwap is a swap that started and hasn't finished.
type PendingSwap struct {
	To string `json:"to"` // the profile whose files are going in
	// Fresh: the profile starts with no files, so the files in place are
	// cleared instead of replaced.
	Fresh bool      `json:"fresh,omitempty"`
	At    time.Time `json:"at"`
}

func (s *Store) SwapsPath() string { return filepath.Join(s.Dir, "swaps.json") }

// PendingSwaps lists the swaps that didn't finish, by game id.
func (s *Store) PendingSwaps() map[string]PendingSwap {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readSwaps()
}

func (s *Store) readSwaps() map[string]PendingSwap {
	out := map[string]PendingSwap{}
	if b, err := os.ReadFile(s.SwapsPath()); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	return out
}

func (s *Store) setSwap(gameID string, p *PendingSwap) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.readSwaps()
	if p == nil {
		if _, ok := st[gameID]; !ok {
			return nil
		}
		delete(st, gameID)
	} else {
		st[gameID] = *p
	}
	if len(st) == 0 {
		err := os.Remove(s.SwapsPath())
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return writeJSONAtomic(s.SwapsPath(), st)
}

// swapIn puts a profile's files in place and records it as the owner, with
// the journal around it. The files in place must already be saved and
// backed up.
func (e *Engine) swapIn(g *Game, playerID string, rs []resolved, fresh bool) error {
	if err := e.Store.setSwap(g.ID, &PendingSwap{To: playerID, Fresh: fresh, At: e.now()}); err != nil {
		return err
	}
	if fresh {
		for _, r := range rs {
			if err := removeAll(r.live); err != nil {
				return fmt.Errorf("couldn't clear %s: %w", r.live, err)
			}
		}
		// Mark the profile as existing (empty) so the next swap restores
		// "nothing" for this player instead of asking again.
		if err := e.saveLive(g, playerID, rs); err != nil {
			return err
		}
	} else if err := e.restore(g, playerID, rs); err != nil {
		return err
	}
	if err := e.Store.SetActive(g.ID, playerID); err != nil {
		return err
	}
	return e.Store.setSwap(g.ID, nil)
}

// finishPending completes a game's interrupted swap, if it has one, so the
// files in place are again exactly one profile's and state.json says whose.
// It reports whether there was one.
func (e *Engine) finishPending(cfg *Config, g *Game, rs []resolved, keep int) (bool, error) {
	p, ok := e.Store.PendingSwaps()[g.ID]
	if !ok {
		return false, nil
	}
	if cfg.Player(p.To) == nil || (!p.Fresh && !e.HasProfile(p.To, g.ID)) {
		// The profile is gone, so the swap can't be done again. Keep the
		// files in place as a backup and ask whose they are next time.
		if _, err := e.Backup(g, keep); err != nil {
			return true, err
		}
		if err := e.Store.SetActive(g.ID, ""); err != nil {
			return true, err
		}
		e.logf("A swap of %s didn't finish and its profile no longer exists; the files on the PC are kept as a backup", g.Name)
		return true, e.Store.setSwap(g.ID, nil)
	}
	if err := e.swapIn(g, p.To, rs, p.Fresh); err != nil {
		return true, fmt.Errorf("finishing the last swap of %s: %w", g.Name, err)
	}
	e.logf("The last swap of %s didn't finish (Lootnest closed or the PC turned off); it's finished now, with %s's files in place", g.Name, e.name(p.To))
	return true, nil
}

// RecoverSwaps finishes every swap that was interrupted. It runs when
// Lootnest starts, before any game is played, and returns the games it
// finished a swap for.
func (e *Engine) RecoverSwaps(cfg *Config, keep int) ([]string, error) {
	var done []string
	var first error
	for id := range e.Store.PendingSwaps() {
		g := cfg.Game(id)
		if g == nil {
			_ = e.Store.setSwap(id, nil) // the game was removed
			continue
		}
		rs, err := e.resolveAll(g)
		if err == nil {
			var ok bool
			ok, err = e.finishPending(cfg, g, rs, keep)
			if ok && err == nil {
				done = append(done, g.ID)
			}
		}
		if err != nil && first == nil {
			first = err
		}
	}
	return done, first
}
