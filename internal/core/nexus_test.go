package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeNexus answers the queries Lootnest sends, by what they ask for.
type fakeNexus struct {
	t      *testing.T
	asked  []map[string]any // each request's variables
	games  map[string]string
	byName map[string]string
}

func (f *fakeNexus) post(addr string, body []byte) ([]byte, error) {
	if addr != NexusGraphQL {
		f.t.Fatalf("posted to %s", addr)
	}
	var req struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		f.t.Fatal(err)
	}
	f.asked = append(f.asked, req.Variables)
	q := req.Query
	switch {
	case strings.Contains(q, "game(domainName"):
		if g, ok := f.games[req.Variables["d"].(string)]; ok {
			return []byte(`{"data":{"game":` + g + `}}`), nil
		}
		return []byte(`{"errors":[{"message":"Game not found.","extensions":{"code":"GAME_NOT_FOUND"}}],"data":{"game":null}}`), nil
	case strings.Contains(q, "games(filter: {name"):
		return []byte(`{"data":{"games":{"nodes":[` + f.byName[req.Variables["n"].(string)] + `]}}}`), nil
	case strings.Contains(q, "games(filter: $f"):
		return []byte(`{"data":{"games":{"nodes":[{"id":952,"name":"The Witcher 3: Wild Hunt","domainName":"witcher3","modCount":10993},
			{"id":150,"name":"The Witcher","domainName":"witcher","modCount":519},{"id":7,"name":"Bad","domainName":"../x","modCount":9999}]}}}`), nil
	case strings.Contains(q, "mods(filter"):
		return []byte(`{"data":{"mods":{"totalCount":3,"nodes":[
			{"modId":10,"name":"Better &amp; Brighter","author":"A","version":"1.2","summary":"Line one<br />line [b]two[/b]","endorsements":5,"downloads":50,
			 "thumbnailUrl":"https://staticdelivery.nexusmods.com/mods/1/images/thumbnails/10/a.png","pictureUrl":"https://evil.example/a.png",
			 "updatedAt":"2026-11-01T10:00:00Z","adultContent":false,"status":"published","modCategory":{"name":"Visuals"}},
			{"modId":11,"name":"Hidden","adultContent":false,"status":"hidden"},
			{"modId":12,"name":"Adult","adultContent":true,"status":"published"}]}}}`), nil
	case strings.Contains(q, "mod(gameId"):
		if req.Variables["m"] == "404" {
			return []byte(`{"errors":[{"message":"Mod not found"}],"data":{"mod":null,"modFiles":null}}`), nil
		}
		return []byte(`{"data":{"mod":{"modId":10,"name":"Better","author":"A","version":"1.2","summary":"S","description":"[size=4]Hi[/size]<br />[img]https://x/y.png[/img]See [url=https://x]the guide[/url].","endorsements":5,"downloads":50,
			"adultContent":false,"status":"published","modRequirements":{"nexusRequirements":{"nodes":[
				{"modId":"1472","modName":"Script Hook","url":"","notes":"Needed","externalRequirement":false},
				{"modId":"0","modName":"Outside tool","url":"http://example.com/tool/","notes":"","externalRequirement":true}]}}},
			"modFiles":[
				{"fileId":1,"name":"Old","version":"1.0","category":"OLD_VERSION","sizeInBytes":"100","date":1000,"primary":0},
				{"fileId":2,"name":"Gone","version":"0.9","category":"REMOVED","sizeInBytes":"100","date":900,"primary":0},
				{"fileId":3,"name":"Extra","version":"1.2","category":"OPTIONAL","sizeInBytes":"300","date":3000,"primary":0},
				{"fileId":4,"name":"Main","version":"1.2","category":"MAIN","sizeInBytes":"5495955","date":2000,"primary":1},
				{"fileId":5,"name":"Main newer","version":"1.3","category":"MAIN","sizeInBytes":"10","date":4000,"primary":0}]}}`), nil
	}
	f.t.Fatalf("unexpected query %s", q)
	return nil, nil
}

func TestNexusFindGame(t *testing.T) {
	f := &fakeNexus{t: t,
		games:  map[string]string{"reddeadredemption2": `{"id":3024,"name":"Red Dead Redemption 2","domainName":"reddeadredemption2","modCount":6664}`},
		byName: map[string]string{"Baldur's Gate 3": `{"id":3474,"name":"Baldur's Gate 3","domainName":"baldursgate3","modCount":20401}`},
	}
	g, err := NexusFindGame(f.post, "Red Dead Redemption 2")
	if err != nil || g == nil || g.ID != 3024 || g.Domain != "reddeadredemption2" {
		t.Fatalf("RDR2: %+v %v", g, err)
	}
	if g, err = NexusFindGame(f.post, "Baldur's Gate 3"); err != nil || g == nil || g.ID != 3474 {
		t.Fatalf("BG3: %+v %v", g, err)
	}
	// Nexus's spelling lacks the mark and the domain differs: found by its words.
	if g, err = NexusFindGame(f.post, "The Witcher® 3: Wild Hunt"); err != nil || g == nil || g.Domain != "witcher3" {
		t.Fatalf("Witcher: %+v %v", g, err)
	}
	if g, err = NexusFindGame(f.post, "Some Game"); err != nil || g != nil {
		t.Fatalf("a game Nexus lacks: %+v %v", g, err)
	}
	if g, err = NexusFindGame(f.post, "  "); err != nil || g != nil {
		t.Fatalf("no name: %+v %v", g, err)
	}
	gs, err := NexusFindGames(f.post, "witcher")
	if err != nil || len(gs) != 2 || gs[0].ID != 952 {
		t.Fatalf("games: %+v %v (an unsafe domain is left out)", gs, err)
	}
}

func TestNexusSearch(t *testing.T) {
	f := &fakeNexus{t: t}
	ms, total, err := NexusSearch(f.post, 3024, "Lenny's trainer", "updated", 24)
	if err != nil || total != 3 || len(ms) != 1 {
		t.Fatalf("%+v %d %v", ms, total, err)
	}
	m := ms[0]
	if m.Name != "Better & Brighter" || m.Summary != "Line one line two" || m.Category != "Visuals" || m.Updated.IsZero() {
		t.Fatalf("mod: %+v", m)
	}
	if m.Thumb == "" || m.Picture != "" {
		t.Fatalf("pictures only from Nexus's servers: %+v", m)
	}
	v := f.asked[0]
	b, _ := json.Marshal(v)
	s := string(b)
	for _, want := range []string{`"value":"lenny"`, `"value":"trainer"`, `"adultContent":[{"op":"EQUALS","value":false}]`, `"updatedAt":{"direction":"DESC"}`, `"o":24`, `"value":"3024"`} {
		if !strings.Contains(s, want) {
			t.Errorf("request lacks %s: %s", want, s)
		}
	}
	if _, _, err := NexusSearch(f.post, 0, "", "", 0); err == nil {
		t.Fatal("no game should fail")
	}
}

func TestNexusModInfo(t *testing.T) {
	f := &fakeNexus{t: t}
	p, err := NexusModInfo(f.post, 3024, 10)
	if err != nil || p == nil {
		t.Fatal(p, err)
	}
	if p.Description != "Hi\nSee the guide." {
		t.Errorf("description %q", p.Description)
	}
	if len(p.Needs) != 2 || p.Needs[0].ModID != 1472 || p.Needs[1].ModID != 0 || !p.Needs[1].External || p.Needs[1].URL != "http://example.com/tool/" {
		t.Errorf("needs %+v", p.Needs)
	}
	var ids []int
	for _, f := range p.Files {
		ids = append(ids, f.ID)
	}
	if got, _ := json.Marshal(ids); string(got) != "[4,5,3,1]" {
		t.Errorf("files in order %s (main primary first, then newest; removed left out)", got)
	}
	if p.Files[0].Size != 5495955 || !p.Files[0].Primary {
		t.Errorf("main file %+v", p.Files[0])
	}
	if p, err := NexusModInfo(f.post, 3024, 404); p != nil || err != nil {
		t.Fatalf("a missing mod is nil without an error: %v %v", p, err)
	}
}

func TestNexusText(t *testing.T) {
	in := "[center][color=#fff]\uFEFF Title[/color][/center]<br />\r\n<br /><br /><br />[youtube]abc[/youtube]Line &amp; more [b]bold[/b] [url=https://x]link words[/url]<script>x</script>\x07"
	if got := NexusText(in); got != "Title\n\nLine & more bold link wordsx" {
		t.Fatalf("%q", got)
	}
	long := strings.Repeat("word ", 3000)
	if got := NexusText(long); !strings.HasSuffix(got, "…") || len([]rune(got)) > nexusTextMax+1 {
		t.Fatalf("long text not cut: %d", len([]rune(got)))
	}
}

func TestNexusAddresses(t *testing.T) {
	for u, ok := range map[string]bool{
		"https://staticdelivery.nexusmods.com/mods/1/a.png": true,
		"https://images.nexusmods.com/mod-images/1/a.png":   true,
		"http://staticdelivery.nexusmods.com/a.png":         false,
		"https://staticdelivery.nexusmods.com.evil/a.png":   false,
		"https://user@images.nexusmods.com/a.png":           false,
	} {
		if SafeNexusImage(u) != ok {
			t.Errorf("SafeNexusImage(%s) != %v", u, ok)
		}
	}
	for u, ok := range map[string]bool{
		"https://www.nexusmods.com/reddeadredemption2/mods/233": true,
		"https://users.nexusmods.com/auth/sign_in":              true,
		"https://nexusmods.com/":                                true,
		"http://www.nexusmods.com/":                             false,
		"https://nexusmods.com.example/":                        false,
		"https://example.com/?u=nexusmods.com":                  false,
		"nxm://reddeadredemption2/mods/233/files/1":             false,
	} {
		if NexusPage(u) != ok {
			t.Errorf("NexusPage(%s) != %v", u, ok)
		}
	}
	if got := NexusFileURL("reddeadredemption2", 233, 32359); got != "https://www.nexusmods.com/reddeadredemption2/mods/233?tab=files&file_id=32359" {
		t.Error(got)
	}
	if NexusFileURL("../x", 1, 1) != "" || NexusModURL("a", 0) != "" {
		t.Error("bad parts make no address")
	}
}

func TestNexusFileName(t *testing.T) {
	for name, want := range map[string]int{
		"Rampage-233-1-6-9-1792931965.zip":              233,
		"ShroudMenu 1.1.0-11287-1-1792965670 (1).7z":    11287,
		"Some Mod - Part A-5304-v2-0b-1700000000.rar":   5304,
		"Game.Update.v1.2.zip":                          0,
		"RDR2-Trainer-12.zip":                           0,
		"Better Lighting-1000-1690000000.zip":           1000,
		"Better Lighting-1000-1-0-1690000000.part1.rar": 0, // a .rar set's part: its stem isn't the name
		"Mod-2-Fix-1690000000.zip":                      2,
	} {
		id, ok := NexusFileName(name)
		if want == 0 && ok || want != 0 && id != want {
			t.Errorf("%s: %d %v, want %d", name, id, ok, want)
		}
	}
}

func TestNexusArrivals(t *testing.T) {
	dir := t.TempDir()
	start := time.Now().Add(-time.Minute)
	put := func(name string, at time.Time) {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("data"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, at, at); err != nil {
			t.Fatal(err)
		}
	}
	put("Rampage-233-1-6-9-1792931965.zip", start.Add(20*time.Second))
	put("Rampage-233-1-6-8-1700000000.zip", start.Add(30*time.Second)) // another file of the same mod, newer on disk
	put("Old Mod-500-1-1600000000.zip", start.Add(-time.Hour))         // there before the ask
	put("Other-777-1-1792000000.7z.crdownload", start.Add(40*time.Second))
	put("notes.txt", start.Add(40*time.Second))
	waits := []NexusWait{
		{Key: "a", ModID: 233, FileID: 1, Date: 1792931965, Since: start},
		{Key: "b", ModID: 500, FileID: 2, Date: 1600000000, Since: start},
		{Key: "c", ModID: 777, FileID: 3, Date: 1792000000, Since: start},
	}
	got := NexusArrivals(dir, waits)
	if len(got) != 1 || got[0].Key != "a" || got[0].Name != "Rampage-233-1-6-9-1792931965.zip" || got[0].Guess {
		t.Fatalf("exact file first: %+v", got)
	}
	// Without the exact file, the mod's other file.
	os.Remove(filepath.Join(dir, "Rampage-233-1-6-9-1792931965.zip"))
	if got = NexusArrivals(dir, waits[:1]); len(got) != 1 || got[0].Name != "Rampage-233-1-6-8-1700000000.zip" || got[0].Guess {
		t.Fatalf("by mod: %+v", got)
	}
	// A file named otherwise, when one ask is open, is a guess; another Nexus mod's file isn't taken.
	os.Remove(filepath.Join(dir, "Rampage-233-1-6-8-1700000000.zip"))
	put("download.zip", start.Add(50*time.Second))
	put("Unrelated-999-1-1792000001.zip", start.Add(55*time.Second))
	if got = NexusArrivals(dir, waits[:1]); len(got) != 1 || got[0].Name != "download.zip" || !got[0].Guess {
		t.Fatalf("guess: %+v", got)
	}
	if got = NexusArrivals(dir, waits[:2]); len(got) != 0 {
		t.Fatalf("no guess with two asks: %+v", got)
	}
	if NexusArrivals(dir, nil) != nil || NexusArrivals(filepath.Join(dir, "none"), waits) != nil {
		t.Fatal("nothing to match")
	}
}

func TestNexusWindow(t *testing.T) {
	for u, want := range map[string]string{
		"https://www.nexusmods.com/reddeadredemption2/mods/233?tab=files": "show",
		"https://users.nexusmods.com/auth/sign_in":                        "show",
		"https://supporter-files.nexus-cdn.com/3024/233/x.zip?md5=1":      "show",
		"https://accounts.google.com/o/oauth2/auth":                       "show",
		"about:blank": "show",
		"NXM://reddeadredemption2/mods/233/files/32359?key=a": "nxm",
		"https://www.dev-c.com/rdr2/scripthookrdr2/":          "browser",
		"http://www.nexusmods.com/":                           "browser",
		"https://www.nexusmods.com:8443/":                     "browser",
		"https://u@www.nexusmods.com/":                        "refuse",
		"file:///C:/Windows":                                  "refuse",
		"javascript:alert(1)":                                 "refuse",
	} {
		if got := NexusWindowGo(u); got != want {
			t.Errorf("%s: %s, want %s", u, got, want)
		}
	}
	arg := NexusWindowArg("reddeadredemption2", 233, 32359)
	if got := NexusWindowURL(arg); got != "https://www.nexusmods.com/reddeadredemption2/mods/233?tab=files&file_id=32359" {
		t.Error(got)
	}
	if got := NexusWindowURL("reddeadredemption2/233/0"); got != "https://www.nexusmods.com/reddeadredemption2/mods/233" {
		t.Error(got)
	}
	for _, bad := range []string{"", "a/b", "../x/1/2", "x/1/2/3", "x/a/1", "evil.com#/1/1"} {
		if NexusWindowURL(bad) != "" {
			t.Errorf("%q made an address", bad)
		}
	}
}

func TestNameMod(t *testing.T) {
	root, game := t.TempDir(), t.TempDir()
	src := filepath.Join(t.TempDir(), "Rampage-233-1-6-9-1792931965")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "Rampage.asi"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := AddMod(root, game, src, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := NameMod(root, m.ID, "  Rampage   Trainer ", &ModNexus{Domain: "reddeadredemption2", ModID: 233, FileID: 32359, Version: "1.6.9"}); err != nil {
		t.Fatal(err)
	}
	got, err := LoadMod(root, m.ID)
	if err != nil || got.Name != "Rampage Trainer" || got.Nexus == nil || got.Nexus.ModID != 233 || len(got.Files) != 1 {
		t.Fatalf("%+v %v", got, err)
	}
	if err := NameMod(root, "m-nope", "x", nil); err == nil {
		t.Fatal("a mod that isn't kept")
	}
}
