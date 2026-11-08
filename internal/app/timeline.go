//go:build windows

package app

import (
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
)

// ---------------------------------------------------------------------------
// The session log (core/sessionlog.go) and a game's timeline. Profile
// sessions are logged where their playtime is counted (tick); Library games
// Lootnest starts are followed by their process here.
// ---------------------------------------------------------------------------

// foundRun is a Library game Lootnest started, followed until it closes.
type foundRun struct {
	proc, build string
	since       time.Time // when its process was first seen
	until       time.Time // given up on if it hasn't shown by then
	missed      int
}

var (
	foundRunsMu sync.Mutex
	foundRuns   = map[string]*foundRun{} // Library key -> its run
)

func (a *App) followFound(key string, f core.FoundGame) {
	if f.ProcessName == "" {
		return
	}
	r := &foundRun{proc: strings.ToLower(strings.TrimSuffix(f.ProcessName, ".exe")), build: a.gameBuild(f.AppID, f.Exe), until: time.Now().Add(3 * time.Minute)}
	foundRunsMu.Lock()
	defer foundRunsMu.Unlock()
	if old := foundRuns[key]; old == nil {
		foundRuns[key] = r
	} else if old.since.IsZero() { // not seen yet: wait from this Play
		old.until = r.until
	} // Play again while it runs is the same session
}

// foundTick logs the session of each followed game that closed.
func (a *App) foundTick(procs map[string]bool) {
	now := time.Now()
	foundRunsMu.Lock()
	defer foundRunsMu.Unlock()
	for key, r := range foundRuns {
		switch {
		case procs[r.proc]:
			if r.since.IsZero() {
				r.since = now
			}
			r.missed = 0
		case r.since.IsZero():
			if now.After(r.until) {
				delete(foundRuns, key)
				go a.stepsAfter(key, false)
			}
		default:
			if r.missed++; r.missed < 2 { // ignore quick restarts (launcher -> game)
				continue
			}
			delete(foundRuns, key)
			go func(key string, rec core.SessionRecord) {
				a.stepsAfter(key, true)
				a.logSession(rec)
				a.apiGameTimeline(key)
			}(key, core.SessionRecord{Game: key, Start: r.since, End: now, Source: "library", Build: r.build})
		}
	}
}

func (a *App) logSession(r core.SessionRecord) {
	if err := a.store.AppendSession(r); err != nil {
		a.logf("Couldn't add a session of %s to the session log: %v", r.Game, err)
	}
	a.scanCapturesSoon() // the clips and screenshots of this session
}

// TimelineItem is one thing that happened in a game.
type TimelineItem struct {
	Kind    string    `json:"kind"` // "session", "point", "unlock", "update" or "shot"
	At      time.Time `json:"at"`
	Start   time.Time `json:"start,omitzero"` // a session's
	Profile string    `json:"profile,omitempty"`
	Seconds int64     `json:"seconds,omitempty"`
	Name    string    `json:"name,omitempty"`    // the point's or achievement's, or the update's or screenshot's file
	Version string    `json:"version,omitempty"` // the version an update brought, when known
	Thumb   string    `json:"thumb,omitempty"`   // a screenshot's thumbnail, as a data: URL
}

const (
	timelineMax = 200
	thumbsMax   = 40 // thumbnails sent with one timeline
)

// apiGameTimeline answers with a "timeline" event: a game's sessions, save
// points, unlocks, updates and screenshots, newest first.
func (a *App) apiGameTimeline(key string) {
	go func() {
		var items []TimelineItem
		sessions, bad, err := a.store.ReadSessions(key, time.Time{})
		if err != nil {
			a.logf("Couldn't read the session log: %v", err)
		}
		if bad > 0 {
			a.logAchOnce("sessionlog|bad", "Session log: %d line(s) couldn't be read and were skipped", bad)
		}
		for _, s := range sessions {
			items = append(items, TimelineItem{Kind: "session", At: s.End, Start: s.Start, Profile: s.Profile, Seconds: s.Seconds})
		}
		for _, p := range core.SavePoints(a.store.SavePointsDir(key)) {
			items = append(items, TimelineItem{Kind: "point", At: p.Created, Profile: p.Player, Name: p.Name})
		}
		updKey, appID, dir, name := "found:"+key, 0, "", ""
		if id, ok := strings.CutPrefix(key, "game:"); ok {
			exe := ""
			a.mu.Lock()
			if g := a.cfg.Game(id); g != nil {
				name, exe = g.Name, g.Launch.Exe
			}
			a.mu.Unlock()
			if exe != "" {
				if p, err := a.env.ResolveExe(exe); err == nil {
					dir = filepath.Dir(p)
				}
			}
			updKey = key
		} else if f, ok := a.foundByKey(key); ok {
			appID, dir, name = f.AppID, f.Dir, f.Name
		}
		if name != "" {
			if set := a.achievements(key, appID, dir, name, false); set != nil {
				seen := map[string]bool{} // a run copied from a profile has its unlocks too
				unlocks := func(list []core.Achievement, profile string) {
					for _, x := range list {
						if k := x.ID + "|" + x.At.String(); x.Unlocked && !x.At.IsZero() && !seen[k] {
							seen[k] = true
							n := x.Name
							if n == "" {
								n = x.ID
							}
							items = append(items, TimelineItem{Kind: "unlock", At: x.At, Profile: profile, Name: n})
						}
					}
				}
				if set.Players != nil {
					a.mu.Lock()
					var order []string // in profile order: a run added later keeps its source's unlocks with the source
					for _, p := range a.cfg.Players {
						order = append(order, p.ID)
					}
					a.mu.Unlock()
					for _, pid := range order {
						if ps := set.Players[pid]; ps != nil {
							unlocks(ps.List, pid)
						}
					}
				} else {
					unlocks(set.List, "")
				}
			}
		}
		if t, err := a.updTarget(updKey); err == nil {
			for _, u := range core.GameUpdates(a.updatesDir(t)) {
				if u.State == "applied" {
					items = append(items, TimelineItem{Kind: "update", At: u.At, Name: u.Source, Version: u.To})
				}
			}
		}
		shotsDir := ""
		if name != "" {
			shotsDir = core.ScreenshotsDir(picturesDir(), name)
			for _, s := range core.Screenshots(shotsDir) {
				items = append(items, TimelineItem{Kind: "shot", At: s.At, Name: s.Name})
			}
		}
		sort.SliceStable(items, func(i, j int) bool { return items[i].At.After(items[j].At) })
		if len(items) > timelineMax {
			items = items[:timelineMax]
		}
		for i, n := 0, 0; i < len(items) && n < thumbsMax; i++ {
			if items[i].Kind == "shot" {
				items[i].Thumb = a.shotThumb(key, shotsDir, items[i].Name)
				n++
			}
		}
		if items == nil {
			items = []TimelineItem{}
		}
		a.emit("timeline", map[string]any{"key": key, "items": items})
	}()
}
