package app

import "lootnest/internal/plat"

// Lootnest's title bar is drawn in the page (#winbar, 15-winbar.js), as in
// Steam: the window has no caption of its own (plat.FrameSize), and the
// page's bar asks for what a caption does.

// winState is what the page's bar needs to know about the window.
func (a *App) winState() map[string]any {
	return map[string]any{
		"frame": plat.HasCaption(a.hwnd), // false when full screen (Big Picture, TV mode)
		"max":   plat.IsZoomed(a.hwnd),
	}
}

// apiWin: "drag" moves the window, "top", "topleft", "topright" resize it
// (the sides and the bottom are the system's own frame), "min", "max",
// "close" are the caption's keys; anything else only answers the state.
// Bound calls run on the window's thread, which the system's move loop
// needs.
func (a *App) apiWin(op string) map[string]any {
	switch op {
	case "drag":
		plat.DragWindow(a.hwnd, "caption")
	case "top", "topleft", "topright":
		plat.DragWindow(a.hwnd, op)
	case "min", "max", "close":
		plat.WindowKey(a.hwnd, op)
	}
	return a.winState()
}

// noteWinSize tells the page when the window was maximized, restored or
// went full screen, so its bar shows the right key.
func (a *App) noteWinSize() {
	st := a.winState()
	key := [2]bool{st["frame"].(bool), st["max"].(bool)}
	if a.winShape == key && a.winShapeSent {
		return
	}
	a.winShape, a.winShapeSent = key, true
	a.emit("winState", st)
}
