package app

import (
	"path/filepath"

	webview2 "github.com/jchv/go-webview2"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// nexusWinClass is the window class of Lootnest's window for Nexus Mods.
const nexusWinClass = "LootnestNexus"

// nxmNote is shown in the page when the user picks Mod Manager Download,
// which Lootnest can't take until it's registered with Nexus.
const nxmNote = `(function () {
  var id = "ln-nxm-note";
  if (document.getElementById(id)) return;
  var d = document.createElement("div");
  d.id = id;
  d.setAttribute("role", "status");
  d.textContent = "Lootnest can't take Mod Manager Download yet. Use Manual download: the file goes to your Downloads folder and Lootnest adds it from there.";
  d.style.cssText = "position:fixed;left:50%;bottom:24px;transform:translateX(-50%);z-index:2147483647;max-width:560px;padding:14px 18px;border-radius:10px;background:#181a21;color:#dfe5eb;font:15px/1.4 system-ui,sans-serif;box-shadow:0 8px 30px rgba(0,0,0,.5)";
  document.documentElement.appendChild(d);
  setTimeout(function () { d.remove(); }, 9000);
})();`

// RunNexusWindow shows nexusmods.com at a mod's file (arg is
// core.NexusWindowArg) in a window of Lootnest's own, until it's closed,
// and returns the process's exit code. It's the site as it is, with the
// user's own sign-in kept under Lootnest's data; it keeps to Nexus's
// pages, its file servers and the sign-in pages, and opens any other link
// in the user's browser. A file downloaded here goes to the Downloads
// folder, where Lootnest finds it (app/nexus.go).
func RunNexusWindow(arg string) int {
	start := core.NexusWindowURL(arg)
	if start == "" {
		return 2
	}
	plat.EnableDPIAwareness()
	dpi := plat.SystemDPI()
	scale := func(n int) uint { return uint(n * dpi / 96) }
	var w webview2.WebView
	w = webview2.NewWithOptions(webview2.WebViewOptions{
		AutoFocus: true,
		DataPath:  filepath.Join(DataDir(), "apps", "nexus"),
		Navigate: func(u string) bool {
			switch core.NexusWindowGo(u) {
			case "show":
				return true
			case "nxm":
				if w != nil {
					w.Eval(nxmNote)
				}
			case "browser":
				_ = plat.ShellOpen(u)
			}
			return false
		},
		NewWindow: func(u string) {
			switch core.NexusWindowGo(u) {
			case "show":
				if w != nil {
					w.Navigate(u)
				}
			case "browser":
				_ = plat.ShellOpen(u)
			case "nxm":
				if w != nil {
					w.Eval(nxmNote)
				}
			}
		},
		// No page here may call Lootnest.
		AllowMessage: func(string) bool { return false },
		WindowOptions: webview2.WindowOptions{
			Title:      "Nexus Mods",
			IconId:     1,
			ClassName:  nexusWinClass,
			Width:      scale(1180),
			Height:     scale(860),
			Center:     true,
			Background: 0x111217,
			OnCreate:   func(hwnd uintptr) { plat.SetWindowIcons(hwnd, dpi) },
		},
	})
	if w == nil {
		return 1
	}
	plat.ForceForeground(uintptr(w.Window()))
	w.Navigate(start)
	w.Run()
	return 0
}

// nexusOwnWindow says whether Lootnest shows Nexus in a window of its own.
const nexusOwnWindow = true
