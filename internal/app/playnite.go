//go:build windows

package app

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"lootnest/internal/core"
)

// Import from Playnite (core/playnite.go): read-only and one way. Playnite's
// files are only read; what changes is Lootnest's own shelf, recent games,
// collections and playnite.json, copied to backups\import\<id> first.

func (a *App) apiPlayniteFind() map[string]any {
	return map[string]any{"dir": core.PlayniteLibraryDir(filepath.Join(os.Getenv("APPDATA"), "Playnite"))}
}

// playniteTargets are the Library's games as the import matches them: a
// game with profiles under its "game:" key, with the details of the copy
// the scan found for it.
func (a *App) playniteTargets() []core.FoundGame {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	found := a.libraryGames()
	skip := map[string]bool{}
	for _, k := range cfg.IgnoredFound {
		skip[k] = true
	}
	var out []core.FoundGame
	for i := range cfg.Games {
		g := &cfg.Games[i]
		t := core.FoundGame{Key: "game:" + g.ID, Name: g.Name, Source: "folder"}
		if g.Launch.Type == "steam" {
			t.Source, t.AppID = "steam", g.Launch.AppID
		} else if exe, err := a.env.ResolveExe(g.Launch.Exe); g.Launch.Exe != "" && err == nil {
			t.Dir = filepath.Dir(exe)
		}
		linked := false
		for _, f := range found {
			if !a.isFoundGame(g, f) {
				continue
			}
			if !linked {
				t.Source, t.AppID, t.SteamID, t.StoreID, t.Dir = f.Source, f.AppID, f.SteamID, f.StoreID, f.Dir
				linked = true
			}
			skip[f.Key] = true
		}
		out = append(out, t)
	}
	for _, f := range found {
		if !skip[f.Key] {
			out = append(out, f)
		}
	}
	return out
}

func (a *App) playnitePlan(dir, playerID string) (string, core.PlaynitePlan, error) {
	lib, games, err := core.ReadPlaynite(dir)
	if err != nil {
		return "", core.PlaynitePlan{}, err
	}
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	recent := a.store.LoadRecent()
	played := map[string]time.Time{}
	for k, p := range recent {
		if t, ok := p.Players[playerID]; ok {
			played[k] = t
		}
	}
	// The Library shows a grouped game's status, rating and note from any
	// of its copies, so a copy counts as having one when any copy does.
	shelf := a.store.LoadShelf()[playerID]
	if shelf == nil {
		shelf = map[string]core.ShelfEntry{}
	}
	for _, grp := range core.GroupCopies(a.visibleFound(a.libraryGames()), cfg.Separate, cfg.Primary, recent) {
		var has core.ShelfEntry
		for _, k := range grp.Keys {
			e := shelf[k]
			has.Status, has.Rating, has.Note = cmp.Or(has.Status, e.Status), max(has.Rating, e.Rating), cmp.Or(has.Note, e.Note)
		}
		for _, k := range grp.Keys {
			e := shelf[k]
			e.Status, e.Rating, e.Note = has.Status, has.Rating, has.Note
			shelf[k] = e
		}
	}
	have := core.PlayniteHave{Shelf: shelf, Imported: a.store.PlayniteImported(playerID), Played: played, Tags: cfg.Tags}
	return lib, core.PlayniteImport(games, a.playniteTargets(), have), nil
}

func (a *App) playniteProfile(playerID string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if p := a.cfg.Player(playerID); p != nil {
		return p.Name, nil
	}
	return "", errors.New("That profile doesn't exist.")
}

// apiPlaynitePreview answers with a "playnitePlan" event: every match and
// what it would bring in, whatever the user chooses to bring in.
func (a *App) apiPlaynitePreview(dir, playerID string) error {
	if _, err := a.playniteProfile(playerID); err != nil {
		return err
	}
	go func() {
		out := map[string]any{"player": playerID, "req": dir}
		lib, plan, err := a.playnitePlan(dir, playerID)
		if err != nil {
			out["error"] = err.Error()
		} else {
			out["dir"], out["plan"] = lib, plan
		}
		a.emit("playnitePlan", out)
	}()
	return nil
}

// apiPlayniteApply makes the import for one profile and answers with a
// "playniteDone" event carrying the id Undo takes.
func (a *App) apiPlayniteApply(dir, playerID string, o core.PlayniteOptions) error {
	name, err := a.playniteProfile(playerID)
	if err != nil {
		return err
	}
	go func() {
		out := map[string]any{}
		defer func() { a.emit("playniteDone", out) }()
		lib, plan, err := a.playnitePlan(dir, playerID)
		if err != nil {
			out["error"] = err.Error()
			return
		}
		changes := plan.Only(o)
		if len(changes) == 0 {
			out["error"] = "With what you chose, there's nothing to bring in."
			return
		}
		id, err := a.store.BackupBeforeImport(time.Now())
		if err != nil {
			out["error"] = "Nothing was imported: Lootnest couldn't copy its own files first (" + err.Error() + ")."
			return
		}
		out["id"] = id
		if err := a.store.ApplyPlaynite(playerID, changes, time.Now()); err != nil {
			out["error"] = "The import stopped part way (" + err.Error() + "). Undo puts back what was there before."
			return
		}
		skipped := 0
		if o.Favourites || o.Categories || o.Tags {
			a.mu.Lock()
			cfg := a.cfg.Clone()
			a.mu.Unlock()
			skipped = cfg.ApplyPlaynite(changes)
			if err := a.saveConfig(cfg); err != nil {
				out["error"] = "The collections weren't saved (" + err.Error() + "). Undo puts back what was there before."
				return
			}
		}
		var secs int64
		for _, c := range changes {
			secs += c.Add
		}
		out["games"], out["seconds"], out["skipped"] = len(changes), secs, skipped
		a.logf("Imported from Playnite (%s) for %s: %d games, %d hours added, %d collections left out; the copy from before is in backups\\import\\%s",
			lib, name, len(changes), secs/3600, skipped, id)
	}()
	return nil
}

// apiPlayniteUndo puts back Lootnest's files from before an import.
func (a *App) apiPlayniteUndo(id string) error {
	if err := a.store.UndoImport(id); err != nil {
		return err
	}
	cfg, _, err := a.store.LoadConfig()
	if err != nil {
		return fmt.Errorf("couldn't read the settings back: %w", err)
	}
	a.mu.Lock()
	a.cfg = cfg
	a.mu.Unlock()
	a.logf("Undid the import from Playnite (backups\\import\\%s); what was there before Undo is in its undone folder", id)
	return nil
}
