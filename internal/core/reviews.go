package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// ---------------------------------------------------------------------------
// Steam reviews for a game's Community section: the most helpful of the
// past year, read from the store's public appreviews endpoint (no key, no
// account). Only with Settings.GameDetails on, and only for the game whose
// page is open. Nothing is posted: the reviews link to Steam to write one.
// ---------------------------------------------------------------------------

const reviewListURL = "https://store.steampowered.com/appreviews/%d?json=1&filter=all&language=%s&review_type=%s&purchase_type=all&num_per_page=%d&day_range=365"

// ReviewsMax is how many reviews a page asks for at a time.
const ReviewsMax = 10

// reviewMaxRunes caps one review's text; the page shows the start.
const reviewMaxRunes = 1500

type Review struct {
	ID      string    `json:"id"`
	Up      bool      `json:"up"`
	Text    string    `json:"text"`
	Hours   float64   `json:"hours"`   // played when the review was written
	Total   float64   `json:"total"`   // played in all
	Helpful int       `json:"helpful"` // people who found it helpful
	Funny   int       `json:"funny"`
	At      time.Time `json:"at"`
	Cut     bool      `json:"cut,omitempty"` // the text was longer
	Free    bool      `json:"free,omitempty"`
	Early   bool      `json:"early,omitempty"` // written in Early Access
}

var (
	reviewBB = regexp.MustCompile(`(?i)\[/?(?:b|i|u|s|h[1-6]|strike|spoiler|noparse|hr|list|olist|\*|quote|code|table|tr|th|td|url)(?:=[^\]]*)?\]`)
	manyNL   = regexp.MustCompile(`\n{3,}`)
	spaceRun = regexp.MustCompile(`[ \t]+`)
	reviewID = regexp.MustCompile(`^[0-9]{1,20}$`)
)

// ReviewText turns a review's BBCode into plain text, keeping paragraphs.
func ReviewText(s string) (string, bool) {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = reviewBB.ReplaceAllString(s, "")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(spaceRun.ReplaceAllString(l, " "))
	}
	s = strings.TrimSpace(manyNL.ReplaceAllString(strings.Join(lines, "\n"), "\n\n"))
	if utf8.RuneCountInString(s) <= reviewMaxRunes {
		return s, false
	}
	r := []rune(s)[:reviewMaxRunes]
	if i := strings.LastIndexAny(string(r), " \n"); i > reviewMaxRunes/2 {
		return strings.TrimSpace(string(r)[:i]) + "…", true
	}
	return string(r) + "…", true
}

// FetchReviews reads the most helpful reviews of the past year; kind is
// "all", "positive" or "negative". Reviews with no text are left out.
func FetchReviews(fetch Fetch, appID int, lang, kind string) ([]Review, error) {
	if appID <= 0 {
		return nil, errors.New("unknown game")
	}
	if !steamLang.MatchString(lang) {
		lang = "english"
	}
	if kind != "positive" && kind != "negative" {
		kind = "all"
	}
	b, err := fetch(fmt.Sprintf(reviewListURL, appID, lang, kind, ReviewsMax))
	if err != nil {
		return nil, err
	}
	return parseReviews(b)
}

func parseReviews(b []byte) ([]Review, error) {
	var res struct {
		Success int `json:"success"`
		Reviews []struct {
			ID     string `json:"recommendationid"`
			Author struct {
				Forever  float64 `json:"playtime_forever"`
				AtReview float64 `json:"playtime_at_review"`
			} `json:"author"`
			Review  string `json:"review"`
			Created int64  `json:"timestamp_created"`
			Up      bool   `json:"voted_up"`
			Helpful int    `json:"votes_up"`
			Funny   int    `json:"votes_funny"`
			Free    bool   `json:"received_for_free"`
			Early   bool   `json:"written_during_early_access"`
		} `json:"reviews"`
	}
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, err
	}
	if res.Success != 1 {
		return nil, errors.New("the store didn't answer")
	}
	out := []Review{}
	for _, r := range res.Reviews {
		text, cut := ReviewText(r.Review)
		if text == "" || !reviewID.MatchString(r.ID) {
			continue
		}
		at := r.Author.AtReview
		if at <= 0 {
			at = r.Author.Forever
		}
		out = append(out, Review{ID: r.ID, Up: r.Up, Text: text, Cut: cut, Hours: hours(at), Total: hours(r.Author.Forever),
			Helpful: r.Helpful, Funny: r.Funny, At: time.Unix(r.Created, 0).UTC(), Free: r.Free, Early: r.Early})
	}
	return out, nil
}

// hours rounds minutes to tenths of an hour.
func hours(min float64) float64 { return float64(int(min/6+0.5)) / 10 }
