//go:build windows

package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"lootnest/internal/core"
)

// ---------------------------------------------------------------------------
// Achievement Doctor (core/achdoctor.go): a read-only check of why a game
// shows no achievements, as an "achDoctor" event. Set the name in the game
// is the one change, and only when the user presses it.
// ---------------------------------------------------------------------------

func (a *App) apiAchDoctor(key string, appID int, dir, name string) {
	go func() {
		out := map[string]any{"key": key}
		defer func() { a.emit("achDoctor", out) }()
		if strings.EqualFold(filepath.Ext(dir), ".exe") {
			dir = filepath.Dir(dir)
		}
		var steps []core.DoctorStep
		add := func(state, title, detail, path string) {
			steps = append(steps, core.DoctorStep{State: state, Title: title, Detail: detail, Path: path})
		}

		// 1. Which game this is on Steam.
		var g *core.Game
		if id, ok := strings.CutPrefix(key, "game:"); ok {
			a.mu.Lock()
			if gg := a.cfg.Game(id); gg != nil {
				c := *gg
				g = &c
			}
			a.mu.Unlock()
		}
		from := "from the Steam game it is"
		if appID <= 0 && g != nil {
			appID = core.GameAppID(g)
			from = "from the game's settings in Lootnest"
		}
		if appID <= 0 && g == nil {
			if f, ok := a.foundByKey(key); ok && f.SteamID > 0 {
				appID = f.SteamID
				from = "from " + sourceName(f.Source)
			}
		}
		if appID <= 0 {
			if appID = core.FolderAppID(dir); appID > 0 {
				from = "from steam_appid.txt, or a tool's settings, next to the game"
			}
		}
		if appID <= 0 {
			if appID = knownStoreAppID(name); appID > 0 {
				from = "from the Steam store, by its name"
			}
		}
		if appID > 0 {
			add("ok", "It's Steam game "+strconv.Itoa(appID), "Lootnest found its Steam id "+from+".", "")
		} else {
			add("fail", "Lootnest can't tell which Steam game it is", "Achievement files are kept under the game's Steam id. Tools usually keep it in steam_appid.txt next to the game; this one has none Lootnest can find. With Download covers and artwork on, Lootnest also looks the game up by its name.", "")
		}

		// 2. The tool next to it.
		tool := core.DetectTool(dir)
		switch {
		case dir == "":
			add("info", "No game folder to look in", "Lootnest looks for the tool in the game's folder.", "")
		case tool == nil:
			add("warn", "No Steam-compatible tool next to the game", "A game from outside the stores only has achievements when a tool keeps them. DRM-free games usually have none. If it has one Lootnest doesn't know, its achievements can't show.", dir)
		case !tool.Reads:
			add("warn", tool.Name+" is next to the game", "Lootnest doesn't read "+tool.Name+"'s achievement files yet. Show recent activity and send it with a note, and it can be added.", dir)
		default:
			add("ok", tool.Name+" is next to the game", "Lootnest reads the achievement files it keeps.", dir)
		}

		// 3. Its achievement file.
		if appID > 0 {
			looked, there := a.env.AchievementFiles(appID)
			switch {
			case len(there) == 0:
				add("warn", "No achievement file yet", "Tools usually write it when the first achievement unlocks. Lootnest looked in "+strconv.Itoa(len(looked))+" places, such as "+firstOr(looked, "")+".", "")
			default:
				best := there[0]
				for _, f := range there {
					if f.Modified.After(best.Modified) {
						best = f
					}
				}
				switch {
				case best.Size == 0:
					add("fail", "Its achievement file is empty", "The tool made it but wrote nothing in it. Starting the game once more usually fills it.", best.Path)
				case best.Unlocked == 0:
					add("warn", "Its achievement file has none unlocked", "Written "+best.Modified.Format("2 Jan 2006, 15:04")+". If you did unlock some, the tool may keep them somewhere else: Show recent activity and send it.", best.Path)
				default:
					add("ok", fmt.Sprintf("%d unlocked, in the file the tool keeps", best.Unlocked), "Written "+best.Modified.Format("2 Jan 2006, 15:04")+".", best.Path)
				}
			}
		}

		// 4. Names and pictures.
		if appID > 0 {
			steamList := 0
			if a.env.SteamUser != nil {
				if u, err := a.env.SteamUser(); err == nil {
					l, _, _ := core.SteamCacheAchievements(u, appID)
					steamList = len(l)
				}
			}
			a.mu.Lock()
			hasKey := strings.TrimSpace(revealSecret(a.cfg.Settings.SteamAPIKey)) != ""
			a.mu.Unlock()
			_, schemaErr := os.Stat(schemaFile(appID))
			switch {
			case steamList > 0:
				add("ok", "Names and pictures from Steam's library cache", "This Steam account has seen the game, so its names are on this PC.", "")
			case len(core.LocalSchema(dir)) > 0:
				add("ok", "Names from the tool's list next to the game", "Pictures show for the ones Steam's servers have.", "")
			case schemaErr == nil:
				add("ok", "Names and pictures from Steam, with your key", "Kept on this PC for a week.", "")
			case hasKey:
				add("warn", "No names yet", "Your Steam Web API key is set; the names come the next time the game's page opens online.", "")
			default:
				add("warn", "Achievements show by their ids", "For their names, descriptions and pictures, add your free Steam Web API key in Settings, Online.", "")
			}
		}

		// 5. Each profile's own.
		if g != nil {
			if g.SharedAchievements {
				add("info", "Shared by every profile", "One set of achievements for everyone who plays it. The game's Manage drawer changes that.", "")
			} else {
				add("info", "Each profile has its own", "The achievement file goes with each profile's saves when Lootnest swaps them.", "")
			}
		}

		// 6. Toasts while playing.
		if ac := core.AntiCheat(dir); ac != "" {
			add("info", "No toasts over this game", "It ships with "+ac+", so Lootnest keeps its windows away from it.", "")
		}

		out["steps"] = steps
		if tool != nil && tool.Settings != "" {
			out["nameInGame"] = map[string]any{"name": core.ToolAccountName(tool.Settings), "settings": tool.Settings}
		}
	}()
}

func firstOr(list []string, d string) string {
	if len(list) > 0 {
		return list[0]
	}
	return d
}

func sourceName(s string) string {
	switch s {
	case "hydra":
		return "Hydra"
	case "":
		return "its launcher"
	}
	return "its launcher (" + s + ")"
}

// apiSetNameInGame sets the name a Goldberg-style tool shows in a game,
// and to friends on LAN, to a profile's name. Its settings files are kept
// in backups first.
func (a *App) apiSetNameInGame(key, dir, playerID string) error {
	if strings.EqualFold(filepath.Ext(dir), ".exe") {
		dir = filepath.Dir(dir)
	}
	tool := core.DetectTool(dir)
	if tool == nil || tool.Settings == "" {
		return errors.New("This game has no steam_settings folder to set a name in.")
	}
	a.mu.Lock()
	p := a.cfg.Player(playerID)
	name := ""
	if p != nil {
		name = p.Name
	}
	a.mu.Unlock()
	if name == "" {
		return errors.New("Choose a profile first.")
	}
	if err := a.env.CheckSafe(tool.Settings); err != nil {
		return err
	}
	keep := filepath.Join(a.store.BackupsDir(), "names-in-games", core.CoverKey(filepath.Base(dir))+"-"+time.Now().Format("20060102-150405"))
	if err := core.SetToolAccountName(tool.Settings, name, keep); err != nil {
		return fmt.Errorf("couldn't set the name: %w", err)
	}
	a.logf("The name in %s is now %s (in %s; the old settings are in %s)", filepath.Base(dir), name, tool.Settings, keep)
	return nil
}
