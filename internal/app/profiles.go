//go:build windows

package app

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"lootnest/internal/core"
)

// ---------------------------------------------------------------------------
// Player profiles: a picture for each player, kept as a small square PNG in
// <data>\avatars, and an optional banner picture in <data>\banners. The
// rest of a profile lives in config.json with the player; their shelf and
// daily activity in shelf.json and activity.json (core/played.go).
// ---------------------------------------------------------------------------

func avatarPath(playerID string) string {
	return filepath.Join(DataDir(), "avatars", filepath.Base(playerID)+".png")
}

func bannerPath(playerID string) string {
	return filepath.Join(DataDir(), "banners", filepath.Base(playerID)+".jpg")
}

// apiSetAvatar makes a profile picture from a picture file the player chose.
// The work happens in the background; an "avatar" event says when it's done.
func (a *App) apiSetAvatar(playerID, path string) error {
	return a.makePicture(playerID, path, avatarPath(playerID), core.MakeAvatar, "avatar", func(p *core.Player) { p.Avatar = true })
}

// apiSetBanner makes a profile banner from a picture file, like apiSetAvatar;
// a "banner" event says when it's done.
func (a *App) apiSetBanner(playerID, path string) error {
	return a.makePicture(playerID, path, bannerPath(playerID), core.MakeBanner, "banner", func(p *core.Player) { p.Banner = "image" })
}

func (a *App) apiRemoveBanner(playerID string) error {
	_ = os.Remove(bannerPath(playerID))
	return a.updatePlayer(playerID, func(p *core.Player) {
		if p.Banner == "image" {
			p.Banner = ""
		}
	})
}

func (a *App) makePicture(playerID, src, dst string, build func(string) ([]byte, error), event string, set func(*core.Player)) error {
	a.mu.Lock()
	known := a.cfg.Player(playerID) != nil
	a.mu.Unlock()
	if !known {
		return errors.New("That profile doesn't exist.")
	}
	go func() {
		out := map[string]any{"id": playerID}
		defer func() { a.emit(event, out) }()
		b, err := build(src)
		if err == nil {
			err = os.MkdirAll(filepath.Dir(dst), 0o755)
		}
		if err == nil {
			err = os.WriteFile(dst+".tmp", b, 0o644)
		}
		if err == nil {
			err = os.Rename(dst+".tmp", dst)
		}
		if err == nil {
			err = a.updatePlayer(playerID, set)
		}
		if err != nil {
			out["error"] = err.Error()
		}
	}()
	return nil
}

// apiSetShelf changes a player's take on a game (status, rating, note, a
// correction of their time played); fields left out keep their value.
func (a *App) apiSetShelf(playerID, key string, p core.ShelfPatch) error {
	a.mu.Lock()
	known := a.cfg.Player(playerID) != nil
	a.mu.Unlock()
	if !known {
		return errors.New("That profile doesn't exist.")
	}
	return a.store.PatchShelf(playerID, key, p, time.Now())
}

func (a *App) apiRemoveAvatar(playerID string) error {
	_ = os.Remove(avatarPath(playerID))
	return a.updatePlayer(playerID, func(p *core.Player) { p.Avatar = false })
}

func (a *App) updatePlayer(playerID string, set func(*core.Player)) error {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	p := cfg.Player(playerID)
	if p == nil {
		return errors.New("That profile doesn't exist.")
	}
	set(p)
	return a.saveConfig(cfg)
}

// apiSetTags puts a Library game in the user's collections ("Co-op", "For
// the kids"); an empty list takes it out of all of them.
func (a *App) apiSetTags(key string, tags []string) error {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	if err := cfg.SetTags(key, tags); err != nil {
		return err
	}
	return a.saveConfig(cfg)
}
