package core

import (
	"strings"
	"testing"
)

func TestReviewText(t *testing.T) {
	got, cut := ReviewText("[h1]Great[/h1]\r\n[b]Really[/b]   good.\n\n\n\n[url=https://x]link[/url] [spoiler]end[/spoiler]")
	if want := "Great\nReally good.\n\nlink end"; got != want || cut {
		t.Fatalf("got %q (cut %v), want %q", got, cut, want)
	}
	long := strings.Repeat("word ", 400)
	got, cut = ReviewText(long)
	if !cut || !strings.HasSuffix(got, "…") || len([]rune(got)) > reviewMaxRunes+1 {
		t.Fatalf("long review not cut: %d runes, cut %v", len([]rune(got)), cut)
	}
}

func TestFetchReviews(t *testing.T) {
	var asked string
	fetch := func(u string) ([]byte, error) {
		asked = u
		return []byte(`{"success":1,"reviews":[
			{"recommendationid":"123","author":{"playtime_forever":600,"playtime_at_review":125},"review":"[b]Fun[/b]","timestamp_created":1700000000,"voted_up":true,"votes_up":42,"votes_funny":3},
			{"recommendationid":"124","author":{"playtime_forever":60},"review":"   ","voted_up":false},
			{"recommendationid":"../x","author":{},"review":"bad id","voted_up":false},
			{"recommendationid":"125","author":{"playtime_forever":61},"review":"Meh","voted_up":false,"received_for_free":true}]}`), nil
	}
	rs, err := FetchReviews(fetch, 570, "english; drop", "negative")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(asked, "/appreviews/570?") || !strings.Contains(asked, "language=english&") || !strings.Contains(asked, "review_type=negative&") {
		t.Fatalf("asked %s", asked)
	}
	if len(rs) != 2 {
		t.Fatalf("want 2 reviews, got %+v", rs)
	}
	if r := rs[0]; r.ID != "123" || !r.Up || r.Text != "Fun" || r.Hours != 2.1 || r.Total != 10 || r.Helpful != 42 || r.At.Year() != 2023 {
		t.Fatalf("first review %+v", r)
	}
	if r := rs[1]; r.Up || !r.Free || r.Hours != 1 {
		t.Fatalf("second review %+v", r)
	}
	if _, err := FetchReviews(func(string) ([]byte, error) { return []byte(`{"success":2}`), nil }, 570, "english", "all"); err == nil {
		t.Fatal("a failed answer should be an error")
	}
	if _, err := FetchReviews(fetch, 0, "english", "all"); err == nil {
		t.Fatal("app id 0 should be refused")
	}
}
