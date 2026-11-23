package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"lootnest/internal/core"
)

// Runs are per game (Player.Game): a run shows and plays only in its game,
// and competitive games get none (core.CompetitiveGame).

var noRunsCache = struct {
	sync.Mutex
	m map[string]noRunsAt
}{m: map[string]noRunsAt{}}

type noRunsAt struct {
	why  string
	exe  string
	when time.Time
}

const noRunsFor = 10 * time.Minute

// runsBlocked says why a game with profiles gets no runs ("" when it may).
// A game's folder is looked at once every ten minutes at most.
func (a *App) runsBlocked(g *core.Game) string {
	noRunsCache.Lock()
	c, ok := noRunsCache.m[g.ID]
	noRunsCache.Unlock()
	if ok && c.exe == g.Launch.Exe && time.Since(c.when) < noRunsFor {
		return c.why
	}
	dir := ""
	if g.Launch.Type != "steam" {
		if exe, err := a.env.ResolveExe(g.Launch.Exe); err == nil && exe != "" {
			dir = filepath.Dir(exe)
		}
	} else if g.Launch.AppID > 0 {
		for _, f := range a.libraryGames() {
			if f.Dir != "" && (f.AppID == g.Launch.AppID || f.SteamID == g.Launch.AppID) {
				dir = f.Dir
				break
			}
		}
	}
	why := core.CompetitiveGame(g.Launch.AppID, g.Name, dir)
	noRunsCache.Lock()
	noRunsCache.m[g.ID] = noRunsAt{why: why, exe: g.Launch.Exe, when: time.Now()}
	noRunsCache.Unlock()
	return why
}

// noRuns is, for the page, each game with profiles that gets no runs.
func (a *App) noRuns(cfg *core.Config) map[string]string {
	out := map[string]string{}
	for i := range cfg.Games {
		if why := a.runsBlocked(&cfg.Games[i]); why != "" {
			out[cfg.Games[i].ID] = why
		}
	}
	return out
}

// hasFiles says whether a profile has files of a game: a profile kept, or
// its files in place now.
func (a *App) hasFiles(active map[string]string) func(player, game string) bool {
	return func(player, game string) bool {
		return active[game] == player || a.eng.HasProfile(player, game)
	}
}

// errRunHere refuses a run in a game it isn't for, or in a game that gets
// no runs.
func (a *App) errRunHere(cfg *core.Config, g *core.Game, playerID string) error {
	p := cfg.Player(playerID)
	if p == nil || p.Of == "" {
		return nil
	}
	if !core.RunFor(p, g.ID, a.hasFiles(a.store.LoadState())) {
		return fmt.Errorf("%s is a run of another game. Pick a profile or a run of %s.", p.Name, g.Name)
	}
	if why := a.runsBlocked(g); why != "" && !a.eng.HasProfile(p.ID, g.ID) {
		return fmt.Errorf("%s has no runs, since %s.", g.Name, runsWhy(why))
	}
	return nil
}

// errNewRun refuses a new run in a game that gets none.
func (a *App) errNewRun(g *core.Game) error {
	if why := a.runsBlocked(g); why != "" {
		return errors.New(g.Name + " has no runs, since " + runsWhy(why) + ". Runs are for story games.")
	}
	return nil
}

// runsWhy turns core.CompetitiveGame's reason into words after "since".
func runsWhy(why string) string {
	if why == "a competitive game" {
		return "it's a competitive game"
	}
	return why
}
