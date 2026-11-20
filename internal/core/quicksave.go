package core

import (
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Save now: before a save point is made while a game runs, Lootnest presses
// the game's own quick save key, waits until the game has written its
// saves, then copies them, so the point is the moment it was asked for and
// not the game's last save. The key is the user's choice per game
// (LaunchOptions.QuickSave); QuickSaveSuggestion only offers one.
// ---------------------------------------------------------------------------

// QuickSaveKeys are the keys a game's quick save can be on.
var QuickSaveKeys = []string{"F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9", "F10", "F11", "F12"}

// QuickSaveVK is a quick save key's Windows virtual-key code.
func QuickSaveVK(name string) (uint16, bool) {
	for i, k := range QuickSaveKeys {
		if strings.EqualFold(k, name) {
			return uint16(0x70 + i), true // VK_F1 onward
		}
	}
	return 0, false
}

// CleanQuickSave is a key name as stored, or "" for none.
func CleanQuickSave(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", true
	}
	for _, k := range QuickSaveKeys {
		if strings.EqualFold(k, name) {
			return k, true
		}
	}
	return "", false
}

// quickSaveByApp are games whose quick save key is known, by Steam app id.
var quickSaveByApp = map[int]string{
	// Bethesda
	22320: "F5", 22330: "F5", 22300: "F5", 22370: "F5", 22380: "F5",
	72850: "F5", 489830: "F5", 377160: "F5", 1716740: "F5",
	// CD Projekt Red
	20920: "F5", 292030: "F5", 1091500: "F5",
	// Larian
	435150: "F5", 1086940: "F5",
	// Arkane
	205100: "F5", 403640: "F5", 480490: "F5",
	// Valve's Source games, and Black Mesa
	70: "F6", 220: "F6", 400: "F6", 620: "F6", 362890: "F6",
	// Owlcat, Obsidian, ZA/UM
	640820: "F5", 1184370: "F5", 291650: "F5", 560130: "F5", 632470: "F5",
	// Eidos Montreal's Deus Ex
	238010: "F5", 337000: "F5",
}

// quickSaveByName are the same games by name, for copies with no app id.
var quickSaveByName = map[string]string{
	"morrowind": "F5", "oblivion": "F5", "fallout3": "F5", "falloutnewvegas": "F5",
	"fallout4": "F5", "skyrim": "F5", "skyrimspecialedition": "F5", "starfield": "F5",
	"thewitcher2": "F5", "thewitcher3wildhunt": "F5", "cyberpunk2077": "F5",
	"divinityoriginalsin2": "F5", "baldursgate3": "F5",
	"dishonored": "F5", "dishonored2": "F5", "prey": "F5",
	"halflife": "F6", "halflife2": "F6", "portal": "F6", "portal2": "F6", "blackmesa": "F6",
	"pathfinderkingmaker": "F5", "pathfinderwrathoftherighteous": "F5",
	"pillarsofeternity": "F5", "pillarsofeternityiideadfire": "F5", "discoelysium": "F5",
	"deusexhumanrevolution": "F5", "deusexmankinddivided": "F5",
	"theelderscrollsiiimorrowind": "F5", "theelderscrollsivoblivion": "F5",
	"theelderscrollsvskyrim": "F5", "theelderscrollsvskyrimspecialedition": "F5",
	"thewitcher3": "F5", "thewitcher2assassinsofkings": "F5", "discoelysiumthefinalcut": "F5",
	"deusexhumanrevolutiondirectorscut": "F5",
}

// QuickSaveSuggestion is the quick save key a game is known to have, or "".
// It is only offered: Lootnest presses a key only once the user chose it.
func QuickSaveSuggestion(appID int, names ...string) string {
	if k := quickSaveByApp[appID]; k != "" {
		return k
	}
	for _, n := range names {
		var b strings.Builder
		for _, r := range strings.ToLower(n) {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				b.WriteRune(r)
			}
		}
		key := b.String()
		for _, cut := range []string{"definitiveedition", "gameoftheyearedition", "goty", "anniversaryedition", "remastered", "directorscut", "enhancededition", "completeedition", "ultimateedition"} {
			key = strings.TrimSuffix(key, cut)
		}
		if k := quickSaveByName[key]; k != "" {
			return k
		}
	}
	return ""
}

// SaveStamps are the stamps of every file in a game's save locations.
type SaveStamps map[string]map[string]FileStamp

// StampSaves reads the stamps of each location's files.
func StampSaves(locs []SaveLoc) (SaveStamps, error) {
	out := SaveStamps{}
	for _, l := range locs {
		s, err := stampOf(l.Live)
		if err != nil {
			return nil, err
		}
		out[l.ID] = s
	}
	return out, nil
}

// SameSaves says whether two readings of the saves are the same.
func SameSaves(a, b SaveStamps) bool { return sameSaves(a, b) }

func sameSaves(a, b SaveStamps) bool {
	if len(a) != len(b) {
		return false
	}
	for id, s := range a {
		if !sameStamps(s, b[id]) {
			return false
		}
	}
	return true
}

// WaitForGameSave waits until the saves differ from before, then until
// they stay the same for settle (the game may write several files, or one
// in parts). It says whether the game saved within limit. An unreadable
// location counts as still being written.
func WaitForGameSave(locs []SaveLoc, before SaveStamps, limit, settle, every time.Duration) bool {
	end := time.Now().Add(limit)
	var last SaveStamps
	for time.Now().Before(end) {
		time.Sleep(every)
		now, err := StampSaves(locs)
		if err != nil || sameSaves(now, before) {
			continue
		}
		last = now
		break
	}
	if last == nil {
		return false
	}
	// Settled: the same stamps for settle in a row, within as long again
	// as limit (a large save written slowly still finishes).
	stillSince, giveUp := time.Now(), time.Now().Add(limit)
	for time.Now().Before(giveUp) {
		time.Sleep(every)
		now, err := StampSaves(locs)
		if err != nil || !sameSaves(now, last) {
			last, stillSince = now, time.Now()
			continue
		}
		if time.Since(stillSince) >= settle {
			return true
		}
	}
	return true // it saved; the copy's own read-back catches a file still moving
}
