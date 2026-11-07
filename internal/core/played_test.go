package core

import (
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
