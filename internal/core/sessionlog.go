package core

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// The session log: one line of JSON per finished play session, in
// sessions.jsonl next to activity.json. It's only ever appended to, so
// history kept there can't be lost to a rewrite; a line that doesn't parse
// is skipped when reading, never removed.
// ---------------------------------------------------------------------------

// SessionRecord is one finished play session.
type SessionRecord struct {
	Game    string    `json:"game"`              // Library key
	Profile string    `json:"profile,omitempty"` // player id, "" for a game without profiles
	Start   time.Time `json:"start"`
	End     time.Time `json:"end"`
	Seconds int64     `json:"seconds"`
	Source  string    `json:"source"` // "profile" or "library"
	Build   string    `json:"build,omitempty"`
}

func (s *Store) SessionLogPath() string { return filepath.Join(s.Dir, "sessions.jsonl") }

// AppendSession adds a session to the log. Sessions under a minute (an
// accidental start) and over a day (a clock jump) aren't recorded, as for
// playtime.
func (s *Store) AppendSession(r SessionRecord) error {
	d := r.End.Sub(r.Start)
	if r.Game == "" || d < time.Minute || d > 24*time.Hour {
		return nil
	}
	r.Start, r.End = r.Start.UTC().Truncate(time.Second), r.End.UTC().Truncate(time.Second)
	r.Seconds = int64(r.End.Sub(r.Start) / time.Second)
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.OpenFile(s.SessionLogPath(), os.O_RDWR|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	// A last line cut short (a crash mid-write) is ended first, so this
	// record doesn't join it.
	if st, err := f.Stat(); err == nil && st.Size() > 0 {
		last := make([]byte, 1)
		if _, err := f.ReadAt(last, st.Size()-1); err == nil && last[0] != '\n' {
			b = append([]byte{'\n'}, b...)
		}
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

// ReadSessions returns the sessions of game ("" for every game) that ended
// at or after since, newest first, and how many lines didn't parse.
func (s *Store) ReadSessions(game string, since time.Time) ([]SessionRecord, int, error) {
	f, err := os.Open(s.SessionLogPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	var out []SessionRecord
	bad := 0
	br := bufio.NewReaderSize(f, 16<<10)
	for {
		line, long, err := br.ReadLine()
		if long {
			// Far longer than a record: skipped to its end, a piece at a time.
			for long && err == nil {
				_, long, err = br.ReadLine()
			}
			bad++
		} else if len(line) > 0 {
			var r SessionRecord
			if json.Unmarshal(line, &r) != nil || r.Game == "" {
				bad++
			} else if (game == "" || r.Game == game) && !r.End.Before(since) {
				out = append(out, r)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, bad, err
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].End.After(out[j].End) })
	return out, bad, nil
}

// libraryTimeKept is the last LibraryTime read, with the log's size and
// time then, so the log is read again only once it changes.
var libraryTimeKept struct {
	sync.Mutex
	path string
	size int64
	mod  time.Time
	out  map[string]int64
}

// LibraryTime is how long each game without profiles was played, by
// Library key: the sum of the sessions the log keeps for it (Profile "").
// A game with profiles counts its time in playtime.json instead.
func (s *Store) LibraryTime() map[string]int64 {
	p := s.SessionLogPath()
	st, err := os.Stat(p)
	if err != nil {
		return map[string]int64{}
	}
	k := &libraryTimeKept
	k.Lock()
	defer k.Unlock()
	if k.out != nil && k.path == p && k.size == st.Size() && k.mod.Equal(st.ModTime()) {
		return maps.Clone(k.out)
	}
	list, _, err := s.ReadSessions("", time.Time{})
	if err != nil {
		return map[string]int64{}
	}
	out := map[string]int64{}
	for _, r := range list {
		if r.Profile == "" && r.Seconds > 0 {
			out[r.Game] += r.Seconds
		}
	}
	k.path, k.size, k.mod, k.out = p, st.Size(), st.ModTime(), out
	return maps.Clone(out)
}
