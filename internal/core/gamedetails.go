package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Game details: a game's summary, genres, developer, publisher, release
// date, screenshots and review score, from the Steam store (appdetails and
// the review summary, no key). Only with Settings.GameDetails on, and only
// for the game whose page is open: nothing here walks the Library. Kept on
// the PC for 30 days; the user's own edits always win.
// ---------------------------------------------------------------------------

const (
	detailsURL = "https://store.steampowered.com/api/appdetails?appids=%d&l=%s&filters=basic,genres,developers,publishers,release_date,screenshots,metacritic"
	reviewsURL = "https://store.steampowered.com/appreviews/%d?json=1&language=all&purchase_type=all&num_per_page=0"
)

// DetailsKeep is how long fetched details are used before they're read again.
const DetailsKeep = 30 * 24 * time.Hour

// At most this many screenshots, genres, developers or publishers.
const maxItems = 12

var steamLang = regexp.MustCompile(`^[a-z]{2,12}$`)

type Screenshot struct {
	Thumb string `json:"thumb"`
	Full  string `json:"full"`
}

type GameDetails struct {
	Summary       string       `json:"summary,omitempty"`
	Genres        []string     `json:"genres,omitempty"`
	Developers    []string     `json:"developers,omitempty"`
	Publishers    []string     `json:"publishers,omitempty"`
	Released      string       `json:"released,omitempty"`
	Screenshots   []Screenshot `json:"screenshots,omitempty"`
	ReviewScore   string       `json:"reviewScore,omitempty"` // Steam's words, such as "Very Positive"
	ReviewPercent int          `json:"reviewPercent,omitempty"`
	ReviewCount   int          `json:"reviewCount,omitempty"`
	Metacritic    int          `json:"metacritic,omitempty"`
	StoreURL      string       `json:"storeUrl,omitempty"`
	Fetched       time.Time    `json:"fetched"`
	// Unknown: the Steam store doesn't have the game, kept so it isn't
	// asked again on every visit.
	Unknown bool `json:"unknown,omitempty"`
}

// DetailsEdit holds only the fields the user changed; nil is Steam's.
type DetailsEdit struct {
	Summary   *string   `json:"summary,omitempty"`
	Developer *string   `json:"developer,omitempty"`
	Publisher *string   `json:"publisher,omitempty"`
	Released  *string   `json:"released,omitempty"`
	Genres    *[]string `json:"genres,omitempty"`
}

func (e DetailsEdit) Empty() bool {
	return e.Summary == nil && e.Developer == nil && e.Publisher == nil && e.Released == nil && e.Genres == nil
}

var htmlTag = regexp.MustCompile(`(?s)<[^>]*>`)

// HTMLText turns a store description's HTML into one paragraph of text.
func HTMLText(s string) string {
	return strings.Join(strings.Fields(html.UnescapeString(htmlTag.ReplaceAllString(s, " "))), " ")
}

// FetchGameDetails reads a game's details from the Steam store, in lang
// (Steam's language name, such as "english"). A game the store doesn't
// have gives nil and no error. The review summary is extra: if it fails,
// the details come without it.
func FetchGameDetails(fetch Fetch, appID int, lang string) (*GameDetails, error) {
	if appID <= 0 {
		return nil, errors.New("unknown game")
	}
	if !steamLang.MatchString(lang) {
		lang = "english"
	}
	b, err := fetch(fmt.Sprintf(detailsURL, appID, lang))
	if err != nil {
		return nil, err
	}
	var res map[string]struct {
		Success bool `json:"success"`
		Data    struct {
			Short  string `json:"short_description"`
			Genres []struct {
				Description string `json:"description"`
			} `json:"genres"`
			Developers []string `json:"developers"`
			Publishers []string `json:"publishers"`
			Release    struct {
				Date string `json:"date"`
			} `json:"release_date"`
			Screenshots []struct {
				Thumb string `json:"path_thumbnail"`
				Full  string `json:"path_full"`
			} `json:"screenshots"`
			Metacritic struct {
				Score int `json:"score"`
			} `json:"metacritic"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, fmt.Errorf("unexpected reply from the Steam store: %w", err)
	}
	r, ok := res[strconv.Itoa(appID)]
	if !ok || !r.Success {
		return nil, nil
	}
	d := r.Data
	g := &GameDetails{Summary: HTMLText(d.Short), Released: HTMLText(d.Release.Date), Metacritic: d.Metacritic.Score,
		Developers: cleanNames(d.Developers), Publishers: cleanNames(d.Publishers), Fetched: time.Now().UTC(),
		StoreURL: fmt.Sprintf("https://store.steampowered.com/app/%d/", appID)}
	for _, x := range d.Genres {
		g.Genres = append(g.Genres, x.Description)
	}
	g.Genres = cleanNames(g.Genres)
	for _, s := range d.Screenshots {
		if len(g.Screenshots) == maxItems {
			break
		}
		if SafeNewsImage(s.Thumb) && SafeNewsImage(s.Full) {
			g.Screenshots = append(g.Screenshots, Screenshot{s.Thumb, s.Full})
		}
	}
	if b, err := fetch(fmt.Sprintf(reviewsURL, appID)); err == nil {
		var rv struct {
			Summary struct {
				Desc     string `json:"review_score_desc"`
				Positive int    `json:"total_positive"`
				Total    int    `json:"total_reviews"`
			} `json:"query_summary"`
		}
		if json.Unmarshal(b, &rv) == nil && rv.Summary.Total > 0 {
			g.ReviewScore = HTMLText(rv.Summary.Desc)
			g.ReviewCount = rv.Summary.Total
			g.ReviewPercent = (rv.Summary.Positive*100 + rv.Summary.Total/2) / rv.Summary.Total
		}
	}
	return g, nil
}

// cleanNames trims names and drops empty and repeated ones, at most maxItems.
func cleanNames(in []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, n := range in {
		n = HTMLText(n)
		if r := []rune(n); len(r) > 100 {
			n = string(r[:100])
		}
		if n == "" || seen[strings.ToLower(n)] || len(out) == maxItems {
			continue
		}
		seen[strings.ToLower(n)] = true
		out = append(out, n)
	}
	return out
}

func detailsFile(dir string, appID int) string {
	return filepath.Join(dir, "details", strconv.Itoa(appID)+".json")
}

func editsFile(dir, key string) string {
	return filepath.Join(dir, "details-edits", gameFileKey(key)+".json")
}

// LoadDetails is the copy kept on this PC (nil when there's none), and
// whether it's older than DetailsKeep.
func LoadDetails(dir string, appID int) (*GameDetails, bool) {
	b, err := os.ReadFile(detailsFile(dir, appID))
	if err != nil {
		return nil, true
	}
	var d GameDetails
	if json.Unmarshal(b, &d) != nil {
		return nil, true
	}
	return &d, time.Since(d.Fetched) > DetailsKeep
}

func SaveDetails(dir string, appID int, d *GameDetails) error {
	return writeJSONAtomic(detailsFile(dir, appID), d)
}

// LoadDetailsEdit is the user's edits of a Library game's details.
func LoadDetailsEdit(dir, key string) DetailsEdit {
	var e DetailsEdit
	if b, err := os.ReadFile(editsFile(dir, key)); err == nil {
		_ = json.Unmarshal(b, &e)
	}
	return e
}

// SaveDetailsEdit keeps the user's edits; with none left, the file goes,
// which is how "Use Steam's again" works.
func SaveDetailsEdit(dir, key string, e DetailsEdit) error {
	p := editsFile(dir, key)
	if e.Empty() {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return writeJSONAtomic(p, e)
}

// CleanDetailsEdit trims what the user typed and keeps only the fields
// that differ from Steam's (fetched may be nil).
func CleanDetailsEdit(e DetailsEdit, fetched *GameDetails) DetailsEdit {
	var f GameDetails
	if fetched != nil {
		f = *fetched
	}
	text := func(p *string, max int, was string) *string {
		if p == nil {
			return nil
		}
		s := strings.Join(strings.Fields(*p), " ")
		if r := []rune(s); len(r) > max {
			s = strings.TrimSpace(string(r[:max]))
		}
		if s == was {
			return nil
		}
		return &s
	}
	out := DetailsEdit{
		Summary:   text(e.Summary, 2000, f.Summary),
		Developer: text(e.Developer, 200, strings.Join(f.Developers, ", ")),
		Publisher: text(e.Publisher, 200, strings.Join(f.Publishers, ", ")),
		Released:  text(e.Released, 60, f.Released),
	}
	if e.Genres != nil {
		g := cleanNames(*e.Genres)
		if strings.Join(g, ",") != strings.Join(f.Genres, ",") {
			if g == nil {
				g = []string{}
			}
			out.Genres = &g
		}
	}
	return out
}

// MergeDetails is fetched with the user's edits over it (fetched may be nil).
func MergeDetails(fetched *GameDetails, e DetailsEdit) *GameDetails {
	var d GameDetails
	if fetched != nil {
		d = *fetched
	}
	if e.Summary != nil {
		d.Summary = *e.Summary
	}
	one := func(s string) []string {
		if s == "" {
			return nil
		}
		return []string{s}
	}
	if e.Developer != nil {
		d.Developers = one(*e.Developer)
	}
	if e.Publisher != nil {
		d.Publishers = one(*e.Publisher)
	}
	if e.Released != nil {
		d.Released = *e.Released
	}
	if e.Genres != nil {
		d.Genres = *e.Genres
	}
	return &d
}
