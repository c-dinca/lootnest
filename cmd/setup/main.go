//go:build windows

// Lootnest-Setup.exe: installs Lootnest for the current Windows user.
package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	webview2 "github.com/jchv/go-webview2"

	"lootnest/internal/app"
	"lootnest/internal/win"
)

//go:embed payload/Lootnest.exe
var payload []byte

//go:embed payload/THIRD_PARTY_NOTICES.txt
var notices []byte

//go:embed setup.html
var page string

// Version is set at build time.
var Version = "1.0.0"

const (
	bg     = 0x141529 // --bg-ground, the page's ground
	border = 0x2A3142 // the app's window edge
)

func init() { runtime.LockOSThread() }

func installDir() string { return app.InstallDir() }

type installOpts struct {
	Desktop   bool `json:"desktop"`
	Autostart bool `json:"autostart"`
}

// install puts Lootnest in place in three steps, calling done(1), done(2)
// and done(3) as each finishes, so the page can tick them off.
func install(o installOpts, done func(step int)) error {
	if done == nil {
		done = func(int) {}
	}
	if !app.StopRunning() {
		return errors.New("Lootnest is running and didn't close. Quit it from its tray icon, then try again.")
	}
	dir := installDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("couldn't create %s: %w", dir, err)
	}
	exe := filepath.Join(dir, app.Name+".exe")
	tmp := exe + ".new"
	if err := os.WriteFile(tmp, payload, 0o755); err != nil {
		return fmt.Errorf("couldn't write Lootnest.exe: %w", err)
	}
	var err error
	for i := 0; i < 20; i++ { // the old exe can stay locked for a moment after it quits
		if err = os.Rename(tmp, exe); err == nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("couldn't replace Lootnest.exe (is it still open?): %w", err)
	}
	_ = os.WriteFile(filepath.Join(dir, "THIRD_PARTY_NOTICES.txt"), notices, 0o644)
	done(1)

	sc := win.Shortcut{Target: exe, WorkDir: dir, Icon: exe, Description: "Separate game settings and saves for everyone on this PC"}
	sc.Path = app.StartMenuShortcut()
	if err := win.CreateShortcut(sc); err != nil {
		return fmt.Errorf("couldn't add Lootnest to the Start menu: %w", err)
	}
	if o.Desktop {
		sc.Path = app.DesktopShortcut()
		if err := win.CreateShortcut(sc); err != nil {
			return fmt.Errorf("couldn't create the desktop shortcut: %w", err)
		}
	} else {
		_ = os.Remove(app.DesktopShortcut())
	}
	if err := win.SetAutostart(app.Name, fmt.Sprintf(`"%s" --tray`, exe), o.Autostart); err != nil {
		return fmt.Errorf("couldn't set Start with Windows: %w", err)
	}
	done(2)
	// Replacing PlaySwap (the old name): Steam launch options still point at
	// PlaySwap.exe until Lootnest rewrites them, so a copy stays there.
	if err := app.ReplaceLegacy(payload); err != nil {
		return fmt.Errorf("couldn't update PlaySwap's old program folder: %w", err)
	}
	if err := win.RegisterUninstall(app.UninstallID, win.UninstallEntry{
		DisplayName:          app.Name,
		DisplayVersion:       Version,
		Publisher:            app.Name,
		DisplayIcon:          exe + ",0",
		InstallLocation:      dir,
		UninstallString:      fmt.Sprintf(`"%s" --uninstall`, exe),
		QuietUninstallString: fmt.Sprintf(`"%s" --uninstall --quiet`, exe),
		EstimatedSizeKB:      uint32(len(payload) / 1024),
	}); err != nil {
		return err
	}
	done(3)
	return nil
}

// currentChoices are the installed Lootnest's desktop shortcut and Start
// with Windows as they are now; on a first install, neither.
func currentChoices() installOpts {
	_, autostart := win.Autostart(app.Name)
	_, err := os.Stat(app.DesktopShortcut())
	return installOpts{Desktop: err == nil, Autostart: autostart}
}

// silentUpdate is "Lootnest-Setup.exe --update", started by Lootnest's own
// "Update now": install over the current version with the same choices,
// with no window, then reopen it.
func silentUpdate() int {
	if err := install(currentChoices(), nil); err != nil {
		win.MessageBox(0, "Lootnest couldn't update:\n"+err.Error()+"\n\nYou can download the installer from the release page instead.", "Lootnest update", win.MB_ICONERROR)
		return 1
	}
	exe := filepath.Join(installDir(), app.Name+".exe")
	if err := win.StartDetached(exe, "", filepath.Dir(exe)); err != nil {
		return 1
	}
	return 0
}

// silentInstall is "Lootnest-Setup.exe --silent" (or /S, /silent), for
// winget and other package managers: install with the current choices, with
// no window and no message box, and don't start Lootnest. The exit code is
// 0 on success and 1 on failure. The reason goes to stderr and to
// Lootnest-Setup.log in %TEMP%, since a windowsgui program's stderr is
// usually nowhere.
func silentInstall() int {
	if err := install(currentChoices(), nil); err != nil {
		msg := "Lootnest couldn't install: " + err.Error()
		fmt.Fprintln(os.Stderr, msg)
		_ = os.WriteFile(filepath.Join(os.TempDir(), "Lootnest-Setup.log"), []byte(time.Now().Format("2006-01-02 15:04:05")+" "+msg+"\r\n"), 0o644)
		return 1
	}
	return 0
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--update" {
		os.Exit(silentUpdate())
	}
	for _, a := range os.Args[1:] {
		if a == "--silent" || strings.EqualFold(a, "/S") || strings.EqualFold(a, "/silent") {
			os.Exit(silentInstall())
		}
	}
	win.EnableDPIAwareness()
	dpi := win.SystemDPI()
	scale := func(v int) int { return v * dpi / 96 }

	if h := win.FindWindow("LootnestSetupWindow"); h != 0 {
		win.Present(h)
		return
	}

	w := webview2.NewWithOptions(webview2.WebViewOptions{
		DataPath: filepath.Join(os.TempDir(), "LootnestSetup-WebView2"),
		WindowOptions: webview2.WindowOptions{
			Title: "Lootnest setup", Width: uint(scale(600)), Height: uint(scale(510)), IconId: 1, Center: true,
			ClassName: "LootnestSetupWindow", Background: bg,
			OnCreate: func(hwnd uintptr) { win.StyleTitleBar(hwnd, bg, border); win.SetWindowIcons(hwnd, dpi) },
		},
	})
	if w == nil {
		if win.MessageBox(0, "Lootnest needs the Microsoft Edge WebView2 Runtime, which isn't installed on this PC.\n\nOpen the download page now? Run this setup again afterwards.",
			"Lootnest setup", win.MB_YESNO|win.MB_ICONWARNING) == win.IDYES {
			_ = win.ShellOpen("https://developer.microsoft.com/microsoft-edge/webview2/")
		}
		os.Exit(1)
	}
	w.SetSize(scale(600), scale(472), webview2.HintFixed)
	defer os.RemoveAll(filepath.Join(os.TempDir(), "LootnestSetup-WebView2"))

	launch := false
	_ = w.Bind("setup_info", func() map[string]any {
		loc := win.InstalledLocation(app.UninstallID)
		legacy := loc == "" && app.HasLegacyInstall()
		return map[string]any{"version": Version, "installDir": installDir(), "existing": loc != "" || legacy, "legacy": legacy}
	})
	_ = w.Bind("setup_install", func(o installOpts) {
		go func() {
			send := func(ev map[string]any) {
				b, _ := json.Marshal(ev)
				w.Dispatch(func() { w.Eval("window.__setup(" + string(b) + ")") })
			}
			res := map[string]any{"type": "installed"}
			if err := install(o, func(n int) { send(map[string]any{"type": "step", "step": n}) }); err != nil {
				res["error"] = err.Error()
			}
			send(res)
		}()
	})
	_ = w.Bind("setup_close", func(open bool) {
		launch = open
		w.Dispatch(func() { w.Terminate() })
	})
	w.SetHtml(strings.Replace(page, "/*@SYSTEM*/", app.SystemCSS(), 1))
	win.Present(uintptr(w.Window()))
	w.Run()

	if launch {
		exe := filepath.Join(installDir(), app.Name+".exe")
		if _, err := os.Stat(exe); err == nil {
			_ = win.StartDetached(exe, "", filepath.Dir(exe))
		}
	}
}
