package core

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestSteamControllers(t *testing.T) {
	for _, v := range []uint32{0x07564427, 0x07564428, 0x07564429} {
		steam := t.TempDir()
		appInfoFile(t, filepath.Join(steam, "appcache", "appinfo.vdf"), v, [][3]string{
			{"730", "Counter-Strike 2", "Game|none"},
			{"1174180", "Red Dead Redemption 2", "Game|partial"},
			{"1086940", "Baldur's Gate 3", "Game|full"},
			{"1245620", "Elden Ring", "Game|Full"}, // controller_support as text
			{"620", "Portal 2", "Game"},            // no categories: nobody says
			{"999", "Not asked for", "Game|full"},
		})
		got := SteamControllers(steam, map[int]bool{730: true, 1174180: true, 1086940: true, 1245620: true, 620: true, 12345: true})
		want := map[int]string{730: "none", 1174180: "partial", 1086940: "full", 1245620: "full"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("version %x: %v, want %v", v, got, want)
		}
	}
	if got := SteamControllers("", map[int]bool{1: true}); len(got) != 0 {
		t.Errorf("no Steam: %v", got)
	}
}

func TestPadGuessAndDetails(t *testing.T) {
	for _, c := range []struct{ source, name, want string }{
		{"riot", "VALORANT", "none"}, {"riot", "League of Legends", "none"}, {"riot", "2XKO", ""},
		{"battlenet", "StarCraft II", "none"}, {"battlenet", "Diablo IV", ""}, {"epic", "Fortnite", ""}, {"folder", "Hearthstone®", "none"},
	} {
		if got := PadGuess(c.source, c.name); got != c.want {
			t.Errorf("PadGuess(%q, %q) = %q, want %q", c.source, c.name, got, c.want)
		}
	}
	for _, c := range []struct {
		d    *GameDetails
		want string
	}{
		{nil, ""}, {&GameDetails{Unknown: true, ReqsRead: true}, ""}, {&GameDetails{Controller: "partial"}, "partial"},
		{&GameDetails{ReqsRead: true}, "none"}, {&GameDetails{}, ""},
	} {
		if got := DetailsPad(c.d); got != c.want {
			t.Errorf("DetailsPad(%+v) = %q, want %q", c.d, got, c.want)
		}
	}
}
