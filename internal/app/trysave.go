//go:build windows

package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lootnest/internal/core"
)

// ---------------------------------------------------------------------------
// Try a save safely (core/trysave.go): a save someone brings (a folder, a
// file, or a .zip, .7z or .rar) goes into a new run of the person, never
// over anyone's files. The run starts as a copy of their files with that
// save in place; playing them again is one click back.
// ---------------------------------------------------------------------------

// apiTrySave starts a run of from's person for a game, with src in the
// save location locID, and returns the run's id.
func (a *App) apiTrySave(gameID, from, locID, src, name string) (string, error) {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	busy := a.sessions[gameID] != nil
	a.mu.Unlock()
	g := cfg.Game(gameID)
	if g == nil {
		return "", errors.New("That game no longer has profiles.")
	}
	if busy {
		return "", fmt.Errorf("%s is busy right now; try again when it's closed.", g.Name)
	}
	base := cfg.Player(from)
	if base == nil {
		return "", errors.New("That profile no longer exists.")
	}
	person := base
	if person.Of != "" {
		if up := cfg.Player(person.Of); up != nil {
			person = up
		}
	}
	src = strings.TrimSpace(src)
	if src == "" {
		return "", errors.New("Choose the save to try first.")
	}
	if _, err := os.Stat(src); err != nil {
		return "", errors.New("The save to try isn't there any more.")
	}
	switch strings.ToLower(filepath.Ext(src)) {
	case ".zip", ".7z", ".rar":
		dir := filepath.Join(DataDir(), "try-"+time.Now().Format("150405.000"))
		defer os.RemoveAll(dir)
		staged, err := core.StageUpdateMax(src, dir, 2<<30)
		if errors.Is(err, core.ErrTooBig) {
			return "", errors.New("That's too big to be a save.")
		}
		if err != nil {
			return "", fmt.Errorf("Couldn't open %s: %v", filepath.Base(src), err)
		}
		src = staged
	}
	name = trimRunes(name, 24)
	if name == "" {
		name = freeName(trimRunes(person.Name, 24-len(" trying"))+" trying", cfg.Players)
	}
	for _, p := range cfg.Players {
		if strings.EqualFold(p.Name, name) {
			return "", fmt.Errorf("There's already a profile called %s.", name)
		}
	}
	run := core.Player{ID: core.NewID("p"), Name: name, Color: person.Color, Of: person.ID}
	players := append(append([]core.Player{}, cfg.Players...), run)
	if err := core.ValidatePlayers(players); err != nil {
		return "", err
	}
	if err := a.eng.ProfileFromImport(g, run.ID, base.ID, locID, src); err != nil {
		return "", err
	}
	if err := a.apiSavePlayers(players); err != nil {
		_ = os.RemoveAll(a.store.ProfileDir(run.ID, g.ID))
		return "", err
	}
	a.logf("New run %s of %s in %s, to try a save (%s's own files are unchanged)", run.Name, person.Name, g.Name, base.Name)
	return run.ID, nil
}
