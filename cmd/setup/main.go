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

const bg = 0x151A24

func init() { runtime.LockOSThread() }

func installDir() string { return app.InstallDir() }

type installOpts struct {
	Desktop   bool `json:"desktop"`
	Autostart bool `json:"autostart"`
}

func install(o installOpts) error {
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
	// Replacing PlaySwap (the old name): Steam launch options still point at
	// PlaySwap.exe until Lootnest rewrites them, so a copy stays there.
	if err := app.ReplaceLegacy(payload); err != nil {
		return fmt.Errorf("couldn't update PlaySwap's old program folder: %w", err)
	}
	return win.RegisterUninstall(app.UninstallID, win.UninstallEntry{
		DisplayName:          app.Name,
		DisplayVersion:       Version,
		Publisher:            app.Name,
		DisplayIcon:          exe + ",0",
		InstallLocation:      dir,
		UninstallString:      fmt.Sprintf(`"%s" --uninstall`, exe),
		QuietUninstallString: fmt.Sprintf(`"%s" --uninstall --quiet`, exe),
		EstimatedSizeKB:      uint32(len(payload) / 1024),
	})
}

func main() {
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
			OnCreate: func(hwnd uintptr) { win.StyleTitleBar(hwnd, bg); win.SetWindowIcons(hwnd, dpi) },
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
			res := map[string]any{"type": "installed"}
			if err := install(o); err != nil {
				res["error"] = err.Error()
			}
			b, _ := json.Marshal(res)
			w.Dispatch(func() { w.Eval("window.__setup(" + string(b) + ")") })
		}()
	})
	_ = w.Bind("setup_close", func(open bool) {
		launch = open
		w.Dispatch(func() { w.Terminate() })
	})
	w.SetHtml(page)
	win.Present(uintptr(w.Window()))
	w.Run()

	if launch {
		exe := filepath.Join(installDir(), app.Name+".exe")
		if _, err := os.Stat(exe); err == nil {
			_ = win.StartDetached(exe, "", filepath.Dir(exe))
		}
	}
}
