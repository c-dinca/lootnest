package core

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Game news and updates (opt-in: Settings > "Show game news and updates").
// Off by default; with it off, nothing here is ever called.
//
// A game's announcements and patch notes come from Steam's public news feed
// for the game, by its Steam app id (for a game outside Steam, the id the
// store gives for its name). Only that id, or the name, is sent.
// ---------------------------------------------------------------------------

const (
	newsURL = "https://api.steampowered.com/ISteamNews/GetNewsForApp/v2/?count=8&maxlength=240&format=json&feeds=steam_community_announcements&appid="
	// Feeds are read again after this long.
	newsMaxAge = 6 * time.Hour
)

type NewsItem struct {
	Title   string    `json:"title"`
	URL     string    `json:"url"`
	Date    time.Time `json:"date"`
	Summary string    `json:"summary"`
	// Update marks patch notes: a new version of the game.
	Update bool `json:"update"`
}

type GameNews struct {
	AppID   int        `json:"appId"`
	Fetched time.Time  `json:"fetched"`
	Items   []NewsItem `json:"items"`
}

var (
	bbTags     = regexp.MustCompile(`\[/?[a-zA-Z0-9*]+(=[^\]]*)?\]`)
	htmlTags   = regexp.MustCompile(`<[^>]+>`)
	spaces     = regexp.MustCompile(`\s+`)
	updateWord = regexp.MustCompile(`(?i)\b(patch|update|hotfix|v?\d+\.\d+(\.\d+)*)\b`)
)

// plainText turns a feed's BBCode or HTML snippet into one line of text.
func plainText(s string) string {
	s = bbTags.ReplaceAllString(s, " ")
	s = htmlTags.ReplaceAllString(s, " ")
	s = strings.NewReplacer("&amp;", "&", "&quot;", `"`, "&#39;", "'", "&lt;", "<", "&gt;", ">", "&nbsp;", " ").Replace(s)
	return strings.TrimSpace(spaces.ReplaceAllString(s, " "))
}

// ParseNews reads Steam's news feed reply.
func ParseNews(b []byte) ([]NewsItem, error) {
	var res struct {
		AppNews struct {
			NewsItems []struct {
				Title    string   `json:"title"`
				URL      string   `json:"url"`
				Contents string   `json:"contents"`
				Date     int64    `json:"date"`
				Tags     []string `json:"tags"`
			} `json:"newsitems"`
		} `json:"appnews"`
	}
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, fmt.Errorf("unexpected reply from Steam's news feed: %w", err)
	}
	out := []NewsItem{}
	for _, n := range res.AppNews.NewsItems {
		if !SafeNewsURL(n.URL) || strings.TrimSpace(n.Title) == "" {
			continue
		}
		it := NewsItem{Title: plainText(n.Title), URL: n.URL, Date: time.Unix(n.Date, 0).UTC(), Summary: plainText(n.Contents)}
		for _, t := range n.Tags {
			if t == "patchnotes" {
				it.Update = true
			}
		}
		if !it.Update && updateWord.MatchString(it.Title) {
			it.Update = true
		}
		out = append(out, it)
	}
	return out, nil
}

// SafeNewsURL accepts only links to Steam's own news pages.
func SafeNewsURL(u string) bool {
	p, err := url.Parse(u)
	if err != nil || p.Scheme != "https" {
		return false
	}
	h := strings.ToLower(p.Hostname())
	for _, ok := range []string{"store.steampowered.com", "steamcommunity.com", "steamstore-a.akamaihd.net", "clan.akamai.steamstatic.com"} {
		if h == ok {
			return true
		}
	}
	return false
}

// FetchNews returns a game's news, from <dir>\<appid>.json when it was read
// less than six hours ago, else from Steam (and saved). When Steam can't be
// reached, an older saved copy is returned with the error.
func FetchNews(fetch Fetch, dir string, appID int) (GameNews, error) {
	path := filepath.Join(dir, strconv.Itoa(appID)+".json")
	var saved GameNews
	if b, err := os.ReadFile(path); err == nil && json.Unmarshal(b, &saved) == nil && saved.AppID == appID {
		if time.Since(saved.Fetched) < newsMaxAge {
			return saved, nil
		}
	}
	b, err := fetch(newsURL + strconv.Itoa(appID))
	if err != nil || b == nil {
		if err == nil {
			err = fmt.Errorf("Steam has no news feed for this game")
		}
		return saved, err
	}
	items, err := ParseNews(b)
	if err != nil {
		return saved, err
	}
	n := GameNews{AppID: appID, Fetched: time.Now().UTC(), Items: items}
	if os.MkdirAll(dir, 0o755) == nil {
		_ = writeJSONAtomic(path, n)
	}
	return n, nil
}

// StoreAppID finds the Steam app id for a game outside Steam by its name,
// remembering the answer (also "not on the store") in <dir>\<name>.appid.
func StoreAppID(fetch Fetch, dir, name string) (int, error) {
	key := CoverKey(name)
	if key == "" {
		return 0, nil
	}
	path := filepath.Join(dir, key+".appid")
	if b, err := os.ReadFile(path); err == nil {
		id, _ := strconv.Atoi(strings.TrimSpace(string(b)))
		return id, nil
	}
	id, err := findStoreAppLoose(fetch, name)
	if err != nil {
		return 0, err
	}
	if os.MkdirAll(dir, 0o755) == nil {
		_ = os.WriteFile(path, []byte(strconv.Itoa(id)), 0o644)
	}
	return id, nil
}

// NewerThanCopy is the most recent update released after a game's files
// were last changed on this PC, if any: for games no store keeps up to date.
func NewerThanCopy(items []NewsItem, filesChanged time.Time) *NewsItem {
	if filesChanged.IsZero() {
		return nil
	}
	for i := range items {
		it := items[i]
		if it.Update && it.Date.After(filesChanged.Add(24*time.Hour)) {
			return &it
		}
	}
	return nil
}
