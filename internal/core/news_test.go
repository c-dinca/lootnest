package core

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const newsReply = `{"appnews":{"appid":730,"newsitems":[
 {"title":"Counter-Strike 2 Update","url":"https://steamstore-a.akamaihd.net/news/externalpost/steam_community_announcements/1","contents":"[list][*]Fixed a bug &amp; more[/list]","date":1793840047,"tags":["patchnotes"]},
 {"title":"Major tournament this weekend","url":"https://store.steampowered.com/news/app/730/view/2","contents":"<p>Watch it live</p>","date":1793000000},
 {"title":"Hotfix 1.2.3","url":"https://steamcommunity.com/games/730/announcements/detail/3","contents":"","date":1792000000},
 {"title":"Bad link","url":"http://evil.example/4","contents":"","date":1791000000}
]}}`

func TestParseNews(t *testing.T) {
	items, err := ParseNews([]byte(newsReply))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("links outside Steam are dropped: %+v", items)
	}
	if !items[0].Update || items[0].Summary != "Fixed a bug & more" {
		t.Fatalf("patch notes, as plain text: %+v", items[0])
	}
	if items[1].Update || items[1].Summary != "Watch it live" {
		t.Fatalf("an announcement isn't an update: %+v", items[1])
	}
	if !items[2].Update {
		t.Fatal("a hotfix in the title is an update")
	}
}

func TestFetchNewsCaches(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "news")
	calls := 0
	fetch := func(u string) ([]byte, error) {
		calls++
		if !strings.HasPrefix(u, newsURL) || !strings.HasSuffix(u, "730") {
			t.Fatalf("unexpected URL %s", u)
		}
		return []byte(newsReply), nil
	}
	n, err := FetchNews(fetch, dir, 730)
	if err != nil || len(n.Items) != 3 || calls != 1 {
		t.Fatalf("first fetch: %+v %v", n, err)
	}
	if n2, _ := FetchNews(fetch, dir, 730); calls != 1 || len(n2.Items) != 3 {
		t.Fatal("a fresh feed is read from disk, not fetched again")
	}
	offline := func(string) ([]byte, error) { return nil, errors.New("offline") }
	if _, err := FetchNews(offline, dir, 570); err == nil {
		t.Fatal("offline with nothing saved is an error")
	}
}

func TestStoreAppID(t *testing.T) {
	dir := t.TempDir()
	calls := 0
	fetch := func(u string) ([]byte, error) {
		calls++
		return []byte(`{"items":[{"id":3240220,"name":"Grand Theft Auto V Enhanced"}]}`), nil
	}
	if id, err := StoreAppID(fetch, dir, "Grand Theft Auto V"); err != nil || id != 3240220 {
		t.Fatalf("got %d %v", id, err)
	}
	if id, _ := StoreAppID(fetch, dir, "Grand Theft Auto V"); id != 3240220 || calls != 1 {
		t.Fatal("the answer is remembered")
	}
}

func TestNewerThanCopy(t *testing.T) {
	items, _ := ParseNews([]byte(newsReply))
	copyDate := time.Unix(1792500000, 0)
	if u := NewerThanCopy(items, copyDate); u == nil || u.Title != "Counter-Strike 2 Update" {
		t.Fatalf("the newest update after the copy: %+v", u)
	}
	if u := NewerThanCopy(items, time.Unix(1793900000, 0)); u != nil {
		t.Fatalf("an up-to-date copy has nothing newer: %+v", u)
	}
}
