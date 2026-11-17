package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// The Sources and updates plugin (plugins/sources-updates), run by the
// real host against feeds served on this PC.

func sourcesPlugin(t *testing.T, svc ScriptServices) (*ScriptHost, string) {
	t.Helper()
	data := t.TempDir()
	src, err := filepath.Abs(filepath.Join("..", "..", "plugins", "sources-updates"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := InstallPlugin(data, src); err != nil {
		t.Fatal(err)
	}
	if err := GrantPlugin(data, "sources-updates"); err != nil {
		t.Fatal(err)
	}
	list := ActivePlugins(data)
	if len(list) != 1 {
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

// pluginTest calls the plugin's "test" handler and decodes its answer.
func pluginTest(t *testing.T, h *ScriptHost, arg map[string]any, dst any) {
	t.Helper()
	out, err := h.Call("test", arg)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(out, dst); err != nil {
		t.Fatalf("%s: %v", out, err)
	}
}

type feedEntry struct {
	T string   `json:"t"`
	U []string `json:"u"`
	D int64    `json:"d"`
	S int64    `json:"s"`
	A int      `json:"a"`
	V string   `json:"v"`
	H string   `json:"h"`
	C string   `json:"c"`
}

type parsedFeed struct {
	Error   string      `json:"error"`
	Name    string      `json:"name"`
	Format  string      `json:"format"`
	Entries []feedEntry `json:"entries"`
	Skipped int         `json:"skipped"`
	HTTP    int         `json:"http"`
}

const testMagnet = "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567&dn=Starfall"

func TestSourcesPluginFeedValidation(t *testing.T) {
	h, _ := sourcesPlugin(t, ScriptServices{})
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC).UnixMilli()
	parse := func(text string) parsedFeed {
		var f parsedFeed
		pluginTest(t, h, map[string]any{"what": "feed", "text": text, "now": now}, &f)
		return f
	}

	good := parse(`{"name": "  My\u0000 feed ", "junk": {"run": "calc.exe"}, "downloads": [
		{"title": "Starfall Tactics v1.2.3 + DLC", "uris": ["` + testMagnet + `", "https://files.example.com/starfall.zip"], "uploadDate": "2026-10-01T12:00:00.000Z", "fileSize": "45.2 GB", "steamAppId": 5, "__proto__": {"x": 1}},
		{"title": "Plain http", "uris": ["http://files.example.com/a.zip"], "uploadDate": "not a date", "fileSize": 1048576},
		{"title": "Bad addresses only", "uris": ["javascript:alert(1)", "file:///c:/x", "ftp://a.example.com/x", "https://user:pw@example.com/x", "magnet:?dn=nohash", "https://exa mple.com/", 7]},
		{"title": "", "uris": ["https://files.example.com/x.zip"]},
		{"title": "No list", "uris": "https://files.example.com/x.zip"},
		{"title": 12, "uris": ["https://files.example.com/x.zip"]},
		"a string", null, 5, []
	]}`)
	if good.Error != "" || good.Format != "hydra" || good.Name != "My feed" || len(good.Entries) != 2 || good.Skipped != 8 || good.HTTP != 1 {
		t.Fatalf("feed: %+v", good)
	}
	e := good.Entries[0]
	if e.T != "Starfall Tactics v1.2.3 + DLC" || len(e.U) != 2 || e.D != time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC).UnixMilli() || e.S != 48533130445 || e.A != 0 {
		t.Errorf("first entry: %+v", e)
	}
	if e := good.Entries[1]; e.D != 0 || e.S != 1048576 {
		t.Errorf("second entry: %+v", e)
	}

	rich := parse(`{"lootnest": 1, "name": "Mine", "downloads": [{"title": "Starfall Tactics", "uris": ["https://files.example.com/s.zip"], "steamAppId": 4242, "version": " 1.2.3 ",
		"sha256": "AB` + strings.Repeat("0", 62) + `", "changelog": "Fixes\nmore"},
		{"title": "Odd extras", "uris": ["https://files.example.com/o.zip"], "steamAppId": "4242", "version": 3, "sha256": "short", "uploadDate": "2090-01-01T00:00:00Z"}]}`)
	if rich.Format != "lootnest" || len(rich.Entries) != 2 {
		t.Fatalf("rich: %+v", rich)
	}
	if e := rich.Entries[0]; e.A != 4242 || e.V != "1.2.3" || e.H != "ab"+strings.Repeat("0", 62) || e.C != "Fixes more" {
		t.Errorf("rich entry: %+v", e)
	}
	if e := rich.Entries[1]; e.A != 0 || e.V != "" || e.H != "" || e.D != 0 {
		t.Errorf("rich entry with bad extras: %+v", e)
	}

	// Long and odd text is cut and cleaned, never refused whole.
	long := parse(fmt.Sprintf(`{"downloads": [{"title": %q, "uris": [%q, %q]}]}`, strings.Repeat("a", 500)+"\u202etxt.exe", "https://files.example.com/"+strings.Repeat("x", 5000), "https://files.example.com/ok"))
	if len(long.Entries) != 1 || len(long.Entries[0].T) != 300 || len(long.Entries[0].U) != 1 {
		t.Errorf("long: %+v", long)
	}
	many := make([]string, 30)
	for i := range many {
		many[i] = fmt.Sprintf(`"https://files.example.com/%d"`, i)
	}
	if f := parse(`{"downloads": [{"title": "Many", "uris": [` + strings.Join(many, ",") + `]}]}`); len(f.Entries) != 1 || len(f.Entries[0].U) != 20 {
		t.Errorf("many addresses: %+v", f)
	}

	for text, want := range map[string]string{
		`not json`:                         "isn't JSON",
		`[1, 2]`:                           "isn't a list of downloads",
		`"downloads"`:                      "isn't a list of downloads",
		`{"name": "x"}`:                    "format this plugin reads",
		`{"downloads": {}}`:                "format this plugin reads",
		`{"lootnest": 1, "downloads": 5}`:  "no list of downloads",
		`{"downloads": []}`:                "lists no downloads",
		`{"downloads": [{"title": "x"}]}`:  "none of its downloads could be read",
		`{"downloads": [1, 2, {"a": 1}]} `: "none of its downloads could be read",
	} {
		if f := parse(text); !strings.Contains(f.Error, want) {
			t.Errorf("%s: %+v, want %q", text, f, want)
		}
	}

	var size int64
	for in, want := range map[string]int64{"45.2 GB": 48533130445, "700 MB": 734003200, "1,5 GB": 1610612736, "12 kb": 12288, "1 TB": 1099511627776, "big": 0, "-5 GB": 0, "5 GB extra": 0, "": 0} {
		if pluginTest(t, h, map[string]any{"what": "size", "size": in}, &size); size != want {
			t.Errorf("size %q: %d, want %d", in, size, want)
		}
	}
	var hash string
	if pluginTest(t, h, map[string]any{"what": "hash", "uri": strings.ToUpper(testMagnet[:60])}, &hash); hash != "0123456789abcdef0123456789abcdef01234567" {
		t.Errorf("info hash: %q", hash)
	}
}

type pluginRow struct {
	ID      string         `json:"id"`
	Title   string         `json:"title"`
	Sub     string         `json:"sub"`
	Lamp    string         `json:"lamp"`
	Tags    []string       `json:"tags"`
	Warn    string         `json:"warn"`
	Error   string         `json:"error"`
	Actions []ScriptAction `json:"actions"`
}

type pluginView struct {
	Sections []struct {
		Title   string `json:"title"`
		Empty   string `json:"empty"`
		Actions []struct {
			ID      string         `json:"id"`
			Form    string         `json:"form"`
			Confirm map[string]any `json:"confirm"`
		} `json:"actions"`
		Rows []pluginRow `json:"rows"`
	} `json:"sections"`
	Forms map[string]any `json:"forms"`
}

type actResult struct {
	Error string `json:"error"`
	Toast string `json:"toast"`
	Form  string `json:"form"`
}

func TestSourcesPluginSources(t *testing.T) {
	var mu sync.Mutex
	feed := `{"name": "Fixture feed", "downloads": [{"title": "Starfall Tactics v1.2.3", "uris": ["` + testMagnet + `"], "uploadDate": "2026-10-01T12:00:00.000Z", "fileSize": "1.5 GB"}]}`
	var conditional, hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		hits++
		switch r.URL.Path {
		case "/feed.json":
			if r.Header.Get("If-None-Match") == `"f1"` {
				conditional++
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.Header().Set("ETag", `"f1"`)
			fmt.Fprint(w, feed)
		case "/broken.json":
			fmt.Fprint(w, `{"downloads": "nope"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base := strings.Replace(srv.URL, "127.0.0.1", "127.0.0.1.nip.invalid", 1) // a name, as the plugin wants one
	_ = base

	changed := make(chan struct{}, 64)
	h, data := sourcesPlugin(t, ScriptServices{Changed: func() { changed <- struct{}{} }, Log: func(s string) { t.Log(s) }})
	view := func() pluginView {
		var v pluginView
		out, err := h.Call("view", map[string]any{"page": "sources"})
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(out, &v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	act := func(arg map[string]any) actResult {
		var r actResult
		out, err := h.Call("action", arg)
		if err != nil {
			t.Fatal(err)
		}
		_ = json.Unmarshal(out, &r)
		return r
	}
	wait := func() {
		t.Helper()
		select {
		case <-changed:
		case <-time.After(10 * time.Second):
			t.Fatal("the plugin never said its page changed")
		}
	}

	// Empty at first, and adding is behind the notice, shown once.
	v := view()
	if len(v.Sections) != 2 || len(v.Sections[0].Rows) != 0 || !strings.Contains(v.Sections[0].Empty, "comes with none") {
		t.Fatalf("first view: %+v", v)
	}
	if a := v.Sections[0].Actions[0]; a.ID != "notice" || a.Confirm == nil || a.Form != "" {
		t.Fatalf("add before the notice: %+v", a)
	}
	if r := act(map[string]any{"id": "add", "values": map[string]any{"url": srv.URL + "/feed.json"}}); r.Error == "" {
		t.Fatal("added before the notice")
	}
	if r := act(map[string]any{"id": "notice"}); r.Form != "add" {
		t.Fatalf("notice: %+v", r)
	}
	if a := view().Sections[0].Actions[0]; a.ID != "add" || a.Form != "add" || a.Confirm != nil {
		t.Fatalf("add after the notice: %+v", a)
	}

	for u, want := range map[string]string{
		"":                                   "starting with https://",
		"feed.json":                          "starting with https://",
		"ftp://files.example.com/f.json":     "starting with https://",
		testMagnet:                           "starting with https://",
		"https://user:pw@example.com/f.json": "starting with https://",
		"javascript:alert(1)":                "starting with https://",
	} {
		if r := act(map[string]any{"id": "add", "values": map[string]any{"url": u}}); !strings.Contains(r.Error, want) {
			t.Errorf("add %q: %+v", u, r)
		}
	}

	// A good source is read at once and its copy kept.
	if r := act(map[string]any{"id": "add", "values": map[string]any{"url": srv.URL + "/feed.json", "name": ""}}); r.Error != "" {
		t.Fatalf("add: %+v", r)
	}
	if r := act(map[string]any{"id": "add", "values": map[string]any{"url": srv.URL + "/feed.json"}}); !strings.Contains(r.Error, "already") {
		t.Errorf("twice: %+v", r)
	}
	wait()
	row := view().Sections[0].Rows[0]
	if row.Title != "Fixture feed" || row.Error != "" || len(row.Tags) != 2 || row.Tags[0] != "1 download" || row.Tags[1] != "Hydra-compatible" || !strings.Contains(row.Warn, "isn't https") {
		t.Fatalf("row: %+v", row)
	}
	if !isFile(filepath.Join(PluginDataDir(data, "sources-updates"), "feed-"+row.ID+".json")) {
		t.Fatal("the feed's copy wasn't kept")
	}

	// Refresh sends the ETag; an unchanged feed isn't read again.
	act(map[string]any{"id": "refresh", "row": row.ID})
	wait()
	mu.Lock()
	if conditional != 1 {
		t.Errorf("conditional requests: %d of %d", conditional, hits)
	}
	// A feed that breaks keeps the last good copy and says why.
	feed = `{"downloads": []}`
	mu.Unlock()
	act(map[string]any{"id": "edit", "row": row.ID, "values": map[string]any{"url": srv.URL + "/feed.json", "name": "Renamed"}})
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"downloads": []}`) })
	act(map[string]any{"id": "refresh", "row": row.ID})
	wait()
	row = view().Sections[0].Rows[0]
	if row.Title != "Renamed" || !strings.Contains(row.Error, "lists no downloads") || row.Lamp != "bad" || row.Tags[0] != "1 download" {
		t.Fatalf("after a broken refresh: %+v", row)
	}
	if !isFile(filepath.Join(PluginDataDir(data, "sources-updates"), "feed-"+row.ID+".json")) {
		t.Fatal("a broken refresh dropped the last good copy")
	}

	// Off, then removed: its copy goes with it.
	act(map[string]any{"id": "toggle", "row": row.ID, "on": false})
	if r := act(map[string]any{"id": "refresh", "row": row.ID}); r.Error == "" {
		t.Error("refreshed a source that's off")
	}
	if row = view().Sections[0].Rows[0]; row.Lamp != "off" {
		t.Errorf("off: %+v", row)
	}
	if r := act(map[string]any{"id": "remove", "row": row.ID}); !strings.Contains(r.Toast, "Removed") {
		t.Errorf("remove: %+v", r)
	}
	if len(view().Sections[0].Rows) != 0 || isFile(filepath.Join(PluginDataDir(data, "sources-updates"), "feed-"+row.ID+".json")) {
		t.Error("a removed source left something")
	}
	if r := act(map[string]any{"id": "nope"}); r.Error == "" {
		t.Error("an unknown action did something")
	}
}
