//go:build windows

package app

import (
	"errors"

	"lootnest/internal/core"
)

// apiCustomArt returns a game's custom art of a kind as a data: URL, or "".
func (a *App) apiCustomArt(key, kind string) string {
	if p := a.store.CustomArtPath(key, kind); p != "" {
		return dataURLMax(p, core.CustomArtMax+1)
	}
	return ""
}

// apiSetCustomArt copies a picture the user chose as a game's cover, wide
// art or logo. Their file stays where it is, as it is.
func (a *App) apiSetCustomArt(key, kind, path string) error {
	w, h, err := a.store.SetCustomArt(key, kind, path)
	if err != nil {
		return errors.New(sentence(err))
	}
	a.logf("Set a custom %s for %s (%dx%d)", kind, key, w, h)
	return nil
}

func (a *App) apiClearCustomArt(key, kind string) error {
	if err := a.store.ClearCustomArt(key, kind); err != nil {
		return errors.New(sentence(err))
	}
	a.logf("Took the custom %s off %s", kind, key)
	return nil
}
