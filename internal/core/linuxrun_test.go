//go:build !windows

package core

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// lookIn finds commands in one folder, as exec.LookPath does on PATH.
func lookIn(dir string) func(string) (string, error) {
	return func(name string) (string, error) {
		p := name
		if !filepath.IsAbs(name) {
			p = filepath.Join(dir, name)
		}
		if st, err := os.Stat(p); err == nil && st.Mode()&0o111 != 0 {
			return p, nil
		}
		return "", os.ErrNotExist
	}
}

func mkCommand(t *testing.T, dir, name string) {
	t.Helper()
	write(t, filepath.Join(dir, name), "#!/bin/sh\n")
	if err := os.Chmod(filepath.Join(dir, name), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestSplitArgs(t *testing.T) {
	for in, want := range map[string][]string{
		``:                                nil,
		`-windowed -fps 60`:               {"-windowed", "-fps", "60"},
		`--profile "My Saves" -x`:         {"--profile", "My Saves", "-x"},
		`'single quoted' "" after`:        {"single quoted", "", "after"},
		`a\"b  -skip`:                     {`a"b`, "-skip"},
		`  --dir="C:\Games\Ashen Ring"  `: {`--dir=C:\Games\Ashen Ring`},
	} {
		if got := SplitArgs(in); !reflect.DeepEqual(got, want) {
			t.Errorf("SplitArgs(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRunnerFor(t *testing.T) {
	bin := t.TempDir()
	look := lookIn(bin)
	pfx := "/home/alex/Games/ashen/pfx"

	native, err := RunnerFor("/home/alex/Games/tux/tuxracer", []string{"-f"}, "", RunnerAuto, 0, look)
	if err != nil || !reflect.DeepEqual(native.Argv, []string{"/home/alex/Games/tux/tuxracer", "-f"}) || native.Env != nil {
		t.Fatalf("a native program starts as is: %+v %v", native, err)
	}
	if _, err := RunnerFor("/g/game.exe", nil, pfx, RunnerAuto, 0, look); err != ErrNoRunner {
		t.Fatalf("no Wine, no umu: %v", err)
	}

	mkCommand(t, bin, "wine")
	w, err := RunnerFor("/g/game.exe", []string{"-dx11"}, pfx, RunnerAuto, 0, look)
	if err != nil || w.Argv[0] != filepath.Join(bin, "wine") || w.Argv[1] != "/g/game.exe" || w.Argv[2] != "-dx11" ||
		!reflect.DeepEqual(w.Env, []string{"WINEPREFIX=" + pfx}) {
		t.Fatalf("Wine, in the game's prefix: %+v %v", w, err)
	}
	if m, _ := RunnerFor("/g/setup.msi", nil, "", RunnerWine, 0, look); strings.Join(m.Argv[1:], " ") != "msiexec /i /g/setup.msi" || m.Env != nil {
		t.Fatalf("an .msi through msiexec, no prefix: %+v", m)
	}
	if _, err := RunnerFor("/g/game.exe", nil, pfx, RunnerUmu, 0, look); err == nil || !strings.Contains(err.Error(), "umu") {
		t.Fatalf("umu asked for and missing: %v", err)
	}

	mkCommand(t, bin, "umu-run")
	u, err := RunnerFor("/g/game.exe", nil, pfx, RunnerAuto, 1245620, look)
	if err != nil || u.Argv[0] != filepath.Join(bin, "umu-run") || !reflect.DeepEqual(u.Env, []string{"GAMEID=umu-1245620", "WINEPREFIX=" + pfx}) {
		t.Fatalf("umu-launcher first, with the game's id: %+v %v", u, err)
	}
	if w2, _ := RunnerFor("/g/game.exe", nil, "", RunnerWine, 1245620, look); w2.Argv[0] != filepath.Join(bin, "wine") {
		t.Fatalf("Wine when it's the choice: %+v", w2)
	}
	if u2, _ := RunnerFor("/g/game.exe", nil, "", RunnerAuto, 0, look); u2.Env[0] != "GAMEID=0" {
		t.Fatalf("no Steam id: %+v", u2)
	}
}

func TestSteamCommand(t *testing.T) {
	bin := t.TempDir()
	look := lookIn(bin)
	if SteamCommand("/home/alex/.local/share/Steam", look) != nil {
		t.Fatal("no steam command: the link is what's left")
	}
	mkCommand(t, bin, "steam")
	if got := SteamCommand("/home/alex/.local/share/Steam", look); !reflect.DeepEqual(got, []string{filepath.Join(bin, "steam")}) {
		t.Fatalf("steam: %v", got)
	}
	flat := "/home/alex/.var/app/com.valvesoftware.Steam/.local/share/Steam"
	mkCommand(t, bin, "flatpak")
	if got := SteamCommand(flat, look); !reflect.DeepEqual(got, []string{filepath.Join(bin, "flatpak"), "run", "com.valvesoftware.Steam"}) {
		t.Fatalf("Flatpak Steam: %v", got)
	}
	if SteamRunURI(570, "") != "steam://rungameid/570" || SteamRunURI(570, "-novid -console") != "steam://run/570//-novid -console/" {
		t.Fatal("Steam's links")
	}
}

func TestLinuxFolderGame(t *testing.T) {
	f := newLinuxFixture(t)
	// A Windows game in a folder, run in Wine's own prefix.
	wine := filepath.Join(f.home, ".wine")
	roaming := mkPrefix(t, wine, "alex")
	write(t, filepath.Join(roaming, "Quiet Isle", "save1.sav"), "s")
	game := filepath.Join(f.home, "Games", "Quiet Isle")
	write(t, filepath.Join(game, "QuietIsle.exe"), "MZ")
	// A native game in a folder.
	tux := filepath.Join(f.home, "Games", "tux")
	mkELF(t, filepath.Join(tux, "tuxracer.x86_64"), 500)

	list := f.env.FinishFound([]FoundGame{
		{Key: "dir:" + strings.ToLower(game), Name: "Quiet Isle", Source: "folder", Dir: game, Exe: filepath.Join(game, "QuietIsle.exe")},
		{Key: "dir:" + strings.ToLower(tux), Name: "Tux", Source: "folder", Dir: tux, Exe: filepath.Join(tux, "tuxracer.x86_64")},
	})
	got := map[string]FoundGame{}
	for _, g := range list {
		got[g.Name] = g
	}
	if q := got["Quiet Isle"]; q.Prefix != "%HOME%/.wine" || len(q.Paths) == 0 || q.Paths[0] != "%APPDATA%/Quiet Isle" {
		t.Fatalf("a Windows folder game, in ~/.wine: %+v", q)
	}
	if x := got["Tux"]; x.Prefix != "" || x.ProcessName != "tuxracer" {
		t.Fatalf("a native folder game: %+v", x)
	}
}
