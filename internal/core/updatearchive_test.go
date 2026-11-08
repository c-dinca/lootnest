package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStage7zUpdate(t *testing.T) {
	game, _ := gameWithUpdate(t)
	src := filepath.Join("testdata", "update.7z") // "PBZ Update v1.0.2 to v1.0.3\ProjectPBZ\Binaries\Win64\PBZ.exe", README.txt
	if !NeedsStaging(src) || NeedsStaging("x.zip") {
		t.Fatal("NeedsStaging")
	}
	staged, err := StageUpdate(src, filepath.Join(t.TempDir(), "stage"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := PlanUpdate(game, staged)
	if err != nil {
		t.Fatal(err)
	}
	p.Name = filepath.Base(src)
	if p.Skipped != "PBZ Update v1.0.2 to v1.0.3" || p.Replace != 1 || p.Added != 0 || len(p.Readme) != 1 {
		t.Fatalf("plan = %+v", p)
	}
	u, err := ApplyUpdate(game, p, filepath.Join(t.TempDir(), "u1"), "PBZ", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(game, "ProjectPBZ", "Binaries", "Win64", "PBZ.exe")) != "new exe" || u.Source != "update.7z" {
		t.Fatalf("applied = %+v", u)
	}

	bad := filepath.Join(t.TempDir(), "broken.rar")
	_ = os.WriteFile(bad, []byte("not a rar"), 0o644)
	if _, err := StageUpdate(bad, filepath.Join(t.TempDir(), "stage2")); err == nil {
		t.Fatal("a broken .rar must be refused")
	}
}

func TestUpdateVersions(t *testing.T) {
	for name, want := range map[string][2]string{
		"Phantom.Blade.Zero.Update.v1.0.2.to.v1.0.3.rar":   {"1.0.2", "1.0.3"},
		"PBZ Update v1.0.2 - v1.0.3.7z":                    {"1.0.2", "1.0.3"},
		"Game_Update_v1.0302256.zip":                       {"", "1.0302256"},
		"Game.Update.Build.16251443.to.Build.16301122.rar": {"build 16251443", "build 16301122"},
		"Game Build 16301122.zip":                          {"", "build 16301122"},
		"Update.part1.rar":                                 {"", ""},
		"Hotfix 2.zip":                                     {"", ""},
		"Hollow Tale v1.3":                                 {"", "1.3"},
		"Game.v1.3.part01.rar":                             {"", "1.3"},
	} {
		f, to := UpdateVersions(name)
		if f != want[0] || to != want[1] {
			t.Errorf("UpdateVersions(%q) = %q, %q; want %q, %q", name, f, to, want[0], want[1])
		}
	}
	if FolderVersion(`D:\Games\Phantom.Blade.Zero.v1.0.2-InsaneRamZes`) != "1.0.2" || FolderVersion(`D:\Games\Phantom.Blade.Zero-InsaneRamZes`) != "" {
		t.Fatal("FolderVersion")
	}
	for _, c := range []struct {
		have, from, to, want string
	}{
		{"1.0.2", "1.0.2", "1.0.3", "ok"},
		{"1.0.1", "1.0.2", "1.0.3", "skips"},
		{"1.0.3", "", "1.0.2", "older"},
		{"1.0.3", "", "1.0.3", "same"},
		{"1.0.9", "", "1.0.10", "ok"},
		{"", "1.0.2", "1.0.3", "unknown"},
		{"build 5", "", "1.2", "unknown"},
	} {
		if got := CheckVersions(c.have, c.from, c.to).State; got != c.want {
			t.Errorf("CheckVersions(%q, %q, %q) = %s, want %s", c.have, c.from, c.to, got, c.want)
		}
	}
	applied := []AppliedUpdate{{State: "undone", To: "1.0.4"}, {State: "applied", To: "1.0.3"}}
	if CopyVersion(`D:\Games\Game.v1.0.2`, applied) != "1.0.3" || CopyVersion(`D:\Games\Game.v1.0.2`, nil) != "1.0.2" {
		t.Fatal("CopyVersion")
	}
}
