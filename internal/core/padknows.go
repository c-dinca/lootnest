package core

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// Whether a game knows PlayStation controllers itself, for the pad
// Lootnest makes of them (app/padbridge.go). A game that does would see a
// DualSense twice once Lootnest adds an Xbox pad for it: as itself and as
// the Xbox pad, so one controller plays for two people (the owner, 23 Nov
// 2026, with two DualSenses in TEKKEN 8). Such a game gets the pads as
// they are; one that doesn't gets an Xbox pad for each.

// PSKnows is what a game knows of PlayStation pads, and how Lootnest knows.
type PSKnows struct {
	DualShock bool   // the DualShock 4
	DualSense bool   // the DualSense and DualSense Edge
	Why       string // "steam" (its store categories), "details", "files", or "" when nothing says
}

// Knows says whether the game reads this pad itself.
func (k PSKnows) Knows(dualsense bool) bool {
	if dualsense {
		return k.DualSense
	}
	return k.DualShock
}

// Steam's store categories for PlayStation pads: 55 and 56 are "DualShock
// Controller Support", 57 and 58 "DualSense Controller Support" (one of
// each pair is the Bluetooth one; either says the game reads the pad).
const (
	steamCatDualShock   = 55
	steamCatDualShockBT = 56
	steamCatDualSense   = 57
	steamCatDualSenseBT = 58
)

// psKnowsOfCats reads a Steam app's categories.
func psKnowsOfCats(cats map[int]bool) PSKnows {
	k := PSKnows{
		DualShock: cats[steamCatDualShock] || cats[steamCatDualShockBT],
		DualSense: cats[steamCatDualSense] || cats[steamCatDualSenseBT],
	}
	if k.DualShock || k.DualSense {
		k.Why = "steam"
	}
	return k
}

// SteamPSPads is what Steam's appinfo.vdf on this PC says of the apps in
// want: only the ones that name a PlayStation pad are in the answer.
func SteamPSPads(steamPath string, want map[int]bool) map[int]PSKnows {
	out := map[int]PSKnows{}
	if steamPath == "" || len(want) == 0 {
		return out
	}
	for id, app := range steamAppInfo(filepath.Join(steamPath, "appcache", "appinfo.vdf"), want) {
		if k := psKnowsOfCats(app.cats); k.Why != "" {
			out[id] = k
		}
	}
	return out
}

// DetailsPS reads a game's Steam store details kept on this PC (1.85's
// Pads, from the same categories by their words): "playstation" counts
// for both pads, since the words don't say which.
func DetailsPS(d *GameDetails) PSKnows {
	if d == nil {
		return PSKnows{}
	}
	for _, p := range d.Pads {
		if p == "playstation" {
			return PSKnows{DualShock: true, DualSense: true, Why: "details"}
		}
	}
	return PSKnows{}
}

// psLibraries are files Sony's PC library for its pads leaves in a game's
// folder (Sony's own ports, and others that use it).
var psLibraries = []string{"libscepad.dll", "libscepadv3.dll", "libscepadex.dll"}

// psFilesMax keeps the look into a big game's folder short.
const psFilesMax = 4000

// FolderPS looks for Sony's pad library in a game's folder, four levels
// down at most (an Unreal game keeps it in <Game>\Binaries\Win64).
func FolderPS(dir string) PSKnows {
	if dir == "" {
		return PSKnows{}
	}
	root := filepath.Clean(dir)
	seen, found := 0, false
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || found {
			return fs.SkipDir
		}
		if seen++; seen > psFilesMax {
			return fs.SkipAll
		}
		if d.IsDir() {
			if rel, err := filepath.Rel(root, p); err == nil && rel != "." && strings.Count(rel, string(filepath.Separator)) >= 3 {
				return fs.SkipDir
			}
			return nil
		}
		name := strings.ToLower(d.Name())
		for _, l := range psLibraries {
			if name == l {
				found = true
				return fs.SkipAll
			}
		}
		return nil
	})
	if !found {
		return PSKnows{}
	}
	return PSKnows{DualShock: true, DualSense: true, Why: "files"}
}
