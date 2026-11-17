package core

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fuGame makes a game folder and a staged new build, and returns a staged
// update for them.
func fuGame(t *testing.T) (*FullUpdate, string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "Games", "Night Tale")
	for name, body := range map[string]string{
		"bin/NightTale.exe":    "old exe",
		"data/pak0.pak":        "old data",
		"data/removed.pak":     "only in the old build",
		"Saves/slot1.sav":      "my save",
		"Saves/deep/slot2.sav": "my other save",
		"config.ini":           "my settings",
		"Mods/cool/mod.pak":    "a mod",
		"steam_api64.dll":      "old store file",
		"steam_settings/x.txt": "tool settings",
	} {
		write(t, filepath.Join(dir, filepath.FromSlash(name)), body)
	}
	staging := filepath.Join(root, "staging")
	for name, body := range map[string]string{
		"Night.Tale.v2/bin/NightTale.exe": "new exe",
		"Night.Tale.v2/data/pak0.pak":     "new data",
		"Night.Tale.v2/data/pak1.pak":     "more new data",
		"Night.Tale.v2/Saves/default.sav": "a save the build ships",
		"Night.Tale.v2/config.ini":        "default settings",
		"Night.Tale.v2/steam_api64.dll":   "new store file",
	} {
		write(t, filepath.Join(staging, filepath.FromSlash(name)), body)
	}
	build, exe, err := FindBuild(staging, "bin/NightTale.exe")
	if err != nil || build != filepath.Join(staging, "Night.Tale.v2") || exe != "bin/NightTale.exe" {
		t.Fatalf("FindBuild: %q %q %v", build, exe, err)
	}
	return &FullUpdate{ID: "u1", Key: "dir:night tale", Name: "Night Tale", Dir: dir, Exe: "bin/NightTale.exe", Staging: staging, Build: build, NewExe: exe, State: FUStaged}, root
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		return "<" + err.Error() + ">"
	}
	return string(b)
}

// oldGameIntact: the game's folder is exactly as fuGame made it.
func oldGameIntact(t *testing.T, dir string) {
	t.Helper()
	for name, want := range map[string]string{"bin/NightTale.exe": "old exe", "data/pak0.pak": "old data", "data/removed.pak": "only in the old build", "Saves/slot1.sav": "my save",
		"Saves/deep/slot2.sav": "my other save", "config.ini": "my settings", "Mods/cool/mod.pak": "a mod", "steam_api64.dll": "old store file"} {
		if got := readFile(t, filepath.Join(dir, filepath.FromSlash(name))); got != want {
			t.Errorf("%s is %q, want %q", name, got, want)
		}
	}
	if isFile(filepath.Join(dir, "data", "pak1.pak")) {
		t.Error("a file of the new build is in the old folder")
	}
}

func TestFindBuild(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "a", "Game.exe"), "x")
	if b, e, err := FindBuild(filepath.Join(root, "a"), "Game.exe"); err != nil || b != filepath.Join(root, "a") || e != "Game.exe" {
		t.Errorf("at the top: %q %q %v", b, e, err)
	}
	// The exe moved inside the build: the root is under the single wrapping folders.
	write(t, filepath.Join(root, "b", "Wrap", "Inner", "Binaries", "Win64", "game.EXE"), "x")
	write(t, filepath.Join(root, "b", "Wrap", "Inner", "readme.txt"), "x")
	if b, e, err := FindBuild(filepath.Join(root, "b"), "Game.exe"); err != nil || b != filepath.Join(root, "b", "Wrap", "Inner") || e != "Binaries/Win64/game.EXE" {
		t.Errorf("moved: %q %q %v", b, e, err)
	}
	// A patch or an installer has no such exe.
	write(t, filepath.Join(root, "c", "setup.exe"), "x")
	write(t, filepath.Join(root, "c", "data", "pak9.pak"), "x")
	if _, _, err := FindBuild(filepath.Join(root, "c"), "bin/Game.exe"); !errors.Is(err, ErrNoBuild) {
		t.Errorf("no build: %v", err)
	}
	if _, _, err := FindBuild(filepath.Join(root, "a"), "../Game.exe"); err == nil {
		t.Error("an exe outside the folder was taken")
	}
}

func TestPlanFullUpdate(t *testing.T) {
	u, _ := fuGame(t)
	p, err := PlanFullUpdate(u.Dir, u.Build, []string{"Saves", "Nowhere", "../outside"}, []string{"config.ini", "Saves"})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Keep) != 2 || p.Keep[0].Rel != "Saves" || p.Keep[0].Why != "saves" || p.Keep[0].Size != int64(len("my save")+len("my other save")) || p.Keep[1].Rel != "config.ini" || p.Keep[1].Why != "yours" {
		t.Errorf("keep: %+v", p.Keep)
	}
	// Only the mods folder is the user's own at the top: the store's file
	// and the tool's settings are never offered, and data\removed.pak isn't
	// at the top.
	if len(p.Extra) != 1 || p.Extra[0].Rel != "Mods" {
		t.Errorf("extra: %+v", p.Extra)
	}
	if p.BuildSize == 0 || p.OldSize == 0 {
		t.Errorf("sizes: %+v", p)
	}
	for _, bad := range []string{"../../Windows", "C:\\Windows", "/etc", "steam_api64.dll", "steam_settings", "steam_settings/configs.user.ini", "bin/steam_emu.ini"} {
		if _, err := PlanFullUpdate(u.Dir, u.Build, nil, []string{bad}); err == nil {
			t.Errorf("%q was taken as a path to keep", bad)
		}
	}
}

func TestFullUpdate(t *testing.T) {
	u, _ := fuGame(t)
	u.Keep = []string{"Saves", "config.ini", "Mods", "gone-since"}
	stateDir := filepath.Join(t.TempDir(), "state")
	var states, logs []string
	backups := 0
	h := FullUpdateHooks{
		Save:   func(x *FullUpdate) error { states = append(states, x.State); return SaveFullUpdate(stateDir, x) },
		Log:    func(s string) { logs = append(logs, s) },
		Backup: func() (string, error) { backups++; return "copy-1", nil },
	}
	if err := RunFullUpdate(u, h); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(states, ","); got != "backup,swap,restore,verify,kept" {
		t.Errorf("states: %s", got)
	}
	if u.State != FUKept || u.Saves != "copy-1" || backups != 1 || u.Old != u.Dir+".lootnest-old" || u.Swapped.IsZero() {
		t.Fatalf("after: %+v", u)
	}
	// The new build is in place, with what was kept carried over it.
	for name, want := range map[string]string{
		"bin/NightTale.exe": "new exe", "data/pak0.pak": "new data", "data/pak1.pak": "more new data", "steam_api64.dll": "new store file",
		"Saves/slot1.sav": "my save", "Saves/deep/slot2.sav": "my other save", "Saves/default.sav": "a save the build ships", "config.ini": "my settings", "Mods/cool/mod.pak": "a mod",
	} {
		if got := readFile(t, filepath.Join(u.Dir, filepath.FromSlash(name))); got != want {
			t.Errorf("%s is %q, want %q", name, got, want)
		}
	}
	if isFile(filepath.Join(u.Dir, "data", "removed.pak")) || isDir(filepath.Join(u.Dir, "steam_settings")) {
		t.Error("something of the old build that wasn't kept is in the new one")
	}
	// The old folder is whole, and the staging folder gave its build away.
	oldGameIntact(t, u.Old)
	if isDir(u.Build) {
		t.Error("the build is still in staging")
	}
	// What's on disk is what's in memory.
	got, err := LoadFullUpdate(stateDir)
	if err != nil || got.State != FUKept || got.Old != u.Old || got.ID != "u1" {
		t.Fatalf("state.json: %+v %v", got, err)
	}
	if len(logs) < 5 {
		t.Errorf("log: %q", logs)
	}

	// A second update while the first one's folder is kept is refused.
	if err := FullUpdateRefused(u.Dir); err == nil || !strings.Contains(err.Error(), "still kept") {
		t.Errorf("a second update: %v", err)
	}

	// Roll back: saves copied first, the old folder back, the new build aside.
	write(t, filepath.Join(u.Dir, "Saves", "slot3.sav"), "made after the update")
	if err := RollbackFullUpdate(u, h); err != nil {
		t.Fatal(err)
	}
	if u.State != FURolledBack || backups != 2 || u.Old != "" || !isDir(u.Aside) {
		t.Fatalf("rolled back: %+v", u)
	}
	oldGameIntact(t, u.Dir)
	if readFile(t, filepath.Join(u.Aside, "Saves", "slot3.sav")) != "made after the update" {
		t.Error("the new build, with the save made in it, isn't kept aside")
	}
	if err := RollbackFullUpdate(u, h); err == nil {
		t.Error("rolled back twice")
	}
}

// A failure at any step after the saves are copied leaves the game exactly
// as it was, and says what happened.
func TestFullUpdateFailures(t *testing.T) {
	for _, at := range []string{FUBackup, FUSwap, FURestore, FUVerify} {
		u, _ := fuGame(t)
		u.Keep = []string{"Saves"}
		restored := ""
		h := FullUpdateHooks{
			Backup:       func() (string, error) { return "copy-1", nil },
			RestoreSaves: func(id string) error { restored = id; return nil },
			After: func(state string) error {
				if state == at {
					return errors.New("the disk said no")
				}
				return nil
			},
		}
		err := RunFullUpdate(u, h)
		if err == nil || !strings.Contains(err.Error(), "the disk said no") || u.State != FUFailed {
			t.Fatalf("%s: %v %+v", at, err, u)
		}
		oldGameIntact(t, u.Dir)
		if isDir(u.Dir + ".lootnest-old") {
			t.Errorf("%s: the old folder's copy is still aside", at)
		}
		if at == FUBackup {
			if restored != "" || u.Aside != "" || !isDir(u.Build) {
				t.Errorf("%s: something moved before the swap: %+v", at, u)
			}
			continue
		}
		// The build that failed is kept aside, never deleted; the saves' copy is put back.
		if restored != "copy-1" || !isFile(filepath.Join(u.Aside, "bin", "NightTale.exe")) {
			t.Errorf("%s: restored %q, aside %q", at, restored, u.Aside)
		}
	}

	// The saves can't be copied: nothing moves at all.
	u, _ := fuGame(t)
	if err := RunFullUpdate(u, FullUpdateHooks{Backup: func() (string, error) { return "", errors.New("no room") }}); err == nil || !strings.Contains(err.Error(), "saves couldn't be copied") {
		t.Fatalf("a failed save copy: %v", err)
	}
	oldGameIntact(t, u.Dir)
	if !isDir(u.Build) || u.State != FUFailed {
		t.Errorf("after a failed save copy: %+v", u)
	}

	// A build without its exe is refused before anything.
	u, _ = fuGame(t)
	os.Remove(filepath.Join(u.Build, "bin", "NightTale.exe"))
	if err := RunFullUpdate(u, FullUpdateHooks{}); err == nil || u.State != FUStaged {
		t.Fatalf("a build without its exe: %v %s", err, u.State)
	}
	// A path to keep that leaves the folder, too.
	u, _ = fuGame(t)
	u.Keep = []string{"../../somewhere"}
	if err := RunFullUpdate(u, FullUpdateHooks{}); err == nil || u.State != FUStaged {
		t.Fatalf("a path outside: %v %s", err, u.State)
	}
	oldGameIntact(t, u.Dir)
}

// Lootnest closes (or the power goes) in the middle: at the next start the
// state on disk is read and the game put back as it was.
func TestFullUpdateRecover(t *testing.T) {
	type stop struct{}
	for _, at := range []string{FUBackup, FUSwap, FURestore, FUVerify} {
		u, _ := fuGame(t)
		u.Keep = []string{"Saves"}
		stateDir := filepath.Join(t.TempDir(), "state")
		h := FullUpdateHooks{
			Save:   func(x *FullUpdate) error { return SaveFullUpdate(stateDir, x) },
			Backup: func() (string, error) { return "copy-1", nil },
			After: func(state string) error {
				if state == at {
					panic(stop{}) // the process is gone: nothing after this runs
				}
				return nil
			},
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					if _, ok := r.(stop); !ok {
						panic(r)
					}
				}
			}()
			_ = RunFullUpdate(u, h)
		}()
		// Next start: only what's on disk is known.
		got, err := LoadFullUpdate(stateDir)
		if err != nil || got == nil || got.State != at {
			t.Fatalf("%s: state on disk %+v %v", at, got, err)
		}
		h.After = nil
		restored := ""
		h.RestoreSaves = func(id string) error { restored = id; return nil }
		changed, err := RecoverFullUpdate(got, h)
		if err != nil || !changed {
			t.Fatalf("%s: recover: %v %v", at, changed, err)
		}
		oldGameIntact(t, got.Dir)
		if isDir(got.Dir + ".lootnest-old") {
			t.Errorf("%s: the old folder's copy is still aside", at)
		}
		if at == FUBackup {
			// Nothing had moved: the update waits again and can be run.
			if got.State != FUStaged {
				t.Errorf("%s: state %s", at, got.State)
			}
			if err := RunFullUpdate(got, h); err != nil || got.State != FUKept {
				t.Errorf("%s: running it again: %v %s", at, err, got.State)
			}
			continue
		}
		if got.State != FUFailed || !strings.Contains(got.Error, "cut short") || restored != "copy-1" || !isDir(got.Aside) {
			t.Errorf("%s: after recovery %+v, restored %q", at, got, restored)
		}
		// Recovering again changes nothing.
		if changed, err := RecoverFullUpdate(got, h); changed || err != nil {
			t.Errorf("%s: recovered twice: %v %v", at, changed, err)
		}
	}
	// States that need nothing.
	for _, st := range []string{FUStaged, FUKept, FUDone, FUFailed, FURolledBack} {
		if changed, err := RecoverFullUpdate(&FullUpdate{State: st}, FullUpdateHooks{}); changed || err != nil {
			t.Errorf("%s: %v %v", st, changed, err)
		}
	}
}

func TestFullUpdateFinishAndRefusals(t *testing.T) {
	u, root := fuGame(t)
	if err := RunFullUpdate(u, FullUpdateHooks{}); err != nil {
		t.Fatal(err)
	}
	old, err := FinishFullUpdate(u, FullUpdateHooks{})
	if err != nil || old != u.Dir+".lootnest-old" || u.State != FUDone || u.Old != "" {
		t.Fatalf("finish: %q %v %+v", old, err, u)
	}
	if _, err := FinishFullUpdate(u, FullUpdateHooks{}); err == nil {
		t.Error("finished twice")
	}
	if err := RollbackFullUpdate(u, FullUpdateHooks{}); err == nil {
		t.Error("rolled back after the old folder was let go")
	}
	for _, dir := range []string{"", "relative/path", filepath.Join(root, "nowhere"), filepath.Join(root, "Program Files", "Game")} {
		if dir != "" && strings.Contains(dir, "Program Files") {
			_ = os.MkdirAll(dir, 0o755)
		}
		if err := FullUpdateRefused(dir); err == nil {
			t.Errorf("%q wasn't refused", dir)
		}
	}
}
