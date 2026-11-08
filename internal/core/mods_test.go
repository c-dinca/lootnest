package core

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// modGame is a game folder with a program, a Data folder and a config.
func modGame(t *testing.T) (game, root string) {
	t.Helper()
	base := t.TempDir()
	game = filepath.Join(base, "Games", "Cool Game")
	write(t, filepath.Join(game, "CoolGame.exe"), "exe")
	write(t, filepath.Join(game, "settings.ini"), "vanilla settings")
	write(t, filepath.Join(game, "Data", "textures", "rock.dds"), "vanilla rock")
	write(t, filepath.Join(game, "Data", "meshes", "tree.nif"), "vanilla tree")
	return game, filepath.Join(base, "data", "mods", "game")
}

// folderHashes is every file in a folder with its sha256.
func folderHashes(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if info.IsDir() {
			out[filepath.ToSlash(rel)+"/"] = ""
			return nil
		}
		sum, err := fileSHA256(p)
		out[filepath.ToSlash(rel)] = sum
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func addMod(t *testing.T, root, game string, files map[string]string, name string) *Mod {
	t.Helper()
	src := filepath.Join(t.TempDir(), name)
	makeZip(t, src, files)
	m, err := AddMod(root, game, src, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	set := append(LoadModSet(root), ModEntry{ID: m.ID, On: true})
	if err := SaveModSet(root, set); err != nil {
		t.Fatal(err)
	}
	return m
}

func modRels(m *Mod) []string {
	var out []string
	for _, f := range m.Files {
		out = append(out, f.Rel)
	}
	return out
}

func TestAddModPlacement(t *testing.T) {
	game, root := modGame(t)
	// Wrapped in a folder of its own, with what's inside the game's Data.
	m := addMod(t, root, game, map[string]string{"Better Rocks v2/textures/rock.dds": "x", "Better Rocks v2/readme.txt": "notes"}, "Better Rocks v2.zip")
	if m.Name != "Better Rocks v2" || m.Into != "Data" || m.AtRoot || !reflect.DeepEqual(modRels(m), []string{"Data/textures/rock.dds"}) {
		t.Fatalf("wrapped mod into Data: %+v", m)
	}
	// Its top level matches the game's own.
	m = addMod(t, root, game, map[string]string{"Data/textures/new.dds": "x", "settings.ini": "modded"}, "fix.zip")
	if m.Into != "" || m.AtRoot || !reflect.DeepEqual(modRels(m), []string{"Data/textures/new.dds", "settings.ini"}) {
		t.Fatalf("mod at the game's level: %+v", m)
	}
	// Nothing matches: at the root as it is, and the mod says so.
	m = addMod(t, root, game, map[string]string{"Reshade/presets/a.ini": "x"}, "other.zip")
	if m.Into != "" || !m.AtRoot || !reflect.DeepEqual(modRels(m), []string{"Reshade/presets/a.ini"}) {
		t.Fatalf("mod at the root: %+v", m)
	}
	// A folder named like the game's own is skipped.
	m = addMod(t, root, game, map[string]string{"Cool Game/extra.pak": "x"}, "named.zip")
	if !reflect.DeepEqual(modRels(m), []string{"extra.pak"}) {
		t.Fatalf("mod in the game's name: %+v", m)
	}
	// From a folder: copied, the original left alone.
	src := filepath.Join(t.TempDir(), "My Mod")
	write(t, filepath.Join(src, "meshes", "tree.nif"), "new tree")
	m, err := AddMod(root, game, src, time.Now())
	if err != nil || m.Name != "My Mod" || !reflect.DeepEqual(modRels(m), []string{"Data/meshes/tree.nif"}) {
		t.Fatalf("folder mod: %+v %v", m, err)
	}
	if read(t, filepath.Join(src, "meshes", "tree.nif")) != "new tree" || read(t, filepath.Join(root, m.ID, "files", "Data", "meshes", "tree.nif")) != "new tree" {
		t.Fatal("the folder mod wasn't copied")
	}
	// A mod added but never kept in the set goes on the next add.
	if _, err := os.Stat(filepath.Join(root, m.ID)); err != nil {
		t.Fatal(err)
	}
	addMod(t, root, game, map[string]string{"settings.ini": "x"}, "last.zip")
	if _, err := os.Stat(filepath.Join(root, m.ID)); !os.IsNotExist(err) {
		t.Fatal("the unfinished add was kept")
	}
	// Into BepInEx when its plugins folder is what matches.
	write(t, filepath.Join(game, "BepInEx", "plugins", "old.dll"), "x")
	m = addMod(t, root, game, map[string]string{"plugins/new.dll": "x"}, "plugin.zip")
	if m.Into != "BepInEx" || !reflect.DeepEqual(modRels(m), []string{"BepInEx/plugins/new.dll"}) {
		t.Fatalf("BepInEx plugin: %+v", m)
	}
}

func TestAddModRefusesPathsOutsideIt(t *testing.T) {
	game, root := modGame(t)
	for _, name := range []string{"../evil.dll", "/abs.dll", "C:/Windows/evil.dll", "file.txt:stream", "a/../../b.dll"} {
		src := filepath.Join(t.TempDir(), "bad.zip")
		makeZip(t, src, map[string]string{name: "x"})
		if _, err := AddMod(root, game, src, time.Now()); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if _, err := AddMod(root, game, game, time.Now()); err == nil {
		t.Error("the game's own folder was accepted as a mod")
	}
	big := filepath.Join(t.TempDir(), "big")
	write(t, filepath.Join(big, "a.bin"), "x")
	f, _ := os.OpenFile(filepath.Join(big, "a.bin"), os.O_WRONLY, 0)
	_ = f.Truncate(ModMax + 1) // sparse
	f.Close()
	if _, err := AddMod(root, game, big, time.Now()); !errors.Is(err, ErrTooBig) {
		t.Errorf("a mod over the limit: %v", err)
	}
	if es, _ := os.ReadDir(root); len(es) != 0 {
		t.Errorf("refused mods left copies: %v", es)
	}
}

func TestPlanModsRefusals(t *testing.T) {
	game, _ := modGame(t)
	mod := func(rels ...string) []*Mod {
		m := &Mod{ID: "m1", Name: "A mod"}
		for _, r := range rels {
			m.Files = append(m.Files, ModFile{Rel: r, SHA256: "x"})
		}
		return []*Mod{m}
	}
	p, err := PlanMods(game, "", mod("Data/textures/rock.dds", "Data/new.esp"), nil)
	if err != nil || p.Added != 1 || p.Replaced != 1 || !reflect.DeepEqual(p.Replaces, []string{"Data/textures/rock.dds"}) {
		t.Fatalf("plan: %+v %v", p, err)
	}
	for _, rel := range []string{"steam_api64.dll", "bin/steam_api.dll", "steamclient64.dll", "EOSSDK-Win64-Shipping.dll", "Galaxy64.dll",
		"uplay_r1_loader64.dll", "steam_settings/configs.user.ini", "steam_emu.ini", "game.cdx", "ColdClientLoader.ini", "OnlineFix64.ini"} {
		if _, err := PlanMods(game, "", mod("Data/ok.esp", rel), nil); err == nil || !strings.Contains(err.Error(), "store") {
			t.Errorf("%s: %v", rel, err)
		}
	}
	if _, err := PlanMods(game, filepath.Join(game, "CoolGame.exe"), mod("CoolGame.exe"), nil); err == nil || !strings.Contains(err.Error(), "program") {
		t.Errorf("replacing the main exe: %v", err)
	}
	if _, err := PlanMods(game, "", mod("CoolGame.exe"), nil); err == nil {
		t.Error("replacing the main exe, found by itself, was accepted")
	}
	if _, err := PlanMods(game, "", mod("helper.exe"), nil); err != nil {
		t.Errorf("a new exe is fine: %v", err)
	}
	for _, rel := range []string{"../x.dll", "/x.dll", "C:/x.dll", "a:b", "Data/../../x"} {
		if _, err := PlanMods(game, "", mod(rel), nil); err == nil {
			t.Errorf("%s: accepted", rel)
		}
	}
	if _, err := PlanMods(game, "", mod("Data"), nil); err == nil {
		t.Error("a file over the game's folder was accepted")
	}
	if _, err := PlanMods(game, "", mod("settings.ini/x"), nil); err == nil {
		t.Error("a folder over the game's file was accepted")
	}
	for _, dir := range []string{`C:\Program Files\Game`, `C:\Program Files (x86)\Steam\steamapps\common\Game`, `C:\Program Files\WindowsApps\Game_1.0`} {
		if ModsRefused(filepath.FromSlash(strings.ReplaceAll(dir, `\`, "/"))) == nil {
			t.Errorf("%s: not refused", dir)
		}
	}
	for file, name := range map[string]string{"EasyAntiCheat/x": "Easy Anti-Cheat", "vgk.sys": "Vanguard", "faceitclient.exe": "FACEIT", "EAAntiCheat.GameServiceLauncher.exe": "EA AntiCheat", "BattlEye/x": "BattlEye"} {
		g := t.TempDir()
		write(t, filepath.Join(g, file), "x")
		if err := ModsRefused(g); err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("%s: %v", file, err)
		}
		if _, err := PlanMods(g, "", mod("x.pak"), nil); err == nil {
			t.Errorf("%s: planned", file)
		}
	}
}

func TestDeployAndPutBack(t *testing.T) {
	game, root := modGame(t)
	if err := os.Chmod(filepath.Join(game, "settings.ini"), 0o444); err != nil {
		t.Fatal(err)
	}
	vanilla := folderHashes(t, game)
	a := addMod(t, root, game, map[string]string{"Data/textures/rock.dds": "rock A", "Data/new/a.esp": "a", "settings.ini": "settings A"}, "a.zip")
	b := addMod(t, root, game, map[string]string{"Data/textures/rock.dds": "rock B"}, "b.zip")
	res, err := ApplyMods(root, game, "", []*Mod{a, b}, time.Now(), nil)
	if err != nil || res.Changed != 0 {
		t.Fatal(res, err)
	}
	if read(t, filepath.Join(game, "Data", "textures", "rock.dds")) != "rock B" || read(t, filepath.Join(game, "Data", "new", "a.esp")) != "a" || read(t, filepath.Join(game, "settings.ini")) != "settings A" {
		t.Fatal("the mods aren't in, or the later one didn't win")
	}
	if read(t, filepath.Join(root, "originals", "Data", "textures", "rock.dds")) != "vanilla rock" || read(t, filepath.Join(root, "originals", "settings.ini")) != "vanilla settings" {
		t.Fatal("the originals weren't kept")
	}
	d := LoadDeployed(root)
	if d == nil || len(d.Files) != 3 || !reflect.DeepEqual(d.Mods, []string{a.ID, b.ID}) || ModJournalPending(root) {
		t.Fatalf("deployed: %+v", d)
	}
	// Apply again with one mod: the first set comes out, the new one goes in.
	if _, err := ApplyMods(root, game, "", []*Mod{a}, time.Now(), nil); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(game, "Data", "textures", "rock.dds")) != "rock A" {
		t.Fatal("redeploy")
	}
	if err := RemoveMod(root, a.ID); err == nil {
		t.Fatal("a mod in the game's folder was removed")
	}
	if _, err := ApplyMods(root, game, "", nil, time.Now(), nil); err != nil {
		t.Fatal(err)
	}
	if got := folderHashes(t, game); !reflect.DeepEqual(got, vanilla) {
		t.Fatalf("not back to vanilla:\n%v\n%v", got, vanilla)
	}
	if LoadDeployed(root) != nil || exists(filepath.Join(root, "originals")) {
		t.Fatal("records left after putting the game back")
	}
	if err := RemoveMod(root, a.ID); err != nil || exists(filepath.Join(root, a.ID)) || len(LoadModSet(root)) != 1 {
		t.Fatal("remove", err)
	}
}

func TestModRunCutShortIsFinished(t *testing.T) {
	game, root := modGame(t)
	vanilla := folderHashes(t, game)
	a := addMod(t, root, game, map[string]string{"Data/textures/rock.dds": "rock A", "Data/new/a.esp": "a", "settings.ini": "settings A", "Data/meshes/tree.nif": "tree A"}, "a.zip")
	defer func() { modStep = nil }()
	stop := errors.New("the PC turned off")
	modStep = func(i int) error {
		if i == 2 {
			return stop
		}
		return nil
	}
	if _, err := ApplyMods(root, game, "", []*Mod{a}, time.Now(), nil); !errors.Is(err, stop) {
		t.Fatal(err)
	}
	if !ModJournalPending(root) || LoadDeployed(root) != nil {
		t.Fatal("the journal should be left and nothing recorded as deployed")
	}
	modStep = nil
	if ok, _, err := FinishModJournal(root); !ok || err != nil {
		t.Fatal(ok, err)
	}
	if ModJournalPending(root) || LoadDeployed(root) == nil || read(t, filepath.Join(game, "settings.ini")) != "settings A" || read(t, filepath.Join(root, "originals", "settings.ini")) != "vanilla settings" {
		t.Fatal("the deploy wasn't finished")
	}
	// Cut short on the way back too.
	modStep = func(i int) error {
		if i == 1 {
			return stop
		}
		return nil
	}
	if _, err := ApplyMods(root, game, "", nil, time.Now(), nil); !errors.Is(err, stop) {
		t.Fatal(err)
	}
	modStep = nil
	if got := ModJournals(filepath.Dir(filepath.Dir(root))); !reflect.DeepEqual(got, []string{root}) {
		t.Fatalf("journals: %v", got)
	}
	if ok, res, err := FinishModJournal(root); !ok || err != nil || res.Changed != 0 {
		t.Fatal(ok, res, err)
	}
	if got := folderHashes(t, game); !reflect.DeepEqual(got, vanilla) {
		t.Fatalf("not back to vanilla:\n%v\n%v", got, vanilla)
	}
}

func TestPutBackKeepsChangedFiles(t *testing.T) {
	game, root := modGame(t)
	a := addMod(t, root, game, map[string]string{"Data/textures/rock.dds": "rock A", "Data/new/a.ini": "a", "Data/new/b.esp": "b"}, "a.zip")
	if _, err := ApplyMods(root, game, "", []*Mod{a}, time.Now(), nil); err != nil {
		t.Fatal(err)
	}
	// A game update rewrote a replaced file, and the user edited an added one.
	write(t, filepath.Join(game, "Data", "textures", "rock.dds"), "updated rock")
	write(t, filepath.Join(game, "Data", "new", "a.ini"), "edited")
	res, err := ApplyMods(root, game, "", nil, time.Now(), nil)
	if err != nil || res.Changed != 2 {
		t.Fatal(res, err)
	}
	if read(t, filepath.Join(game, "Data", "textures", "rock.dds")) != "vanilla rock" {
		t.Fatal("the original isn't back")
	}
	if read(t, filepath.Join(res.ChangedDir, "Data", "textures", "rock.dds")) != "updated rock" || read(t, filepath.Join(res.ChangedDir, "Data", "new", "a.ini")) != "edited" {
		t.Fatal("the changed files weren't kept")
	}
	if exists(filepath.Join(game, "Data", "new")) {
		t.Fatal("the added files or their folder are still in the game")
	}
}

func TestDeployKeepsAFileThatAppeared(t *testing.T) {
	game, root := modGame(t)
	a := addMod(t, root, game, map[string]string{"Data/new.esp": "a"}, "a.zip")
	// Planned as new, but the user put a file there before it went in.
	p, _ := PlanMods(game, "", []*Mod{a}, nil)
	write(t, filepath.Join(game, "Data", "new.esp"), "the user's")
	j := &modJournal{Op: "deploy", Game: game, At: time.Now(), Plan: &ModsDeployed{Mods: []string{a.ID}, Files: p.files}}
	if err := writeJSONSynced(modJournalPath(root), j); err != nil {
		t.Fatal(err)
	}
	if _, _, err := FinishModJournal(root); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyMods(root, game, "", nil, time.Now(), nil); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(game, "Data", "new.esp")) != "the user's" {
		t.Fatal("the user's file wasn't kept")
	}
}

// stuckApply applies a mod whose staged settings.ini is gone, so the run
// stops after rock.dds is in and settings.ini's original is kept.
func stuckApply(t *testing.T) (game, root string, a *Mod, hidden func()) {
	t.Helper()
	game, root = modGame(t)
	a = addMod(t, root, game, map[string]string{"Data/textures/rock.dds": "rock A", "settings.ini": "settings A"}, "a.zip")
	staged := filepath.Join(root, a.ID, "files", "settings.ini")
	if err := os.Rename(staged, staged+".away"); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyMods(root, game, "", []*Mod{a}, time.Now(), nil); err == nil || !ModJournalPending(root) {
		t.Fatal("the run should stop part way", err)
	}
	if read(t, filepath.Join(game, "Data", "textures", "rock.dds")) != "rock A" || !exists(filepath.Join(root, "originals", "settings.ini")) {
		t.Fatal("the run didn't stop where the test needs it")
	}
	return game, root, a, func() { _ = os.Rename(staged+".away", staged) }
}

func TestDeployResumedKeepsAFileTheGameChanged(t *testing.T) {
	game, root, _, back := stuckApply(t)
	write(t, filepath.Join(game, "settings.ini"), "the game rewrote its settings")
	back()
	ok, res, err := FinishModJournal(root)
	if !ok || err != nil || res.Changed != 1 || read(t, filepath.Join(res.ChangedDir, "settings.ini")) != "the game rewrote its settings" {
		t.Fatal(ok, res, err)
	}
	if read(t, filepath.Join(game, "settings.ini")) != "settings A" {
		t.Fatal("the mod file isn't in")
	}
}

func TestPutBackWhenADeployCantFinish(t *testing.T) {
	game, root, _, _ := stuckApply(t)
	if _, _, err := FinishModJournal(root); err == nil {
		t.Fatal("the deploy can't finish without its file")
	}
	if _, err := ApplyMods(root, game, "", nil, time.Now(), nil); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(game, "Data", "textures", "rock.dds")) != "vanilla rock" || read(t, filepath.Join(game, "settings.ini")) != "vanilla settings" {
		t.Fatal("the game isn't back as it was")
	}
	if ModJournalPending(root) || LoadDeployed(root) != nil || exists(filepath.Join(root, "originals")) || exists(filepath.Join(game, "Data", "textures", "rock.dds"+tmpSuffix)) {
		t.Fatal("records left after putting the game back")
	}
}
