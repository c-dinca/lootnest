//go:build windows

package app

import (
	"errors"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Find games (core/findgames.go): search any game, see its prices in the
// official stores and its requirements beside this PC's hardware, and open
// a store's page in the browser. Only with Settings.FindGames on.
// ---------------------------------------------------------------------------

var (
	findMu     sync.Mutex
	findOffers = map[int]findCached{}
	findStores map[string]string
	findStoreT time.Time
	pcSpecs    *core.PCSpecs
)

type findCached struct {
	offer *core.GameOffer
	at    time.Time
}

const findKeep = time.Hour

var errFindOff = errors.New("Turn on Find games in Settings, Online, first.")

func (a *App) findOn() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg.Settings.FindGames
}

// storeCountry is the country Windows is set to, for prices in its currency.
func storeCountry() string { return core.StoreCountry(win.UserCountry()) }

// ownedIndex maps the games on this PC by Steam app id and by name, to
// "game:<id>" (with profiles) or "found:<key>" (in the Library).
func (a *App) ownedIndex() (byID map[int]string, byName map[string]string) {
	byID, byName = map[int]string{}, map[string]string{}
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	for _, g := range cfg.Games {
		if g.Launch.Type == "steam" && g.Launch.AppID > 0 {
			byID[g.Launch.AppID] = "game:" + g.ID
		}
		byName[core.CoverKey(g.Name)] = "game:" + g.ID
	}
	for _, f := range a.visibleFound(a.libraryGames()) {
		if id := max(f.AppID, f.SteamID); id > 0 {
			if _, ok := byID[id]; !ok {
				byID[id] = "found:" + f.Key
			}
		}
		if k := core.CoverKey(f.Name); k != "" {
			if _, ok := byName[k]; !ok {
				byName[k] = "found:" + f.Key
			}
		}
	}
	return byID, byName
}

func ownedOf(byID map[int]string, byName map[string]string, appID int, name string) string {
	if o := byID[appID]; o != "" {
		return o
	}
	return byName[core.CoverKey(name)]
}

// apiFindGames searches the Steam store and answers with a "findGames"
// event.
func (a *App) apiFindGames(q string) error {
	if !a.findOn() {
		return errFindOff
	}
	go func() {
		out := map[string]any{"q": q}
		defer func() { a.emit("findGames", out) }()
		hits, err := core.SearchStore(fetchURL, q, storeCountry())
		if err != nil {
			a.logf("Find games: %v", err)
			out["error"] = "Couldn't reach the Steam store. Check the connection and try again."
			return
		}
		byID, byName := a.ownedIndex()
		for i := range hits {
			hits[i].Owned = ownedOf(byID, byName, hits[i].AppID, hits[i].Name)
		}
		out["hits"] = hits
	}()
	return nil
}

// apiGameOffer gathers one game's prices and requirements, and this PC's
// hardware, as a "gameOffer" event.
func (a *App) apiGameOffer(appID int, name string) error {
	if !a.findOn() {
		return errFindOff
	}
	go func() {
		out := map[string]any{"appId": appID}
		defer func() { a.emit("gameOffer", out) }()
		out["pc"] = a.pcSpecs()
		byID, byName := a.ownedIndex()
		out["owned"] = ownedOf(byID, byName, appID, name)
		findMu.Lock()
		c, ok := findOffers[appID]
		findMu.Unlock()
		if ok && time.Since(c.at) < findKeep {
			out["offer"] = c.offer
			return
		}
		o, err := core.FindOffer(fetchURL, appID, storeCountry(), sharkStores())
		if err != nil {
			a.logf("Find games: %v", err)
			out["error"] = "Couldn't get this game's prices. Check the connection and try again."
			return
		}
		findMu.Lock()
		findOffers[appID] = findCached{o, time.Now()}
		findMu.Unlock()
		out["offer"] = o
	}()
	return nil
}

// sharkStores is CheapShark's list of stores, kept for a day.
func sharkStores() map[string]string {
	findMu.Lock()
	if findStores != nil && time.Since(findStoreT) < 24*time.Hour {
		s := findStores
		findMu.Unlock()
		return s
	}
	findMu.Unlock()
	s, err := core.SharkStores(fetchURL)
	if err != nil {
		return nil
	}
	findMu.Lock()
	findStores, findStoreT = s, time.Now()
	findMu.Unlock()
	return s
}

// pcSpecs reads this PC's hardware once (the free space each time).
func (a *App) pcSpecs() core.PCSpecs {
	findMu.Lock()
	p := pcSpecs
	findMu.Unlock()
	if p == nil {
		gpu, vram := win.GPU()
		p = &core.PCSpecs{CPU: win.CPUName(), GPU: gpu, VRAM: vram, RAM: win.TotalRAM(), Windows: win.WindowsName()}
		findMu.Lock()
		pcSpecs = p
		findMu.Unlock()
	}
	out := *p
	out.FreeOn, out.Free = win.MostFree()
	return out
}

// openStore opens a store's page for a game in the browser.
func openStore(u string) error {
	if !core.SafeStoreLink(u) {
		return errors.New("That store link can't be opened.")
	}
	return win.ShellOpen(strings.TrimSpace(u))
}
