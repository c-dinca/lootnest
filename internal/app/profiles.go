//go:build windows

package app

import (
	"errors"
	"os"
	"path/filepath"

	"lootnest/internal/core"
)

// ---------------------------------------------------------------------------
// Player profiles: a picture for each player, kept as a small square PNG in
// <data>\avatars. The rest of a profile (line, banner, favourites) lives in
// config.json with the player.
// ---------------------------------------------------------------------------

func avatarPath(playerID string) string {
	return filepath.Join(DataDir(), "avatars", filepath.Base(playerID)+".png")
}

// apiSetAvatar makes a profile picture from a picture file the player chose.
// The work happens in the background; an "avatar" event says when it's done.
func (a *App) apiSetAvatar(playerID, path string) error {
	a.mu.Lock()
	known := a.cfg.Player(playerID) != nil
	a.mu.Unlock()
	if !known {
		return errors.New("That player doesn't exist.")
	}
	go func() {
		out := map[string]any{"id": playerID}
		defer func() { a.emit("avatar", out) }()
		b, err := core.MakeAvatar(path)
		if err == nil {
			err = os.MkdirAll(filepath.Dir(avatarPath(playerID)), 0o755)
		}
		if err == nil {
			err = os.WriteFile(avatarPath(playerID)+".tmp", b, 0o644)
		}
		if err == nil {
			err = os.Rename(avatarPath(playerID)+".tmp", avatarPath(playerID))
		}
		if err == nil {
			err = a.setPlayerAvatar(playerID, true)
		}
		if err != nil {
			out["error"] = err.Error()
		}
	}()
	return nil
}

func (a *App) apiRemoveAvatar(playerID string) error {
	_ = os.Remove(avatarPath(playerID))
	return a.setPlayerAvatar(playerID, false)
}

func (a *App) setPlayerAvatar(playerID string, on bool) error {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	p := cfg.Player(playerID)
	if p == nil {
		return errors.New("That player doesn't exist.")
	}
	p.Avatar = on
	return a.saveConfig(cfg)
}
