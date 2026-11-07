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
	Updated time.Time `json:"updated"`
}

func (e ShelfEntry) empty() bool { return e.Status == "" && e.Rating == 0 && e.Note == "" }

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

// SetShelf saves a player's take on a game; an empty entry removes it.
func (s *Store) SetShelf(playerID, key string, e ShelfEntry, now time.Time) error {
	if playerID == "" || key == "" || len(key) > 400 || strings.ContainsAny(key, "\r\n\x00") {
		return errors.New("invalid game")
	}
	if err := e.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sh := s.readShelf()
	if sh[playerID] == nil {
		sh[playerID] = map[string]ShelfEntry{}
	}
	if e.empty() {
		delete(sh[playerID], key)
	} else {
		e.Updated = now.UTC()
		sh[playerID][key] = e
	}
	return writeJSONAtomic(s.ShelfPath(), sh)
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
