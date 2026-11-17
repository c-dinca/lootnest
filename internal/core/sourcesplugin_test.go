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

// seed is what the plugin already keeps when it starts: storage key ->
// JSON.
func sourcesPlugin(t *testing.T, svc ScriptServices, seed ...map[string]string) (*ScriptHost, string) {
	t.Helper()
	data := t.TempDir()
	for _, m := range seed {
		for k, v := range m {
			write(t, filepath.Join(PluginDataDir(data, "sources-updates"), k+".json"), v)
		}
	}
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
	// waitFor waits until the plugin's page, drawn again each time it says
	// it changed, shows what's expected.
	waitFor := func(what string, ok func() bool) {
		t.Helper()
		deadline := time.After(10 * time.Second)
		for !ok() {
			select {
			case <-changed:
			case <-time.After(50 * time.Millisecond):
			case <-deadline:
				t.Fatalf("never happened: %s", what)
			}
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
	waitFor("the first read", func() bool { r := view().Sections[0].Rows; return len(r) == 1 && len(r[0].Tags) == 2 })
	row := view().Sections[0].Rows[0]
	if row.Title != "Fixture feed" || row.Error != "" || len(row.Tags) != 2 || row.Tags[0] != "1 download" || row.Tags[1] != "Hydra-compatible" || !strings.Contains(row.Warn, "isn't https") {
		t.Fatalf("row: %+v", row)
	}
	if !isFile(filepath.Join(PluginDataDir(data, "sources-updates"), "feed-"+row.ID+".json")) {
		t.Fatal("the feed's copy wasn't kept")
	}

	// Refresh sends the ETag; an unchanged feed isn't read again.
	act(map[string]any{"id": "refresh", "row": row.ID})
	waitFor("a conditional request", func() bool { mu.Lock(); defer mu.Unlock(); return conditional == 1 })
	mu.Lock()
	if hits != 2 {
		t.Errorf("requests: %d, %d conditional", hits, conditional)
	}
	// A feed that breaks keeps the last good copy and says why.
	feed = `{"downloads": []}`
	mu.Unlock()
	act(map[string]any{"id": "edit", "row": row.ID, "values": map[string]any{"url": srv.URL + "/feed.json", "name": "Renamed"}})
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"downloads": []}`) })
	act(map[string]any{"id": "refresh", "row": row.ID})
	waitFor("the broken refresh", func() bool { return view().Sections[0].Rows[0].Error != "" })
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

func TestSourcesPluginTitles(t *testing.T) {
	h, _ := sourcesPlugin(t, ScriptServices{})
	for title, want := range map[string]string{
		"Starfall Tactics": "starfall tactics",
		"STARFALL TACTICS II: Gold Edition v1.2.3 + 5 DLCs [MULTi12]":  "starfall tactics 2",
		"Starfall Tactics 2 (v1.0.4 + Bonus OST, MULTi8) [Some Group]": "starfall tactics 2",
		"The Starfall Tactics - Deluxe Edition Build 12345":            "starfall tactics",
		"Starfall.Tactics.v1.2.0":                                      "starfall tactics",
		"Starfall Tactics: Game of the Year Edition":                   "starfall tactics",
		"Night's Edge & Dawn":                                          "nights edge dawn",
		"Caf\u00e9 Ren\u00e9e":                                         "cafe renee",
		"Legend IV 1.0.2.1":                                            "legend 4",
		"Moonbase Update v1.0 to v1.1":                                 "moonbase",
		"Moonbase 2026-10-01":                                          "moonbase",
		"Echo Remastered":                                              "echo remastered",
		"":                                                             "",
	} {
		var got []string
		if pluginTest(t, h, map[string]any{"what": "norm", "title": title}, &got); strings.Join(got, " ") != want {
			t.Errorf("norm %q: %q, want %q", title, strings.Join(got, " "), want)
		}
	}
	for _, c := range []struct {
		a, b     string
		min, max float64
	}{
		{"Starfall Tactics", "Starfall Tactics v1.2.3 + DLC", 1, 1},
		{"Starfall Tactics II", "starfall tactics 2 Deluxe Edition", 1, 1},
		{"Starfall Tactics 2", "Starfall Tactics 3", 0, 0.3},
		{"Starfall Tactics", "Starfall Tactics 2", 0, 0.3},
		{"Starfall Tactics Online Arena", "Starfall Tactics Arena", 0.6, 0.89},
		{"Starfall", "Starfall Tactics", 0, 0.59},
		{"Starfall Tactics", "Moonbase", 0, 0},
		{"", "Moonbase", 0, 0},
	} {
		var got float64
		if pluginTest(t, h, map[string]any{"what": "score", "a": c.a, "b": c.b}, &got); got < c.min || got > c.max {
			t.Errorf("score %q against %q: %.2f, want %.2f to %.2f", c.a, c.b, got, c.min, c.max)
		}
	}
}

type titleVer struct {
	Sem   string `json:"sem"`
	From  string `json:"from"`
	Build int    `json:"build"`
	Date  int64  `json:"date"`
}

func TestSourcesPluginVersions(t *testing.T) {
	h, _ := sourcesPlugin(t, ScriptServices{})
	day := func(y, m, d int) int64 { return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC).UnixMilli() }
	for title, want := range map[string]titleVer{
		"Starfall Tactics v1.2.3 + DLC":       {Sem: "1.2.3"},
		"Starfall Tactics V 1.2":              {Sem: "1.2"},
		"Starfall Tactics (v1.0.4b, MULTi8)":  {Sem: "1.0.4b"},
		"Starfall Tactics 1.2.3.4":            {Sem: "1.2.3.4"},
		"Starfall.Tactics.v1.2.0-GROUP":       {Sem: "1.2.0"},
		"Starfall Tactics Build 16251443":     {Build: 16251443},
		"Starfall Tactics v1.4 Build 9001":    {Sem: "1.4", Build: 9001},
		"Starfall Tactics b1234567":           {Build: 1234567},
		"Starfall Tactics 2026-10-01":         {Date: day(2026, 10, 1)},
		"Starfall Tactics (01.10.2026)":       {Date: day(2026, 10, 1)},
		"Starfall Tactics Update v1.0 to 1.1": {Sem: "1.1", From: "1.0"},
		"Starfall Tactics Update v1.1 -> 1.0": {Sem: "1.1"}, // not a step forward, so not a patch
		"Starfall Tactics 2":                  {},
		"Starfall Tactics 2.5":                {},
		"Area 51":                             {},
		"Vault 13 build":                      {},
		"":                                    {},
	} {
		var got titleVer
		if pluginTest(t, h, map[string]any{"what": "version", "title": title}, &got); got != want {
			t.Errorf("version %q: %+v, want %+v", title, got, want)
		}
	}
	one, zero, minus := 1, 0, -1
	for _, c := range []struct {
		a, b string
		want *int
	}{
		{"G v1.0.10", "G v1.0.9", &one},
		{"G v1.2", "G v1.2.0", &zero},
		{"G v1.2a", "G v1.2b", &minus},
		{"G v2", "G v1.9", nil}, // "v2" alone isn't read as a version
		{"G Build 200", "G Build 1000", &minus},
		{"G v1.1 Build 5000", "G v1.2 Build 4000", &minus}, // version numbers before builds
		{"G 2026-10-02", "G 2026-10-01", &one},
		{"G v1.0", "G Build 500", nil},
		{"G v1.0", "G", nil},
	} {
		var got *int
		pluginTest(t, h, map[string]any{"what": "compare", "a": c.a, "b": c.b}, &got)
		if (got == nil) != (c.want == nil) || (got != nil && *got != *c.want) {
			t.Errorf("compare %q with %q: %v, want %v", c.a, c.b, got, c.want)
		}
	}

	// What's installed: this PC's own reading before the stored baseline, and
	// the exe's version only when it continues the baseline's numbering.
	type inst struct {
		V    titleVer `json:"v"`
		From string   `json:"from"`
	}
	base := map[string]any{"base": map[string]any{"title": "G v1.2.0", "date": 5, "from": "hydra"}}
	for name, c := range map[string]struct {
		game map[string]any
		link map[string]any
		want string
		from string
	}{
		"the baseline":            {map[string]any{}, base, "1.2.0", "hydra"},
		"the folder's name":       {map[string]any{"version": "1.3.0"}, base, "1.3.0", "pc"},
		"an exe that continues":   {map[string]any{"exeVersion": "1.4.1"}, base, "1.4.1", "exe"},
		"an engine's exe version": {map[string]any{"exeVersion": "2021.3.15"}, base, "1.2.0", "hydra"},
		"an older exe version":    {map[string]any{"exeVersion": "1.0.0"}, base, "1.2.0", "hydra"},
		"an exe with no baseline": {map[string]any{"exeVersion": "1.4.1"}, map[string]any{}, "", ""},
	} {
		var got inst
		if pluginTest(t, h, map[string]any{"what": "installed", "game": c.game, "link": c.link}, &got); got.V.Sem != c.want || got.From != c.from {
			t.Errorf("%s: %+v", name, got)
		}
	}
}

type detected struct {
	Offers []struct {
		ID, Game, Title, Version, Have string
	} `json:"offers"`
	Pending []struct {
		Game, Name, K, Title, Src string
	} `json:"pending"`
}

func TestSourcesPluginDetect(t *testing.T) {
	const hash = "0123456789abcdef0123456789abcdef01234567"
	entry := func(title, uri, date string) string {
		return fmt.Sprintf(`{"title": %q, "uris": [%q], "uploadDate": %q, "fileSize": "2 GB"}`, title, uri, date)
	}
	// The plugin first saw these games a while ago; an entry uploaded
	// yesterday is newer than that, one from January isn't.
	recent := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	longAgo := time.Now().Add(-60 * 24 * time.Hour).UnixMilli()
	feeds := map[string]string{
		"/a.json": `{"name": "Feed A", "downloads": [` + strings.Join([]string{
			entry("Night Tale v1.2.0", "magnet:?xt=urn:btih:"+hash, "2026-06-01T00:00:00Z"),
			entry("Night Tale v1.3.0 + DLC", "https://files.example.com/nt130.zip", "2026-09-01T00:00:00Z"),
			entry("Night Tale v1.3.1 + DLC", "https://files.example.com/nt131.zip", "2026-10-01T00:00:00Z"),
			entry("Night Tale 2 v1.0", "https://files.example.com/nt2.zip", "2026-10-05T00:00:00Z"),
			entry("Moonbase Build 500", "https://files.example.com/mb500.zip", "2026-10-01T00:00:00Z"),
			entry("Moonbase Build 400", "https://files.example.com/mb400.zip", "2026-08-01T00:00:00Z"),
			entry("Harbor Lights", "https://files.example.com/hl-old.zip", "2026-01-01T00:00:00Z"),
			entry("Harbor Lights", "https://files.example.com/hl-new.zip", recent),
			entry("Echo Valley Online Arena", "https://files.example.com/ev.zip", recent),
			entry("Quiet Road Update v1.0 to v1.1", "https://files.example.com/qr-patch.zip", "2026-10-01T00:00:00Z"),
			entry("Quiet Road Update v1.4 to v1.5", "https://files.example.com/qr-patch2.zip", "2026-10-02T00:00:00Z"),
			entry("Zeta Protocol Reloaded Collection", "https://files.example.com/zp.zip", recent),
			entry("Store Game v9.9", "https://files.example.com/sg.zip", recent),
		}, ",") + `]}`,
		"/b.json": `{"lootnest": 1, "name": "Feed B", "downloads": [
			{"title": "NT (the new build)", "uris": ["https://b.example.com/nt.zip"], "steamAppId": 777, "version": "1.3.0", "uploadDate": "2026-09-02T00:00:00Z"}]}`,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f, ok := feeds[r.URL.Path]; ok {
			fmt.Fprint(w, f)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	seen := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	games := []ScriptGame{
		{Key: "dir:c:\\games\\night tale", Name: "Night Tale", Source: "hydra", SteamID: 777, Folder: "Night Tale",
			Origin: &ScriptOrigin{URI: "magnet:?xt=urn:btih:" + strings.ToUpper(hash) + "&dn=x", Folder: "Night.Tale", Time: seen}},
		{Key: "dir:c:\\games\\moonbase", Name: "Moonbase", Source: "folder", Folder: "Moonbase", Version: "build 450"},
		{Key: "dir:c:\\games\\harbor", Name: "Harbor Lights", Source: "folder", Folder: "Harbor"},
		{Key: "dir:c:\\games\\echo", Name: "Echo Valley Arena", Source: "folder", Folder: "Echo"},
		{Key: "dir:c:\\games\\quiet", Name: "Quiet Road", Source: "folder", Folder: "Quiet Road v1.0", Version: "1.0"},
		{Key: "dir:c:\\games\\zeta", Name: "ZP", Source: "added", Folder: "zp"},
		{Key: "steam:42", Name: "Store Game", Source: "steam", Folder: "Store Game"},
	}
	var mu sync.Mutex
	var offers []ScriptOffer
	var said []string
	h, _ := sourcesPlugin(t, ScriptServices{
		Games:  func() []ScriptGame { return games },
		Offers: func(l []ScriptOffer) { mu.Lock(); offers = l; mu.Unlock() },
		Notify: func(s string) { mu.Lock(); said = append(said, s); mu.Unlock() },
	}, map[string]string{"links": fmt.Sprintf(`{"dir:c:\\games\\harbor": {"seen": %d}, "dir:c:\\games\\echo": {"seen": %d}, "dir:c:\\games\\zeta": {"seen": %d}}`, longAgo, longAgo, longAgo)})
	act := func(arg map[string]any) actResult {
		var r actResult
		out, err := h.Call("action", arg)
		if err != nil {
			t.Fatal(err)
		}
		_ = json.Unmarshal(out, &r)
		return r
	}
	act(map[string]any{"id": "notice"})
	for _, f := range []string{"/a.json", "/b.json"} {
		if r := act(map[string]any{"id": "add", "values": map[string]any{"url": srv.URL + f}}); r.Error != "" {
			t.Fatal(r.Error)
		}
	}
	// Both feeds are read before looking.
	deadline := time.Now().Add(10 * time.Second)
	for {
		var v pluginView
		out, _ := h.Call("view", map[string]any{"page": "sources"})
		_ = json.Unmarshal(out, &v)
		if rows := v.Sections[0].Rows; len(rows) == 2 && len(rows[0].Tags) == 2 && len(rows[1].Tags) == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the feeds were never read")
		}
		time.Sleep(30 * time.Millisecond)
	}
	detect := func() detected {
		var d detected
		pluginTest(t, h, map[string]any{"what": "detect"}, &d)
		return d
	}
	d := detect()
	byGame := map[string][]string{}
	for _, o := range d.Offers {
		byGame[o.Game] = append(byGame[o.Game], o.Title+" | "+o.Version+" | have "+o.Have)
	}
	// Night Tale: Hydra's record is the 1.2.0 entry (same info hash), so
	// 1.3.1 is newer in Feed A, and Feed B's entry matches by Steam id.
	if got := byGame["dir:c:\\games\\night tale"]; len(got) != 2 || got[0] != "Night Tale v1.3.1 + DLC | 1.3.1 | have 1.2.0" || got[1] != "NT (the new build) | 1.3.0 | have 1.2.0" {
		t.Errorf("Night Tale: %q", got)
	}
	// Moonbase: build 450 on this PC, so Build 500 and not Build 400.
	if got := byGame["dir:c:\\games\\moonbase"]; len(got) != 1 || got[0] != "Moonbase Build 500 | Build 500 | have Build 450" {
		t.Errorf("Moonbase: %q", got)
	}
	// Harbor Lights: no versions anywhere, so only the entry uploaded after
	// the game was first seen.
	if got := byGame["dir:c:\\games\\harbor"]; len(got) != 1 || !strings.HasPrefix(got[0], "Harbor Lights |  |") {
		t.Errorf("Harbor Lights: %q", got)
	}
	// Quiet Road 1.0: the patch from 1.0, not the one from 1.4.
	if got := byGame["dir:c:\\games\\quiet"]; len(got) != 1 || got[0] != "Quiet Road Update v1.0 to v1.1 | 1.1 | have 1.0" {
		t.Errorf("Quiet Road: %q", got)
	}
	// A close name is asked about, never linked by itself; a store's game
	// and a name that doesn't match get nothing.
	if len(byGame["dir:c:\\games\\echo"]) != 0 || len(byGame["steam:42"]) != 0 || len(byGame["dir:c:\\games\\zeta"]) != 0 {
		t.Errorf("offers that shouldn't be: %+v", byGame)
	}
	if len(d.Pending) != 1 || d.Pending[0].Name != "Echo Valley Arena" || d.Pending[0].Title != "Echo Valley Online Arena" {
		t.Fatalf("pending: %+v", d.Pending)
	}
	mu.Lock()
	if len(offers) != len(d.Offers) || len(said) != 1 || !strings.Contains(said[0], "updates for your games") {
		t.Errorf("told Lootnest %d offers, said %q", len(offers), said)
	}
	for _, o := range offers {
		if len(o.Actions) != 3 || o.Source == "" || o.Size != 2<<30 && o.Game != "dir:c:\\games\\night tale" {
			t.Errorf("offer: %+v", o)
		}
	}
	mu.Unlock()

	// Yes to the close name: its entry is offered from then on.
	if r := act(map[string]any{"id": "match-yes", "row": "m0"}); r.Error != "" {
		t.Fatal(r.Error)
	}
	d = detect()
	if len(d.Pending) != 0 {
		t.Errorf("still pending: %+v", d.Pending)
	}
	found := false
	for _, o := range d.Offers {
		found = found || o.Game == "dir:c:\\games\\echo"
	}
	if !found {
		t.Error("a confirmed match wasn't offered")
	}

	// Linking by hand, for a name too different to match.
	if r := act(map[string]any{"id": "link", "values": map[string]any{"game": "dir:c:\\games\\zeta", "words": "nothing like it"}}); r.Error == "" {
		t.Error("linked to words no source has")
	}
	if r := act(map[string]any{"id": "link", "values": map[string]any{"game": "dir:c:\\games\\zeta", "words": "zeta protocol"}}); r.Form != "pick" {
		t.Fatalf("link: %+v", r)
	}
	if r := act(map[string]any{"id": "pick", "values": map[string]any{"k": "zeta protocol reloaded collection"}}); r.Error != "" {
		t.Fatal(r.Error)
	}
	d = detect()
	found = false
	for _, o := range d.Offers {
		found = found || o.Game == "dir:c:\\games\\zeta"
	}
	if !found {
		t.Error("a game linked by hand wasn't offered")
	}

	// Ignore this version: 1.3.1 goes and stays gone; the one before it in
	// the same source shows instead. Ignore the source: nothing from it.
	offer := func(id, action string) actResult {
		var r actResult
		out, err := h.Call("offer", map[string]any{"id": id, "action": action})
		if err != nil {
			t.Fatal(err)
		}
		_ = json.Unmarshal(out, &r)
		return r
	}
	var nt131, ntB string
	for _, o := range d.Offers {
		if o.Title == "Night Tale v1.3.1 + DLC" {
			nt131 = o.ID
		}
		if o.Title == "NT (the new build)" {
			ntB = o.ID
		}
	}
	if r := offer(nt131, "ignore-version"); r.Error != "" || r.Toast == "" {
		t.Fatalf("ignore-version: %+v", r)
	}
	d = detect()
	var titles []string
	for _, o := range d.Offers {
		if o.Game == "dir:c:\\games\\night tale" {
			titles = append(titles, o.Title)
		}
	}
	if strings.Join(titles, ",") != "Night Tale v1.3.0 + DLC,NT (the new build)" {
		t.Errorf("after ignoring 1.3.1: %q", titles)
	}
	if r := offer(ntB, "ignore-source"); r.Error != "" {
		t.Fatalf("ignore-source: %+v", r)
	}
	d = detect()
	for _, o := range d.Offers {
		if o.Title == "NT (the new build)" {
			t.Error("an ignored source still offers")
		}
	}
	if r := offer("nope", "update"); r.Error == "" {
		t.Error("an offer that isn't there did something")
	}
	// Stop ignoring brings both back.
	act(map[string]any{"id": "unignore", "row": "g:dir:c:\\games\\night tale"})
	d = detect()
	n := 0
	for _, o := range d.Offers {
		if o.Game == "dir:c:\\games\\night tale" {
			n++
		}
	}
	if n != 2 {
		t.Errorf("after Stop ignoring: %d offers for Night Tale", n)
	}
	// Told once: looking again says nothing new.
	mu.Lock()
	before := len(said)
	mu.Unlock()
	detect()
	mu.Lock()
	if len(said) != before {
		t.Errorf("told again: %q", said)
	}
	mu.Unlock()
}
