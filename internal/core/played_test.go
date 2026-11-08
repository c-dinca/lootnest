package core

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestShelf(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	if err := s.SetShelf("p1", "steam:620", ShelfEntry{Status: "finished", Rating: 5, Note: " Great \r\n co-op "}, now); err != nil {
		t.Fatal(err)
	}
	e := s.LoadShelf()["p1"]["steam:620"]
	if e.Status != "finished" || e.Rating != 5 || e.Note != "Great \n co-op" || !e.Updated.Equal(now) {
		t.Fatalf("%+v", e)
	}
	for _, bad := range []ShelfEntry{{Status: "owned"}, {Rating: 6}, {Rating: -1}} {
		if err := s.SetShelf("p1", "steam:620", bad, now); err == nil {
			t.Fatalf("%+v accepted", bad)
		}
	}
	if err := s.SetShelf("p1", "steam:620", ShelfEntry{}, now); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.LoadShelf()["p1"]["steam:620"]; ok {
		t.Fatal("an empty entry removes the game from the shelf")
	}
}

func TestShelfPatch(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	str := func(v string) *string { return &v }
	four, off := 4, int64(6*3600)
	_ = s.SetShelf("p2", "steam:620", ShelfEntry{Status: "next"}, now)
	_ = s.PatchShelf("p1", "steam:620", ShelfPatch{Status: str("playing")}, now)
	if err := s.PatchShelf("p1", "steam:620", ShelfPatch{Rating: &four, Note: str(" Boss at 3 "), Offset: &off}, now); err != nil {
		t.Fatal(err)
	}
	sh := s.LoadShelf()
	e := sh["p1"]["steam:620"]
	if e.Status != "playing" || e.Rating != 4 || e.Note != "Boss at 3" || e.Offset != off || sh["p2"]["steam:620"].Status != "next" {
		t.Fatalf("a patch keeps the other fields and entries: %+v", sh)
	}
	zero := 0
	_ = s.PatchShelf("p1", "steam:620", ShelfPatch{Rating: &zero}, now)
	if e := s.LoadShelf()["p1"]["steam:620"]; e.Rating != 0 || e.Note != "Boss at 3" || e.Status != "playing" || e.Offset != off {
		t.Fatalf("clearing the rating keeps the rest: %+v", e)
	}
	big := int64(maxOffset + 1)
	if s.PatchShelf("p1", "steam:620", ShelfPatch{Offset: &big}, now) == nil {
		t.Fatal("an offset over the limit is refused")
	}
	if s.LoadShelf()["p1"]["steam:620"].Offset != off {
		t.Fatal("a refused patch changes nothing")
	}
}

func TestShelfBrokenKept(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	_ = os.WriteFile(s.ShelfPath(), []byte(`{"p1":{"steam:620":{"status":"pla`), 0o644)
	four := 4
	if err := s.PatchShelf("p1", "game:a", ShelfPatch{Rating: &four}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(s.ShelfPath() + ".broken"); err != nil || !strings.Contains(string(b), "steam:620") {
		t.Fatalf("a shelf.json that doesn't parse is kept: %q %v", b, err)
	}
	if s.LoadShelf()["p1"]["game:a"].Rating != 4 {
		t.Fatal("the change is still saved")
	}
}

func TestWithOffsets(t *testing.T) {
	pt := map[string]map[string]int64{"game:a": {"p1": 44 * 3600, "p2": 3600}}
	shelf := map[string]map[string]ShelfEntry{
		"p1": {"game:a": {Offset: 6 * 3600}, "game:b": {Offset: 1800}},
		"p2": {"game:a": {Offset: -2 * 3600, Rating: 3}},
		"p3": {"game:a": {Status: "playing"}},
	}
	got := WithOffsets(pt, shelf)
	if got["game:a"]["p1"] != 50*3600 || got["game:a"]["p2"] != 0 || got["game:b"]["p1"] != 1800 {
		t.Fatalf("positive, clamped at 0, and a game with no sessions: %+v", got)
	}
	if _, ok := got["game:a"]["p3"]; ok {
		t.Fatal("no offset, no change")
	}
}

func TestActivity(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	loc := time.Local
	start := time.Date(2026, 11, 6, 23, 0, 0, 0, loc)
	if err := s.AddActivity("game:a", "p1", start, start.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	_ = s.AddActivity("game:b", "p1", start.Add(3*time.Hour), start.Add(3*time.Hour+30*time.Minute))
	_ = s.AddActivity("game:b", "p1", start, start.Add(20*time.Second)) // too short
	a := s.LoadActivity()
	d := a.Days["p1"]
	if d["2026-11-06"]["game:a"] != 3600 || d["2026-11-07"]["game:a"] != 3600 || d["2026-11-07"]["game:b"] != 1800 {
		t.Fatalf("split over days: %+v", d)
	}
	if a.Sessions["p1"] != 2 || a.Longest["p1"].Key != "game:a" || a.Longest["p1"].Seconds != 7200 {
		t.Fatalf("%+v %+v", a.Sessions, a.Longest)
	}
	// Days older than activityDays are dropped.
	later := start.AddDate(0, 0, activityDays+5)
	_ = s.AddActivity("game:a", "p1", later, later.Add(time.Hour))
	if _, ok := s.LoadActivity().Days["p1"]["2026-11-06"]; ok {
		t.Fatal("old days are dropped")
	}
}

func TestProfileFields(t *testing.T) {
	ps := []Player{{Name: "Ana", Color: "#123456", BannerY: 140,
		Sections: []string{"stats", "bogus", "stats", "recent"}, Bio: "line\r\nline"}}
	if err := ValidatePlayers(ps); err != nil {
		t.Fatal(err)
	}
	p := ps[0]
	if p.BannerY != 100 || len(p.Sections) != 2 || p.Sections[1] != "recent" || p.Bio != "line\nline" {
		t.Fatalf("%+v", p)
	}
	ps[0].Bio = "1\n2\n3\n4\n5\n6\n7"
	if ValidatePlayers(ps) == nil {
		t.Fatal("a 7-line about is refused")
	}
}
