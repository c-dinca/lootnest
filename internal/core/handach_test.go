package core

import (
	"os"
	"strings"
	"testing"
	"time"
)

// The shape of Steam's IPlayerService/GetGameAchievements answer (the
// percentage comes as text), with made-up achievements.
const gameAchJSON = `{"response":{"achievements":[
 {"internal_name":"ACH_ONE","localized_name":"First Steps","localized_desc":"Finish the first chapter.","icon":"aa11.jpg","icon_gray":"gg.jpg","hidden":true,"player_percent_unlocked":"72.7","internal_key":1},
 {"internal_name":"ACH_TWO","localized_name":"Fisher","localized_desc":"Catch a fish.","icon":"bb22.jpg","hidden":false,"player_percent_unlocked":"1.4"},
 {"internal_name":"ACH_TWO","localized_name":"Twice","icon":"cc.jpg"},
 {"internal_name":"","localized_name":"No id"},
 {"internal_name":"ACH_THREE","localized_name":"Legend","icon":"https://evil.example/x.jpg","player_percent_unlocked":"bad"}
]}}`

func TestParseGameAchievements(t *testing.T) {
	l := ParseGameAchievements([]byte(gameAchJSON), 4242)
	if len(l) != 3 || l[0].ID != "ACH_ONE" || l[1].ID != "ACH_TWO" || l[2].ID != "ACH_THREE" {
		t.Fatalf("list %+v", l)
	}
	if !l[0].Hidden || l[0].Name != "First Steps" || l[0].Description != "Finish the first chapter." || l[0].Percent != 72.7 {
		t.Fatalf("first %+v", l[0])
	}
	if !strings.HasSuffix(l[0].Icon, "/apps/4242/aa11.jpg") || l[1].Percent != 1.4 || l[1].Hidden {
		t.Fatalf("icon or second %+v %+v", l[0], l[1])
	}
	if l[2].Icon != "" || l[2].Percent != 0 {
		t.Fatalf("an icon off Steam's servers, or a percentage that isn't one: %+v", l[2])
	}
	if ParseGameAchievements([]byte("<html>"), 1) != nil {
		t.Fatal("not JSON")
	}
}

func TestHandMarks(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	key := `dir:d:\games\rdr`
	now := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	if err := s.UpdateHandAch(func(h *HandAchievements) error { return h.Mark(key, "p1", "ACH_ONE", true, now) }); err == nil {
		t.Fatal("marking before marking by hand is on")
	}
	err := s.UpdateHandAch(func(h *HandAchievements) error {
		h.SetHandOn(key, 4242, true)
		if err := h.Mark(key, "p1", "ACH_TWO", true, now); err != nil {
			return err
		}
		if err := h.Mark(key, "p1", "ACH_TWO", true, now.Add(time.Hour)); err != nil { // a second mark keeps the first time
			return err
		}
		return h.Mark(key, "p2", "ACH_ONE", true, now)
	})
	if err != nil {
		t.Fatal(err)
	}
	h := s.LoadHandAch()
	g := h.On(key)
	if g == nil || g.AppID != 4242 || !g.Marks["p1"]["ACH_TWO"].Equal(now) {
		t.Fatalf("saved %+v", g)
	}

	list := ParseGameAchievements([]byte(gameAchJSON), 4242)
	set := HandSet(list, 4242, []string{"p1", "p2", "p3"}, g)
	if set.Source != "hand" || set.Total != 3 || len(set.Players) != 3 {
		t.Fatalf("set %+v", set)
	}
	p1, p3 := set.Players["p1"], set.Players["p3"]
	if p1.Unlocked != 1 || p1.List[0].ID != "ACH_TWO" || !p1.List[0].Unlocked || p1.List[0].Seq != 2 || p3.Unlocked != 0 || set.Unlocked != 1 {
		t.Fatalf("each profile's own: %+v / %+v", p1, p3)
	}
	if list[1].Unlocked {
		t.Fatal("the shared list must not change")
	}

	// Off keeps the marks; on again brings them back.
	_ = s.UpdateHandAch(func(h *HandAchievements) error { h.SetHandOn(key, 0, false); return nil })
	if h := s.LoadHandAch(); h.On(key) != nil || len(h.Games[key].Marks["p1"]) != 1 {
		t.Fatal("off keeps the marks")
	}
	_ = s.UpdateHandAch(func(h *HandAchievements) error {
		h.SetHandOn(key, 0, true)
		return h.Mark(key, "p2", "ACH_ONE", false, now)
	})
	if h := s.LoadHandAch(); h.On(key).AppID != 4242 || h.On(key).Marks["p2"] != nil {
		t.Fatalf("unmark: %+v", h.On(key))
	}

	// A new key (the game moved) takes the marks; RekeyGame does it.
	to := `dir:e:\games\rdr`
	if err := s.RekeyGame(key, to); err != nil {
		t.Fatal(err)
	}
	if h := s.LoadHandAch(); h.Games[key] != nil || h.On(to) == nil || len(h.On(to).Marks["p1"]) != 1 {
		t.Fatalf("rekeyed %+v", h.Games)
	}

	// A broken file is kept aside, never written over.
	if err := os.WriteFile(s.HandAchPath(), []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateHandAch(func(h *HandAchievements) error { h.SetHandOn(key, 1, true); return nil }); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(s.HandAchPath() + ".broken"); err != nil || string(b) != "{broken" {
		t.Fatal("the broken file must be kept")
	}
}

func TestHandRekeyMerges(t *testing.T) {
	a, b := time.Unix(100, 0), time.Unix(200, 0)
	h := HandAchievements{Games: map[string]*HandGame{
		"old": {AppID: 7, Marks: map[string]map[string]time.Time{"p1": {"X": a, "Y": b}}},
		"new": {Off: true, Marks: map[string]map[string]time.Time{"p1": {"X": b}}},
	}}
	if !h.Rekey("old", "new") || h.Games["old"] != nil {
		t.Fatal("moved")
	}
	n := h.Games["new"]
	if n.AppID != 7 || n.Off || !n.Marks["p1"]["X"].Equal(a) || !n.Marks["p1"]["Y"].Equal(b) {
		t.Fatalf("merged %+v", n)
	}
}
