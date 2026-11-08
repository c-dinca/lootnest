package core

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// What each player thinks of their games and when they play: the shelf
// (playing, up next, finished, stopped; a rating and a note) in shelf.json,
// and time played per day in activity.json. Both stay on this PC.
// ---------------------------------------------------------------------------

// ShelfStatuses are the places a game can have on a player's shelf.
var ShelfStatuses = []string{"playing", "next", "finished", "complete", "stopped"}

// ShelfEntry is one player's take on one game.
type ShelfEntry struct {
	Status  string    `json:"status,omitempty"`
	Rating  int       `json:"rating,omitempty"` // 1-5 stars, 0 for none
	Note    string    `json:"note,omitempty"`
	Offset  int64     `json:"offset,omitempty"` // seconds the player added to (or took from) their time played
	Updated time.Time `json:"updated"`
}

func (e ShelfEntry) empty() bool {
	return e.Status == "" && e.Rating == 0 && e.Note == "" && e.Offset == 0
}

// ShelfPatch changes some fields of an entry; a nil field keeps its value.
type ShelfPatch struct {
	Status *string `json:"status"`
	Rating *int    `json:"rating"`
	Note   *string `json:"note"`
	Offset *int64  `json:"offset"`
}

func (p ShelfPatch) apply(e *ShelfEntry) {
	if p.Status != nil {
		e.Status = *p.Status
	}
	if p.Rating != nil {
		e.Rating = *p.Rating
	}
	if p.Note != nil {
		e.Note = *p.Note
	}
	if p.Offset != nil {
		e.Offset = *p.Offset
	}
}

// maxOffset bounds a correction of the time played: 100,000 hours.
const maxOffset = 100000 * 3600

// Validate cleans an entry and checks its limits.
func (e *ShelfEntry) Validate() error {
	e.Note = strings.TrimSpace(strings.ReplaceAll(e.Note, "\r\n", "\n"))
	if len([]rune(e.Note)) > 500 {
		return errors.New("Keep the note under 500 characters.")
	}
	if e.Status != "" {
		ok := false
		for _, s := range ShelfStatuses {
			ok = ok || s == e.Status
		}
		if !ok {
			return errors.New("unknown shelf status")
		}
	}
	if e.Rating < 0 || e.Rating > 5 {
		return errors.New("A rating is 1 to 5 stars.")
	}
	if e.Offset < -maxOffset || e.Offset > maxOffset {
		return errors.New("A correction is at most 100,000 hours.")
	}
	return nil
}

func (s *Store) ShelfPath() string { return filepath.Join(s.Dir, "shelf.json") }

// LoadShelf returns every player's shelf: player id -> Library key -> entry.
func (s *Store) LoadShelf() map[string]map[string]ShelfEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readShelf()
}

func (s *Store) readShelf() map[string]map[string]ShelfEntry {
	out := map[string]map[string]ShelfEntry{}
	if b, err := os.ReadFile(s.ShelfPath()); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	return out
}

// readShelfToChange reads shelf.json before a write. A file that doesn't
// parse is kept as shelf.json.broken, so the write never replaces it.
func (s *Store) readShelfToChange() (map[string]map[string]ShelfEntry, error) {
	out := map[string]map[string]ShelfEntry{}
	b, err := os.ReadFile(s.ShelfPath())
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	if json.Unmarshal(b, &out) != nil {
		if err := os.Rename(s.ShelfPath(), s.ShelfPath()+".broken"); err != nil {
			return nil, err
		}
		out = nil
	}
	if out == nil {
		out = map[string]map[string]ShelfEntry{}
	}
	return out, nil
}

// SetShelf saves a player's take on a game; an empty entry removes it.
func (s *Store) SetShelf(playerID, key string, e ShelfEntry, now time.Time) error {
	return s.updateShelf(playerID, key, func(old *ShelfEntry) { *old = e }, now)
}

// PatchShelf changes the fields p sets in a player's take on a game and
// keeps the others.
func (s *Store) PatchShelf(playerID, key string, p ShelfPatch, now time.Time) error {
	return s.updateShelf(playerID, key, p.apply, now)
}

func (s *Store) updateShelf(playerID, key string, set func(*ShelfEntry), now time.Time) error {
	if playerID == "" || key == "" || len(key) > 400 || strings.ContainsAny(key, "\r\n\x00") {
		return errors.New("invalid game")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sh, err := s.readShelfToChange()
	if err != nil {
		return err
	}
	if sh[playerID] == nil {
		sh[playerID] = map[string]ShelfEntry{}
	}
	e := sh[playerID][key]
	set(&e)
	if err := e.Validate(); err != nil {
		return err
	}
	if e.empty() {
		delete(sh[playerID], key)
	} else {
		e.Updated = now.UTC()
		sh[playerID][key] = e
	}
	return writeJSONAtomic(s.ShelfPath(), sh)
}

// WithOffsets adds each player's own correction of their time played
// (ShelfEntry.Offset) to pt, Library key -> player id -> seconds, never
// going below 0. pt is changed in place and returned.
func WithOffsets(pt map[string]map[string]int64, shelf map[string]map[string]ShelfEntry) map[string]map[string]int64 {
	for pid, games := range shelf {
		for key, e := range games {
			if e.Offset == 0 {
				continue
			}
			if pt[key] == nil {
				pt[key] = map[string]int64{}
			}
			pt[key][pid] = max(0, pt[key][pid]+e.Offset)
		}
	}
	return pt
}

// Activity is when each player played: Days is player id -> "2006-01-02"
// (local) -> Library key -> seconds; Longest is each player's longest
// session; Sessions counts them.
type Activity struct {
	Days     map[string]map[string]map[string]int64 `json:"days"`
	Longest  map[string]LongSession                 `json:"longest"`
	Sessions map[string]int                         `json:"sessions"`
}

type LongSession struct {
	Key     string    `json:"key"`
	Seconds int64     `json:"seconds"`
	At      time.Time `json:"at"`
}

// activityDays is how long daily activity is kept.
const activityDays = 400

func (s *Store) ActivityPath() string { return filepath.Join(s.Dir, "activity.json") }

func (s *Store) LoadActivity() Activity {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readActivity()
}

func (s *Store) readActivity() Activity {
	var a Activity
	if b, err := os.ReadFile(s.ActivityPath()); err == nil {
		_ = json.Unmarshal(b, &a)
	}
	if a.Days == nil {
		a.Days = map[string]map[string]map[string]int64{}
	}
	if a.Longest == nil {
		a.Longest = map[string]LongSession{}
	}
	if a.Sessions == nil {
		a.Sessions = map[string]int{}
	}
	return a
}

// AddActivity records a finished session from start to end, split over the
// local days it covers. The same limits as AddPlaytime apply.
func (s *Store) AddActivity(key, playerID string, start, end time.Time) error {
	d := end.Sub(start)
	if playerID == "" || key == "" || d < time.Minute || d > 24*time.Hour {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	a := s.readActivity()
	days := a.Days[playerID]
	if days == nil {
		days = map[string]map[string]int64{}
		a.Days[playerID] = days
	}
	for t := start.Local(); t.Before(end); {
		y, m, dd := t.Date()
		next := time.Date(y, m, dd+1, 0, 0, 0, 0, t.Location())
		if next.After(end) {
			next = end
		}
		day := t.Format("2006-01-02")
		if days[day] == nil {
			days[day] = map[string]int64{}
		}
		days[day][key] += int64(next.Sub(t) / time.Second)
		t = next
	}
	a.Sessions[playerID]++
	if sec := int64(d / time.Second); sec > a.Longest[playerID].Seconds {
		a.Longest[playerID] = LongSession{Key: key, Seconds: sec, At: start.UTC()}
	}
	// Keep the last activityDays days.
	cut := end.Local().AddDate(0, 0, -activityDays).Format("2006-01-02")
	for _, pd := range a.Days {
		for day := range pd {
			if day < cut {
				delete(pd, day)
			}
		}
	}
	return writeJSONAtomic(s.ActivityPath(), a)
}
