package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dop251/goja"
)

// ---------------------------------------------------------------------------
// Script plugins (the owner's change of 17 Nov 2026: a plugin may hold code).
// A plugin that provides "script" has one .js, run inside Lootnest by a
// JavaScript engine written in Go (goja): no process is started, no file of
// the plugin's is loaded as a program, and the script reaches nothing but
// the `lootnest` object below. plugin.json lists what it asks to do
// (permissions) and the keys it keeps (secrets); it runs only once the user
// allowed exactly that list, and a version that asks for something else is
// off until they allow that too.
//
// What a script can do, by permission:
//
//	(always)  lootnest.on, log, schedule, hash, storage.*, ui.changed
//	net       lootnest.net.fetch: http and https, never this PC or its
//	          network, with size and time limits; a secret goes only to
//	          the host plugin.json names for it
//	library   lootnest.library.games: names, folder names, versions
//	updates   lootnest.updates.offer, and the update calls of
//	          pluginupdate.go, which always ask the user
//	notify    lootnest.notify
//
// Only JSON crosses between the script and Lootnest. Every call into the
// script has a time limit; the engine has no memory limit, which is why a
// script plugin is never on by default.
// ---------------------------------------------------------------------------

// PluginSecret is a key a script plugin keeps (a service's token). The
// user types it into a form Lootnest draws; the script never reads it, and
// Lootnest sends it only to Host, over https.
type PluginSecret struct {
	Name  string `json:"name"`
	Host  string `json:"host"`
	Label string `json:"label"`
}

// PluginPage is a Settings category a script plugin draws.
type PluginPage struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

var (
	scriptNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,40}\.js$`)
	scriptKeyRe  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,39}$`)
	scriptHostRe = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,24}$`)
	headerNameRe = regexp.MustCompile(`^[A-Za-z0-9-]{1,40}$`)
)

// scriptPerms are the permissions, in the order they're shown, with what
// each means in words.
var scriptPerms = [][2]string{
	{"net", "Go online, to any address it or you give it"},
	{"library", "Read the list of your games: their names, folders and versions"},
	{"updates", "Offer updates for your games, download them, and replace a game's folder once you say yes"},
	{"notify", "Show notices"},
}

func scriptPerm(name string) string {
	for _, p := range scriptPerms {
		if p[0] == name {
			return p[1]
		}
	}
	return ""
}

func (m *PluginManifest) hasScript() bool { return slices.Contains(m.Provides, "script") }

// checkScriptManifest checks the parts of plugin.json only a script plugin
// has.
func checkScriptManifest(m *PluginManifest, script bool) error {
	if !script {
		if m.Script != "" || len(m.Permissions) > 0 || len(m.Secrets) > 0 || len(m.Pages) > 0 {
			return errors.New(`it has a script, permissions, secrets or pages, but doesn't provide "script"`)
		}
		return nil
	}
	if !scriptNameRe.MatchString(m.Script) {
		return errors.New("its script must be one .js at its top, named in lowercase letters, digits, hyphens or underscores")
	}
	seen := map[string]bool{}
	for _, p := range m.Permissions {
		if scriptPerm(p) == "" || seen[p] {
			return fmt.Errorf("it asks for %q, which isn't net, library, updates or notify", p)
		}
		seen[p] = true
	}
	if len(m.Secrets) > 5 {
		return errors.New("it keeps more than 5 keys")
	}
	names := map[string]bool{}
	for i := range m.Secrets {
		s := &m.Secrets[i]
		s.Label = strings.Join(strings.Fields(s.Label), " ")
		switch {
		case !scriptKeyRe.MatchString(s.Name) || names[s.Name]:
			return errors.New("a key's name must be lowercase letters, digits or hyphens, and used once")
		case !scriptHostRe.MatchString(s.Host):
			return fmt.Errorf("the key %q must name the one site it's sent to, such as api.example.com", s.Name)
		case s.Label == "" || len([]rune(s.Label)) > 60:
			return fmt.Errorf("the key %q needs a label of 1 to 60 characters", s.Name)
		case !seen["net"]:
			return errors.New("it keeps a key but doesn't ask to go online")
		}
		names[s.Name] = true
	}
	if len(m.Pages) > 3 {
		return errors.New("it has more than 3 pages")
	}
	ids := map[string]bool{}
	for i := range m.Pages {
		p := &m.Pages[i]
		p.Title = strings.Join(strings.Fields(p.Title), " ")
		if !scriptKeyRe.MatchString(p.ID) || ids[p.ID] || p.Title == "" || len([]rune(p.Title)) > 24 {
			return errors.New("each page needs its own id and a title of 1 to 24 characters")
		}
		ids[p.ID] = true
	}
	return nil
}

// checkPluginScript refuses a script that doesn't parse, at install.
func checkPluginScript(dir string, m *PluginManifest) error {
	src, err := os.ReadFile(filepath.Join(dir, m.Script))
	if err != nil {
		return errors.New("its script, " + m.Script + ", isn't in it")
	}
	if _, err := goja.Compile(m.Script, string(src), true); err != nil {
		return errors.New("its script can't be read: " + shortErr(err))
	}
	return nil
}

// pluginGrant is what allowing a script plugin covers: its permissions and
// the sites its keys go to. A version with another list asks again.
func pluginGrant(m *PluginManifest) []string {
	out := append([]string{"script"}, m.Permissions...)
	for _, s := range m.Secrets {
		out = append(out, "secret:"+s.Name+"@"+s.Host)
	}
	sort.Strings(out)
	return out
}

// PluginAsks is what a script plugin asks to do, in words, for the user to
// read before turning it on.
func PluginAsks(m *PluginManifest) []string {
	out := []string{"Run its own script inside Lootnest, and keep its own data there"}
	for _, p := range scriptPerms {
		if slices.Contains(m.Permissions, p[0]) {
			out = append(out, p[1])
		}
	}
	for _, s := range m.Secrets {
		out = append(out, "Keep your "+s.Label+", sent only to "+s.Host)
	}
	return out
}

// GrantPlugin records that the user allowed what the installed version of
// a script plugin asks, and turns it on.
func GrantPlugin(dataDir, id string) error {
	if !pluginIDRe.MatchString(id) {
		return errors.New("that plugin isn't installed")
	}
	m, err := CheckPluginDir(filepath.Join(PluginsDir(dataDir), id))
	if err != nil || !m.hasScript() {
		return errors.New("that plugin has nothing to allow")
	}
	st := loadPlugins(dataDir)
	st.Granted[id], st.On[id] = pluginGrant(m), true
	return writeJSONAtomic(pluginsState(dataDir), st)
}

// PluginDataDir is where a script plugin keeps its own data. It stays when
// the plugin is removed, so installing it again finds it.
func PluginDataDir(dataDir, id string) string { return filepath.Join(dataDir, "plugin-data", id) }

// ScriptGame is a Library game as a script sees it: no full paths.
type ScriptGame struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Source  string `json:"source"`
	SteamID int    `json:"steamId,omitempty"`
	Folder  string `json:"folder,omitempty"`  // the install folder's name
	Version string `json:"version,omitempty"` // read on this PC, when it can be
	// ExeVersion is the exe's own file version. For many games it's the
	// engine's, so a script should only trust it when it fits what else is
	// known.
	ExeVersion string `json:"exeVersion,omitempty"`
	Size       int64  `json:"size,omitempty"`
}

// ScriptAction is a button a script names; Lootnest draws it.
type ScriptAction struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// ScriptOffer is an update a script offers for a Library game.
type ScriptOffer struct {
	ID      string         `json:"id"`
	Game    string         `json:"game"` // the Library key
	Title   string         `json:"title"`
	Version string         `json:"version,omitempty"`
	Have    string         `json:"have,omitempty"` // the version on this PC, in words
	Source  string         `json:"source,omitempty"`
	Size    int64          `json:"size,omitempty"`
	Date    string         `json:"date,omitempty"`
	Note    string         `json:"note,omitempty"`
	Status  string         `json:"status,omitempty"`
	Busy    bool           `json:"busy,omitempty"`
	Actions []ScriptAction `json:"actions,omitempty"`
}

// ScriptServices is what the app gives a script's host. A nil function is a
// service that isn't there (in tests).
type ScriptServices struct {
	Log    func(string)
	Secret func(name string) string
	// DropSecret forgets a key.
	DropSecret func(name string)
	Games      func() []ScriptGame
	Notify     func(text string)
	Changed    func()
	Offers     func([]ScriptOffer)
	// Update starts or steers an update the script asked for
	// (pluginupdate.go); nil until the app can do them.
	Update func(op string, req json.RawMessage) (any, error)

	AllowLocal bool          // tests only: fetch may reach this PC
	CallLimit  time.Duration // the most one call into the script may take
}

const (
	scriptCallLimit  = 3 * time.Minute
	scriptFetchMax   = 64 << 20
	scriptFetchDef   = 8 << 20
	scriptStoreMax   = 64 << 20  // one stored value
	scriptStoreTotal = 512 << 20 // all of a plugin's stored values
	scriptTimersMax  = 64
	scriptViewMax    = 2 << 20
	scriptOffersMax  = 500
)

// ErrNoHandler: the script has no handler of that name.
var ErrNoHandler = errors.New("the plugin has no such handler")

// ScriptHost runs one plugin's script.
type ScriptHost struct {
	p    Plugin
	data string
	svc  ScriptServices

	mu        sync.Mutex // one call into the script at a time
	vm        *goja.Runtime
	on        map[string]goja.Callable
	parse     goja.Callable
	stringify goja.Callable
	closed    bool

	tmu    sync.Mutex
	timers map[string]*time.Timer
	client *http.Client
}

// StartScript loads a plugin's script and runs its top level, which
// registers its handlers. dataDir is Lootnest's data folder.
func StartScript(p Plugin, dataDir string, svc ScriptServices) (*ScriptHost, error) {
	if !p.hasScript() {
		return nil, errors.New("that plugin has no script")
	}
	src, err := os.ReadFile(filepath.Join(p.Dir, p.Script))
	if err != nil {
		return nil, err
	}
	if svc.CallLimit <= 0 {
		svc.CallLimit = scriptCallLimit
	}
	h := &ScriptHost{p: p, data: PluginDataDir(dataDir, p.ID), svc: svc, on: map[string]goja.Callable{}, timers: map[string]*time.Timer{}}
	h.client = &http.Client{
		Transport: &http.Transport{Proxy: nil, DialContext: guardedDial(svc.AllowLocal), TLSHandshakeTimeout: 15 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second, MaxIdleConns: 4, IdleConnTimeout: 30 * time.Second},
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			if r.URL.Scheme != "https" && r.URL.Scheme != "http" {
				return errors.New("a redirect left http")
			}
			if r.URL.Host != via[0].URL.Host {
				r.Header.Del("Authorization") // a key never follows a redirect to another site
			}
			return nil
		},
	}
	vm := goja.New()
	vm.SetMaxCallStackSize(2000)
	h.vm = vm
	j := vm.Get("JSON").ToObject(vm)
	h.parse, _ = goja.AssertFunction(j.Get("parse"))
	h.stringify, _ = goja.AssertFunction(j.Get("stringify"))
	h.install()
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.limited(func() error {
		_, err := vm.RunScript(p.Script, string(src))
		return err
	}); err != nil {
		return nil, err
	}
	return h, nil
}

// Has says whether the script registered a handler.
func (h *ScriptHost) Has(name string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.on[name] != nil
}

// Call runs one of the script's handlers with arg (as JSON) and returns
// what it returned, as JSON. Calls wait for each other.
func (h *ScriptHost) Call(name string, arg any) (json.RawMessage, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return nil, errors.New("the plugin is off")
	}
	fn := h.on[name]
	if fn == nil {
		return nil, ErrNoHandler
	}
	var out json.RawMessage
	err := h.limited(func() error {
		var av goja.Value = goja.Undefined()
		if arg != nil {
			b, err := json.Marshal(arg)
			if err != nil {
				return err
			}
			if av, err = h.parse(goja.Undefined(), h.vm.ToValue(string(b))); err != nil {
				return err
			}
		}
		v, err := fn(goja.Undefined(), av)
		if err != nil {
			return err
		}
		if pr, ok := v.Export().(*goja.Promise); ok {
			if pr.State() == goja.PromiseStatePending {
				_, _ = h.vm.RunString("") // lets what it awaits finish
			}
			switch pr.State() {
			case goja.PromiseStateRejected:
				return errors.New(pr.Result().String())
			case goja.PromiseStatePending:
				return errors.New("it returned a promise that never settles; Lootnest's calls don't wait")
			}
			v = pr.Result()
		}
		if v == nil || goja.IsUndefined(v) {
			return nil
		}
		s, err := h.stringify(goja.Undefined(), v)
		if err != nil {
			return err
		}
		if s != nil && !goja.IsUndefined(s) {
			if out = json.RawMessage(s.String()); len(out) > scriptViewMax {
				return errors.New("its answer is over 2 MB")
			}
		}
		return nil
	})
	return out, err
}

// limited runs f, which calls into the script, and stops the script when it
// takes longer than the limit. h.mu is held.
func (h *ScriptHost) limited(f func() error) (err error) {
	t := time.AfterFunc(h.svc.CallLimit, func() { h.vm.Interrupt("it took too long") })
	defer func() {
		t.Stop()
		h.vm.ClearInterrupt()
		if r := recover(); r != nil {
			err = fmt.Errorf("the plugin's script failed: %v", r)
		}
		var ex *goja.Exception
		var ie *goja.InterruptedError
		switch {
		case errors.As(err, &ie):
			err = errors.New("the plugin's script took too long and was stopped")
		case errors.As(err, &ex):
			err = errors.New(scriptErrText(ex))
		}
	}()
	return f()
}

func scriptErrText(ex *goja.Exception) string {
	v := ex.Value()
	if o, ok := v.(*goja.Object); ok {
		if m := o.Get("message"); m != nil && !goja.IsUndefined(m) {
			return m.String()
		}
	}
	return v.String()
}

// Close stops the script: its timers go and later calls are refused.
func (h *ScriptHost) Close() {
	h.tmu.Lock()
	for k, t := range h.timers {
		t.Stop()
		delete(h.timers, k)
	}
	h.tmu.Unlock()
	h.vm.Interrupt("the plugin was turned off")
	h.mu.Lock()
	h.closed = true
	h.vm.ClearInterrupt()
	h.mu.Unlock()
	h.client.CloseIdleConnections()
}

func (h *ScriptHost) log(format string, args ...any) {
	if h.svc.Log != nil {
		h.svc.Log(fmt.Sprintf(format, args...))
	}
}

// fail throws msg into the script as an Error.
func (h *ScriptHost) fail(format string, args ...any) {
	e, err := h.vm.New(h.vm.Get("Error"), h.vm.ToValue(fmt.Sprintf(format, args...)))
	if err != nil {
		panic(h.vm.ToValue(fmt.Sprintf(format, args...)))
	}
	panic(e)
}

func (h *ScriptHost) need(perm string) {
	if !slices.Contains(h.p.Permissions, perm) {
		h.fail("this plugin didn't ask for %q in its plugin.json", perm)
	}
}

// toJS gives the script v, by way of JSON.
func (h *ScriptHost) toJS(v any) goja.Value {
	b, err := json.Marshal(v)
	if err != nil {
		h.fail("%v", err)
	}
	out, err := h.parse(goja.Undefined(), h.vm.ToValue(string(b)))
	if err != nil {
		h.fail("%v", err)
	}
	return out
}

// fromJS reads a value of the script's into dst, by way of JSON.
func (h *ScriptHost) fromJS(v goja.Value, dst any) {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return
	}
	s, err := h.stringify(goja.Undefined(), v)
	if err != nil || s == nil || goja.IsUndefined(s) {
		h.fail("that value can't be read")
	}
	if err := json.Unmarshal([]byte(s.String()), dst); err != nil {
		h.fail("that value isn't what Lootnest expects: %s", shortErr(err))
	}
}

func cutRunes(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

// install puts the `lootnest` object, and console.log, into the engine.
func (h *ScriptHost) install() {
	vm := h.vm
	obj := func(fns map[string]any) *goja.Object {
		o := vm.NewObject()
		for k, f := range fns {
			_ = o.Set(k, f)
		}
		return o
	}
	logFn := func(c goja.FunctionCall) goja.Value {
		parts := make([]string, len(c.Arguments))
		for i, a := range c.Arguments {
			parts[i] = a.String()
		}
		h.log("%s", cutRunes(strings.Join(parts, " "), 500))
		return goja.Undefined()
	}
	ln := obj(map[string]any{
		"plugin": map[string]any{"id": h.p.ID, "version": h.p.Version},
		"log":    logFn,
		"on": func(c goja.FunctionCall) goja.Value {
			name := c.Argument(0).String()
			fn, ok := goja.AssertFunction(c.Argument(1))
			if !scriptKeyRe.MatchString(name) || !ok {
				h.fail("lootnest.on takes a name in lowercase letters, digits or hyphens, and a function")
			}
			h.on[name] = fn
			return goja.Undefined()
		},
		"schedule": h.jsSchedule,
		"hash": func(c goja.FunctionCall) goja.Value {
			sum := sha256.Sum256([]byte(c.Argument(0).String()))
			return vm.ToValue(hex.EncodeToString(sum[:]))
		},
		"notify": func(c goja.FunctionCall) goja.Value {
			h.need("notify")
			if t := cutRunes(c.Argument(0).String(), 200); t != "" && h.svc.Notify != nil {
				h.svc.Notify(t)
			}
			return goja.Undefined()
		},
	})
	_ = ln.Set("storage", obj(map[string]any{"get": h.jsGet, "set": h.jsSet, "remove": h.jsRemove}))
	_ = ln.Set("net", obj(map[string]any{"fetch": h.jsFetch}))
	_ = ln.Set("ui", obj(map[string]any{"changed": func(goja.FunctionCall) goja.Value {
		if h.svc.Changed != nil {
			h.svc.Changed()
		}
		return goja.Undefined()
	}}))
	_ = ln.Set("secrets", obj(map[string]any{
		"has": func(c goja.FunctionCall) goja.Value {
			name := c.Argument(0).String()
			return vm.ToValue(h.secret(name) != nil && h.svc.Secret != nil && h.svc.Secret(name) != "")
		},
		// A script can forget a key (one that was refused, or on the user's
		// Disconnect), never read or set one.
		"remove": func(c goja.FunctionCall) goja.Value {
			if s := h.secret(c.Argument(0).String()); s != nil && h.svc.DropSecret != nil {
				h.svc.DropSecret(s.Name)
			}
			return goja.Undefined()
		},
	}))
	_ = ln.Set("library", obj(map[string]any{"games": func(goja.FunctionCall) goja.Value {
		h.need("library")
		var games []ScriptGame
		if h.svc.Games != nil {
			games = h.svc.Games()
		}
		if games == nil {
			games = []ScriptGame{}
		}
		return h.toJS(games)
	}}))
	_ = ln.Set("updates", obj(map[string]any{
		"offer": func(c goja.FunctionCall) goja.Value {
			h.need("updates")
			var list []ScriptOffer
			h.fromJS(c.Argument(0), &list)
			if len(list) > scriptOffersMax {
				list = list[:scriptOffersMax]
			}
			for i := range list {
				o := &list[i]
				if !scriptIDRe.MatchString(o.ID) {
					h.fail("an offer's id must be 1 to 80 letters, digits, dots, colons, hyphens or underscores")
				}
				o.Title, o.Version, o.Source, o.Have = cutRunes(o.Title, 300), cutRunes(o.Version, 60), cutRunes(o.Source, 60), cutRunes(o.Have, 60)
				o.Date, o.Note, o.Status = cutRunes(o.Date, 40), cutRunes(o.Note, 300), cutRunes(o.Status, 120)
				if len(o.Actions) > 4 {
					o.Actions = o.Actions[:4]
				}
				for k := range o.Actions {
					a := &o.Actions[k]
					if a.Label = cutRunes(a.Label, 40); !scriptKeyRe.MatchString(a.ID) || a.Label == "" {
						h.fail("an offer's action needs an id and a label")
					}
				}
			}
			if h.svc.Offers != nil {
				h.svc.Offers(list)
			}
			return goja.Undefined()
		},
		"run": func(c goja.FunctionCall) goja.Value {
			h.need("updates")
			if h.svc.Update == nil {
				h.fail("updates aren't available here")
			}
			op := c.Argument(0).String()
			var req json.RawMessage
			h.fromJS(c.Argument(1), &req)
			out, err := h.svc.Update(op, req)
			if err != nil {
				h.fail("%s", err.Error())
			}
			if out == nil {
				return goja.Undefined()
			}
			return h.toJS(out)
		},
	}))
	_ = vm.Set("lootnest", ln)
	_ = vm.Set("console", obj(map[string]any{"log": logFn, "warn": logFn, "error": logFn}))
}

var scriptIDRe = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,80}$`)

func (h *ScriptHost) secret(name string) *PluginSecret {
	for i := range h.p.Secrets {
		if h.p.Secrets[i].Name == name {
			return &h.p.Secrets[i]
		}
	}
	return nil
}

// jsSchedule: lootnest.schedule(name, ms, arg) calls the handler `name`
// with arg after ms. One pending call per name: a new one replaces it, and
// ms under 0 only cancels.
func (h *ScriptHost) jsSchedule(c goja.FunctionCall) goja.Value {
	name := c.Argument(0).String()
	if !scriptKeyRe.MatchString(name) {
		h.fail("lootnest.schedule takes a handler's name")
	}
	ms := c.Argument(1).ToInteger()
	var arg json.RawMessage
	h.fromJS(c.Argument(2), &arg)
	h.tmu.Lock()
	defer h.tmu.Unlock()
	if t := h.timers[name]; t != nil {
		t.Stop()
		delete(h.timers, name)
	}
	if ms < 0 {
		return goja.Undefined()
	}
	if len(h.timers) >= scriptTimersMax {
		h.fail("too many scheduled calls")
	}
	d := min(max(time.Duration(ms)*time.Millisecond, 20*time.Millisecond), 30*24*time.Hour)
	var t *time.Timer
	t = time.AfterFunc(d, func() {
		h.tmu.Lock()
		mine := h.timers[name] == t
		if mine {
			delete(h.timers, name)
		}
		h.tmu.Unlock()
		if !mine {
			return
		}
		var a any
		if arg != nil {
			a = arg
		}
		if _, err := h.Call(name, a); err != nil && !errors.Is(err, ErrNoHandler) {
			h.log("%s: %v", name, err)
		}
	})
	h.timers[name] = t
	return goja.Undefined()
}

// ---- storage: one JSON file per key in the plugin's own data folder ----

func (h *ScriptHost) storePath(c goja.FunctionCall) string {
	key := c.Argument(0).String()
	if !scriptKeyRe.MatchString(key) {
		h.fail("a storage key is 1 to 40 lowercase letters, digits or hyphens")
	}
	return filepath.Join(h.data, key+".json")
}

func (h *ScriptHost) jsGet(c goja.FunctionCall) goja.Value {
	b, err := os.ReadFile(h.storePath(c))
	if err != nil {
		return goja.Null()
	}
	v, err := h.parse(goja.Undefined(), h.vm.ToValue(string(b)))
	if err != nil {
		return goja.Null() // a damaged file reads as nothing; set writes a new one
	}
	return v
}

func (h *ScriptHost) jsSet(c goja.FunctionCall) goja.Value {
	p := h.storePath(c)
	s, err := h.stringify(goja.Undefined(), c.Argument(1))
	if err != nil || s == nil || goja.IsUndefined(s) {
		h.fail("that value can't be stored")
	}
	b := []byte(s.String())
	if len(b) > scriptStoreMax {
		h.fail("that value is over 64 MB")
	}
	var total int64
	ents, _ := os.ReadDir(h.data)
	for _, e := range ents {
		if info, err := e.Info(); err == nil && !e.IsDir() && e.Name() != filepath.Base(p) {
			total += info.Size()
		}
	}
	if total+int64(len(b)) > scriptStoreTotal {
		h.fail("this plugin's data would be over 512 MB")
	}
	if err := os.MkdirAll(h.data, 0o755); err != nil {
		h.fail("%v", err)
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		h.fail("%v", err)
	}
	if err := os.Rename(tmp, p); err != nil {
		_ = os.Remove(tmp)
		h.fail("%v", err)
	}
	return goja.Undefined()
}

func (h *ScriptHost) jsRemove(c goja.FunctionCall) goja.Value {
	_ = os.Remove(h.storePath(c))
	return goja.Undefined()
}

// ---- net ----

var errLocalAddress = errors.New("plugins can't reach this PC or its network")

// guardedDial refuses connections to this PC, its network and other
// addresses that aren't the public internet, whatever name led there.
func guardedDial(allowLocal bool) func(ctx context.Context, network, addr string) (net.Conn, error) {
	_, cgnat, _ := net.ParseCIDR("100.64.0.0/10")
	d := &net.Dialer{Timeout: 15 * time.Second, Control: func(network, address string, _ syscall.RawConn) error {
		if allowLocal {
			return nil
		}
		host, _, err := net.SplitHostPort(address)
		ip := net.ParseIP(host)
		if err != nil || ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsMulticast() ||
			ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || cgnat.Contains(ip) {
			return errLocalAddress
		}
		return nil
	}}
	return d.DialContext
}

type scriptFetch struct {
	Method   string            `json:"method"`
	Headers  map[string]string `json:"headers"`
	Body     string            `json:"body"`
	Form     map[string]string `json:"form"`
	MaxBytes int64             `json:"maxBytes"`
	Timeout  int               `json:"timeout"` // seconds
	Secret   string            `json:"secret"`
}

// jsFetch: lootnest.net.fetch(url, {method, headers, body, form, maxBytes,
// timeout, secret}) gives {status, headers, text}. It waits for the answer.
func (h *ScriptHost) jsFetch(c goja.FunctionCall) goja.Value {
	h.need("net")
	var o scriptFetch
	h.fromJS(c.Argument(1), &o)
	u, err := url.Parse(strings.TrimSpace(c.Argument(0).String()))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil {
		h.fail("that address isn't http or https")
	}
	method := strings.ToUpper(o.Method)
	if method == "" {
		method = "GET"
	}
	if !slices.Contains([]string{"GET", "POST", "PUT", "DELETE", "HEAD"}, method) {
		h.fail("that method isn't allowed")
	}
	body := o.Body
	if len(o.Form) > 0 {
		f := url.Values{}
		for k, v := range o.Form {
			f.Set(k, v)
		}
		body = f.Encode()
	}
	if len(body) > 1<<20 {
		h.fail("a request's body can't be over 1 MB")
	}
	limit := o.MaxBytes
	if limit <= 0 {
		limit = scriptFetchDef
	}
	limit = min(limit, scriptFetchMax)
	secs := o.Timeout
	if secs <= 0 {
		secs = 30
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(min(secs, 180))*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, u.String(), strings.NewReader(body))
	if err != nil {
		h.fail("that address can't be read")
	}
	for k, v := range o.Headers {
		lk := strings.ToLower(k)
		if !headerNameRe.MatchString(k) || strings.ContainsAny(v, "\r\n") || lk == "host" || lk == "cookie" || lk == "content-length" || (lk == "authorization" && o.Secret != "") {
			h.fail("the header %q can't be set", k)
		}
		req.Header.Set(k, v)
	}
	if len(o.Form) > 0 {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Lootnest")
	}
	if o.Secret != "" {
		s := h.secret(o.Secret)
		var val string
		if s != nil && h.svc.Secret != nil {
			val = h.svc.Secret(s.Name)
		}
		switch {
		case s == nil:
			h.fail("plugin.json doesn't list the key %q", o.Secret)
		case u.Scheme != "https" || !strings.EqualFold(u.Hostname(), s.Host):
			h.fail("the key %q only goes to https://%s", s.Name, s.Host)
		case val == "":
			h.fail("the key %q isn't set yet", s.Name)
		}
		req.Header.Set("Authorization", "Bearer "+val)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err // the address may hold a token
		}
		h.log("%s %s: %v", method, u.Host, err)
		if errors.Is(err, errLocalAddress) {
			h.fail("%s", errLocalAddress.Error())
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.fail("%s took too long to answer", u.Host)
		}
		h.fail("couldn't reach %s (%s)", u.Host, shortErr(err))
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		h.log("%s %s: %v", method, u.Host, err)
		h.fail("the answer from %s was cut short", u.Host)
	}
	h.log("%s %s: %d, %d bytes", method, u.Host, resp.StatusCode, len(b))
	if int64(len(b)) > limit {
		h.fail("the answer from %s is over %d MB", u.Host, max(limit>>20, 1))
	}
	out := h.vm.NewObject()
	hd := h.vm.NewObject()
	for k, v := range resp.Header {
		if len(v) > 0 {
			_ = hd.Set(strings.ToLower(k), v[0])
		}
	}
	_ = out.Set("status", resp.StatusCode)
	_ = out.Set("headers", hd)
	_ = out.Set("text", strings.ToValidUTF8(string(b), "�"))
	return out
}
