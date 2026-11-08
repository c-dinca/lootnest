//go:build windows

package app

import (
	"os"
	"path/filepath"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// apiSaveRecap saves a profile's year in games (a PNG the window drew) in
// Pictures\Lootnest and shows it in Explorer. It returns the file's path.
func (a *App) apiSaveRecap(profile string, year int, dataURL string) (string, error) {
	b, err := core.DecodePNGDataURL(dataURL)
	if err != nil {
		return "", err
	}
	if year < 2000 || year > time.Now().Year()+1 {
		year = time.Now().Year()
	}
	dir := filepath.Join(picturesDir(), "Lootnest")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	p := filepath.Join(dir, core.RecapFileName(profile, year))
	if err := os.WriteFile(p+".tmp", b, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(p+".tmp", p); err != nil {
		_ = os.Remove(p + ".tmp")
		return "", err
	}
	a.logf("Saved %s's %d in games to %s", profile, year, p)
	_ = win.RevealInExplorer(p)
	return p, nil
}
