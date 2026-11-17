package app

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Script plugins in the app (core/pluginscript.go). Each plugin that's on
// and allowed has a host; safe mode starts none. A plugin's Settings pages
// are described by its script and drawn by the page (79-pluginscript.js);
// bound calls run on the window's thread, so every call into a script is a
// goroutine that answers with an event.
// ---------------------------------------------------------------------------

type scriptRun struct {
	host    *core.ScriptHost
	plugin  core.Plugin
	views   map[string]json.RawMessage // page id -> the last view drawn
	offers  []core.ScriptOffer
	problem string
}

var scripts struct {
	sync.Mutex
	m map[string]*scriptRun
}

// PluginPageInfo is a Settings category a running script plugin draws.
type PluginPageInfo struct {
	Plugin string `json:"plugin"`
	Name   string `json:"name"`
	ID     string `json:"id"`
	Title  string `json:"title"`
}

func (a *App) pluginPages() []PluginPageInfo {
	scripts.Lock()
	defer scripts.Unlock()
	out := []PluginPageInfo{}
	for _, r := range scripts.m {
		for _, pg := range r.plugin.Pages {
			out = append(out, PluginPageInfo{Plugin: r.plugin.ID, Name: r.plugin.Name, ID: pg.ID, Title: pg.Title})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Plugin+out[i].ID < out[j].Plugin+out[j].ID })
	return out
}

// syncScripts starts the script of every plugin that's on and allowed, and
// stops the others. A plugin whose version changed starts again.
func (a *App) syncScripts() {
	if a.safe {
		return
	}
	want := map[string]core.Plugin{}
	for _, p := range core.ActivePlugins(DataDir()) {
		if p.Has("script") && p.Granted {
			want[p.ID] = p
		}
	}
	scripts.Lock()
	if scripts.m == nil {
		scripts.m = map[string]*scriptRun{}
	}
	var stop []*scriptRun
	for id, r := range scripts.m {
		if p, ok := want[id]; !ok || p.Version != r.plugin.Version || p.Key != r.plugin.Key {
			stop = append(stop, r)
			delete(scripts.m, id)
		} else {
			delete(want, id)
		}
	}
	scripts.Unlock()
	for _, r := range stop {
		r.host.Close()
		a.logf("Plugin %s: its script stopped", r.plugin.Name)
		a.pluginOffersChanged()
	}
	for _, p := range want {
		h, err := core.StartScript(p, DataDir(), a.scriptServices(p))
		if err != nil {
			a.logf("Plugin %s: its script didn't start: %v", p.Name, err)
			a.emit("toast", map[string]any{"kind": "error", "text": p.Name + " didn't start: " + sentence(err)})
			continue
		}
		scripts.Lock()
		scripts.m[p.ID] = &scriptRun{host: h, plugin: p, views: map[string]json.RawMessage{}}
		scripts.Unlock()
		a.logf("Plugin %s %s: its script runs", p.Name, p.Version)
		if _, err := h.Call("start", nil); err != nil && !errors.Is(err, core.ErrNoHandler) {
			a.logf("Plugin %s: start: %v", p.Name, err)
		}
	}
	if len(stop) > 0 || len(want) > 0 {
		a.emit("pluginPages", map[string]any{"pages": a.pluginPages()})
	}
}

func (a *App) scriptRun(id string) *scriptRun {
	scripts.Lock()
	defer scripts.Unlock()
	return scripts.m[id]
}

func (a *App) scriptServices(p core.Plugin) core.ScriptServices {
	id, name := p.ID, p.Name
	return core.ScriptServices{
		Log:    func(s string) { a.logf("Plugin %s: %s", name, s) },
		Secret: func(n string) string { return pluginSecret(id, n) },
		Games:  a.scriptGames,
		Notify: func(text string) {
			a.emit("toast", map[string]any{"text": text})
			a.notify(name, text)
		},
		Changed: func() { a.emit("pluginChanged", map[string]any{"id": id}) },
		Offers: func(list []core.ScriptOffer) {
			if r := a.scriptRun(id); r != nil {
				scripts.Lock()
				r.offers = list
				scripts.Unlock()
				a.pluginOffersChanged()
			}
		},
	}
}

// pluginOffersChanged tells the page the updates plugins offer changed.
func (a *App) pluginOffersChanged() { a.emit("pluginOffers", nil) }

// scriptGames are the Library's games as a script sees them.
func (a *App) scriptGames() []core.ScriptGame {
	var out []core.ScriptGame
	for _, f := range a.libraryGames() {
		g := core.ScriptGame{Key: f.Key, Name: f.Name, Source: f.Source, SteamID: f.SteamID}
		if f.Dir != "" {
			g.Folder = filepath.Base(f.Dir)
			g.Version = core.FolderVersion(f.Dir)
		}
		out = append(out, g)
	}
	return out
}

// ---- secrets: plugin-secrets.json, each value encrypted for this Windows
// user (DPAPI). Never sent to the page or to a script. ----

var secretMu sync.Mutex

func pluginSecretsPath() string { return filepath.Join(DataDir(), "plugin-secrets.json") }

func loadPluginSecrets() map[string]map[string]string {
	out := map[string]map[string]string{}
	if b, err := os.ReadFile(pluginSecretsPath()); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	return out
}

func pluginSecret(id, name string) string {
	secretMu.Lock()
	defer secretMu.Unlock()
	enc, err := base64.StdEncoding.DecodeString(loadPluginSecrets()[id][name])
	if err != nil || len(enc) == 0 {
		return ""
	}
	b, err := win.Unprotect(enc)
	if err != nil {
		return ""
	}
	return string(b)
}

func setPluginSecret(id, name, value string) error {
	secretMu.Lock()
	defer secretMu.Unlock()
	all := loadPluginSecrets()
	if value = strings.TrimSpace(value); value == "" {
		delete(all[id], name)
	} else {
		enc, err := win.Protect([]byte(value))
		if err != nil {
			return err
		}
		if all[id] == nil {
			all[id] = map[string]string{}
		}
		all[id][name] = base64.StdEncoding.EncodeToString(enc)
	}
	b, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	tmp := pluginSecretsPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, pluginSecretsPath())
}

// ---- the page's calls ----

// apiPluginGrant: the user read what a script plugin asks and allowed it.
func (a *App) apiPluginGrant(id string) error {
	if a.safe {
		return errPluginsSafe
	}
	p, ok := a.pluginByID(id)
	if !ok || !p.Has("script") {
		return errors.New("That plugin isn't installed.")
	}
	if err := core.GrantPlugin(DataDir(), id); err != nil {
		return err
	}
	a.logf("Plugin %s allowed: %s", p.Name, strings.Join(p.Asks, "; "))
	a.pluginsChanged(p.Has("sources"), true)
	return nil
}

// apiPluginView answers the last view of a plugin's page at once (or
// nothing), and asks the script for a fresh one, which comes as the event
// pluginView.
func (a *App) apiPluginView(id, page string) json.RawMessage {
	r := a.scriptRun(id)
	if r == nil {
		return nil
	}
	go a.drawPluginView(r, page)
	scripts.Lock()
	defer scripts.Unlock()
	return r.views[page]
}

func (a *App) drawPluginView(r *scriptRun, page string) {
	v, err := r.host.Call("view", map[string]any{"page": page})
	if err != nil {
		a.logf("Plugin %s: view %s: %v", r.plugin.Name, page, err)
		a.emit("pluginView", map[string]any{"id": r.plugin.ID, "page": page, "error": sentence(err)})
		return
	}
	scripts.Lock()
	r.views[page] = v
	scripts.Unlock()
	a.emit("pluginView", map[string]any{"id": r.plugin.ID, "page": page, "view": v})
}

// pluginActionReq is a button or a form of a plugin's page. Secrets are
// kept by Lootnest; the script only learns which were set.
type pluginActionReq struct {
	Req     string            `json:"req"`
	Page    string            `json:"page"`
	ID      string            `json:"id"`
	Row     string            `json:"row"`
	On      *bool             `json:"on,omitempty"`
	Value   string            `json:"value"`
	Values  map[string]any    `json:"values"`
	Secrets map[string]string `json:"secrets"`
}

// apiPluginAction runs a page's action in the script; the answer comes as
// the event pluginActed, then a fresh view.
func (a *App) apiPluginAction(id string, req pluginActionReq) error {
	r := a.scriptRun(id)
	if r == nil {
		return errors.New("That plugin isn't running.")
	}
	set := map[string]bool{}
	for name, val := range req.Secrets {
		ok := false
		for _, s := range r.plugin.Secrets {
			ok = ok || s.Name == name
		}
		if !ok {
			return errors.New("That plugin doesn't keep such a key.")
		}
		if err := setPluginSecret(id, name, val); err != nil {
			return err
		}
		set[name] = strings.TrimSpace(val) != ""
	}
	go func() {
		arg := map[string]any{"id": req.ID, "row": req.Row, "value": req.Value, "values": req.Values, "secrets": set}
		if req.On != nil {
			arg["on"] = *req.On
		}
		out, err := r.host.Call("action", arg)
		ev := map[string]any{"id": id, "req": req.Req}
		if err != nil {
			a.logf("Plugin %s: action %s: %v", r.plugin.Name, req.ID, err)
			ev["error"] = sentence(err)
		} else if out != nil {
			ev["result"] = out
		}
		a.emit("pluginActed", ev)
		if req.Page != "" {
			a.drawPluginView(r, req.Page)
		}
	}()
	return nil
}
