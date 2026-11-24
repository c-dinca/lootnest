package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lootnest/internal/core"
)

// A Windows game with profiles starts through the runner on PATH, in its
// own Wine prefix, with its arguments kept whole.
func TestRunProgramLinux(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home", "alex")
	bin := filepath.Join(root, "bin")
	out := filepath.Join(root, "ran.txt")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	// A stand-in for Wine: it writes what it was given.
	script := "#!/bin/sh\nprintf '%s\\n' \"WINEPREFIX=$WINEPREFIX\" \"$@\" > " + out + "\n"
	if err := os.WriteFile(filepath.Join(bin, "wine"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)

	pfx := filepath.Join(home, "Games", "ashen")
	for _, d := range []string{filepath.Join(pfx, "drive_c", "windows"), filepath.Join(pfx, "drive_c", "users", "alex")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	exe := filepath.Join(pfx, "drive_c", "Games", "Ashen", "Ashen.exe")
	if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}

	env := core.NewLinuxEnv(home, func(k string) string { return map[string]string{"HOME": home, "USER": "alex"}[k] }, filepath.Join(home, ".local", "share", "lootnest"))
	cfg := core.DefaultConfig()
	cfg.Settings.LinuxRunner = core.RunnerWine
	cfg.Games = []core.Game{{ID: "ashen", Name: "Ashen", Prefix: env.Tokenize(pfx), Launch: core.Launch{Type: "exe", Exe: env.Tokenize(exe)}}}
	a := &App{env: env, cfg: cfg, store: &core.Store{Dir: t.TempDir()}}

	if got := a.prefixRootOf("game:ashen"); got != pfx {
		t.Fatalf("the game's prefix: %q", got)
	}
	if err := a.runProgram("game:ashen", exe, `-windowed --profile "My Saves"`, filepath.Dir(exe), false); err != nil {
		t.Fatal(err)
	}
	var ran []byte
	for i := 0; i < 50 && len(ran) == 0; i++ {
		time.Sleep(50 * time.Millisecond)
		ran, _ = os.ReadFile(out)
	}
	want := "WINEPREFIX=" + pfx + "\n" + exe + "\n-windowed\n--profile\nMy Saves\n"
	if string(ran) != want {
		t.Fatalf("ran:\n%s\nwant:\n%s", ran, want)
	}

	// A native program starts as itself; no runner asked for.
	cfg.Settings.LinuxRunner = core.RunnerAuto
	native := filepath.Join(root, "tux.sh")
	if err := os.WriteFile(native, []byte("#!/bin/sh\necho native > "+out+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := a.runProgram("game:none", native, "", root, false); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 50; i++ {
		time.Sleep(50 * time.Millisecond)
		if b, _ := os.ReadFile(out); strings.TrimSpace(string(b)) == "native" {
			return
		}
	}
	t.Fatal("the native program didn't run")
}
