package core

import "strings"

// Runs are saves of their own in a story (Player.Of, Player.Game). A
// competitive game has no story to keep a run of, and its anti-cheat
// watches its files, so Lootnest offers no runs there (the owner, 23 Nov
// 2026). Profiles, one per person, still swap its settings.

// competitiveAppIDs are games with no story, by Steam app id.
var competitiveAppIDs = map[int]bool{
	730: true, 570: true, 440: true, // Counter-Strike 2, Dota 2, Team Fortress 2
	1422450: true, // Deadlock
	2357570: true, // Overwatch 2
	1172470: true, // Apex Legends
	578080:  true, // PUBG
	359550:  true, // Rainbow Six Siege
	2073850: true, // The Finals
	2767030: true, // Marvel Rivals
	252950:  true, // Rocket League
	381210:  true, // Dead by Daylight
	1203220: true, // Naraka: Bladepoint
	252490:  true, // Rust
}

// competitiveWords are the same by words of a game's name.
var competitiveWords = []string{
	"counter-strike", "counter strike", "valorant", "league of legends", "teamfight tactics", "dota 2",
	"deadlock", "overwatch", "apex legends", "pubg", "fortnite", "rainbow six siege", "the finals",
	"marvel rivals", "rocket league", "team fortress 2", "escape from tarkov", "dead by daylight",
	"naraka", "2xko", "warzone", "xdefiant", "splitgate",
}

// kernelAntiCheats run with the system's rights: games with one are
// competitive whatever their name says.
var kernelAntiCheats = map[string]bool{"Vanguard": true, "FACEIT": true, "EA AntiCheat": true}

// CompetitiveGame says why a game gets no runs ("" when it may): its Steam
// app id or name on Lootnest's list, or a kernel anti-cheat in its folder.
func CompetitiveGame(appID int, name, dir string) string {
	if competitiveAppIDs[appID] {
		return "a competitive game"
	}
	n := " " + strings.Join(strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-')
	}), " ") + " "
	for _, w := range competitiveWords {
		if strings.Contains(n, " "+w+" ") {
			return "a competitive game"
		}
	}
	if ac := AntiCheat(dir); kernelAntiCheats[ac] {
		return "it has " + ac
	}
	return ""
}
