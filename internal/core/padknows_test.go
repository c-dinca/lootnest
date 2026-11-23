package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPSKnowsOfCats(t *testing.T) {
	// TEKKEN 8's categories on Steam (Nov 2026): 55 to 58 beside Partial.
	tekken := psKnowsOfCats(map[int]bool{2: true, 1: true, 18: true, 55: true, 56: true, 57: true, 58: true})
	if !tekken.DualSense || !tekken.DualShock || tekken.Why != "steam" {
		t.Fatalf("TEKKEN 8: %+v", tekken)
	}
	// ELDEN RING: Full controller support only, so an Xbox pad.
	if k := psKnowsOfCats(map[int]bool{2: true, 28: true}); k.DualSense || k.DualShock || k.Why != "" {
		t.Fatalf("ELDEN RING: %+v", k)
	}
	only4 := psKnowsOfCats(map[int]bool{56: true})
	if !only4.Knows(false) || only4.Knows(true) {
		t.Fatalf("DualShock only: %+v", only4)
	}
	if k := psKnowsOfCats(nil); k.Why != "" {
		t.Fatalf("no categories: %+v", k)
	}
}

func TestDetailsPS(t *testing.T) {
	if k := DetailsPS(&GameDetails{Pads: []string{"playstation"}}); !k.DualSense || !k.DualShock || k.Why != "details" {
		t.Fatalf("playstation: %+v", k)
	}
	if k := DetailsPS(&GameDetails{Controller: "full"}); k.Why != "" {
		t.Fatalf("full only: %+v", k)
	}
	if k := DetailsPS(nil); k.Why != "" {
		t.Fatalf("nil: %+v", k)
	}
}

func TestFolderPS(t *testing.T) {
	dir := t.TempDir()
	if k := FolderPS(dir); k.Why != "" {
		t.Fatalf("empty folder: %+v", k)
	}
	// An Unreal game keeps it in <Game>\Binaries\Win64, three levels down.
	bin := filepath.Join(dir, "Polaris", "Binaries", "Win64")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "libScePad.dll"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if k := FolderPS(dir); !k.DualSense || !k.DualShock || k.Why != "files" {
		t.Fatalf("libScePad.dll: %+v", k)
	}
	// Deeper than four levels isn't looked into.
	deep := t.TempDir()
	far := filepath.Join(deep, "a", "b", "c", "d", "e")
	if err := os.MkdirAll(far, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(far, "libScePad.dll"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if k := FolderPS(deep); k.Why != "" {
		t.Fatalf("too deep: %+v", k)
	}
	if k := FolderPS(""); k.Why != "" {
		t.Fatalf("no folder: %+v", k)
	}
}

func TestLaunchOptionsPadOneOrOther(t *testing.T) {
	o, err := CleanLaunchOptions(LaunchOptions{KeepPad: true, XboxPad: true})
	if err != nil {
		t.Fatal(err)
	}
	if !o.KeepPad || o.XboxPad {
		t.Fatalf("both: %+v", o)
	}
	if o, _ := CleanLaunchOptions(LaunchOptions{XboxPad: true}); !o.XboxPad || o.Empty() {
		t.Fatalf("Xbox: %+v", o)
	}
}
