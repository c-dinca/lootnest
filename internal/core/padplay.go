package core

import (
	"path/filepath"
	"strings"
)

// A game plays with a controller ("full" or "partial"), not ("none"), or
// nobody says (""). Big Picture keeps games with "none" out of its rows
// (app/padplay.go). Only what's on this PC and Steam's store details the
// user already has are read here.

// padOfCommon reads a Steam app's controller support from appinfo.vdf's
// common section: its controller_support, else its store categories (28
// is Full controller support, 18 Partial). An app whose categories are
// there with neither has none; one with no categories, nobody says.
func padOfCommon(common map[string]any) string {
	switch v := strings.ToLower(vdfStr(common, "controller_support")); v {
	case "full", "partial":
		return v
	}
	cats, ok := common["category"].(map[string]any)
	if !ok || len(cats) == 0 {
		return ""
	}
	if _, ok := cats["category_28"]; ok {
		return "full"
	}
	if _, ok := cats["category_18"]; ok {
		return "partial"
	}
	return "none"
}

// SteamControllers is the controller support Steam's appinfo.vdf on this
// PC knows for the apps in want (app id -> full, partial or none). Steam
// keeps there the apps of its account and the ones it looked at.
func SteamControllers(steamPath string, want map[int]bool) map[int]string {
	out := map[int]string{}
	if steamPath == "" || len(want) == 0 {
		return out
	}
	for id, app := range steamAppInfo(filepath.Join(steamPath, "appcache", "appinfo.vdf"), want) {
		if app.pad != "" {
			out[id] = app.pad
		}
	}
	return out
}

// noPadGames are games outside Steam played only with a keyboard and a
// mouse, by name, for launchers that say nothing about controllers.
var noPadGames = map[string]bool{
	"league of legends": true, "valorant": true, "teamfight tactics": true, "legends of runeterra": true,
	"starcraft": true, "starcraft ii": true, "starcraft remastered": true, "hearthstone": true,
	"heroes of the storm": true, "world of tanks": true, "world of warships": true,
}

// PadGuess is what a game's launcher and name say when nothing else does:
// Riot's games are played with a keyboard and a mouse (2XKO, a fighting
// game, aside), and so are a few well-known others.
func PadGuess(source, name string) string {
	n := strings.ToLower(strings.TrimSpace(strings.NewReplacer("®", "", "™", "").Replace(name)))
	if strings.Contains(n, "2xko") {
		return ""
	}
	if noPadGames[n] || source == "riot" {
		return "none"
	}
	return ""
}

// DetailsPad reads controller support from a game's Steam store details
// kept on this PC: full or partial as Steam says, none when the details
// were read with their categories (ReqsRead, since 2.1.0) and name none.
func DetailsPad(d *GameDetails) string {
	if d == nil || d.Unknown {
		return ""
	}
	if d.Controller != "" {
		return d.Controller
	}
	if d.ReqsRead {
		return "none"
	}
	return ""
}
