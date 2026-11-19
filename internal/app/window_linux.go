package app

import (
	"os"

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
	serveInstance(a)
	return nil
}

// onClose is the window's close button: Lootnest quits (asking first when
// a game is running). With a tray (docs/ROADMAP.md, section 9, L2b) it will
// go to the tray as on Windows.
func (a *App) onClose() bool {
	if a.quitting {
		return false
	}
	a.quit(false)
	return true
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
