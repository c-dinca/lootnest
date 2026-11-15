package core

import (
	"net/url"
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
		return []byte(`{"response":{"query_summary":{"num_reviews":4,"review_score_desc":"Mostly Positive","total_positive":75,"total_reviews":100},"reviews":[
			{"recommendationid":123,"author":{"playtime_forever":600,"playtime_at_review":125},"review":"[b]Fun[/b]","timestamp_created":1700000000,"voted_up":true,"votes_up":42,"votes_funny":3},
			{"recommendationid":"124","author":{"playtime_forever":60},"review":"   ","voted_up":false},
			{"recommendationid":"../x","author":{},"review":"bad id","voted_up":false},
			{"recommendationid":"125","author":{"playtime_forever":61},"review":"Meh","voted_up":false,"received_for_free":true}]}}`), nil
	}
	rs, err := FetchReviews(fetch, 570, "english; drop", "negative")
	if err != nil {
		t.Fatal(err)
	}
	q, _ := url.QueryUnescape(asked)
	if !strings.Contains(q, "IUserReviewsService/GetAppReviews/v1/?input_json=") || !strings.Contains(q, `"appid":570`) || !strings.Contains(q, `"languages":["english"]`) || !strings.Contains(q, `"review_type":2`) || !strings.Contains(q, `"day_range":365`) {
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
	if page, _ := FetchReviewPage(fetch, 570, "english", "all"); page.Score != "Mostly Positive" || page.Percent != 75 || page.Count != 100 || len(page.Items) != 2 {
		t.Fatalf("page %+v", page)
	}
	if _, err := FetchReviews(func(string) ([]byte, error) { return []byte(`{"error":"x"}`), nil }, 570, "english", "all"); err == nil {
		t.Fatal("a failed answer should be an error")
	}
	if _, err := FetchReviews(fetch, 0, "english", "all"); err == nil {
		t.Fatal("app id 0 should be refused")
	}
}
