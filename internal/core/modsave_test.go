package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestModLooksLikeSave(t *testing.T) {
	mod := func(atRoot bool, rels ...string) *Mod {
		m := &Mod{AtRoot: atRoot}
		for _, r := range rels {
			m.Files = append(m.Files, ModFile{Rel: r, Size: 1})
		}
		return m
	}
	rdr := []string{"SRDR30000", "SRDR30001", "SRDR30014", "settings.xml"}
	cases := []struct {
		what     string
		m        *Mod
		category string
		saves    []string
		want     bool
	}{
		{"an RDR2 prologue skip", mod(true, "SRDR30005"), "", nil, true},
		{"with its picture", mod(true, "SRDR30005", "preview.jpg"), "", nil, true},
		{"in a folder of its own", mod(true, "Prologue Skip/SRDR30003"), "", nil, true},
		{"a GTA V save", mod(true, "SGTA50000"), "", nil, true},
		{"Skyrim's", mod(true, "Save 12 - Dovahkiin.ess"), "", nil, true},
		{"Elden Ring's", mod(true, "ER0000.sl2"), "", nil, true},
		{"shaped like the game's saves", mod(true, "PROFILE_07.dat"), "", []string{"PROFILE_01.dat", "PROFILE_02.dat"}, true},
		{"Nexus says saved games", mod(true, "whatever.bin"), "Saved Games", nil, true},
		{"a mod that matched the game's folder", mod(false, "SRDR30005"), "Saved Games", nil, false},
		{"a mod beside a save", mod(true, "SRDR30005", "dinput8.dll"), "", nil, false},
		{"a dll at the root", mod(true, "dinput8.dll", "ScriptHookRDR2.dll"), "", rdr, false},
		{"a settings file named as the game's", mod(true, "settings.xml"), "", rdr, false},
		{"only pictures", mod(true, "preview.jpg"), "", nil, false},
		{"a save editor", mod(true, "editor.exe"), "Save Editing", nil, false},
		{"no mod", nil, "", nil, false},
	}
	for _, c := range cases {
		if got := ModLooksLikeSave(c.m, c.category, c.saves); got != c.want {
			t.Errorf("%s: %v, want %v", c.what, got, c.want)
		}
	}
}

func TestSaveNames(t *testing.T) {
	dir := t.TempDir()
	deep := filepath.Join(dir, "Profiles", "A1B2C3")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{filepath.Join(deep, "SRDR30000"), filepath.Join(dir, "settings.xml")} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := SaveNames([]string{dir, filepath.Join(dir, "gone")})
	if len(got) != 2 {
		t.Fatalf("names = %v", got)
	}
}

// One save brought into a folder goes beside the saves like it, as Red Dead
// Redemption 2 keeps them in Profiles\<id>.
func TestSaveHome(t *testing.T) {
	dir := t.TempDir()
	prof := filepath.Join(dir, "A1B2C3D4")
	for _, p := range []string{filepath.Join(prof, "SRDR30000"), filepath.Join(prof, "SRDR30001"), filepath.Join(dir, "settings.xml"), filepath.Join(dir, "Other", "slot1.sav")} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for name, want := range map[string]string{
		"SRDR30005":   prof,
		"SRDR30000":   prof,
		"slot9.sav":   filepath.Join(dir, "Other"),
		"mystery.bin": dir,
	} {
		if got := saveHome(dir, name); got != want {
			t.Errorf("saveHome(%s) = %s, want %s", name, got, want)
		}
	}
}
