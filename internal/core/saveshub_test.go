package core

import (
	"testing"
	"time"
)

func TestSavesOverview(t *testing.T) {
	now := time.Date(2026, 11, 8, 20, 0, 0, 0, time.UTC)
	ago := func(h int) time.Time { return now.Add(-time.Duration(h) * time.Hour) }
	in := []SaveCopy{
		{Key: "game:bg3", Kind: "swap", ID: "s1", At: ago(5), Size: 10},
		{Key: "dir:hollow", Kind: "net", ID: "n1", At: ago(1), Size: 3},
		{Key: "game:bg3", Kind: "point", ID: "p1", At: ago(2), Size: 20},
		{Key: "game:bg3", Kind: "point", ID: "p2", At: ago(9), Size: 20},
		{Key: "dir:hollow", Kind: "net", ID: "n2", At: ago(30), Size: 4},
	}
	recent, games := SavesOverview(in, 3)
	if len(recent) != 3 || recent[0].ID != "n1" || recent[1].ID != "p1" || recent[2].ID != "s1" {
		t.Fatalf("recent: %+v", recent)
	}
	if in[0].ID != "s1" {
		t.Fatal("the input was reordered")
	}
	if len(games) != 2 || games[0].Key != "dir:hollow" || games[1].Key != "game:bg3" {
		t.Fatalf("games: %+v", games)
	}
	if g := games[1]; g.Copies != 3 || g.Points != 2 || g.Size != 50 || !g.Last.Equal(ago(2)) {
		t.Fatalf("bg3: %+v", g)
	}
	if g := games[0]; g.Copies != 2 || g.Points != 0 || g.Size != 7 || !g.Last.Equal(ago(1)) {
		t.Fatalf("hollow: %+v", g)
	}
	if r, g := SavesOverview(nil, 10); len(r) != 0 || len(g) != 0 || g == nil {
		t.Fatalf("empty: %+v %+v", r, g)
	}
}
