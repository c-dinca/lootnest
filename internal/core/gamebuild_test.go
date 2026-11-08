package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGameBuild(t *testing.T) {
	steam := t.TempDir()
	write(t, filepath.Join(steam, "steamapps", "appmanifest_570.acf"), "\"AppState\"\n{\n\t\"appid\"\t\t\"570\"\n\t\"buildid\"\t\t\"16480123\"\n}\n")
	if b := GameBuild(steam, 570, ""); b != "build:16480123" {
		t.Fatalf("steam build %q", b)
	}
	write(t, filepath.Join(steam, "steamapps", "appmanifest_440.acf"), "\"AppState\"\n{\n\t\"buildid\"\t\t\"1\"\n\t\"TargetBuildID\"\t\t\"2\"\n}\n")
	if b := GameBuild(steam, 440, ""); b != "build:2" {
		t.Fatalf("an update Steam has waiting counts now: %q", b)
	}
	if GameBuild(steam, 730, "") != "" {
		t.Fatal("a game Steam doesn't list has no build")
	}

	exe := filepath.Join(t.TempDir(), "Game.exe")
	write(t, exe, "v1")
	first := GameBuild("", 0, exe)
	if first == "" {
		t.Fatalf("exe stamp %q", first)
	}
	if GameBuild("", 0, exe) != first {
		t.Fatal("the same exe gives the same stamp")
	}
	write(t, exe, "version 2")
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(exe, later, later); err != nil {
		t.Fatal(err)
	}
	if GameBuild("", 0, exe) == first {
		t.Fatal("a changed exe gives a new stamp")
	}
	if GameBuild("", 0, filepath.Join(t.TempDir(), "gone.exe")) != "" {
		t.Fatal("a missing exe has no stamp")
	}
}

func TestNoteBuild(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	note := func(key, build string) bool {
		t.Helper()
		changed, err := s.NoteBuild(key, build)
		if err != nil {
			t.Fatal(err)
		}
		return changed
	}
	if note("game:a", "build:1") {
		t.Fatal("the first start is not an update")
	}
	if note("game:a", "build:1") {
		t.Fatal("the same build is not an update")
	}
	if !note("game:a", "build:2") {
		t.Fatal("a new build is an update")
	}
	if note("game:a", "build:2") {
		t.Fatal("the update is noted once")
	}
	if note("game:a", "") || note("game:b", "") {
		t.Fatal("an unknown build is never an update")
	}
	if !note("game:a", "build:3") {
		t.Fatal("an unknown build keeps the last one seen")
	}
}
