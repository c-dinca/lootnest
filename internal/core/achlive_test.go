package core

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestNewUnlocks(t *testing.T) {
	t0 := time.Unix(1000, 0)
	before := map[string]time.Time{"A": t0}
	after := map[string]time.Time{"A": t0, "C": t0.Add(2 * time.Second), "B": t0.Add(time.Second)}
	if got := NewUnlocks(before, after); !reflect.DeepEqual(got, []string{"B", "C"}) {
		t.Fatalf("got %v", got)
	}
	if got := NewUnlocks(after, after); len(got) != 0 {
		t.Fatalf("nothing new expected, got %v", got)
	}
}

func TestGlobalPercentages(t *testing.T) {
	b := []byte(`{"achievementpercentages":{"achievements":[{"name":"ACH_WIN","percent":"4.2"},{"name":"ach_start","percent":87.5},{"name":"bad","percent":"x"}]}}`)
	p := ParseGlobalPercentages(b)
	if p["ACH_WIN"] != 4.2 || p["ach_start"] != 87.5 || len(p) != 2 {
		t.Fatalf("got %v", p)
	}
	list := []Achievement{{ID: "ACH_WIN"}, {ID: "ACH_START"}, {ID: "OTHER", Percent: 3}}
	ApplyPercentages(list, p)
	if list[0].Percent != 4.2 || list[1].Percent != 87.5 || list[2].Percent != 3 {
		t.Fatalf("got %+v", list)
	}
	if ParseGlobalPercentages([]byte(`{}`)) != nil {
		t.Fatal("empty expected")
	}
}

func TestRarity(t *testing.T) {
	cases := map[float64]string{0: "", 1.2: "Ultra rare, 1.2% of players have it", 4.25: "Rare, 4.2% of players have it", 12.6: "Uncommon, 13% of players have it", 80: "Common, 80% of players have it"}
	for p, want := range cases {
		if got := RarityText(p); got != want {
			t.Errorf("%v: got %q, want %q", p, got, want)
		}
	}
}

func TestAntiCheat(t *testing.T) {
	d := t.TempDir()
	if AntiCheat(d) != "" {
		t.Fatal("none expected")
	}
	if err := os.MkdirAll(filepath.Join(d, "Binaries", "EasyAntiCheat"), 0o755); err != nil {
		t.Fatal(err)
	}
	if AntiCheat(d) != "Easy Anti-Cheat" {
		t.Fatal("EAC one level down")
	}
	d2 := t.TempDir()
	_ = os.WriteFile(filepath.Join(d2, "BEService_x64.exe"), []byte("x"), 0o644)
	if AntiCheat(d2) != "BattlEye" {
		t.Fatal("BattlEye")
	}
}
