package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompetitiveGame(t *testing.T) {
	for _, c := range []struct {
		id   int
		name string
		want bool
	}{
		{730, "Counter-Strike 2", true},
		{0, "Counter-Strike 2", true},
		{0, "VALORANT", true},
		{0, "League of Legends", true},
		{1174180, "Red Dead Redemption 2", false},
		{0, "Baldur's Gate 3", false},
		{0, "Rusty Lake Hotel", false},
		{0, "Deadlock", true},
	} {
		if got := CompetitiveGame(c.id, c.name, "") != ""; got != c.want {
			t.Errorf("%d %q: %v, want %v", c.id, c.name, got, c.want)
		}
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "vgc.exe"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if why := CompetitiveGame(0, "Some Game", dir); why != "it has Vanguard" {
		t.Errorf("Vanguard: %q", why)
	}
}

func TestAssignRunGames(t *testing.T) {
	c := &Config{
		Players: []Player{{ID: "p1", Name: "Cristian"}, {ID: "r1", Name: "Skip Prologue", Of: "p1"}, {ID: "r2", Name: "Two", Of: "p1"}, {ID: "r3", Name: "None", Of: "p1"}},
		Games:   []Game{{ID: "rdr2"}, {ID: "cs2"}},
	}
	files := map[string]bool{"p1/rdr2": true, "p1/cs2": true, "r1/rdr2": true, "r2/rdr2": true, "r2/cs2": true}
	has := func(p, g string) bool { return files[p+"/"+g] }
	if !AssignRunGames(c, has) {
		t.Fatal("nothing changed")
	}
	got := map[string]string{}
	for _, p := range c.Players {
		got[p.ID] = p.Game
	}
	if got["p1"] != "" || got["r1"] != "rdr2" || got["r2"] != "" || got["r3"] != "" {
		t.Errorf("games: %v", got)
	}
	if AssignRunGames(c, has) {
		t.Error("changed twice")
	}
	r1, r2 := c.Player("r1"), c.Player("r2")
	if !RunFor(r1, "rdr2", has) || RunFor(r1, "cs2", has) {
		t.Error("r1 is for RDR2 only")
	}
	if !RunFor(r2, "cs2", has) || !RunFor(r2, "rdr2", has) {
		t.Error("an older run shows where it has files")
	}
	if !RunFor(c.Player("p1"), "cs2", has) {
		t.Error("a person shows everywhere")
	}
}

func TestValidatePlayersRuns(t *testing.T) {
	ps := []Player{{ID: "p1", Name: "Cristian"}, {ID: "r1", Name: "Hardcore", Of: "p1", Game: "rdr2"}, {ID: "r2", Name: "Hardcore", Of: "p1", Game: "bg3"}, {ID: "p2", Name: "Mihai", Game: "x"}}
	if err := ValidatePlayers(ps); err != nil {
		t.Fatalf("same run name in two games: %v", err)
	}
	if ps[3].Game != "" {
		t.Error("a person has no game")
	}
	ps = append(ps, Player{ID: "r3", Name: "hardcore", Of: "p1", Game: "rdr2"})
	if err := ValidatePlayers(ps); err == nil {
		t.Error("two runs with one name in one game")
	}
	many := []Player{{ID: "p1", Name: "A"}}
	for i := 0; i <= MaxRunsPerGame; i++ {
		many = append(many, Player{ID: "r" + string(rune('a'+i)), Name: "Run " + string(rune('a'+i)), Of: "p1", Game: "g"})
	}
	if err := ValidatePlayers(many); err == nil {
		t.Error("too many runs in one game")
	}
}
