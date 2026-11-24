package plat

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProcessNames(t *testing.T) {
	proc := t.TempDir()
	add := func(pid, comm, cmdline string) {
		dir := filepath.Join(proc, pid)
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(dir, "comm"), []byte(comm+"\n"), 0o644)
		_ = os.WriteFile(filepath.Join(dir, "cmdline"), []byte(cmdline), 0o644)
	}
	add("100", "tuxracer", "/usr/games/tuxracer\x00--fullscreen\x00")
	// A game run through Proton: comm is cut to 15 characters, the command
	// line names the Windows program.
	add("200", "AshenRing-Win64", "C:\\Games\\Ashen Ring\\AshenRing-Win64-Shipping.exe\x00-dx12\x00")
	add("300", "wine64-preload", "/usr/bin/wine64-preload\x00Z:\\home\\alex\\Games\\Quiet Isle\\QuietIsle.EXE\x00")
	add("400", "TinyGame.x86_6", "/home/alex/Games/tiny/TinyGame.x86_64\x00")
	add("self", "x", "x")
	got, err := processNamesIn(proc)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"tuxracer", "ashenring-win64-shipping", "quietisle", "wine64-preload", "tinygame", "tinygame.x86_64"} {
		if !got[want] {
			t.Errorf("%s isn't listed: %v", want, got)
		}
	}
	if got["x"] {
		t.Error("only numbered folders are processes")
	}
	if free, total := DiskSpace(proc); free <= 0 || total < free {
		t.Errorf("disk space: %d of %d", free, total)
	}
	if d := Documents(); d == "" {
		t.Error("Documents")
	}
}
