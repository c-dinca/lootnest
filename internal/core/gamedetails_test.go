package core

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const hadesDetails = `{"1145350":{"success":true,"data":{"name":"Hades II",
"short_description":"Battle beyond the Underworld &amp; <b>defy</b> the Titan of Time<br>\r\n in this   sequel.",
"genres":[{"id":"1","description":"Action"},{"id":"25","description":"Adventure"},{"id":"1","description":"Action"}],
"developers":["Supergiant Games"],"publishers":["Supergiant Games"],
"release_date":{"coming_soon":false,"date":"25 Sep, 2025"},
"screenshots":[{"id":0,"path_thumbnail":"https://shared.akamai.steamstatic.com/a/ss_1.600x338.jpg","path_full":"https://shared.akamai.steamstatic.com/a/ss_1.1920x1080.jpg"},
{"id":1,"path_thumbnail":"https://evil.example/ss.jpg","path_full":"https://evil.example/ss.jpg"}],
"metacritic":{"score":95,"url":"https://www.metacritic.com/game/hades-ii"}}}}`

const hadesReviews = `{"success":1,"query_summary":{"num_reviews":0,"review_score":9,"review_score_desc":"Overwhelmingly Positive","total_positive":12100,"total_negative":245,"total_reviews":12345}}`

// steamServer stands in for the Steam store, and fetch sends to it what
// would go to store.steampowered.com.
func steamServer(t *testing.T, details, reviews string, asked *[]string) Fetch {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*asked = append(*asked, r.URL.String())
		switch {
		case r.URL.Path == "/api/appdetails":
			_, _ = io.WriteString(w, details)
		case strings.HasPrefix(r.URL.Path, "/appreviews/") && reviews != "":
			_, _ = io.WriteString(w, reviews)
		default:
			http.Error(w, "no", http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)
	return func(u string) ([]byte, error) {
		resp, err := http.Get(strings.Replace(u, "https://store.steampowered.com", srv.URL, 1))
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, io.ErrUnexpectedEOF
		}
		return io.ReadAll(resp.Body)
	}
}

func TestFetchGameDetails(t *testing.T) {
	var asked []string
	d, err := FetchGameDetails(steamServer(t, hadesDetails, hadesReviews, &asked), 1145350, "english")
	if err != nil || d == nil {
		t.Fatal(d, err)
	}
	if len(asked) != 2 || !strings.Contains(asked[0], "appids=1145350") || !strings.Contains(asked[0], "screenshots") || !strings.Contains(asked[1], "num_per_page=0") {
		t.Fatalf("asked %v", asked)
	}
	if d.Summary != "Battle beyond the Underworld & defy the Titan of Time in this sequel." {
		t.Fatalf("summary %q", d.Summary)
	}
	if strings.Join(d.Genres, ",") != "Action,Adventure" || d.Developers[0] != "Supergiant Games" || d.Released != "25 Sep, 2025" || d.Metacritic != 95 {
		t.Fatalf("details %+v", d)
	}
	if len(d.Screenshots) != 1 || !strings.HasSuffix(d.Screenshots[0].Full, "1920x1080.jpg") {
		t.Fatalf("pictures come from Steam only: %+v", d.Screenshots)
	}
	if d.ReviewScore != "Overwhelmingly Positive" || d.ReviewPercent != 98 || d.ReviewCount != 12345 {
		t.Fatalf("reviews %+v", d)
	}
	if d.StoreURL != "https://store.steampowered.com/app/1145350/" || d.Fetched.IsZero() {
		t.Fatalf("%+v", d)
	}
}

func TestFetchGameDetailsWithoutReviews(t *testing.T) {
	var asked []string
	d, err := FetchGameDetails(steamServer(t, hadesDetails, "", &asked), 1145350, "")
	if err != nil || d == nil || d.Summary == "" || d.ReviewScore != "" || d.ReviewCount != 0 {
		t.Fatalf("details without reviews: %+v %v", d, err)
	}
	if !strings.Contains(asked[0], "l=english") {
		t.Fatalf("asked %v", asked)
	}
}

func TestFetchGameDetailsUnknown(t *testing.T) {
	var asked []string
	d, err := FetchGameDetails(steamServer(t, `{"999":{"success":false}}`, hadesReviews, &asked), 999, "english")
	if err != nil || d != nil {
		t.Fatalf("a game Steam doesn't know: %+v %v", d, err)
	}
	if _, err := FetchGameDetails(steamServer(t, `<html>`, "", &asked), 999, "english"); err == nil {
		t.Fatal("a broken reply is an error")
	}
}

func TestHTMLText(t *testing.T) {
	cases := map[string]string{
		"<p>One &quot;two&quot;</p>\n<ul><li>three</li></ul>": `One "two" three`,
		"Rock &amp; roll&nbsp;now":                            "Rock & roll now",
		"plain":                                               "plain",
		"&lt;b&gt;kept as text&lt;/b&gt;":                     "<b>kept as text</b>",
	}
	for in, want := range cases {
		if got := HTMLText(in); got != want {
			t.Errorf("HTMLText(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDetailsCache(t *testing.T) {
	dir := t.TempDir()
	if d, stale := LoadDetails(dir, 5); d != nil || !stale {
		t.Fatal("nothing kept yet")
	}
	if err := SaveDetails(dir, 5, &GameDetails{Summary: "Fresh", Fetched: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if d, stale := LoadDetails(dir, 5); d == nil || d.Summary != "Fresh" || stale {
		t.Fatalf("fresh copy: %+v %v", d, stale)
	}
	_ = SaveDetails(dir, 5, &GameDetails{Summary: "Old", Fetched: time.Now().Add(-31 * 24 * time.Hour)})
	if d, stale := LoadDetails(dir, 5); d == nil || d.Summary != "Old" || !stale {
		t.Fatalf("a month-old copy is shown and read again: %+v %v", d, stale)
	}
}

func TestDetailsEditsWin(t *testing.T) {
	dir := t.TempDir()
	steam := &GameDetails{Summary: "Steam's", Developers: []string{"Supergiant Games"}, Publishers: []string{"Supergiant Games"}, Released: "25 Sep, 2025", Genres: []string{"Action"}}
	str := func(s string) *string { return &s }
	genres := []string{" Roguelike ", "", "Action", "roguelike"}
	e := CleanDetailsEdit(DetailsEdit{Summary: str("  Mine,\n  really "), Developer: str("Supergiant Games"), Released: str("2025"), Genres: &genres}, steam)
	if e.Developer != nil || e.Publisher != nil {
		t.Fatalf("a field the same as Steam's isn't an edit: %+v", e)
	}
	if *e.Summary != "Mine, really" || *e.Released != "2025" || strings.Join(*e.Genres, ",") != "Roguelike,Action" {
		t.Fatalf("edit %+v", e)
	}
	if err := SaveDetailsEdit(dir, "dir:d:\\games\\hades ii", e); err != nil {
		t.Fatal(err)
	}
	got := LoadDetailsEdit(dir, "dir:d:\\games\\hades ii")
	m := MergeDetails(steam, got)
	if m.Summary != "Mine, really" || m.Released != "2025" || m.Developers[0] != "Supergiant Games" || strings.Join(m.Genres, ",") != "Roguelike,Action" {
		t.Fatalf("merged %+v", m)
	}
	if steam.Summary != "Steam's" {
		t.Fatal("merging changed Steam's copy")
	}
	if m := MergeDetails(nil, got); m.Summary != "Mine, really" {
		t.Fatalf("edits without Steam's details: %+v", m)
	}
	if !LoadDetailsEdit(dir, "game:other").Empty() {
		t.Fatal("another game has no edits")
	}
	// Use Steam's again: no edits left, no file left.
	if err := SaveDetailsEdit(dir, "dir:d:\\games\\hades ii", CleanDetailsEdit(DetailsEdit{Summary: str("Steam's")}, steam)); err != nil {
		t.Fatal(err)
	}
	if !LoadDetailsEdit(dir, "dir:d:\\games\\hades ii").Empty() {
		t.Fatal("the edit is gone")
	}
	if err := SaveDetailsEdit(dir, "dir:d:\\games\\hades ii", DetailsEdit{}); err != nil {
		t.Fatal("clearing twice is fine:", err)
	}
}

func TestControllerSupport(t *testing.T) {
	cat := func(names ...string) string {
		var parts []string
		for i, n := range names {
			parts = append(parts, fmt.Sprintf(`{"id":%d,"description":%q}`, i+1, n))
		}
		return "[" + strings.Join(parts, ",") + "]"
	}
	cases := []struct {
		name, data, want string
		pads             []string
	}{
		{"the field says full", `"controller_support":"full","categories":` + cat("Single-player", "Partial Controller Support"), "full", nil},
		{"partial from the categories", `"categories":` + cat("Single-player", "partial controller support"), "partial", nil},
		{"full beats partial in the categories", `"categories":` + cat("Partial Controller Support", "Full controller support"), "full", nil},
		{"pad categories", `"categories":` + cat("Full controller support", "PlayStation controllers", "DualSense controller support", "Steam Achievements"), "full", []string{"playstation"}},
		{"DualShock alone", `"categories":` + cat("DualShock controller support"), "", []string{"playstation"}},
		{"none", `"categories":` + cat("Single-player", "Steam Cloud", "Tracked Controller Support"), "", nil},
		{"no categories at all", `"short_description":"x"`, "", nil},
	}
	for _, c := range cases {
		var asked []string
		body := `{"7":{"success":true,"data":{` + c.data + `}}}`
		d, err := FetchGameDetails(steamServer(t, body, "", &asked), 7, "english")
		if err != nil || d == nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if d.Controller != c.want || strings.Join(d.Pads, ",") != strings.Join(c.pads, ",") {
			t.Errorf("%s: got %q %v, want %q %v", c.name, d.Controller, d.Pads, c.want, c.pads)
		}
		if !strings.Contains(asked[0], "categories") {
			t.Fatalf("the categories are asked for in the same request: %v", asked)
		}
	}
}

func TestControllerOldCache(t *testing.T) {
	dir := t.TempDir()
	old := `{"summary":"Kept before 1.85","genres":["Action"],"fetched":"` + time.Now().Add(-24*time.Hour).UTC().Format(time.RFC3339) + `"}`
	if err := os.MkdirAll(filepath.Join(dir, "details"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "details", "9.json"), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	d, stale := LoadDetails(dir, 9)
	if d == nil || stale || d.Summary != "Kept before 1.85" || d.Controller != "" || d.Pads != nil {
		t.Fatalf("an old copy stays valid, with nothing known about controllers: %+v %v", d, stale)
	}
}

func TestControllerEdit(t *testing.T) {
	str := func(s string) *string { return &s }
	steam := &GameDetails{Controller: "full", Pads: []string{"playstation"}}
	if e := CleanDetailsEdit(DetailsEdit{Controller: str("full")}, steam); !e.Empty() {
		t.Fatalf("the same as Steam's isn't an edit: %+v", e)
	}
	if e := CleanDetailsEdit(DetailsEdit{Controller: str("")}, steam); !e.Empty() {
		t.Fatalf("Steam's is no edit: %+v", e)
	}
	if e := CleanDetailsEdit(DetailsEdit{Controller: str("wheel")}, steam); !e.Empty() {
		t.Fatalf("an unknown value is dropped: %+v", e)
	}
	e := CleanDetailsEdit(DetailsEdit{Controller: str("none")}, steam)
	if e.Empty() || *e.Controller != "none" {
		t.Fatalf("keyboard and mouse only is kept: %+v", e)
	}
	if m := MergeDetails(steam, e); m.Controller != "none" || m.Pads != nil {
		t.Fatalf("merged %+v", m)
	}
	if m := MergeDetails(nil, DetailsEdit{Controller: str("partial")}); m.Controller != "partial" {
		t.Fatalf("an edit without Steam's details: %+v", m)
	}
	if steam.Controller != "full" || len(steam.Pads) != 1 {
		t.Fatal("merging changed Steam's copy")
	}
}
