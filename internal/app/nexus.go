package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// ---------------------------------------------------------------------------
// Mods from Nexus Mods (core/nexus.go; the owner, 20 Nov 2026). A game's
// Mods tab can browse Nexus: search, a mod's page with what it needs and
// its files. Get opens the file's page on nexusmods.com in a window of
// Lootnest's own (nexuswin_windows.go; the browser where there's none),
// where the user downloads it as on the site. Lootnest waits for the file
// in Downloads, then closes that window and shows what adding it would do,
// as for a mod the user brings. Lootnest itself downloads nothing; it only
// reads Nexus's public API, and only with Settings, Online, Nexus Mods on.
// ---------------------------------------------------------------------------

const (
	nexusWaitFor   = 2 * time.Hour    // an ask is forgotten after this
	nexusLook      = 3 * time.Second  // how often Downloads is looked at while an ask is open
	nexusKeepPages = 10 * time.Minute // a mod's page is asked again after this
	nexusKeepGame  = time.Hour        // a game found (or not) on Nexus
)

var nexusHTTP = &http.Client{Timeout: 20 * time.Second}

// nexusPost sends one request to Nexus's API.
func nexusPost(addr string, body []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", addr, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", Name+"/"+Version)
	req.Header.Set("Application-Name", Name)
	req.Header.Set("Application-Version", Version)
	resp, err := nexusHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK && len(b) == 0 {
		return nil, fmt.Errorf("Nexus Mods answered %s", resp.Status)
	}
	return b, nil
}

// nexusFile is what's kept in nexus.json: the game the user chose on Nexus
// for a Library game, and the files they went to download.
type nexusFile struct {
	Picked map[string]*core.NexusGame `json:"picked,omitempty"`
	Waits  []core.NexusWait           `json:"waits,omitempty"`
}

type nexusGameKept struct {
	game *core.NexusGame
	at   time.Time
}

type nexusPageKept struct {
	page *core.NexusModPage
	at   time.Time
}

var nexus = struct {
	sync.Mutex
	loaded   bool
	file     nexusFile
	games    map[string]nexusGameKept // by Library key
	pages    map[string]nexusPageKept // by "<game id>/<mod id>"
	watching bool
	window   uint32 // the Nexus window's process, while it's open
}{games: map[string]nexusGameKept{}, pages: map[string]nexusPageKept{}}

func nexusPath() string { return filepath.Join(DataDir(), "nexus.json") }

// nexusLoad reads nexus.json once; call it with nexus held.
func nexusLoad() {
	if nexus.loaded {
		return
	}
	nexus.loaded = true
	if b, err := os.ReadFile(nexusPath()); err == nil {
		_ = json.Unmarshal(b, &nexus.file)
	}
	if nexus.file.Picked == nil {
		nexus.file.Picked = map[string]*core.NexusGame{}
	}
}

// nexusSave writes nexus.json; call it with nexus held.
func (a *App) nexusSave() {
	b, err := json.MarshalIndent(nexus.file, "", "  ")
	if err == nil {
		tmp := nexusPath() + ".tmp"
		if err = os.WriteFile(tmp, b, 0o644); err == nil {
			err = os.Rename(tmp, nexusPath())
		}
	}
	if err != nil {
		a.logf("ERROR saving nexus.json: %v", err)
	}
}

func (a *App) nexusOn() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg.Settings.NexusMods
}

var errNexusOff = errors.New("Turn on Nexus Mods in Settings, Online, to browse mods.")

// nexusGameOf is the game on Nexus a Library game is: the one the user
// chose, else the one found by its name (nil when Nexus has none).
func (a *App) nexusGameOf(key string) (*core.NexusGame, string, error) {
	t, err := a.modTarget(key)
	if err != nil {
		return nil, "", err
	}
	nexus.Lock()
	nexusLoad()
	if g := nexus.file.Picked[key]; g != nil {
		nexus.Unlock()
		return g, t.Name, nil
	}
	k, ok := nexus.games[key]
	nexus.Unlock()
	if ok && time.Since(k.at) < nexusKeepGame {
		return k.game, t.Name, nil
	}
	g, err := core.NexusFindGame(nexusPost, t.Name)
	if err != nil {
		return nil, t.Name, err
	}
	nexus.Lock()
	nexus.games[key] = nexusGameKept{g, time.Now()}
	nexus.Unlock()
	return g, t.Name, nil
}

// nexusKnown is the game on Nexus already chosen or found for a Library
// game, without asking Nexus.
func (a *App) nexusKnown(key string) (*core.NexusGame, string) {
	name := ""
	if t, err := a.modTarget(key); err == nil {
		name = t.Name
	}
	nexus.Lock()
	defer nexus.Unlock()
	nexusLoad()
	if g := nexus.file.Picked[key]; g != nil {
		return g, name
	}
	return nexus.games[key].game, name
}

func nexusSay(err error) string {
	if errors.Is(err, errNexusOff) {
		return err.Error()
	}
	return "Couldn't reach Nexus Mods. Check the connection and try again."
}

// apiNexusGame finds the game on Nexus; the answer is a "nexusGame" event.
func (a *App) apiNexusGame(key string) {
	go func() {
		out := map[string]any{"key": key}
		defer func() { a.emit("nexusGame", out) }()
		if !a.nexusOn() {
			out["off"] = true
			return
		}
		g, name, err := a.nexusGameOf(key)
		out["name"] = name
		if err != nil {
			a.logf("Nexus Mods, finding %s: %v", name, err)
			out["error"] = nexusSay(err)
			return
		}
		nexus.Lock()
		out["picked"] = nexus.file.Picked[key] != nil
		nexus.Unlock()
		if g == nil {
			out["none"] = true
			return
		}
		out["game"] = g
	}()
}

// apiNexusGames lists Nexus's games for words, for choosing one by hand;
// the answer is a "nexusGames" event.
func (a *App) apiNexusGames(q string) {
	go func() {
		out := map[string]any{"q": q}
		defer func() { a.emit("nexusGames", out) }()
		if !a.nexusOn() {
			out["error"] = errNexusOff.Error()
			return
		}
		gs, err := core.NexusFindGames(nexusPost, q)
		if err != nil {
			a.logf("Nexus Mods, games for %q: %v", q, err)
			out["error"] = nexusSay(err)
			return
		}
		out["games"] = gs
	}()
}

// apiNexusPick makes a game on Nexus the one for a Library game; nil goes
// back to the one found by its name.
func (a *App) apiNexusPick(key string, g *core.NexusGame) error {
	if g != nil && (g.ID <= 0 || core.NexusModURL(g.Domain, 1) == "") {
		return errors.New("That isn't a game on Nexus Mods.")
	}
	nexus.Lock()
	nexusLoad()
	if g == nil {
		delete(nexus.file.Picked, key)
	} else {
		nexus.file.Picked[key] = &core.NexusGame{ID: g.ID, Name: g.Name, Domain: g.Domain, Mods: g.Mods}
	}
	delete(nexus.games, key)
	a.nexusSave()
	nexus.Unlock()
	a.apiNexusGame(key)
	return nil
}

// apiNexusSearch is one page of a game's mods on Nexus; the answer is a
// "nexusMods" event carrying seq back, so the page keeps only the newest.
func (a *App) apiNexusSearch(key, q, order string, offset, seq int) {
	go func() {
		out := map[string]any{"key": key, "q": q, "sort": order, "offset": offset, "seq": seq}
		defer func() { a.emit("nexusMods", out) }()
		if !a.nexusOn() {
			out["error"] = errNexusOff.Error()
			return
		}
		g, _, err := a.nexusGameOf(key)
		if err == nil && g == nil {
			out["none"] = true
			return
		}
		var mods []core.NexusMod
		var total int
		if err == nil {
			mods, total, err = core.NexusSearch(nexusPost, g.ID, q, order, offset)
		}
		if err != nil {
			a.logf("Nexus Mods, search %q: %v", q, err)
			out["error"] = nexusSay(err)
			return
		}
		out["mods"], out["total"], out["domain"] = mods, total, g.Domain
	}()
}

// nexusPage is a mod's page, kept a few minutes.
func (a *App) nexusPage(g *core.NexusGame, modID int, fresh bool) (*core.NexusModPage, error) {
	ck := strconv.Itoa(g.ID) + "/" + strconv.Itoa(modID)
	nexus.Lock()
	k, ok := nexus.pages[ck]
	nexus.Unlock()
	if ok && !fresh && time.Since(k.at) < nexusKeepPages {
		return k.page, nil
	}
	p, err := core.NexusModInfo(nexusPost, g.ID, modID)
	if err != nil {
		return nil, err
	}
	nexus.Lock()
	nexus.pages[ck] = nexusPageKept{p, time.Now()}
	nexus.Unlock()
	return p, nil
}

// apiNexusMod is a mod's page; the answer is a "nexusMod" event.
func (a *App) apiNexusMod(key string, modID int) {
	go func() {
		out := map[string]any{"key": key, "modId": modID}
		defer func() { a.emit("nexusMod", out) }()
		if !a.nexusOn() {
			out["error"] = errNexusOff.Error()
			return
		}
		g, _, err := a.nexusGameOf(key)
		if err == nil && g == nil {
			err = errors.New("no game")
		}
		var p *core.NexusModPage
		if err == nil {
			p, err = a.nexusPage(g, modID, false)
		}
		if err != nil {
			a.logf("Nexus Mods, mod %d: %v", modID, err)
			out["error"] = nexusSay(err)
			return
		}
		if p == nil {
			out["none"] = true
			return
		}
		out["mod"], out["url"], out["domain"] = p, core.NexusModURL(g.Domain, modID), g.Domain
	}()
}

// apiNexusGet opens a mod's file on nexusmods.com for the user to download,
// and waits for it in Downloads (fileID 0 opens the mod's page and waits
// for nothing). The answer is a "nexusWaits" event.
func (a *App) apiNexusGet(key string, modID, fileID int) error {
	if !a.nexusOn() {
		return errNexusOff
	}
	// Bound calls run on the window's thread: only what's known already, no asking Nexus here.
	g, name := a.nexusKnown(key)
	if g == nil {
		return errors.New("Open the game's Nexus Mods pane again, then choose the file.")
	}
	if fileID <= 0 {
		return a.nexusOpen(g.Domain, modID, 0)
	}
	nexus.Lock()
	ck := strconv.Itoa(g.ID) + "/" + strconv.Itoa(modID)
	k := nexus.pages[ck]
	nexus.Unlock()
	if k.page == nil {
		return errors.New("Open the mod again, then choose its file.")
	}
	var f *core.NexusFile
	for i := range k.page.Files {
		if k.page.Files[i].ID == fileID {
			f = &k.page.Files[i]
		}
	}
	if f == nil {
		return errors.New("That file isn't on the mod's page any more.")
	}
	w := core.NexusWait{Key: key, Game: name, Domain: g.Domain, ModID: modID, FileID: fileID, Mod: k.page.Name, File: f.Name, Date: f.Date, Since: time.Now()}
	nexus.Lock()
	nexusLoad()
	waits := nexus.file.Waits[:0]
	for _, o := range nexus.file.Waits {
		if !(o.Key == key && o.ModID == modID) {
			waits = append(waits, o)
		}
	}
	nexus.file.Waits = append(waits, w)
	a.nexusSave()
	nexus.Unlock()
	a.logf("Waiting for %s (%s) from Nexus Mods in Downloads, for %s", w.Mod, w.File, name)
	if err := a.nexusOpen(g.Domain, modID, fileID); err != nil {
		return err
	}
	a.nexusWatch()
	a.emitNexusWaits()
	return nil
}

// nexusOpen shows a page of nexusmods.com: in Lootnest's window for it,
// else in the browser.
func (a *App) nexusOpen(domain string, modID, fileID int) error {
	arg := core.NexusWindowArg(domain, modID, fileID)
	if !nexusOwnWindow {
		return plat.ShellOpen(core.NexusWindowURL(arg))
	}
	nexus.Lock()
	old := nexus.window
	nexus.Unlock()
	if old != 0 && plat.ProcessAlive(old) {
		plat.CloseWindowsOfPID(old)
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	plat.AllowAnyForeground()
	pid, err := plat.StartTracked(self, "--nexus "+arg, filepath.Dir(self))
	if err != nil {
		a.logf("Couldn't open Lootnest's window for Nexus Mods: %v", err)
		return plat.ShellOpen(core.NexusWindowURL(arg))
	}
	nexus.Lock()
	nexus.window = pid
	nexus.Unlock()
	return nil
}

// apiNexusForget stops waiting for a mod's file.
func (a *App) apiNexusForget(key string, modID int) {
	nexus.Lock()
	nexusLoad()
	waits := nexus.file.Waits[:0]
	for _, w := range nexus.file.Waits {
		if !(w.Key == key && w.ModID == modID) {
			waits = append(waits, w)
		}
	}
	nexus.file.Waits = waits
	a.nexusSave()
	nexus.Unlock()
	a.emitNexusWaits()
}

// apiNexusWaits sends the files being waited for ("nexusWaits").
func (a *App) apiNexusWaits() { a.emitNexusWaits() }

func (a *App) emitNexusWaits() {
	nexus.Lock()
	nexusLoad()
	waits := append([]core.NexusWait{}, nexus.file.Waits...)
	nexus.Unlock()
	a.emit("nexusWaits", map[string]any{"waits": waits})
}

// nexusWatch looks at Downloads every few seconds while a file is awaited
// (from a start too, for asks made before Lootnest closed).
func (a *App) nexusWatch() {
	nexus.Lock()
	nexusLoad()
	if nexus.watching || len(nexus.file.Waits) == 0 {
		nexus.Unlock()
		return
	}
	nexus.watching = true
	nexus.Unlock()
	go func() {
		t := time.NewTicker(nexusLook)
		defer t.Stop()
		for range t.C {
			if !a.nexusLook() {
				return
			}
		}
	}()
}

// nexusLook matches what arrived in Downloads to the asks open, and says
// whether any are still open.
func (a *App) nexusLook() bool {
	nexus.Lock()
	now := time.Now()
	var open []core.NexusWait
	for _, w := range nexus.file.Waits {
		if now.Sub(w.Since) < nexusWaitFor {
			open = append(open, w)
		}
	}
	waits := append([]core.NexusWait{}, open...)
	nexus.Unlock()
	var got []core.NexusArrival
	if !a.nexusOn() {
		waits = nil
	}
	for _, dir := range downloadDirs() {
		if len(waits) == 0 {
			break
		}
		for _, arr := range core.NexusArrivals(dir, waits) {
			got = append(got, arr)
			for i, w := range waits {
				if w.Key == arr.Key && w.ModID == arr.ModID {
					waits = append(waits[:i], waits[i+1:]...)
					break
				}
			}
		}
	}
	on := a.nexusOn()
	nexus.Lock()
	// Only what was found or forgotten goes: an ask made meanwhile stays.
	if len(got) > 0 || len(open) != len(nexus.file.Waits) || !on {
		keep := []core.NexusWait{}
		for _, w := range nexus.file.Waits {
			done := now.Sub(w.Since) >= nexusWaitFor || !on
			for _, arr := range got {
				done = done || w.Key == arr.Key && w.ModID == arr.ModID
			}
			if !done {
				keep = append(keep, w)
			}
		}
		nexus.file.Waits = keep
		a.nexusSave()
	}
	left := len(nexus.file.Waits) > 0
	if !left {
		nexus.watching = false
	}
	win := nexus.window
	nexus.Unlock()
	for _, arr := range got {
		a.logf("%s from Nexus Mods arrived in Downloads: %s", arr.Mod, arr.Path)
		if !left && win != 0 && plat.ProcessAlive(win) {
			plat.CloseWindowsOfPID(win)
		}
		a.ui(a.show)
		a.emit("nexusArrived", map[string]any{"arrival": arr})
	}
	if len(got) > 0 {
		a.emitNexusWaits()
	}
	return left
}

// apiNexusAdd adds a file that arrived from Nexus to the game's mods, named
// after its page; the answer is "modPlan", as for any mod.
func (a *App) apiNexusAdd(key, path string, modID, fileID int) error {
	if st, err := os.Stat(path); err != nil || st.IsDir() {
		return errors.New("That file isn't in your Downloads any more.")
	}
	go func() {
		name, from := "", (*core.ModNexus)(nil)
		if g, _, err := a.nexusGameOf(key); err == nil && g != nil {
			from = &core.ModNexus{Domain: g.Domain, ModID: modID, FileID: fileID}
			if p, err := a.nexusPage(g, modID, false); err == nil && p != nil {
				name = p.Name
				for _, f := range p.Files {
					if f.ID == fileID {
						from.Version = f.Version
					}
				}
			}
		}
		if err := a.addMod(key, path, name, from); err != nil {
			a.emit("modPlan", map[string]any{"key": key, "path": path, "refused": err.Error()})
		}
	}()
	return nil
}

// nexusAtStart picks up asks still open from before.
func (a *App) nexusAtStart() {
	if a.safe || !a.nexusOn() {
		return
	}
	a.nexusWatch()
}
