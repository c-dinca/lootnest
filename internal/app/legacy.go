//go:build windows

package app

import (
	"os"
	"path/filepath"
	"strings"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Moving over from PlaySwap, Lootnest's old name.
//
// PlaySwap kept its data in %LOCALAPPDATA%\PlaySwap and lived in
// %LOCALAPPDATA%\Programs\PlaySwap. Steam's launch options and the game
// shortcuts it made point at that PlaySwap.exe, so the installer leaves a
// copy of Lootnest.exe there until nothing uses it any more.
// ---------------------------------------------------------------------------

const (
	LegacyName        = "PlaySwap"
	LegacyWindowClass = "PlaySwapMainWindow"
	LegacyUninstallID = "PlaySwap"
)

func localAppData() string {
	if base := os.Getenv("LOCALAPPDATA"); base != "" {
		return base
	}
	return filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
}

// InstallDir is where the installer puts Lootnest.exe.
func InstallDir() string { return filepath.Join(localAppData(), "Programs", Name) }

// LegacyInstallDir is where PlaySwap was installed.
func LegacyInstallDir() string { return filepath.Join(localAppData(), "Programs", LegacyName) }

// DataDir is %LOCALAPPDATA%\Lootnest. Until PlaySwap's folder has been moved
// there (MigrateLegacyData), it is PlaySwap's folder.
func DataDir() string {
	dir := filepath.Join(localAppData(), Name)
	old := filepath.Join(localAppData(), LegacyName)
	if !dirExists(dir) && dirExists(old) {
		return old
	}
	return dir
}

// MigrateLegacyData moves %LOCALAPPDATA%\PlaySwap to %LOCALAPPDATA%\Lootnest
// in one rename, so nothing is copied or lost. If the folder is in use, it
// stays where it is (DataDir keeps using it) and the move is tried again on
// the next start.
func MigrateLegacyData() error {
	dir := filepath.Join(localAppData(), Name)
	old := filepath.Join(localAppData(), LegacyName)
	if dirExists(dir) || !dirExists(old) {
		return nil
	}
	if err := os.Rename(old, dir); err != nil {
		return err
	}
	if log := filepath.Join(dir, "playswap.log"); fileExists(log) && !fileExists(filepath.Join(dir, "lootnest.log")) {
		_ = os.Rename(log, filepath.Join(dir, "lootnest.log"))
	}
	return nil
}

// AppExe is the Lootnest.exe to point Steam and shortcuts at. When this
// process is the copy left at PlaySwap's old location, that is the installed
// Lootnest.exe, so the old location can be retired.
func AppExe() string {
	exe, _ := os.Executable()
	if strings.EqualFold(filepath.Base(filepath.Dir(exe)), LegacyName) {
		if cand := filepath.Join(InstallDir(), Name+".exe"); fileExists(cand) {
			return cand
		}
	}
	return exe
}

// cleanupLegacy replaces PlaySwap's game shortcuts with Lootnest ones and
// removes PlaySwap's program folder once no Steam launch option uses it.
func (a *App) cleanupLegacy() {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()

	// "<game> (PlaySwap).lnk" -> "<game> (Lootnest).lnk"
	suffix := " (" + strings.ToLower(LegacyName) + ").lnk"
	changed := false
	for _, s := range cfg.Shortcuts {
		base := strings.ToLower(filepath.Base(s))
		if !strings.HasSuffix(base, suffix) {
			continue
		}
		name := strings.TrimSuffix(base, suffix)
		for _, g := range cfg.Games {
			if strings.EqualFold(strings.TrimSpace(badFileChars.ReplaceAllString(g.Name, " ")), name) {
				if _, err := a.apiShortcut(g.ID); err != nil {
					a.logf("Couldn't replace the PlaySwap shortcut for %s: %v", g.Name, err)
				}
				break
			}
		}
		_ = os.Remove(s)
		a.logf("Removed old shortcut %s", s)
		changed = true
	}
	if changed {
		a.mu.Lock()
		live := a.cfg.Clone()
		var list []string
		for _, s := range live.Shortcuts {
			if !strings.HasSuffix(strings.ToLower(filepath.Base(s)), suffix) {
				list = append(list, s)
			}
		}
		live.Shortcuts = list
		a.mu.Unlock()
		_ = a.saveConfig(live)
	}

	// PlaySwap's program folder, once Steam no longer starts anything from it.
	old := LegacyInstallDir()
	exe, _ := os.Executable()
	if !dirExists(old) || strings.EqualFold(filepath.Dir(exe), old) {
		return
	}
	if a.env.SteamPath != "" && steamUsesDir(a.env.SteamPath, old) {
		return
	}
	if err := os.RemoveAll(old); err != nil {
		a.logf("Couldn't remove PlaySwap's old program folder %s: %v", old, err)
		return
	}
	a.logf("Removed PlaySwap's old program folder %s", old)
}

// steamUsesDir reports whether any Steam account's launch options mention dir.
func steamUsesDir(steamPath, dir string) bool {
	l := strings.ToLower(dir)
	escaped := strings.ReplaceAll(l, `\`, `\\`) // how localconfig.vdf stores it
	for _, acct := range core.SteamAccounts(steamPath) {
		b, err := os.ReadFile(filepath.Join(steamPath, "userdata", acct, "config", "localconfig.vdf"))
		if err != nil {
			continue
		}
		text := strings.ToLower(string(b))
		if strings.Contains(text, escaped) || strings.Contains(text, l) {
			return true
		}
	}
	return false
}

// removeLegacyInstall takes PlaySwap's Start menu entry, desktop shortcut,
// "Start with Windows" entry and Apps & features entry away (used by the
// installer when it replaces PlaySwap).
func removeLegacyInstall() {
	_ = os.Remove(filepath.Join(win.StartMenuPrograms(), LegacyName+".lnk"))
	_ = os.Remove(filepath.Join(win.Desktop(), LegacyName+".lnk"))
	_ = win.SetAutostart(LegacyName, "", false)
	win.UnregisterUninstall(LegacyUninstallID)
}

// ReplaceLegacy is the installer's half of the move: it removes PlaySwap's
// entries and, if PlaySwap's program folder exists, puts the new exe there
// as PlaySwap.exe so Steam launch options and old shortcuts keep working.
func ReplaceLegacy(payload []byte) error {
	removeLegacyInstall()
	old := LegacyInstallDir()
	if !dirExists(old) {
		return nil
	}
	exe := filepath.Join(old, LegacyName+".exe")
	tmp := exe + ".new"
	if err := os.WriteFile(tmp, payload, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp, exe); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// HasLegacyInstall reports whether PlaySwap is installed on this PC.
func HasLegacyInstall() bool {
	return win.InstalledLocation(LegacyUninstallID) != "" || dirExists(LegacyInstallDir())
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
