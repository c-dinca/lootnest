package core

import (
	"path/filepath"
	"testing"
)

func TestSuggestPathsFromSaveDB(t *testing.T) {
	f := newFixture(t)
	inst := filepath.Join(f.root, "Games", "Night Tale")
	db := &SaveDB{Games: []DBGame{
		{Name: "Night Tale", Files: []string{
			"<winAppData>/NightStudio/Tale/*.sav", // two files: their folder
			"<winLocalAppData>/NightTale/settings.ini",
			"<base>/game/saves",    // Ren'Py, inside the game's folder
			"<base>/*.cfg",         // files right in the game's folder stay files
			"<winDocuments>/*.dat", // Documents itself is never swapped
		}},
		{Name: "Other", Steam: 777, Files: []string{"<winAppData>/Other"}},
	}}
	db.index()
	write(t, filepath.Join(f.appdata, "NightStudio", "Tale", "slot1.sav"), "1")
	write(t, filepath.Join(f.appdata, "NightStudio", "Tale", "slot2.sav"), "2")
	write(t, filepath.Join(f.home, "AppData", "Local", "NightTale", "settings.ini"), "x")
	write(t, filepath.Join(inst, "game", "saves", "1-1-LT1.save"), "s")
	write(t, filepath.Join(inst, "a.cfg"), "a")
	write(t, filepath.Join(inst, "b.cfg"), "b")
	write(t, filepath.Join(f.home, "Documents", "one.dat"), "d")
	write(t, filepath.Join(f.home, "Documents", "two.dat"), "d")

	// Off (no list): only name matches.
	for _, s := range f.env.SuggestPaths("Night Tale", inst, 0) {
		if s.Reason == "Listed on PCGamingWiki" {
			t.Fatalf("no list, yet %+v", s)
		}
	}

	f.env.SaveDB = func() *SaveDB { return db }
	got := map[string]bool{}
	sug := f.env.SuggestPaths("Night Tale", inst, 0)
	for _, s := range sug {
		if s.Reason == "Listed on PCGamingWiki" {
			got[s.Resolved] = true
		}
	}
	want := []string{
		filepath.Join(f.appdata, "NightStudio", "Tale"),
		filepath.Join(f.home, "AppData", "Local", "NightTale", "settings.ini"),
		filepath.Join(inst, "game", "saves"),
		filepath.Join(inst, "a.cfg"),
		filepath.Join(inst, "b.cfg"),
		filepath.Join(f.home, "Documents", "one.dat"), // files, never Documents
		filepath.Join(f.home, "Documents", "two.dat"),
	}
	for _, w := range want {
		if !got[w] {
			t.Errorf("missing %s in %+v", w, sug)
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %d listed suggestions, want %d: %+v", len(got), len(want), sug)
	}
	if got[inst] || got[filepath.Join(f.home, "Documents")] {
		t.Error("the game's folder or Documents must never be suggested")
	}
	if sug[0].Reason != "Listed on PCGamingWiki" {
		t.Errorf("the list's locations should come first: %+v", sug[0])
	}

	// Found by Steam id, whatever the name.
	write(t, filepath.Join(f.appdata, "Other", "x"), "1")
	sug = f.env.SuggestPaths("Renamed", "Renamed", 777)
	if len(sug) == 0 || sug[0].Resolved != filepath.Join(f.appdata, "Other") {
		t.Fatalf("by Steam id: %+v", sug)
	}
}
