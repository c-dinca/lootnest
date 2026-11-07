package app

import (
	"testing"

	"lootnest/internal/core"
)

func TestTrimRunes(t *testing.T) {
	for _, c := range []struct {
		in   string
		n    int
		want string
	}{
		{"  Cristian  ", 24, "Cristian"},
		{"Ștefănescu-Ionescu", 5, "Ștefă"},
		{"Cristian from Before the boss", 24, "Cristian from Before the"},
		{"Alexandra from ", 14, "Alexandra from"},
		{"", 4, ""},
	} {
		if got := trimRunes(c.in, c.n); got != c.want {
			t.Errorf("trimRunes(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
		}
	}
}

func TestTrimWords(t *testing.T) {
	for _, c := range []struct {
		in   string
		n    int
		want string
	}{
		{"Before going back to Forge", 10, "Before"},
		{"Save point, 7 Nov, 14:05", 10, "Save point"},
		{"Save point, 7 Nov, 14:05", 11, "Save point"},
		{"Quick save point, 15:04", 10, "Quick save"},
		{"Bossfightnumberthree", 10, "Bossfightn"},
		{"Boss", 10, "Boss"},
	} {
		if got := trimWords(c.in, c.n); got != c.want {
			t.Errorf("trimWords(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
		}
	}
}

// Two unnamed points branched in a row get different names.
func TestFreeName(t *testing.T) {
	ps := []core.Player{{Name: "Cristian"}, {Name: "cristian from save point"}}
	if got := freeName("Cristian from Save point", ps); got != "Cristian from Save 2" {
		t.Fatalf("got %q", got)
	}
	ps = append(ps, core.Player{Name: "Cristian from Save 2"})
	if got := freeName("Cristian from Save point", ps); got != "Cristian from Save 3" {
		t.Fatalf("got %q", got)
	}
	if got := freeName("Ana trying", ps); got != "Ana trying" {
		t.Fatalf("got %q", got)
	}
}
