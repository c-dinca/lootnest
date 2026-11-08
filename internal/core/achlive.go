package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Achievements as they unlock. While a game runs, Lootnest reads the
// achievement file a Steam-compatible tool keeps for it (the same files the
// game page reads) every few seconds, and shows each new one in a small
// window of its own, over the game. Nothing is injected into the game and
// nothing reads its memory.
//
// How rare each one is comes from Steam's public global percentages (no key
// needed), only when the user turns that on.
// ---------------------------------------------------------------------------

// NewUnlocks lists the achievements in after that weren't in before, in the
// order they were unlocked.
func NewUnlocks(before, after map[string]time.Time) []string {
	var out []string
	for id := range after {
		if _, had := before[id]; !had {
			out = append(out, id)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		ti, tj := after[out[i]], after[out[j]]
		if !ti.Equal(tj) {
			return ti.Before(tj)
		}
		return out[i] < out[j]
	})
	return out
}

// ParseGlobalPercentages reads Steam's GetGlobalAchievementPercentagesForApp:
// each achievement's id and the share of players who have it. Steam has sent
// the percentage both as a number and as a string.
func ParseGlobalPercentages(b []byte) map[string]float64 {
	var r struct {
		P struct {
			A []struct {
				Name    string          `json:"name"`
				Percent json.RawMessage `json:"percent"`
			} `json:"achievements"`
		} `json:"achievementpercentages"`
	}
	if json.Unmarshal(b, &r) != nil {
		return nil
	}
	out := map[string]float64{}
	for _, a := range r.P.A {
		s := strings.Trim(string(a.Percent), `"`)
		if p, err := strconv.ParseFloat(s, 64); err == nil && a.Name != "" && p >= 0 && p <= 100 {
			out[a.Name] = p
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ApplyPercentages fills in how rare each achievement is, where it isn't
// known yet. Ids match exactly, else ignoring case.
func ApplyPercentages(list []Achievement, pct map[string]float64) {
	if len(pct) == 0 {
		return
	}
	lower := map[string]float64{}
	for k, v := range pct {
		lower[strings.ToLower(k)] = v
	}
	for i := range list {
		if list[i].Percent > 0 {
			continue
		}
		if p, ok := pct[list[i].ID]; ok {
			list[i].Percent = p
		} else if p, ok := lower[strings.ToLower(list[i].ID)]; ok {
			list[i].Percent = p
		}
	}
}

// Rarity names how rare an achievement is from the share of players who
// have it ("" when that isn't known).
func Rarity(percent float64) string {
	switch {
	case percent <= 0:
		return ""
	case percent < 2:
		return "Ultra rare"
	case percent < 10:
		return "Rare"
	case percent < 30:
		return "Uncommon"
	}
	return "Common"
}

// RarityText is the line under an achievement's name: "Rare, 4.2% of
// players have it".
func RarityText(percent float64) string {
	r := Rarity(percent)
	if r == "" {
		return ""
	}
	p := strconv.FormatFloat(percent, 'f', 1, 64)
	if percent >= 10 {
		p = strconv.Itoa(int(percent + 0.5))
	}
	return r + ", " + p + "% of players have it"
}

// AntiCheat names the anti-cheat a game's folder ships with (Easy
// Anti-Cheat, BattlEye, Vanguard, FACEIT, EA AntiCheat), or "". Lootnest
// keeps its windows away from those games unless the user turns them on for
// one, and never puts mods into them.
func AntiCheat(dir string) string {
	if dir == "" {
		return ""
	}
	names := map[string]string{
		"easyanticheat": "Easy Anti-Cheat", "easyanticheat_eos": "Easy Anti-Cheat",
		"easyanticheat_eos_setup.exe": "Easy Anti-Cheat", "easyanticheat_setup.exe": "Easy Anti-Cheat",
		"start_protected_game.exe": "Easy Anti-Cheat",
		"battleye":                 "BattlEye", "beservice.exe": "BattlEye", "beservice_x64.exe": "BattlEye",
		"vgk.sys": "Vanguard", "vgc.exe": "Vanguard",
	}
	look := func(d string) string {
		es, err := os.ReadDir(d)
		if err != nil {
			return ""
		}
		for _, e := range es {
			n := strings.ToLower(e.Name())
			if ac, ok := names[n]; ok {
				return ac
			}
			if ext := filepath.Ext(n); strings.HasPrefix(n, "faceit") && (ext == ".exe" || ext == ".sys") {
				return "FACEIT"
			}
			if strings.HasPrefix(n, "eaanticheat") {
				return "EA AntiCheat"
			}
		}
		return ""
	}
	if n := look(dir); n != "" {
		return n
	}
	es, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range es {
		if e.IsDir() {
			if n := look(filepath.Join(dir, e.Name())); n != "" {
				return n
			}
		}
	}
	return ""
}

// RarestUnlocked are the n unlocked achievements fewest players have, rarest
// first (only the ones whose share is known).
func RarestUnlocked(list []Achievement, n int) []Achievement {
	var out []Achievement
	for _, a := range list {
		if a.Unlocked && a.Percent > 0 {
			out = append(out, a)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Percent < out[j].Percent })
	if len(out) > n {
		out = out[:n]
	}
	if out == nil {
		out = []Achievement{}
	}
	return out
}

// RarityColor is the colour of an achievement's tier ("#RRGGBB"), the same
// as the app's tier tokens: gold for ultra rare, blue for rare, green for
// uncommon, silver for common and when it isn't known.
func RarityColor(percent float64) string {
	switch Rarity(percent) {
	case "Ultra rare":
		return "#F2C14E"
	case "Rare":
		return "#6FA8F5"
	case "Uncommon":
		return "#5FC28E"
	}
	return "#AEB6C2"
}
