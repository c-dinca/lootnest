//go:build windows

package app

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// StopRunning asks a running Lootnest to quit and waits for it to go away.
// It also stops PlaySwap, Lootnest's old name, when that is still running.
func StopRunning() bool {
	for _, class := range []string{WindowClass, LegacyWindowClass} {
		h := win.FindWindow(class)
		if h == 0 {
			continue
		}
		win.SendCommand(h, "quit-now")
		for i := 0; i < 50 && win.FindWindow(class) != 0; i++ {
			time.Sleep(100 * time.Millisecond)
		}
		if win.FindWindow(class) != 0 {
			return false
		}
		time.Sleep(300 * time.Millisecond) // let the process exit fully
	}
	return true
}

// StartMenuShortcut is where the installer puts Lootnest's Start menu entry.
func StartMenuShortcut() string {
	return filepath.Join(win.StartMenuPrograms(), Name+".lnk")
}

func DesktopShortcut() string {
	return filepath.Join(win.Desktop(), Name+".lnk")
}

// Uninstall removes Lootnest from this PC (run as "Lootnest.exe --uninstall").
func Uninstall(quiet bool) {
	if !quiet && win.MessageBox(0, "Remove Lootnest from this PC?", Name+" uninstall", win.MB_YESNO|win.MB_ICONQUESTION) != win.IDYES {
		return
	}
	if !StopRunning() {
		win.MessageBox(0, "Lootnest is still running and didn't close. Quit it from the tray icon, then try again.", Name+" uninstall", win.MB_ICONWARNING)
		return
	}

	deleteData := false
	if !quiet {
		msg := "Also delete everyone's saved profiles and backups?\n\n" +
			"Yes: delete them.\nNo: keep them in " + DataDir() + " (Lootnest picks them up again if you reinstall).\n\n" +
			"Either way, each game's folders keep the files of whoever played last."
		deleteData = win.MessageBox(0, msg, Name+" uninstall", win.MB_YESNO|win.MB_ICONQUESTION|win.MB_DEFBUTTON2) == win.IDYES
	}

	// Take Lootnest back out of Steam's launch options, or games wouldn't start.
	restoreHooksForUninstall(quiet)

	// Shortcuts Lootnest created on the desktop for games.
	store := &core.Store{Dir: DataDir()}
	if cfg, first, err := store.LoadConfig(); err == nil && !first {
		for _, s := range cfg.Shortcuts {
			if l := strings.ToLower(s); strings.HasSuffix(l, " (lootnest).lnk") || strings.HasSuffix(l, " (playswap).lnk") {
				_ = os.Remove(s)
			}
		}
	}
	removeLegacyInstall()
	if old := LegacyInstallDir(); dirExists(old) {
		_ = os.RemoveAll(old) // the PlaySwap.exe copy left for old launch options
	}
	_ = os.Remove(StartMenuShortcut())
	_ = os.Remove(DesktopShortcut())
	_ = win.SetAutostart(Name, "", false)
	win.UnregisterUninstall(UninstallID)

	if deleteData {
		_ = os.RemoveAll(DataDir())
	} else {
		_ = os.RemoveAll(filepath.Join(DataDir(), "WebView2"))
	}

	if !quiet {
		win.MessageBox(0, "Lootnest was removed.", Name+" uninstall", win.MB_ICONINFORMATION)
	}
	// Remove the program folder right after this process exits.
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	if strings.EqualFold(filepath.Base(dir), Name) && strings.EqualFold(filepath.Base(exe), Name+".exe") {
		cmd := filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
		_ = win.StartHidden(cmd, `/d /c ping -n 3 127.0.0.1 >nul & rmdir /s /q "`+dir+`"`)
	}
}
