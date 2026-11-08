package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// A new copy of the same game: a game in a folder is reinstalled as a whole
// new version (often in a new folder, as a full download), and the old
// folder is deleted or left beside it. Lootnest notices the new copy is the
// same game and carries everything over: the saves kept inside the game's
// folder, and what's recorded under its Library key (playtime, when it was
// played, its status, collections and settings).
//
// copies.json is a ledger of the folder games Lootnest has seen, so the old
// copy is still known once its folder is gone.
// ---------------------------------------------------------------------------

// SeenGame is a folder game in the ledger.
type SeenGame struct {
	Key     string    `json:"key"`
	Name    string    `json:"name"`
	Dir     string    `json:"dir"`
	SteamID int       `json:"steamId,omitempty"`
	First   time.Time `json:"first"`          // when Lootnest first saw this copy
	Seen    time.Time `json:"seen"`           // the last scan that found it
	Gone    bool      `json:"gone,omitempty"` // its folder isn't there any more
	// Kept are the in-folder saves copied when it was removed from this
	// PC (Remove from this PC), by place inside the folder: the backup.
	Kept map[string]string `json:"kept,omitempty"`
}

// CopyLedger is copies.json.
type CopyLedger struct {
	Games     map[string]SeenGame `json:"games"`
	Dismissed map[string]bool     `json:"dismissed,omitempty"` // "old|new": not the same game
	Carried   map[string]string   `json:"carried,omitempty"`   // old key -> the new key it went to
}

func (s *Store) CopiesPath() string { return filepath.Join(s.Dir, "copies.json") }

func (s *Store) LoadCopies() CopyLedger {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readCopies()
}

func (s *Store) readCopies() CopyLedger {
	l := CopyLedger{}
	if b, err := os.ReadFile(s.CopiesPath()); err == nil {
		_ = json.Unmarshal(b, &l)
	}
	if l.Games == nil {
		l.Games = map[string]SeenGame{}
	}
	if l.Dismissed == nil {
		l.Dismissed = map[string]bool{}
	}
	if l.Carried == nil {
		l.Carried = map[string]string{}
	}
	return l
}

// UpdateCopies changes copies.json under the store's lock.
func (s *Store) UpdateCopies(change func(*CopyLedger)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	l := s.readCopies()
	change(&l)
	return writeJSONAtomic(s.CopiesPath(), l)
}

// NoteSeen records the folder games a scan found, and marks the ones whose
// folder is gone.
func (l *CopyLedger) NoteSeen(found []FoundGame, now time.Time, dirThere func(string) bool) {
	here := map[string]bool{}
	for _, f := range found {
		if f.Dir == "" || f.AppID > 0 {
			continue
		}
		if o := l.Games[f.Key]; o.Gone && len(o.Kept) > 0 {
			if !dirThere(f.Dir) {
				continue // still listed, but its folder is gone
			}
			// Removed, then added again in the same folder: the removed copy
			// stays as the old one, under a key of its own for each removal,
			// so its kept saves are offered back, unless they already went
			// to another copy.
			delete(l.Games, f.Key)
			if l.Carried[f.Key] == "" {
				o.Key += removedSuffix + strconv.FormatInt(o.Seen.Unix(), 10)
				l.rekeyDismissed(f.Key, o.Key)
				l.Games[o.Key] = o
			}
		}
		here[f.Key] = true
		first := l.Games[f.Key].First
		if first.IsZero() {
			first = now.UTC()
		}
		l.Games[f.Key] = SeenGame{Key: f.Key, Name: f.Name, Dir: f.Dir, SteamID: f.SteamID, First: first, Seen: now.UTC()}
	}
	for k, g := range l.Games {
		if !here[k] && len(g.Kept) == 0 { // a removed copy stays gone, even with a new one in its folder
			g.Gone = !dirThere(g.Dir)
			l.Games[k] = g
		}
	}
	// Forget copies gone for half a year.
	for k, g := range l.Games {
		if g.Gone && now.Sub(g.Seen) > 183*24*time.Hour {
			delete(l.Games, k)
		}
	}
}

const removedSuffix = "#removed-"

// NoteRemoved records a game removed from this PC, with its kept saves.
func (l *CopyLedger) NoteRemoved(f FoundGame, kept map[string]string, now time.Time) {
	g := l.Games[f.Key]
	if g.Key == "" {
		g = SeenGame{Key: f.Key, Name: f.Name, Dir: f.Dir, SteamID: f.SteamID, First: now.UTC()}
	}
	g.Seen, g.Gone, g.Kept = now.UTC(), true, kept
	l.Games[f.Key] = g
	delete(l.Carried, f.Key) // what an earlier copy in its place carried or dismissed is history
	l.rekeyDismissed(f.Key, "")
}

// rekeyDismissed moves the "Not the same game" answers about old copy from
// to to, or forgets them when to is "".
func (l *CopyLedger) rekeyDismissed(from, to string) {
	for k := range l.Dismissed {
		if rest, ok := strings.CutPrefix(k, from+"|"); ok {
			delete(l.Dismissed, k)
			if to != "" {
				l.Dismissed[to+"|"+rest] = true
			}
		}
	}
}

// CopyMatch is a game that looks like a new copy of one Lootnest knew.
type CopyMatch struct {
	Old     SeenGame `json:"old"`
	New     SeenGame `json:"new"`
	OldGone bool     `json:"oldGone"` // the old folder is gone (else both are on the PC)
}

// sameGame: the same Steam id, or the same name (as covers match names).
func sameGame(a, b SeenGame) bool {
	if a.SteamID > 0 && b.SteamID > 0 {
		return a.SteamID == b.SteamID
	}
	return CoverKey(a.Name) != "" && CoverKey(a.Name) == CoverKey(b.Name)
}

// Matches lists new copies: a game the ledger saw first after another with
// the same name or Steam id, not dismissed and not carried over yet. The
// newest old copy is the one matched.
func (l CopyLedger) Matches() []CopyMatch {
	var out []CopyMatch
	for _, n := range l.Games {
		if n.Gone {
			continue
		}
		var best *SeenGame
		for _, o := range l.Games {
			o := o
			if o.Key == n.Key || l.Carried[o.Key] != "" || l.Dismissed[o.Key+"|"+n.Key] || !sameGame(o, n) {
				continue
			}
			if len(o.Kept) == 0 && (isWithin(o.Dir, n.Dir) || isWithin(n.Dir, o.Dir)) {
				continue // one inside the other: the same install found twice
			}
			if !o.First.Before(n.First) {
				continue // the copy Lootnest saw first is the old one
			}
			if best == nil || o.First.After(best.First) {
				best = &o
			}
		}
		if best != nil {
			out = append(out, CopyMatch{Old: *best, New: n, OldGone: best.Gone})
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].New.Name) < strings.ToLower(out[j].New.Name) })
	return out
}

// RekeyGame moves what Lootnest recorded for a Library game from one key to
// another: when it was played, playtime, daily activity, statuses, its
// captures from other apps and custom art. Anything already recorded under the new key is kept and added to.
func (s *Store) RekeyGame(from, to string) error {
	if from == "" || to == "" || from == to {
		return errors.New("nothing to move")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := map[string]Played{}
	if b, err := os.ReadFile(s.RecentPath()); err == nil {
		_ = json.Unmarshal(b, &rec)
	}
	if o, ok := rec[from]; ok {
		n := rec[to]
		if o.At.After(n.At) {
			n.At, n.Player = o.At, o.Player
		}
		players := map[string]time.Time{}
		for _, p := range []Played{o.withLast(), rec[to].withLast()} {
			for id, t := range p.Players {
				if t.After(players[id]) {
					players[id] = t
				}
			}
		}
		if len(players) > 0 {
			n.Players = players
		}
		rec[to] = n
		delete(rec, from)
		if err := writeJSONAtomic(s.RecentPath(), rec); err != nil {
			return err
		}
	}

	pt := map[string]map[string]int64{}
	if b, err := os.ReadFile(s.PlaytimePath()); err == nil {
		_ = json.Unmarshal(b, &pt)
	}
	if o, ok := pt[from]; ok {
		if pt[to] == nil {
			pt[to] = map[string]int64{}
		}
		for p, sec := range o {
			pt[to][p] += sec
		}
		delete(pt, from)
		if err := writeJSONAtomic(s.PlaytimePath(), pt); err != nil {
			return err
		}
	}

	a := s.readActivity()
	moved := false
	for _, days := range a.Days {
		for _, games := range days {
			if sec, ok := games[from]; ok {
				games[to] += sec
				delete(games, from)
				moved = true
			}
		}
	}
	for p, ls := range a.Longest {
		if ls.Key == from {
			ls.Key = to
			a.Longest[p] = ls
			moved = true
		}
	}
	if moved {
		if err := writeJSONAtomic(s.ActivityPath(), a); err != nil {
			return err
		}
	}

	shelf, err := s.readShelfToChange()
	if err != nil {
		return err
	}
	changed := false
	for p, games := range shelf {
		if e, ok := games[from]; ok {
			// The newer take wins, the older fills what it leaves empty, and
			// both corrections of the time played stay, as the time does.
			if n, has := games[to]; has {
				if !e.Updated.After(n.Updated) {
					e, n = n, e
				}
				e.Offset += n.Offset
				if e.Status == "" {
					e.Status = n.Status
				}
				if e.Rating == 0 {
					e.Rating = n.Rating
				}
				if e.Note == "" {
					e.Note = n.Note
				}
			}
			games[to] = e
			delete(games, from)
			shelf[p] = games
			changed = true
		}
	}
	if changed {
		if err := writeJSONAtomic(s.ShelfPath(), shelf); err != nil {
			return err
		}
	}

	caps, recapped := s.readCaptures(), false
	for i := range caps {
		if caps[i].Key == from {
			caps[i].Key, recapped = to, true
		}
	}
	if recapped {
		if err := writeJSONAtomic(s.CapturesPath(), captureIndex{Captures: caps}); err != nil {
			return err
		}
	}
	return s.rekeyCustomArt(from, to)
}

// RekeyConfig moves a Library game's settings, launch options and
// collections to its new key. It reports whether cfg changed.
func RekeyConfig(cfg *Config, from, to string) bool {
	changed := false
	if p, ok := cfg.LibraryPrefs[from]; ok {
		if _, has := cfg.LibraryPrefs[to]; !has {
			if cfg.LibraryPrefs == nil {
				cfg.LibraryPrefs = map[string]LibraryPref{}
			}
			cfg.LibraryPrefs[to] = p
		}
		delete(cfg.LibraryPrefs, from)
		changed = true
	}
	if o, ok := cfg.LaunchOpts[from]; ok {
		if _, has := cfg.LaunchOpts[to]; !has {
			cfg.LaunchOpts[to] = o
		}
		delete(cfg.LaunchOpts, from)
		changed = true
	}
	if tags, ok := cfg.Tags[from]; ok {
		if cfg.Tags == nil {
			cfg.Tags = map[string][]string{}
		}
		seen := map[string]bool{}
		var all []string
		for _, t := range append(append([]string{}, cfg.Tags[to]...), tags...) {
			if !seen[strings.ToLower(t)] {
				seen[strings.ToLower(t)] = true
				all = append(all, t)
			}
		}
		cfg.Tags[to] = all
		delete(cfg.Tags, from)
		changed = true
	}
	return changed
}

// CarriedSave is one save location moved into the new copy.
type CarriedSave struct {
	Rel      string `json:"rel"` // inside the game's folder, with "/"
	Size     int64  `json:"size"`
	Replaced bool   `json:"replaced"` // the new copy had saves there, kept in backups first
}

// CarrySaves puts the saves the old copy kept inside its folder (rels, from
// the old folder, or from from[rel] when the folder is gone) into the new
// copy's folder at the same place. Saves the new copy already has there are
// copied to keep first.
func CarrySaves(newDir string, from map[string]string, keep string) ([]CarriedSave, error) {
	var out []CarriedSave
	rels := make([]string, 0, len(from))
	for r := range from {
		rels = append(rels, r)
	}
	sort.Strings(rels)
	for _, rel := range rels {
		src := from[rel]
		r, err := updateRel(rel)
		if err != nil {
			return out, err
		}
		if !HasFiles(src) {
			continue
		}
		dst := filepath.Join(newDir, filepath.FromSlash(r))
		c := CarriedSave{Rel: r, Size: Size(src)}
		if HasFiles(dst) {
			if err := Mirror(dst, filepath.Join(keep, filepath.FromSlash(r))); err != nil {
				return out, fmt.Errorf("couldn't keep the new copy's own %s first: %w", r, err)
			}
			c.Replaced = true
		}
		if err := Mirror(src, dst); err != nil {
			return out, fmt.Errorf("%s: %w", r, err)
		}
		out = append(out, c)
	}
	return out, nil
}

// Reseat puts the files of the profile that owns a game back in place, as
// a new copy of the game was installed: whatever the new copy has there is
// backed up first. It reports whose files went in ("" for none).
func (e *Engine) Reseat(cfg *Config, gameID string, keep int) (string, error) {
	g := cfg.Game(gameID)
	if g == nil {
		return "", errors.New("unknown game")
	}
	owner := e.Store.LoadState()[gameID]
	if owner == "" || !e.HasProfile(owner, gameID) {
		return "", nil
	}
	rs, err := e.resolveAll(g)
	if err != nil {
		return "", err
	}
	if _, err := e.Backup(g, keep); err != nil {
		return "", err
	}
	return owner, e.swapIn(g, owner, rs, false)
}
