package core

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Achievements from the user's own Steam account, for games they also own on
// Steam (Settings, Online; off by default). Steam's Web API answers
// ISteamUserStats/GetPlayerAchievements with the user's own key and their
// SteamID64. Steam only answers for a profile whose game details are public.
// ---------------------------------------------------------------------------

// SteamID64 is the SteamID64 of a Steam\userdata\<account id> folder.
func SteamID64(userDir string) string {
	id, err := strconv.ParseUint(filepath.Base(filepath.Clean(userDir)), 10, 64)
	if err != nil || id == 0 || id >= steamID64Base {
		return ""
	}
	return strconv.FormatUint(id+steamID64Base, 10)
}

// ErrSteamPrivate: Steam won't show the account's achievements, because
// the profile's game details aren't public.
var ErrSteamPrivate = errors.New("Steam keeps this account's achievements private. In Steam, set the profile's Game details to Public.")

// ErrSteamNoStats: the game has no achievements on Steam, or the account
// doesn't own it there.
var ErrSteamNoStats = errors.New("This Steam account has no achievements for this game.")

// ParsePlayerAchievements reads GetPlayerAchievements' answer: every
// achievement it lists (name and description when asked with l=), with
// unlocked ones carrying their time.
func ParsePlayerAchievements(b []byte) ([]Achievement, error) {
	var v struct {
		Stats struct {
			Success      bool   `json:"success"`
			Error        string `json:"error"`
			Achievements []struct {
				ID       string `json:"apiname"`
				Achieved int    `json:"achieved"`
				Unlock   int64  `json:"unlocktime"`
				Name     string `json:"name"`
				Desc     string `json:"description"`
			} `json:"achievements"`
		} `json:"playerstats"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, errors.New("Steam's answer couldn't be read.")
	}
	if !v.Stats.Success {
		if strings.Contains(strings.ToLower(v.Stats.Error), "not public") {
			return nil, ErrSteamPrivate
		}
		return nil, ErrSteamNoStats
	}
	var out []Achievement
	for _, a := range v.Stats.Achievements {
		id := strings.TrimSpace(a.ID)
		if id == "" {
			continue
		}
		x := Achievement{ID: id, Name: strings.TrimSpace(a.Name), Description: strings.TrimSpace(a.Desc), Unlocked: a.Achieved == 1}
		if x.Unlocked && a.Unlock > 0 {
			x.At = time.Unix(a.Unlock, 0).UTC()
		}
		out = append(out, x)
	}
	if len(out) == 0 {
		return nil, ErrSteamNoStats
	}
	return out, nil
}

// AccountUnlocks are the unlocked ones of a player's list, by id.
func AccountUnlocks(list []Achievement) map[string]time.Time {
	out := map[string]time.Time{}
	for _, a := range list {
		if a.Unlocked {
			out[a.ID] = a.At
		}
	}
	return out
}
