//go:build windows

package app

import (
	"time"

	"lootnest/internal/core"
)

// hubRecent is how many copies the Saves page lists under Recent copies.
const hubRecent = 30

// apiSavesOverview sends what Lootnest keeps of every game's saves as a
// "savesOverview" event: the copies before swaps, the safety net's copies
// and the save points, newest first, and their sums per game. It only reads.
func (a *App) apiSavesOverview() {
	go func() {
		a.mu.Lock()
		cfg := a.cfg.Clone()
		a.mu.Unlock()
		var copies []core.SaveCopy
		points := func(key string) {
			for _, p := range core.SavePoints(a.store.SavePointsDir(key)) {
				copies = append(copies, core.SaveCopy{Key: key, Kind: "point", ID: p.ID, At: p.Created, Size: p.Size, Name: p.Name, Owner: p.Player})
			}
		}
		for _, g := range cfg.Games {
			key := "game:" + g.ID
			for _, b := range a.eng.Backups(g.ID) {
				at, _ := time.Parse(time.RFC3339, b.When)
				copies = append(copies, core.SaveCopy{Key: key, Kind: "swap", ID: core.BackupID(b), At: at, Size: core.Size(b.Dir), Owner: b.Owner})
			}
			points(key)
		}
		net := netStore()
		for _, f := range a.visibleFound(a.libraryGames()) {
			if gameSavesOn(f) {
				for _, v := range net.GameVersions(gameSavesKey(f)) {
					copies = append(copies, core.SaveCopy{Key: f.Key, Kind: "net", ID: v.ID, At: v.Created, Size: v.Size})
				}
			}
			points(f.Key)
		}
		recent, games := core.SavesOverview(copies, hubRecent)
		a.emit("savesOverview", map[string]any{"recent": recent, "games": games})
	}()
}
