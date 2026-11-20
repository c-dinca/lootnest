package core

import (
	"path/filepath"
	"testing"
	"time"
)

func TestQuickSaveKeys(t *testing.T) {
	if vk, ok := QuickSaveVK("f5"); !ok || vk != 0x74 {
		t.Fatalf("F5 is VK 0x74, got %x %v", vk, ok)
	}
	if vk, _ := QuickSaveVK("F12"); vk != 0x7B {
		t.Fatalf("F12 %x", vk)
	}
	if _, ok := QuickSaveVK("Enter"); ok {
		t.Fatal("only function keys")
	}
	if k, ok := CleanQuickSave(" f9 "); !ok || k != "F9" {
		t.Fatalf("clean %q", k)
	}
	if _, ok := CleanQuickSave("ctrl+s"); ok {
		t.Fatal("refused")
	}
	if k, ok := CleanQuickSave(""); !ok || k != "" {
		t.Fatal("none")
	}
	for _, c := range []struct {
		id   int
		name string
		want string
	}{
		{489830, "", "F5"},
		{220, "", "F6"},
		{0, "The Elder Scrolls V: Skyrim Special Edition", "F5"},
		{0, "Skyrim Special Edition", "F5"},
		{0, "Baldur's Gate 3", "F5"},
		{0, "Cyberpunk 2077", "F5"},
		{0, "Half-Life 2", "F6"},
		{0, "Disco Elysium - The Final Cut", "F5"},
		{0, "Fallout 4: Game of the Year Edition", "F5"},
		{0, "Counter-Strike 2", ""},
	} {
		if got := QuickSaveSuggestion(c.id, c.name); got != c.want {
			t.Errorf("%d %q: %q, want %q", c.id, c.name, got, c.want)
		}
	}
}

func TestWaitForGameSave(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "saves")
	write(t, filepath.Join(live, "quick.sav"), "before")
	locs := []SaveLoc{{ID: "a", Live: live}}
	before, err := StampSaves(locs)
	if err != nil {
		t.Fatal(err)
	}
	// Nothing written: no save.
	if WaitForGameSave(locs, before, 120*time.Millisecond, 50*time.Millisecond, 10*time.Millisecond) {
		t.Fatal("the game didn't save")
	}
	// The game writes two files, one after the other.
	go func() {
		time.Sleep(40 * time.Millisecond)
		write(t, filepath.Join(live, "quick.sav"), "after, longer")
		time.Sleep(40 * time.Millisecond)
		write(t, filepath.Join(live, "quick.png"), "picture")
	}()
	start := time.Now()
	if !WaitForGameSave(locs, before, 2*time.Second, 150*time.Millisecond, 10*time.Millisecond) {
		t.Fatal("the game saved")
	}
	if time.Since(start) < 200*time.Millisecond {
		t.Fatal("it waits until both files are written and still")
	}
	if !exists(filepath.Join(live, "quick.png")) {
		t.Fatal("returned before the second file")
	}
}
