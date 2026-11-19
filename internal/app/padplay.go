package app

import (
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
)

// Which games play with a controller, for Big Picture to leave out the
// ones that don't (Counter-Strike 2, Riot's games). Per Library key:
// "full", "partial", "none", or no entry when nothing says. In order: the
// user's own word (Edit details, or Big Picture's right-click on the PC),
// Steam's appinfo.vdf on this PC, the game's store details already kept
// (only with Game details on), then the launcher and name (core.PadGuess).
// Nothing goes online for it.

var padKept = struct {
	sync.Mutex
	at  time.Time
	m   map[string]string
	gen int
}{}

// padStale makes the next ask read everything again (after a scan, or the
// user's word on a game).
func padStale() {
	padKept.Lock()
	padKept.at = time.Time{}
	padKept.gen++
	padKept.Unlock()
}

// apiPadSupport answers with a "padSupport" event; read again at most every
// half hour unless something changed.
func (a *App) apiPadSupport() {
	go func() {
		padKept.Lock()
		m, fresh, gen := padKept.m, time.Since(padKept.at) < 30*time.Minute, padKept.gen
		padKept.Unlock()
		if !fresh || m == nil {
			m = a.padSupport()
			padKept.Lock()
			if padKept.gen == gen {
				padKept.m, padKept.at = m, time.Now()
			}
			padKept.Unlock()
		}
		a.emit("padSupport", map[string]any{"pads": m})
	}()
}

func (a *App) padSupport() map[string]string {
	type item struct {
		key, name, source string
		id                int
	}
	var items []item
	for _, f := range a.libraryGames() {
		id := max(f.AppID, f.SteamID)
		if id <= 0 {
			id = core.FolderAppID(f.Dir)
		}
		if id <= 0 {
			id = knownStoreAppID(f.Name)
		}
		items = append(items, item{f.Key, f.Name, f.Source, id})
	}
	a.mu.Lock()
	games := append([]core.Game{}, a.cfg.Games...)
	a.mu.Unlock()
	for i := range games {
		id := core.GameAppID(&games[i])
		if id <= 0 {
			id = knownStoreAppID(games[i].Name)
		}
		items = append(items, item{"game:" + games[i].ID, games[i].Name, "", id})
	}
	want := map[int]bool{}
	for _, it := range items {
		if it.id > 0 {
			want[it.id] = true
		}
	}
	steam := core.SteamControllers(a.env.SteamPath, want)
	dir, out := DataDir(), map[string]string{}
	none := 0
	for _, it := range items {
		v := ""
		if e := core.LoadDetailsEdit(dir, it.key); e.Controller != nil {
			v = *e.Controller
		}
		if v == "" && it.id > 0 {
			v = steam[it.id]
		}
		if v == "" && it.id > 0 {
			d, _ := core.LoadDetails(dir, it.id)
			v = core.DetailsPad(d)
		}
		if v == "" {
			v = core.PadGuess(it.source, it.name)
		}
		if v != "" {
			out[it.key] = v
		}
		if v == "none" {
			none++
		}
	}
	a.logf("Controllers: known for %d of %d games, %d without controller support", len(out), len(items), none)
	return out
}

// apiSetPadPlay keeps the user's word on whether a game plays with a
// controller ("full" or "none"; "" gives the say back to Steam), as the
// Controller of its Edit details.
func (a *App) apiSetPadPlay(key, pad string) error {
	e := core.LoadDetailsEdit(DataDir(), key)
	switch strings.TrimSpace(pad) {
	case "full", "partial", "none":
		e.Controller = &pad
	default:
		e.Controller = nil
	}
	if err := a.apiSaveDetailsEdit(key, e); err != nil {
		return err
	}
	padStale()
	a.apiPadSupport()
	return nil
}
