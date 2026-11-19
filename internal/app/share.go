package app

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// apiShareImage takes a picture the window drew to share (achievements, for
// a whole profile or one game): "save" keeps it in Pictures\Lootnest\
// Achievements and shows it in Explorer, "copy" puts it on the clipboard to
// paste into Discord or a chat. Nothing is sent anywhere.
func (a *App) apiShareImage(how, title, dataURL string) (string, error) {
	b, err := core.DecodePNGDataURL(dataURL)
	if err != nil {
		return "", err
	}
	switch how {
	case "copy":
		if err := plat.SetClipboardImage(a.hwnd, b); err != nil {
			return "", err
		}
		return "", nil
	case "save":
		dir := filepath.Join(picturesDir(), "Lootnest", "Achievements")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		p := filepath.Join(dir, core.ShareFileName(title, time.Now().Format("2006-01-02")))
		if err := os.WriteFile(p+".tmp", b, 0o644); err != nil {
			return "", err
		}
		if err := os.Rename(p+".tmp", p); err != nil {
			_ = os.Remove(p + ".tmp")
			return "", err
		}
		_ = plat.RevealInExplorer(p)
		return p, nil
	}
	return "", errors.New("unknown way to share")
}
