//go:build windows

package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Plugins (core/plugins.go): data only, read when they're installed or
// turned on and kept here until the next change. In safe mode none is on.
// ---------------------------------------------------------------------------

// PluginTheme is an on theme plugin's colours, checked in Go, for the page
// to set on :root while a profile that chose it is seated.
type PluginTheme struct {
	Name   string            `json:"name"`
	Tokens map[string]string `json:"tokens"`
}

type pluginMeta struct {
	name string
	meta core.PluginMeta
}

type pluginData struct {
	on      []core.Plugin
	sources map[string]bool // ids of the on plugins with sources
	saves   []core.PluginSaveDB
	meta    []pluginMeta
	themes  map[string]PluginTheme // by plugin id
}

var (
	plugMu   sync.Mutex
	plugRead *pluginData
)

func (a *App) plugins() *pluginData {
	if a.safe {
		return &pluginData{sources: map[string]bool{}, themes: map[string]PluginTheme{}}
	}
	plugMu.Lock()
	defer plugMu.Unlock()
	if plugRead != nil {
		return plugRead
	}
	d := &pluginData{on: core.ActivePlugins(DataDir()), sources: map[string]bool{}, themes: map[string]PluginTheme{}}
	for _, p := range d.on {
		if p.Has("sources") {
			d.sources[p.ID] = true
		}
		if p.Has("saves") {
			if db, err := core.PluginSaves(p.Dir); err == nil {
				d.saves = append(d.saves, core.PluginSaveDB{Name: p.Name, DB: db})
			}
		}
		if p.Has("metadata") {
			if m, err := core.PluginMetadata(p.Dir); err == nil {
				d.meta = append(d.meta, pluginMeta{p.Name, m})
			}
		}
		if p.Has("theme") {
			if t, err := core.PluginTheme(p.Dir); err == nil {
				d.themes[p.ID] = PluginTheme{Name: t.Name, Tokens: t.Tokens}
			}
		}
	}
	plugRead = d
	return d
}

// pluginsChanged reads the plugins again, and brings the Library up to
// date when a source plugin changed: a scan when one came on, its games
// out of the list at once when one went off.
func (a *App) pluginsChanged(sources, on bool) {
	plugMu.Lock()
	plugRead = nil
	plugMu.Unlock()
	if !sources {
		return
	}
	if on {
		a.startScan()
		return
	}
	a.emit("found", map[string]any{"scanning": false})
}

// recheckPlugins forgets the plugins read before when one of them no
// longer checks out in list (its files changed since), so it's off from
// then on, and says whether that one had sources.
func (a *App) recheckPlugins(list []core.Plugin) (dropped, sources bool) {
	plugMu.Lock()
	defer plugMu.Unlock()
	if plugRead == nil {
		return false, false
	}
	for _, c := range plugRead.on {
		for _, p := range list {
			if p.ID == c.ID && p.Problem != "" {
				dropped, sources = true, sources || c.Has("sources")
			}
		}
	}
	if dropped {
		plugRead = nil
	}
	return dropped, sources
}

// pluginGames are the games of the on source plugins, read on this PC,
// checked again first since they're read from the plugin's files.
func (a *App) pluginGames() []core.FoundGame {
	a.recheckPlugins(core.ListPlugins(DataDir()))
	var out []core.FoundGame
	for _, p := range a.plugins().on {
		if p.Has("sources") {
			games := core.PluginGames(p, os.Getenv, win.ReadRegKey)
			a.logf("Plugin %s: %d games", p.Name, len(games))
			out = append(out, games...)
		}
	}
	return out
}

// pluginDetails are the details a metadata plugin has for a Library game,
// and the plugin's name.
func (a *App) pluginDetails(key string) (*core.GameDetails, string) {
	meta := a.plugins().meta
	if len(meta) == 0 {
		return nil, ""
	}
	var names []string
	if id, ok := strings.CutPrefix(key, "game:"); ok {
		a.mu.Lock()
		if g := a.cfg.Game(id); g != nil {
			names = append(names, g.Name)
		}
		a.mu.Unlock()
	} else if f, ok := a.foundByKey(key); ok {
		names = append(names, f.Name, f.ScanName)
		if f.Dir != "" {
			names = append(names, filepath.Base(f.Dir))
		}
	}
	appID := a.detailsAppID(key)
	for _, m := range meta {
		if d := m.meta.Find(appID, names...); d != nil {
			return d, m.name
		}
	}
	return nil, ""
}

var errPluginsSafe = errors.New("Lootnest is in safe mode, so plugins stay off. Restart it normally to change them.")

func (a *App) apiPlugins() map[string]any {
	list := core.ListPlugins(DataDir())
	if dropped, sources := a.recheckPlugins(list); dropped {
		a.pluginsChanged(sources, false)
	}
	if list == nil {
		list = []core.Plugin{}
	}
	return map[string]any{"plugins": list}
}

func (a *App) pluginByID(id string) (core.Plugin, bool) {
	for _, p := range core.ListPlugins(DataDir()) {
		if p.ID == id {
			return p, true
		}
	}
	return core.Plugin{}, false
}

// apiInstallPlugin answers {plugin} once installed, or {keyChange} when
// the version is signed with another key than the pinned one, for the page
// to ask; only that dialog calls apiInstallPluginNewKey.
func (a *App) apiInstallPlugin(path string) (map[string]any, error) {
	return a.installPlugin(path, core.InstallPlugin)
}

// apiInstallPluginNewKey installs path only when it's signed with the key
// the user accepted (its fingerprint); any other key is asked about again.
func (a *App) apiInstallPluginNewKey(path, key string) (map[string]any, error) {
	return a.installPlugin(path, func(dataDir, src string) (*core.PluginInstalled, error) {
		return core.InstallPluginNewKey(dataDir, src, key)
	})
}

func (a *App) installPlugin(path string, install func(string, string) (*core.PluginInstalled, error)) (map[string]any, error) {
	if a.safe {
		return nil, errPluginsSafe
	}
	m, err := install(DataDir(), path)
	var kc *core.PluginKeyChange
	if errors.As(err, &kc) {
		a.logf("Plugin %s from %s not installed: signed with key %s, not %s", kc.Name, path, kc.Key, kc.Pinned)
		return map[string]any{"keyChange": kc}, nil
	}
	if err != nil {
		a.logf("Plugin from %s not installed: %v", path, err)
		return nil, errors.New("Lootnest didn't install it: " + err.Error() + ".")
	}
	a.logf("Plugin %s %s installed from %s, key %q", m.Name, m.Version, path, m.Key)
	p, _ := a.pluginByID(m.ID)
	a.pluginsChanged(p.Has("sources"), p.On)
	return map[string]any{"plugin": m}, nil
}

func (a *App) apiSetPluginOn(id string, on bool) error {
	if a.safe {
		return errPluginsSafe
	}
	p, ok := a.pluginByID(id)
	if !ok {
		return errors.New("That plugin isn't installed.")
	}
	if err := core.SetPluginOn(DataDir(), id, on); err != nil {
		return err
	}
	a.pluginsChanged(p.Has("sources"), on)
	return nil
}

func (a *App) apiRemovePlugin(id string) error {
	p, ok := a.pluginByID(id)
	if !ok {
		return errors.New("That plugin isn't installed.")
	}
	if err := core.RemovePlugin(DataDir(), id); err != nil {
		return err
	}
	a.logf("Plugin %s removed", p.Name)
	a.pluginsChanged(p.Has("sources"), false)
	return nil
}

func (a *App) apiRollbackPlugin(id string) (*core.PluginManifest, error) {
	if a.safe {
		return nil, errPluginsSafe
	}
	m, err := core.RollbackPlugin(DataDir(), id)
	if err != nil {
		return nil, errors.New("Lootnest couldn't roll it back: " + err.Error() + ".")
	}
	a.logf("Plugin %s rolled back to %s", m.Name, m.Version)
	p, _ := a.pluginByID(id)
	a.pluginsChanged(p.Has("sources"), p.On)
	return m, nil
}
