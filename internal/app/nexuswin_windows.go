package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	webview2 "github.com/jchv/go-webview2"
	"github.com/jchv/go-webview2/pkg/edge"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// nexusWinClass is the window class of Lootnest's window for Nexus Mods.
const nexusWinClass = "LootnestNexus"

// nexusNotes is added to every page of the window: window.__lnNote(text)
// shows a short note at the bottom of the page, and a page opened from a
// Mod Manager Download link (marked ln=manual) says to use Manual
// download, which Lootnest takes.
const nexusNotes = `(function () {
  if (window.top !== window || window.__lnNote) return;
  window.__lnNote = function (text) {
    var id = "ln-note", d = document.getElementById(id);
    if (!d) {
      d = document.createElement("div");
      d.id = id;
      d.setAttribute("role", "status");
      d.style.cssText = "position:fixed;left:50%;bottom:24px;transform:translateX(-50%);z-index:2147483647;max-width:560px;padding:14px 18px;border-radius:10px;background:#181a21;color:#dfe5eb;font:15px/1.4 system-ui,sans-serif;box-shadow:0 8px 30px rgba(0,0,0,.5)";
      (document.body || document.documentElement).appendChild(d);
    }
    d.textContent = text;
    clearTimeout(window.__lnNoteT);
    window.__lnNoteT = setTimeout(function () { d.remove(); }, 12000);
  };
  if (/[?&]ln=manual\b/.test(location.search)) {
    var show = function () { window.__lnNote("Press Manual download for this file. Lootnest takes the download from there."); };
    if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", show); else show();
  }
})();`

func jsString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// RunNexusWindow shows nexusmods.com at a mod's file (arg is
// core.NexusWindowArg) in a window of Lootnest's own, until it's closed,
// and returns the process's exit code. It's the site as it is, with the
// user's own sign-in kept under Lootnest's data; it keeps to Nexus's
// pages, its file servers and the sign-in pages, and opens any other link
// in the user's browser. A file downloaded from Nexus here goes into
// Lootnest's folder for it (core.NexusIncomingDir) with a note of the mod
// beside it, and the browser's download list stays hidden; Lootnest shows
// the progress and adds it (app/nexus.go). A WebView2 runtime too old to
// say when a download starts leaves it in the Downloads folder, where
// Lootnest finds it by its name. A Mod Manager Download link opens the
// same file's page, to use Manual download.
func RunNexusWindow(arg string) int {
	start := core.NexusWindowURL(arg)
	if start == "" {
		return 2
	}
	parts := strings.Split(arg, "/")
	modID, _ := strconv.Atoi(parts[1])
	fileID, _ := strconv.Atoi(parts[2])
	plat.EnableDPIAwareness()
	dpi := plat.SystemDPI()
	scale := func(n int) uint { return uint(n * dpi / 96) }
	var w webview2.WebView
	note := func(text string) {
		if w != nil {
			w.Eval("window.__lnNote&&window.__lnNote(" + jsString(text) + ")")
		}
	}
	// A Mod Manager Download link: the same file's page, for Manual download.
	manual := func(u string) {
		if d, m, f, ok := core.NexusNXM(u); ok && w != nil {
			to := core.NexusFileURL(d, m, f) + "&ln=manual"
			w.Dispatch(func() { w.Navigate(to) }) // after this navigation is turned down
			return
		}
		note("Use Manual download for this file. Lootnest takes the download from there.")
	}
	w = webview2.NewWithOptions(webview2.WebViewOptions{
		AutoFocus: true,
		DataPath:  filepath.Join(DataDir(), "apps", "nexus"),
		Navigate: func(u string) bool {
			switch core.NexusWindowGo(u) {
			case "show":
				return true
			case "nxm":
				manual(u)
			case "browser":
				_ = plat.ShellOpen(u)
			}
			return false
		},
		NewWindow: func(u string) {
			switch core.NexusWindowGo(u) {
			case "show":
				if w != nil {
					w.Dispatch(func() { w.Navigate(u) })
				}
			case "browser":
				_ = plat.ShellOpen(u)
			case "nxm":
				manual(u)
			}
		},
		Download: func(d edge.Download) (string, bool, bool) {
			if core.NexusWindowGo(d.URI) != "show" {
				return "", false, false // not from Nexus: as WebView2 would
			}
			dir := core.NexusIncomingDir(DataDir())
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return "", false, false
			}
			p := core.NexusIncomingPath(dir, d.Path)
			// The mod is the one the window was opened for unless the file's name says otherwise.
			mod, file := modID, fileID
			if m, ok := core.NexusFileName(filepath.Base(p)); ok && m != modID {
				mod, file = m, 0
			}
			if err := core.WriteNexusNote(p, core.NexusNote{ModID: mod, FileID: file, Total: d.Total, At: time.Now()}); err != nil {
				return "", false, false
			}
			note("Lootnest is taking this download. Its progress shows in Lootnest, and this window closes by itself when it's done.")
			return p, true, false
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
	w.Init(nexusNotes)
	plat.ForceForeground(uintptr(w.Window()))
	w.Navigate(start)
	w.Run()
	return 0
}

// nexusOwnWindow says whether Lootnest shows Nexus in a window of its own.
const nexusOwnWindow = true
