package core

import (
	"reflect"
	"testing"
	"time"
)

func TestGroupCopies(t *testing.T) {
	steam := FoundGame{Key: "steam:1145350", Name: "Hades II", Source: "steam", AppID: 1145350}
	epic := FoundGame{Key: "dir:c:\\epic\\hades ii", Name: "Hades II™", Source: "epic"}
	hydra := FoundGame{Key: "dir:d:\\hydra\\hades2", Name: "Hades 2", Source: "hydra", SteamID: 1145350}
	doom := FoundGame{Key: "steam:379720", Name: "DOOM", Source: "steam", AppID: 379720}
	eternal := FoundGame{Key: "dir:c:\\gog\\doom eternal", Name: "DOOM Eternal", Source: "gog"}
	w3 := FoundGame{Key: "dir:c:\\gog\\witcher 3", Name: "The Witcher 3: Wild Hunt", Source: "gog"}
	w3goty := FoundGame{Key: "dir:d:\\games\\witcher 3 goty", Name: "The Witcher 3: Wild Hunt - Game of the Year Edition", Source: "folder"}
	remaster := FoundGame{Key: "dir:c:\\epic\\w3r", Name: "The Witcher 3: Wild Hunt Remastered", Source: "epic"}
	f1 := FoundGame{Key: "dir:d:\\a\\celeste", Name: "Celeste", Source: "folder"}
	f2 := FoundGame{Key: "dir:e:\\b\\celeste", Name: "Celeste", Source: "shortcut"}

	got := GroupCopies([]FoundGame{steam, epic, hydra, doom, eternal, w3, w3goty, remaster, f1, f2}, nil, nil, nil)
	want := []CopyGroup{
		{Key: "steam:1145350", Keys: []string{steam.Key, epic.Key, hydra.Key}, Primary: steam.Key},
		{Key: "name:thewitcher3wildhunt", Keys: []string{w3.Key, w3goty.Key}, Primary: w3.Key},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("groups:\n got %+v\nwant %+v", got, want)
	}

	// Two Steam ids with the same name are different games.
	other := FoundGame{Key: "dir:x", Name: "Hades II", Source: "gog", SteamID: 99}
	if g := GroupCopies([]FoundGame{steam, other}, nil, nil, nil); len(g) != 0 {
		t.Fatalf("different Steam ids grouped: %+v", g)
	}
	// Split apart by the user.
	if g := GroupCopies([]FoundGame{steam, epic}, []string{"steam:1145350"}, nil, nil); len(g) != 0 {
		t.Fatalf("a separate group came back: %+v", g)
	}
}

func TestGroupCopiesPrimary(t *testing.T) {
	gog := FoundGame{Key: "dir:c:\\gog\\hk", Name: "Hollow Knight", Source: "gog"}
	steam := FoundGame{Key: "steam:367520", Name: "Hollow Knight", Source: "steam", AppID: 367520}
	folder := FoundGame{Key: "dir:d:\\hk", Name: "Hollow Knight", Source: "folder"}
	list := []FoundGame{gog, steam, folder}
	now := time.Now()
	primary := func(chosen map[string]string, recent map[string]Played) string {
		g := GroupCopies(list, nil, chosen, recent)
		if len(g) != 1 {
			t.Fatalf("groups: %+v", g)
		}
		return g[0].Primary
	}
	if p := primary(nil, nil); p != steam.Key {
		t.Fatalf("no plays: %s, want the Steam copy", p)
	}
	recent := map[string]Played{gog.Key: {At: now.Add(-time.Hour)}, folder.Key: {At: now}}
	if p := primary(nil, recent); p != folder.Key {
		t.Fatalf("played: %s, want the one played last", p)
	}
	if p := primary(map[string]string{"steam:367520": gog.Key}, recent); p != gog.Key {
		t.Fatalf("chosen: %s", p)
	}
	if p := primary(map[string]string{"steam:367520": "dir:gone"}, recent); p != folder.Key {
		t.Fatalf("a chosen copy that's gone: %s", p)
	}
	noSteam := GroupCopies([]FoundGame{gog, folder}, nil, nil, nil)
	if len(noSteam) != 1 || noSteam[0].Primary != gog.Key || noSteam[0].Key != "name:hollowknight" {
		t.Fatalf("no Steam copy: %+v", noSteam)
	}
}

func TestCopyName(t *testing.T) {
	for in, want := range map[string]string{
		"DOOM®":                      "doom",
		"Fallout 4: GOTY":            "fallout4",
		"Fallout 4 GOTY Edition":     "fallout4",
		"GOTY":                       "goty",
		"Batman™: Arkham City":       "batmanarkhamcity",
		"Mass Effect 2 (Remastered)": "masseffect2remastered",
	} {
		if got := CopyName(in); got != want {
			t.Errorf("CopyName(%q) = %q, want %q", in, got, want)
		}
	}
}
