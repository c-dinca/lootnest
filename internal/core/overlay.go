package core

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ---------------------------------------------------------------------------
// The FPS counter inside a game (native/overlay, app/overlay.go): a DLL of
// Lootnest's own hooks the game's Present, as Steam's and Discord's
// overlays do, counts frames there and draws the number into the frame,
// so it shows over exclusive fullscreen too and needs no rights at all.
// Only in story games (CLAUDE.md, Story games and online games): never in
// a game with an anti-cheat anywhere in its folder, a Steam game with
// Valve Anti-Cheat, or a game known to be played online; and never while
// any anti-cheat runs. Denuvo and other anti-tamper are allowed (the owner,
// 20 Nov 2026): the DLL touches only the graphics API's Present, as those
// overlays do. Everything else keeps the counter outside the game.
// ---------------------------------------------------------------------------

// OverlayShare is the 64-byte block "Local\LootnestOverlay-<pid>" Lootnest
// makes before the DLL goes in; native/overlay/overlay.c's ln_share.
const (
	OverlayMagic   = 0x564F4E4C // "LNOV"
	OverlayVersion = 1
	OverlaySize    = 64
	OverOn         = 8  // int32, Lootnest: draw the number
	OverCorner     = 12 // int32, Lootnest: 0 tl, 1 tr, 2 bl, 3 br
	OverFPS        = 16 // int32, the DLL: frames in the last second
	OverFrameUS    = 20 // int32, the DLL: a frame's time in microseconds
	OverAPI        = 24 // int32, the DLL: OverlayAPIs
	OverDrawn      = 28 // int32, the DLL: frames the number was drawn on
	OverBeat       = 32 // int64, the DLL: QueryPerformanceCounter at the last frame
	OverError      = 40 // int32, the DLL: what failed (OverlayErrors)
	OverHooked     = 44 // int32, the DLL: 1 DXGI, 2 Direct3D 9, 4 OpenGL
)

// OverlayAPIs name the DLL's api numbers; those it draws in are true.
var OverlayAPIs = map[int32]struct {
	Name  string
	Draws bool
}{1: {"Direct3D 11", true}, 2: {"Direct3D 12", false}, 3: {"Direct3D 9", true}, 4: {"OpenGL", true}}

// OverlayErrors name the DLL's error numbers.
var OverlayErrors = map[int32]string{
	1: "it found no shared block", 2: "MinHook didn't start", 3: "DXGI's Present couldn't be hooked",
	4: "Direct3D 9's Present couldn't be hooked", 5: "OpenGL's SwapBuffers couldn't be hooked",
	6: "it couldn't draw on the game's frame",
}

// FPSCornerIndex is the DLL's number for a corner.
func FPSCornerIndex(c string) int32 {
	switch CleanFPSCorner(c) {
	case "tr":
		return 1
	case "bl":
		return 2
	case "br":
		return 3
	}
	return 0
}

// Steam's store categories that say a game is played online, and Valve
// Anti-Cheat.
const (
	steamCatVAC          = 8
	steamCatSinglePlayer = 2
)

var steamCatsOnline = []int{1, 20, 36, 49} // Multi-player, MMO, Online PvP, PvP

// SteamOnline is, for the apps in want, why Steam's files on this PC say
// each is an online game: "Valve Anti-Cheat", or "online only" for one
// with online categories and no Single-player. Apps Steam says nothing
// about are left out.
func SteamOnline(steamPath string, want map[int]bool) map[int]string {
	out := map[int]string{}
	if steamPath == "" || len(want) == 0 {
		return out
	}
	for id, app := range steamAppInfo(filepath.Join(steamPath, "appcache", "appinfo.vdf"), want) {
		if why := onlineByCategories(app.cats); why != "" {
			out[id] = why
		}
	}
	return out
}

func onlineByCategories(cats map[int]bool) string {
	if cats[steamCatVAC] {
		return "Valve Anti-Cheat"
	}
	if cats[steamCatSinglePlayer] {
		return ""
	}
	for _, c := range steamCatsOnline {
		if cats[c] {
			return "online only"
		}
	}
	return ""
}

// onlineGames are games known to be played online, or whose story shares
// its program with an online mode watched by an anti-cheat that leaves no
// file to find (CLAUDE.md): by Steam app id and by name.
var onlineAppIDs = map[int]bool{
	271590: true, 3240220: true, // Grand Theft Auto V (and Enhanced)
	1174180: true,                // Red Dead Redemption 2
	1245620: true,                // Elden Ring
	1938090: true,                // Call of Duty
	1085660: true,                // Destiny 2
	230410:  true,                // Warframe
	238960:  true, 2694490: true, // Path of Exile 1 and 2
	2344520: true,                       // Diablo IV
	2357570: true,                       // Overwatch 2
	1172470: true,                       // Apex Legends
	578080:  true,                       // PUBG
	730:     true, 570: true, 440: true, // CS2, Dota 2, TF2
	1422450: true, // Deadlock
}

var onlineNameWords = []string{
	"grand theft auto v", "gta v", "gta 5", "red dead redemption 2", "red dead online", "call of duty",
	"destiny 2", "warframe", "path of exile", "diablo iv", "overwatch", "apex legends", "pubg",
	"counter-strike", "valorant", "league of legends", "dota 2", "fortnite", "rainbow six", "the finals",
	"genshin impact", "honkai", "zenless zone zero", "wuthering waves", "world of warcraft", "lost ark",
	"new world", "escape from tarkov", "rust", "dead by daylight", "battlefield", "halo infinite",
	"marvel rivals", "deadlock", "naraka", "the division", "fifa", "ea sports fc", "nba 2k",
	"elden ring", "dark souls iii", "for honor", "sea of thieves", "forza horizon",
}

// OnlineGame says why a game is online by Lootnest's list ("" when it
// isn't on it): its Steam app id, else words of its name.
func OnlineGame(appID int, name string) string {
	if onlineAppIDs[appID] {
		return "played online"
	}
	n := " " + strings.Join(strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-')
	}), " ") + " "
	for _, w := range onlineNameWords {
		if strings.Contains(n, " "+w+" ") {
			return "played online"
		}
	}
	return ""
}

// AntiCheatDeep looks for an anti-cheat anywhere in a game's folders, a
// few levels down (AntiCheat looks only at the top and one level), as the
// in-game counter needs to be sure. It stops after 20,000 entries and then
// answers as if one were found, since it couldn't make sure.
func AntiCheatDeep(dirs ...string) string {
	seen := 0
	for _, root := range dirs {
		if root == "" {
			continue
		}
		if ac := AntiCheat(root); ac != "" {
			return ac
		}
		found := ""
		depth0 := strings.Count(filepath.Clean(root), string(filepath.Separator))
		filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			seen++
			if seen > 20000 {
				found = "unknown (too many files to look through)"
				return fs.SkipAll
			}
			n := strings.ToLower(d.Name())
			if ac := antiCheatName(n); ac != "" {
				found = ac
				return fs.SkipAll
			}
			if d.IsDir() && strings.Count(filepath.Clean(p), string(filepath.Separator))-depth0 >= 4 {
				return fs.SkipDir
			}
			return nil
		})
		if found != "" {
			return found
		}
	}
	return ""
}

// antiCheatName is the anti-cheat a file or folder name belongs to.
func antiCheatName(n string) string {
	switch {
	case n == "easyanticheat" || n == "easyanticheat_eos" || strings.HasPrefix(n, "easyanticheat") || n == "start_protected_game.exe":
		return "Easy Anti-Cheat"
	case n == "battleye" || strings.HasPrefix(n, "beservice") || strings.HasPrefix(n, "beclient"):
		return "BattlEye"
	case n == "vgk.sys" || n == "vgc.exe":
		return "Vanguard"
	case strings.HasPrefix(n, "faceit"):
		return "FACEIT"
	case strings.HasPrefix(n, "eaanticheat"):
		return "EA AntiCheat"
	case n == "gameguard" || n == "gameguard.des" || n == "gamemon.des":
		return "nProtect GameGuard"
	case strings.HasPrefix(n, "xigncode") || n == "x3.xem":
		return "XIGNCODE3"
	case n == "pnkbstra.exe" || n == "pnkbstrb.exe" || n == "pb":
		return "PunkBuster"
	case n == "mhyprot2.sys" || n == "mhyprot3.sys":
		return "mhyprot"
	case n == "ace-base64.sys" || strings.HasPrefix(n, "sguard"):
		return "Anti-Cheat Expert"
	case n == "nprotect" || n == "hackshield" || n == "ahnlab":
		return "HackShield"
	}
	return ""
}

// antiCheatProcs are anti-cheat programs and services by process name
// (lower case, without .exe).
var antiCheatProcs = map[string]string{
	"easyanticheat": "Easy Anti-Cheat", "easyanticheat_eos": "Easy Anti-Cheat", "eac_launcher": "Easy Anti-Cheat",
	"beservice": "BattlEye", "beservice_x64": "BattlEye", "bedaisy": "BattlEye",
	"vgc": "Vanguard", "vgtray": "Vanguard",
	"faceit": "FACEIT", "faceitclient": "FACEIT", "faceitservice": "FACEIT",
	"eaanticheat.gameservice": "EA AntiCheat", "eaanticheatservice": "EA AntiCheat",
	"gameguard": "nProtect GameGuard", "gamemon": "nProtect GameGuard", "gamemon64": "nProtect GameGuard",
	"pnkbstra": "PunkBuster", "pnkbstrb": "PunkBuster",
	"sguard64": "Anti-Cheat Expert", "ace-tray": "Anti-Cheat Expert",
	"xigncode": "XIGNCODE3", "xxd-0.xem": "XIGNCODE3",
	"mhyprot2": "mhyprot",
}

// AntiCheatRunning names an anti-cheat among the running processes (lower
// case, without .exe), "" for none.
func AntiCheatRunning(procs map[string]bool) string {
	for p := range procs {
		if ac, ok := antiCheatProcs[p]; ok {
			return ac
		}
	}
	return ""
}

// InGameFacts is what decides whether the counter may go into a game.
type InGameFacts struct {
	Dirs        []string // the game's folder and its program's
	AppID       int      // Steam app id, 0 when none
	Name        string
	SteamOnline string // SteamOnline's answer for the app id
	Running     map[string]bool
}

// InGameAllowed says whether the counter may go into the game, and if not,
// why (for the log and Settings).
func InGameAllowed(f InGameFacts) (bool, string) {
	if ac := AntiCheatDeep(f.Dirs...); ac != "" {
		return false, "it has an anti-cheat (" + ac + ")"
	}
	if f.SteamOnline != "" {
		return false, "Steam says it's played online (" + f.SteamOnline + ")"
	}
	if why := OnlineGame(f.AppID, f.Name); why != "" {
		return false, "it's " + why
	}
	if ac := AntiCheatRunning(f.Running); ac != "" {
		return false, "an anti-cheat runs (" + ac + ")"
	}
	return true, ""
}

// OverlayDLLName is the DLL's file name under Lootnest's data folder, by
// the first bytes of its SHA-256, so a newer Lootnest never writes over a
// copy a game still has loaded.
func OverlayDLLName(sum string) string {
	if len(sum) > 12 {
		sum = sum[:12]
	}
	return "lootnest-overlay64-" + sum + ".dll"
}

// PruneOverlayDLLs removes the DLL copies in dir other than keep; a copy a
// running game has loaded can't be removed and stays until next time.
func PruneOverlayDLLs(dir, keep string) {
	es, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range es {
		n := e.Name()
		if n != keep && strings.HasPrefix(n, "lootnest-overlay64-") && strings.HasSuffix(n, ".dll") {
			os.Remove(filepath.Join(dir, n))
		}
	}
}
