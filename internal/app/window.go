package app

import "unsafe"

// view is what the app needs of its window and the page in it. On Windows
// it's a WebView2 (window_windows.go); docs/ROADMAP.md, section 9, gives
// Linux a WebKitGTK one.
type view interface {
	// Bind makes f callable from the page as window[name].
	Bind(name string, f interface{}) error
	// Dispatch runs f on the window's thread.
	Dispatch(f func())
	// Eval runs a script in the page.
	Eval(js string)
	// RefreshView sizes the page to the window and shows it.
	RefreshView()
	// Run runs the window's loop until it closes.
	Run()
	// SetHtml loads the page.
	SetHtml(html string)
	// Window is the system's handle of the window.
	Window() unsafe.Pointer
}
