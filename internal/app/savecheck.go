//go:build windows

package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// The save check on Home: how Lootnest looks after the saves of every game
// in the Library, and the games whose saves it can't find yet. It only
// reads, and is worked out at most every ten minutes.
// ---------------------------------------------------------------------------

type checkGame struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type saveCheck struct {
	Profiles int         `json:"profiles"` // games with profiles: a copy before every swap
	Net      int         `json:"net"`      // games whose saves the safety net keeps copies of
	Points   int         `json:"points"`   // games with save points
	Cloud    bool        `json:"cloud"`    // cloud saves are set up
	Gone     []checkGame `json:"gone"`     // saves gone from their folder; copies kept
	Unknown  []checkGame `json:"unknown"`  // where they keep their saves isn't known
	UnknownN int         `json:"unknownN"` // how many in all
	// OneDrive are the games that keep saves in Documents or Saved Games
	// when OneDrive syncs that folder; OneDriveIn names the folders.
	OneDrive   []checkGame `json:"oneDrive"`
	OneDriveIn string      `json:"oneDriveIn,omitempty"`
	CheckedAt  time.Time   `json:"checkedAt"` // when this was worked out
}

var (
	checkMu       sync.Mutex
	lastSaveCheck *saveCheck
)

func (a *App) apiSaveCheck(fresh bool) {
	go func() {
		checkMu.Lock()
		defer checkMu.Unlock()
		if lastSaveCheck == nil || fresh || time.Since(lastSaveCheck.CheckedAt) > 10*time.Minute {
			c := a.saveCheck()
			lastSaveCheck = &c
		}
		a.emit("saveCheck", map[string]any{"check": lastSaveCheck})
	}()
}

func (a *App) saveCheck() saveCheck {
	c := saveCheck{Gone: []checkGame{}, Unknown: []checkGame{}, OneDrive: []checkGame{}, CheckedAt: time.Now()}
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	synced := core.OneDriveFolders([]string{a.env.Documents, a.env.SavedGames}, []string{os.Getenv("OneDrive"), os.Getenv("OneDriveConsumer"), os.Getenv("OneDriveCommercial")})
	var in []string
	for _, f := range synced {
		if f == a.env.Documents {
			in = append(in, "Documents")
		} else {
			in = append(in, "Saved Games")
		}
	}
	c.OneDriveIn = strings.Join(in, " and ")
	if len(synced) > 0 {
		for i := range cfg.Games {
			g := &cfg.Games[i]
			locs, _ := a.eng.SaveLocs(g)
			for _, l := range locs {
				if core.WithinAny(l.Live, synced) {
					c.OneDrive = append(c.OneDrive, checkGame{"game:" + g.ID, g.Name})
					break
				}
			}
		}
	}
	c.Profiles = len(cfg.Games)
	c.Cloud = cfg.Settings.BackupDir != "" && (cfg.Settings.CloudSync || cfg.Settings.GameSaves)
	net := loadNet()
	if es, err := os.ReadDir(filepath.Join(a.store.Dir, "savepoints")); err == nil {
		for _, e := range es {
			if e.IsDir() && len(core.SavePoints(filepath.Join(a.store.Dir, "savepoints", e.Name()))) > 0 {
				c.Points++
			}
		}
	}
	for _, f := range a.visibleFound(a.libraryGames()) {
		if !gameSavesOn(f) {
			continue
		}
		if len(synced) > 0 {
			for _, l := range a.savesOf(f).Locs {
				if core.WithinAny(l.Resolved, synced) {
					c.OneDrive = append(c.OneDrive, checkGame{f.Key, f.Name})
					break
				}
			}
		}
		e := net[gameSavesKey(f)]
		if e.Gone {
			c.Gone = append(c.Gone, checkGame{f.Key, f.Name})
		}
		if e.Hash != "" {
			c.Net++
			continue
		}
		if f.URI != "" || f.Source == "epic" || f.Source == "gog" {
			continue // their launcher may keep them; not asked for
		}
		if !e.Gone && f.Dir != "" && len(a.vaultLocs(f)) == 0 {
			c.UnknownN++
			c.Unknown = append(c.Unknown, checkGame{f.Key, f.Name})
		}
	}
	for _, l := range [][]checkGame{c.Unknown, c.OneDrive} {
		sort.Slice(l, func(i, j int) bool { return strings.ToLower(l[i].Name) < strings.ToLower(l[j].Name) })
	}
	return c
}

// Stage says Lootnest is in beta, next to its version.
const Stage = "beta"

const issuesURL = "https://github.com/c-dinca/lootnest/issues/new"

// apiReportProblem copies Lootnest's version and its recent activity, for
// the user to paste, and opens a new issue on GitHub in the browser.
// Nothing is sent by Lootnest itself.
func (a *App) apiReportProblem() error {
	text := "Lootnest " + Version + " " + Stage + "\n\nRecent activity:\n" + a.apiLog()
	if err := win.SetClipboardText(a.hwnd, text); err != nil {
		return err
	}
	return win.ShellOpen(issuesURL)
}
