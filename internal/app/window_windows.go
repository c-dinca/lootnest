package app

import (
	"errors"
	"path/filepath"

	webview2 "github.com/jchv/go-webview2"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// openWindow makes Lootnest's window, a WebView2 over a Win32 window, and
// keeps it in a.w and a.hwnd.
func (a *App) openWindow(opt Options, dpi int, scale func(int) int) error {
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		AutoFocus: true,
		DataPath:  filepath.Join(DataDir(), "WebView2"),
		// The window only ever shows Lootnest's own page: nothing else can be
		// loaded or navigated to, so the page's access to Lootnest can't leak.
		AllowRequest: core.AllowedPageRequest,
		AllowMessage: a.fromPage,
		WindowOptions: webview2.WindowOptions{
			Title:       Name,
			Width:       uint(scale(1100)),
			Height:      uint(scale(740)),
			IconId:      1,
			Center:      true,
			ClassName:   WindowClass,
			Hidden:      opt.StartHidden && opt.LaunchGame == "",
			Background:  groundColor,
			OnCreate:    func(hwnd uintptr) { plat.StyleTitleBar(hwnd, bgColor, borderColor); plat.SetWindowIcons(hwnd, dpi) },
			MessageHook: a.hook,
		},
	})
	if w == nil {
		r := plat.MessageBox(0, "Lootnest needs the Microsoft Edge WebView2 Runtime, which isn't installed on this PC.\n\nOpen the download page now?",
			Name, plat.MB_YESNO|plat.MB_ICONWARNING)
		if r == plat.IDYES {
			_ = plat.ShellOpen("https://developer.microsoft.com/microsoft-edge/webview2/")
		}
		return errors.New("WebView2 runtime missing")
	}
	a.w = w
	a.hwnd = uintptr(w.Window())
	w.SetSize(scale(860), scale(580), webview2.HintMin)
	a.taskbarCreated = plat.RegisterWindowMessage("TaskbarCreated")
	return nil
}

// present brings the window up, in front.
func (a *App) present() { plat.Present(a.hwnd) }

// closeWindow closes the window, which ends Run (hook lets it through once
// a.quitting is set).
func (a *App) closeWindow() { plat.PostMessage(a.hwnd, plat.WM_CLOSE, 0, 0) }

// platWindow: on Windows, a.hwnd is the window.
type platWindow struct{}

// trayReady: on Windows the tray's menu is made when it's clicked (hook).
func (a *App) trayReady() {}
