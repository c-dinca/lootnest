package core

import (
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// ---------------------------------------------------------------------------
// Steam reviews for a game's Community section: the most helpful of the
// past year, from Steam's public IUserReviewsService (no key, no account;
// the store's appreviews endpoint was switched off on 22 Oct 2026). Only
// with Settings.GameDetails on, and only for the game whose page is open.
// Nothing is posted: the reviews link to Steam to write one.
// ---------------------------------------------------------------------------

// reviewsQuery is the GetAppReviews address for these inputs.
func reviewsQuery(in map[string]any) string {
	b, _ := json.Marshal(in)
	return reviewsAPI + url.QueryEscape(string(b))
}

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
	page, err := FetchReviewPage(fetch, appID, lang, kind)
	return page.Items, err
}

// ReviewPage is one page of reviews and, for kind "all", Steam's words
// and share of positive reviews for the game.
type ReviewPage struct {
	Items   []Review `json:"items"`
	Score   string   `json:"score,omitempty"`   // such as "Very Positive"
	Percent int      `json:"percent,omitempty"` // positive, 0 to 100
	Count   int      `json:"count,omitempty"`   // every review counted
}

// FetchReviewPage is FetchReviews with the summary Steam sends beside them.
func FetchReviewPage(fetch Fetch, appID int, lang, kind string) (ReviewPage, error) {
	if appID <= 0 {
		return ReviewPage{}, errors.New("unknown game")
	}
	if !steamLang.MatchString(lang) {
		lang = "english"
	}
	if kind != "positive" && kind != "negative" {
		kind = "all"
	}
	types := map[string]int{"all": 0, "positive": 1, "negative": 2}
	b, err := fetch(reviewsQuery(map[string]any{"appid": appID, "filter": 0, "languages": []string{lang}, "review_type": types[kind],
		"purchase_type": 1, "num_per_page": ReviewsMax, "day_range": 365}))
	if err != nil {
		return ReviewPage{}, err
	}
	return parseReviews(b)
}

func parseReviews(b []byte) (ReviewPage, error) {
	var res struct {
		Response *struct {
			Summary struct {
				Desc     string `json:"review_score_desc"`
				Positive int    `json:"total_positive"`
				Total    int    `json:"total_reviews"`
			} `json:"query_summary"`
			Reviews []struct {
				ID     json.RawMessage `json:"recommendationid"` // a string, or a number
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
		} `json:"response"`
	}
	if err := json.Unmarshal(b, &res); err != nil {
		return ReviewPage{}, err
	}
	if res.Response == nil {
		return ReviewPage{}, errors.New("Steam didn't answer")
	}
	out := []Review{}
	for _, r := range res.Response.Reviews {
		text, cut := ReviewText(r.Review)
		id := strings.Trim(string(r.ID), `"`)
		if text == "" || !reviewID.MatchString(id) {
			continue
		}
		at := r.Author.AtReview
		if at <= 0 {
			at = r.Author.Forever
		}
		out = append(out, Review{ID: id, Up: r.Up, Text: text, Cut: cut, Hours: hours(at), Total: hours(r.Author.Forever),
			Helpful: r.Helpful, Funny: r.Funny, At: time.Unix(r.Created, 0).UTC(), Free: r.Free, Early: r.Early})
	}
	page := ReviewPage{Items: out}
	if sum := res.Response.Summary; sum.Total > 0 {
		page.Score, page.Count = HTMLText(sum.Desc), sum.Total
		page.Percent = (sum.Positive*100 + sum.Total/2) / sum.Total
	}
	return page, nil
}

// hours rounds minutes to tenths of an hour.
func hours(min float64) float64 { return float64(int(min/6+0.5)) / 10 }
