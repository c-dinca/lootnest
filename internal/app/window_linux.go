package app

import (
	"os"
	"reflect"
	"time"

	"lootnest/internal/desk"
	"lootnest/internal/plat"
	"lootnest/internal/webkit"
)

// obsidian is the window's ground before the page draws (DESIGN.md).
const obsidian = 0x111217

// openWindow makes Lootnest's window on Linux, a WebKitGTK view in a GTK
// window (internal/webkit), and starts taking commands from a second
// Lootnest (instance_linux.go).
func (a *App) openWindow(opt Options, dpi int, scale func(int) int) error {
	w, err := webkit.New(webkit.Options{
		Title:        Name,
		Width:        1100,
		Height:       740,
		MinWidth:     860,
		MinHeight:    580,
		Hidden:       opt.StartHidden && opt.LaunchGame == "",
		DataDir:      webkitDir(),
		Background:   obsidian,
		Debug:        os.Getenv("LOOTNEST_DEBUG") != "",
		AllowMessage: a.fromPage,
		OnClose:      a.onClose,
	})
	if err != nil {
		return err
	}
	a.w = w
	a.lw = w
	desk.SetParent(w.Window(), plat.IconPath())
	serveInstance(a)
	return nil
}

// onClose is the window's close button, as WM_CLOSE is on Windows (hook):
// with Close to tray on and a tray the desktop shows, the window hides and
// Lootnest keeps watching games; else Lootnest quits (asking first when a
// game is running).
func (a *App) onClose() bool {
	if a.quitting {
		return false
	}
	a.mu.Lock()
	toTray := a.cfg.Settings.CloseToTray
	a.mu.Unlock()
	if toTray && a.tray.Shown() {
		a.cancelRequests()
		a.lw.Hide()
		if !a.trayHintShown {
			a.trayHintShown = true
			a.tray.Notify("Lootnest is still running", "It keeps watching your games from the tray. Open its menu there to quit.")
		}
		return true
	}
	a.quit(false)
	return true
}

// trayReady keeps the tray's menu current: an AppIndicator menu is made
// before it's opened, so it's made again whenever its lines change.
func (a *App) trayReady() {
	if !a.tray.Shown() {
		return
	}
	var last []plat.MenuItem
	set := func() {
		items, ids := a.trayItems()
		if reflect.DeepEqual(items, last) {
			return
		}
		last = items
		a.tray.SetMenu(items, func(id int) { a.trayChoose(id, ids) })
	}
	set()
	go func() {
		for range time.Tick(2 * time.Second) {
			a.ui(set)
		}
	}()
}

// present brings the window up, in front.
func (a *App) present() {
	if a.lw != nil {
		a.lw.Present()
	}
}

// closeWindow ends Run.
func (a *App) closeWindow() {
	if a.lw != nil {
		a.lw.Terminate()
	}
}

// platWindow is the Linux window, for what the view interface doesn't
// cover (present, close).
type platWindow struct{ lw *webkit.WebView }
