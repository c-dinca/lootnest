package app

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// StopRunning asks a running Lootnest to quit and waits for it to go away.
// It also stops PlaySwap, Lootnest's old name, when that is still running.
func StopRunning() bool {
	for _, class := range []string{WindowClass, LegacyWindowClass} {
		h := plat.FindWindow(class)
		if h == 0 {
			continue
		}
		plat.SendCommand(h, "quit-now")
		for i := 0; i < 50 && plat.FindWindow(class) != 0; i++ {
			time.Sleep(100 * time.Millisecond)
		}
		if plat.FindWindow(class) != 0 {
			return false
		}
		time.Sleep(300 * time.Millisecond) // let the process exit fully
	}
	return true
}

// StartMenuShortcut is where the installer puts Lootnest's Start menu entry.
func StartMenuShortcut() string {
	return filepath.Join(plat.StartMenuPrograms(), Name+".lnk")
}

func DesktopShortcut() string {
	return filepath.Join(plat.Desktop(), Name+".lnk")
}

// Uninstall removes Lootnest from this PC (run as "Lootnest.exe --uninstall").
func Uninstall(quiet bool) {
	if !quiet && plat.MessageBox(0, "Remove Lootnest from this PC?", Name+" uninstall", plat.MB_YESNO|plat.MB_ICONQUESTION) != plat.IDYES {
		return
	}
	if !StopRunning() {
		plat.MessageBox(0, "Lootnest is still running and didn't close. Quit it from the tray icon, then try again.", Name+" uninstall", plat.MB_ICONWARNING)
		return
	}

	deleteData := false
	if !quiet {
		msg := "Also delete everyone's saved profiles and backups?\n\n" +
			"Yes: delete them.\nNo: keep them in " + DataDir() + " (Lootnest picks them up again if you reinstall).\n\n" +
			"Either way, each game's folders keep the files of whoever played last."
		deleteData = plat.MessageBox(0, msg, Name+" uninstall", plat.MB_YESNO|plat.MB_ICONQUESTION|plat.MB_DEFBUTTON2) == plat.IDYES
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
	_ = plat.SetAutostart(Name, "", false)
	plat.UnregisterUninstall(UninstallID)

	if deleteData {
		_ = os.RemoveAll(DataDir())
	} else {
		_ = os.RemoveAll(filepath.Join(DataDir(), "WebView2"))
	}

	if !quiet {
		plat.MessageBox(0, "Lootnest was removed.", Name+" uninstall", plat.MB_ICONINFORMATION)
	}
	// Remove the program folder right after this process exits.
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	if strings.EqualFold(filepath.Base(dir), Name) && strings.EqualFold(filepath.Base(exe), Name+".exe") {
		cmd := filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
		_ = plat.StartHidden(cmd, `/d /c ping -n 3 127.0.0.1 >nul & rmdir /s /q "`+dir+`"`)
	}
}
