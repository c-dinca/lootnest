package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// scriptManifest is a script plugin's plugin.json.
func scriptManifest(id, version string, perms []string, secrets ...PluginSecret) string {
	b, _ := json.Marshal(PluginManifest{ID: id, Name: "Night " + id, Version: version, Provides: []string{"script"},
		Script: "main.js", Permissions: perms, Secrets: secrets, Pages: []PluginPage{{ID: "main", Title: "Main"}}})
	return string(b)
}

// startScript installs a script plugin, allows it and starts it.
func startScript(t *testing.T, perms []string, js string, svc ScriptServices, secrets ...PluginSecret) (*ScriptHost, string) {
	t.Helper()
	data := t.TempDir()
	if _, err := InstallPlugin(data, pluginZip(t, map[string]string{"plugin.json": scriptManifest("tester", "1.0.0", perms, secrets...), "main.js": js})); err != nil {
		t.Fatal(err)
	}
	if err := GrantPlugin(data, "tester"); err != nil {
		t.Fatal(err)
	}
	list := ActivePlugins(data)
	if len(list) != 1 || !list[0].Granted {
		t.Fatalf("not active: %+v", ListPlugins(data))
	}
	svc.AllowLocal = true
	h, err := StartScript(list[0], data, svc)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)
	return h, data
}

func callStr(t *testing.T, h *ScriptHost, name string, arg any) string {
	t.Helper()
	out, err := h.Call(name, arg)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return string(out)
}

func TestScriptPluginInstall(t *testing.T) {
	data := t.TempDir()
	js := `lootnest.on("start", () => 1)`
	for _, c := range []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"a script in a data plugin", map[string]string{"plugin.json": manifest("has-js", "1.0.0", "theme"), "theme.json": goodTheme, "main.js": js}, "isn't data (main.js)"},
		{"a second script", map[string]string{"plugin.json": scriptManifest("two-js", "1.0.0", nil), "main.js": js, "lib/extra.js": js}, "isn't data (lib/extra.js)"},
		{"no script file", map[string]string{"plugin.json": scriptManifest("no-js", "1.0.0", nil)}, "isn't in it"},
		{"a script that doesn't parse", map[string]string{"plugin.json": scriptManifest("bad-js", "1.0.0", nil), "main.js": "function ("}, "can't be read"},
		{"an unknown permission", map[string]string{"plugin.json": scriptManifest("bad-perm", "1.0.0", []string{"files"}), "main.js": js}, `asks for "files"`},
		{"a key without net", map[string]string{"plugin.json": scriptManifest("key-no-net", "1.0.0", nil, PluginSecret{Name: "k", Host: "api.example.com", Label: "Key"}), "main.js": js}, "doesn't ask to go online"},
		{"a key for no site", map[string]string{"plugin.json": scriptManifest("key-no-host", "1.0.0", []string{"net"}, PluginSecret{Name: "k", Host: "localhost", Label: "Key"}), "main.js": js}, "one site"},
		{"permissions in a data plugin", map[string]string{"plugin.json": `{"id":"sneaky","name":"Sneaky","version":"1.0.0","provides":["theme"],"permissions":["net"]}`, "theme.json": goodTheme}, `doesn't provide "script"`},
		{"an exe", map[string]string{"plugin.json": scriptManifest("has-exe", "1.0.0", nil), "main.js": js, "tool.exe": "MZ"}, "isn't data (tool.exe)"},
	} {
		if _, err := InstallPlugin(data, pluginZip(t, c.files)); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
	if ents, _ := os.ReadDir(PluginsDir(data)); len(ents) != 0 {
		t.Fatalf("refused plugins left files: %v", ents)
	}

	// A script plugin is off until what it asks is allowed.
	zipOf := func(version string, perms []string) string {
		return pluginZip(t, map[string]string{"plugin.json": scriptManifest("asker", version, perms), "main.js": js})
	}
	if _, err := InstallPlugin(data, zipOf("1.0.0", []string{"net"})); err != nil {
		t.Fatal(err)
	}
	p := ListPlugins(data)[0]
	if p.On || p.Granted || len(p.Asks) != 2 || !strings.Contains(p.Asks[1], "online") {
		t.Fatalf("fresh install: %+v", p)
	}
	if err := SetPluginOn(data, "asker", true); err != nil {
		t.Fatal(err)
	}
	if p = ListPlugins(data)[0]; p.On || len(ActivePlugins(data)) != 0 {
		t.Fatalf("on without being allowed: %+v", p)
	}
	if err := GrantPlugin(data, "asker"); err != nil {
		t.Fatal(err)
	}
	if p = ListPlugins(data)[0]; !p.On || !p.Granted {
		t.Fatalf("allowed: %+v", p)
	}
	// The same asks in a new version stay allowed; more asks don't.
	if _, err := InstallPlugin(data, zipOf("1.1.0", []string{"net"})); err != nil {
		t.Fatal(err)
	}
	if p = ListPlugins(data)[0]; !p.On || !p.Granted || p.Version != "1.1.0" {
		t.Fatalf("same asks: %+v", p)
	}
	if _, err := InstallPlugin(data, zipOf("1.2.0", []string{"net", "updates"})); err != nil {
		t.Fatal(err)
	}
	if p = ListPlugins(data)[0]; p.On || p.Granted || len(p.Asks) != 3 {
		t.Fatalf("more asks: %+v", p)
	}
	if err := RemovePlugin(data, "asker"); err != nil {
		t.Fatal(err)
	}
	if st := loadPlugins(data); len(st.Granted) != 0 {
		t.Fatalf("grant kept after remove: %+v", st)
	}
}

func TestScriptCalls(t *testing.T) {
	var logs []string
	h, data := startScript(t, nil, `
		lootnest.on("echo", (a) => ({ got: a, id: lootnest.plugin.id }));
		lootnest.on("store", (a) => { lootnest.storage.set("box", a); return lootnest.storage.get("box"); });
		lootnest.on("read", () => lootnest.storage.get("box"));
		lootnest.on("drop", () => { lootnest.storage.remove("box"); return lootnest.storage.get("box"); });
		lootnest.on("bad-key", () => lootnest.storage.set("../up", 1));
		lootnest.on("throw", () => { throw new Error("no luck") });
		lootnest.on("spin", () => { for (;;) {} });
		lootnest.on("later", async (a) => { await null; return a + 1; });
		lootnest.on("net", () => lootnest.net.fetch("https://example.com/"));
		lootnest.on("games", () => lootnest.library.games());
		lootnest.on("say", () => lootnest.log("hello", 2));
		lootnest.on("escape", () => [typeof require, typeof process, typeof fetch, typeof XMLHttpRequest, typeof WebAssembly, typeof globalThis.lootnest.net.fetch].join());
	`, ScriptServices{Log: func(s string) { logs = append(logs, s) }, CallLimit: 300 * time.Millisecond})

	if got := callStr(t, h, "echo", map[string]any{"a": []int{1, 2}}); got != `{"got":{"a":[1,2]},"id":"tester"}` {
		t.Errorf("echo: %s", got)
	}
	if got := callStr(t, h, "store", map[string]any{"n": 1}); got != `{"n":1}` {
		t.Errorf("store: %s", got)
	}
	if b, err := os.ReadFile(filepath.Join(PluginDataDir(data, "tester"), "box.json")); err != nil || string(b) != `{"n":1}` {
		t.Errorf("stored file: %s %v", b, err)
	}
	if got := callStr(t, h, "read", nil); got != `{"n":1}` {
		t.Errorf("read: %s", got)
	}
	if got := callStr(t, h, "drop", nil); got != "null" {
		t.Errorf("drop: %s", got)
	}
	if got := callStr(t, h, "later", 41); got != "42" {
		t.Errorf("an async handler: %s", got)
	}
	if got := callStr(t, h, "escape", nil); got != `"undefined,undefined,undefined,undefined,undefined,function"` {
		t.Errorf("what a script can reach: %s", got)
	}
	callStr(t, h, "say", nil)
	if len(logs) != 1 || logs[0] != "hello 2" {
		t.Errorf("log: %q", logs)
	}
	for name, want := range map[string]string{
		"bad-key": "storage key",
		"throw":   "no luck",
		"spin":    "took too long",
		"net":     `didn't ask for "net"`,
		"games":   `didn't ask for "library"`,
		"nothing": "no such handler",
	} {
		if _, err := h.Call(name, nil); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v, want %q", name, err, want)
		}
	}
	// A script stopped for taking too long still answers afterwards.
	if got := callStr(t, h, "echo", 1); !strings.Contains(got, `"got":1`) {
		t.Errorf("after a stop: %s", got)
	}
	h.Close()
	if _, err := h.Call("echo", 1); err == nil {
		t.Error("a closed script answered")
	}
}

func TestScriptSchedule(t *testing.T) {
	var mu sync.Mutex
	var said []string
	h, _ := startScript(t, []string{"notify"}, `
		let n = 0;
		lootnest.on("tick", (a) => { n++; lootnest.notify(a.word + n); if (n < 3) lootnest.schedule("tick", 20, a); });
		lootnest.on("never", () => lootnest.notify("never"));
		lootnest.on("go", () => { lootnest.schedule("tick", 20, { word: "t" }); lootnest.schedule("never", 30); lootnest.schedule("never", -1); });
	`, ScriptServices{Notify: func(s string) { mu.Lock(); said = append(said, s); mu.Unlock() }})
	callStr(t, h, "go", nil)
	deadline := time.Now().Add(5 * time.Second)
	for {
		mu.Lock()
		got := strings.Join(said, ",")
		mu.Unlock()
		if got == "t1,t2,t3" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("scheduled calls: %q", got)
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(80 * time.Millisecond)
	if len(said) != 3 {
		t.Fatalf("a cancelled call ran: %q", said)
	}
}

func TestScriptFetch(t *testing.T) {
	var auth []string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = append(auth, r.URL.Path+" "+r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/big":
			w.Write(make([]byte, 3<<20))
		case "/slow":
			time.Sleep(1500 * time.Millisecond)
		case "/etag":
			if r.Header.Get("If-None-Match") == `"v1"` {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.Header().Set("ETag", `"v1"`)
			fmt.Fprint(w, "fresh")
		case "/form":
			r.ParseForm()
			fmt.Fprint(w, r.Method+" "+r.Form.Get("link"))
		default:
			fmt.Fprint(w, "ok")
		}
	}))
	defer srv.Close()
	host := strings.TrimPrefix(srv.URL, "https://")
	js := `
		lootnest.on("get", (a) => { const r = lootnest.net.fetch(a.url, a.opts); return { status: r.status, text: r.text, etag: r.headers.etag || "" }; });
		lootnest.on("has", () => lootnest.secrets.has("svc"));
	`
	secret := ""
	h, _ := startScript(t, []string{"net"}, js, ScriptServices{Secret: func(n string) string { return secret }},
		PluginSecret{Name: "svc", Host: "api.example.com", Label: "Service key"})
	h.client = srv.Client()
	h.client.CheckRedirect = nil
	get := func(path string, opts map[string]any) (string, error) {
		out, err := h.Call("get", map[string]any{"url": srv.URL + path, "opts": opts})
		return string(out), err
	}
	if got, err := get("/", nil); err != nil || got != `{"status":200,"text":"ok","etag":""}` {
		t.Errorf("get: %s %v", got, err)
	}
	if got, err := get("/etag", nil); err != nil || !strings.Contains(got, `"etag":"\"v1\""`) {
		t.Errorf("etag: %s %v", got, err)
	}
	if got, err := get("/etag", map[string]any{"headers": map[string]string{"If-None-Match": `"v1"`}}); err != nil || !strings.Contains(got, `"status":304`) {
		t.Errorf("not modified: %s %v", got, err)
	}
	if got, err := get("/form", map[string]any{"method": "post", "form": map[string]string{"link": "m&x=1"}}); err != nil || !strings.Contains(got, `POST m\u0026x=1`) && !strings.Contains(got, "POST m&x=1") {
		t.Errorf("form: %s %v", got, err)
	}
	for name, c := range map[string]struct {
		path string
		opts map[string]any
		want string
	}{
		"too big":       {"/big", map[string]any{"maxBytes": 1 << 20}, "is over 1 MB"},
		"too slow":      {"/slow", map[string]any{"timeout": 1}, "took too long"},
		"a cookie":      {"/", map[string]any{"headers": map[string]string{"Cookie": "a=b"}}, "can't be set"},
		"a method":      {"/", map[string]any{"method": "CONNECT"}, "isn't allowed"},
		"another's key": {"/", map[string]any{"secret": "other"}, "doesn't list the key"},
		"a key astray":  {"/", map[string]any{"secret": "svc"}, "only goes to https://api.example.com"},
	} {
		if got, err := get(c.path, c.opts); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %s %v, want %q", name, got, err, c.want)
		}
	}
	for _, u := range []string{"file:///c:/windows/win.ini", "ftp://example.com/x", "https://user:pw@example.com/", "javascript:alert(1)", ""} {
		if _, err := h.Call("get", map[string]any{"url": u}); err == nil || !strings.Contains(err.Error(), "isn't http or https") {
			t.Errorf("%q: %v", u, err)
		}
	}
	for _, a := range auth {
		if !strings.HasSuffix(a, " ") {
			t.Errorf("a key was sent: %q", a)
		}
	}

	// The key goes to its own site, and the script can only ask whether
	// it's set.
	if got := callStr(t, h, "has", nil); got != "false" {
		t.Errorf("has before: %s", got)
	}
	secret = "tok-123"
	if got := callStr(t, h, "has", nil); got != "true" {
		t.Errorf("has after: %s", got)
	}
	h.p.Secrets[0].Host = strings.Split(host, ":")[0]
	auth = nil
	if got, err := get("/", map[string]any{"secret": "svc"}); err != nil || len(auth) != 1 || auth[0] != "/ Bearer tok-123" {
		t.Errorf("with the key: %s %v %q", got, err, auth)
	}
	if _, err := get("/", map[string]any{"secret": "svc", "headers": map[string]string{"Authorization": "x"}}); err == nil {
		t.Error("the script set Authorization beside a key")
	}
}

// Without AllowLocal, a script can't reach this PC or its network.
func TestScriptFetchStaysOffThisPC(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("reached this PC") }))
	defer srv.Close()
	h, _ := startScript(t, []string{"net"}, `lootnest.on("get", (u) => lootnest.net.fetch(u).status)`, ScriptServices{})
	h.client.Transport.(*http.Transport).DialContext = guardedDial(false)
	for _, u := range []string{srv.URL, "http://localhost:47990/", "http://192.168.1.1/", "http://10.0.0.5/", "http://169.254.169.254/", "http://0.0.0.0/"} {
		if _, err := h.Call("get", u); err == nil || !strings.Contains(err.Error(), "can't reach this PC") {
			t.Errorf("%s: %v", u, err)
		}
	}
}

func TestScriptOffers(t *testing.T) {
	var got []ScriptOffer
	h, _ := startScript(t, []string{"updates", "library"}, `
		lootnest.on("offer", (list) => lootnest.updates.offer(list));
		lootnest.on("games", () => lootnest.library.games().map((g) => g.name + ":" + (g.folder || "")).join());
	`, ScriptServices{
		Offers: func(l []ScriptOffer) { got = l },
		Games: func() []ScriptGame {
			return []ScriptGame{{Key: "dir:c:\\games\\a", Name: "A"}, {Key: "dir:c:\\games\\b", Name: "B", Folder: "B v2"}}
		},
	})
	long := strings.Repeat("x", 400)
	callStr(t, h, "offer", []map[string]any{{"id": "s1:abc", "game": "dir:c:\\games\\a", "title": "A  v2\n" + long, "size": 12, "extra": "ignored",
		"actions": []map[string]string{{"id": "update", "label": "Update"}}}})
	if len(got) != 1 || got[0].ID != "s1:abc" || len([]rune(got[0].Title)) != 300 || strings.Contains(got[0].Title, "\n") || got[0].Size != 12 || got[0].Actions[0].Label != "Update" {
		t.Fatalf("offers: %+v", got)
	}
	if _, err := h.Call("offer", []map[string]any{{"id": "has space", "game": "x", "title": "t"}}); err == nil {
		t.Error("an offer with a bad id was taken")
	}
	if s := callStr(t, h, "games", nil); s != `"A:,B:B v2"` {
		t.Errorf("games: %s", s)
	}
}
