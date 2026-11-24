package plat

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKeyNames(t *testing.T) {
	for vk, want := range map[uint32]uint32{VK_F5: 0xffc2, VK_F12: 0xffc9, 'A': 'a', '7': '7', 0x20: 0} {
		if got := keysym(vk); got != want {
			t.Errorf("keysym(%#x) = %#x, want %#x", vk, got, want)
		}
	}
	if got := trigger(ModControl|ModShift, VK_F12); got != "CTRL+SHIFT+F12" {
		t.Errorf("trigger = %q", got)
	}
	if got := trigger(ModAlt, 'K'); got != "ALT+K" {
		t.Errorf("trigger = %q", got)
	}
}

// A key the app didn't declare isn't held.
func TestUndeclaredKey(t *testing.T) {
	SetHotKeys(nil, func(int) {}, nil)
	if RegisterHotKey(MainWindow, 99, ModControl, VK_F5) {
		t.Fatal("an undeclared key was held")
	}
}

// A window's process is known by the names ProcessNames gives: a Wine
// game by the .exe its command line runs.
func TestPidNames(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "4242")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "comm"), []byte("Game-Win64-Ship\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "cmdline"), []byte(`C:\Games\Hades II\Ship\Hades2.exe`+"\x00-dx12\x00"), 0o644)
	names, err := pidNames(dir)
	if err != nil || !names["hades2"] || !names["game-win64-ship"] {
		t.Fatalf("names %v, %v", names, err)
	}
	if _, err := pidNames(filepath.Join(dir, "gone")); err == nil {
		t.Fatal("a process that's gone has names")
	}
}
