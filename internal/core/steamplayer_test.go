package core

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestSteamID64(t *testing.T) {
	if got := SteamID64(filepath.Join("C:", "Steam", "userdata", "12345")); got != "76561197960278073" {
		t.Fatalf("got %q", got)
	}
	for _, bad := range []string{"", "anonymous", "0", filepath.Join("x", "76561197960278073")} {
		if SteamID64(bad) != "" {
			t.Fatalf("%q should give none", bad)
		}
	}
}

func TestParsePlayerAchievements(t *testing.T) {
	l, err := ParsePlayerAchievements([]byte(`{"playerstats":{"steamID":"7656","gameName":"G","achievements":[
		{"apiname":"ACH_A","achieved":1,"unlocktime":1700000000,"name":"First","description":"Do it"},
		{"apiname":"ACH_B","achieved":0,"unlocktime":0,"name":"Second"},
		{"apiname":"","achieved":1}],"success":true}}`))
	if err != nil || len(l) != 2 || !l[0].Unlocked || l[0].Name != "First" || l[0].Description != "Do it" || !l[0].At.Equal(time.Unix(1700000000, 0)) || l[1].Unlocked {
		t.Fatalf("%+v %v", l, err)
	}
	if u := AccountUnlocks(l); len(u) != 1 || !u["ACH_A"].Equal(time.Unix(1700000000, 0)) {
		t.Fatalf("unlocks %v", u)
	}
	if _, err := ParsePlayerAchievements([]byte(`{"playerstats":{"error":"Profile is not public","success":false}}`)); !errors.Is(err, ErrSteamPrivate) {
		t.Fatalf("private: %v", err)
	}
	if _, err := ParsePlayerAchievements([]byte(`{"playerstats":{"error":"Requested app has no stats","success":false}}`)); !errors.Is(err, ErrSteamNoStats) {
		t.Fatalf("no stats: %v", err)
	}
	if _, err := ParsePlayerAchievements([]byte(`<html>`)); err == nil {
		t.Fatal("not JSON")
	}
}
