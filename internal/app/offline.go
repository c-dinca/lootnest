package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// ---------------------------------------------------------------------------
// Keep a game offline (core/offline.go): Windows Firewall rules for its
// programs, made as administrator after Windows asks the user. The result
// comes back as an "offline" event.
// ---------------------------------------------------------------------------

// offlineTarget is the game's folder and main program.
func (a *App) offlineTarget(key string) (name, dir, exe string, err error) {
	if id, ok := strings.CutPrefix(key, "game:"); ok {
		a.mu.Lock()
		g := a.cfg.Game(id)
		var launch core.Launch
		if g != nil {
			name, launch = g.Name, g.Launch
		}
		a.mu.Unlock()
		if g == nil {
			return "", "", "", errors.New("That game no longer exists.")
		}
		if launch.Type != "exe" {
			return "", "", "", errors.New("Steam starts this game, so Lootnest doesn't know its programs. Use Windows Firewall for Steam games.")
		}
		p, rerr := a.env.ResolveExe(launch.Exe)
		if rerr != nil {
			return "", "", "", rerr
		}
		return name, filepath.Dir(p), p, nil
	}
	f, ok := a.foundByKey(key)
	if !ok {
		return "", "", "", errFoundGone
	}
	if f.AppID > 0 || f.URI != "" {
		return "", "", "", errors.New("Its launcher starts this game, and keeps it up to date online. Lootnest keeps only games it starts itself offline.")
	}
	if f.Exe != "" {
		if p, rerr := a.env.ResolveExe(f.Exe); rerr == nil {
			exe = p
		}
	}
	dir = f.Dir
	if dir == "" && exe != "" {
		dir = filepath.Dir(exe)
	}
	if dir == "" {
		return "", "", "", errors.New("Lootnest doesn't know where this game is installed.")
	}
	return f.Name, dir, exe, nil
}

func (a *App) apiSetOffline(key string, on bool) error {
	name, dir, exe, err := a.offlineTarget(key)
	if err != nil {
		return err
	}
	var exes []string
	if on {
		if exes = core.GameExes(dir, exe); len(exes) == 0 {
			return errors.New("There's no program in the game's folder to keep offline.")
		}
		for i := range exes {
			exes[i] = plat.LongPath(exes[i]) // the firewall refuses short 8.3 names
		}
	}
	go func() {
		out := map[string]any{"key": key, "on": on}
		defer func() { a.emit("offline", out) }()
		if err := a.runFirewall(core.FirewallScript(core.OfflineTag(key), name, exes, on)); err != nil {
			out["error"] = err.Error()
			return
		}
		a.mu.Lock()
		cfg := a.cfg.Clone()
		a.mu.Unlock()
		if on {
			if cfg.Offline == nil {
				cfg.Offline = map[string][]string{}
			}
			cfg.Offline[key] = exes
		} else {
			delete(cfg.Offline, key)
		}
		if err := a.saveConfig(cfg); err != nil {
			out["error"] = err.Error()
			return
		}
		out["exes"] = exes
		if on {
			a.logf("%s is kept offline: Windows Firewall blocks %d of its programs", name, len(exes))
		} else {
			a.logf("%s can go online again", name)
		}
	}()
	return nil
}

// runFirewall runs a FirewallScript as administrator, after Windows asks
// the user. The error says what happened in the user's words.
func (a *App) runFirewall(script string) error {
	ps := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	args := "-NoProfile -NonInteractive -ExecutionPolicy Bypass -WindowStyle Hidden -EncodedCommand " + core.EncodePowerShell(script)
	code, err := plat.RunElevatedAndWait(a.hwnd, ps, args)
	switch {
	case errors.Is(err, plat.ErrCancelledByUser):
		return errors.New("Nothing changed: Windows' prompt was cancelled.")
	case err != nil:
		return errors.New("Windows Firewall couldn't be changed: " + err.Error())
	case code != 0:
		return fmt.Errorf("Windows Firewall didn't take the rules (code %d). Nothing is blocked.", code)
	}
	return nil
}

// offlineMoved keeps a game that was kept offline offline in the folder it
// was just moved to: one script, run once as administrator, takes away the
// rules for the old places (under the old key) and blocks the programs in
// the new folder (under the new key). If that can't be done, the game is
// no longer recorded as kept offline, and the answer says so.
func (a *App) offlineMoved(oldKey, newKey string) string {
	name, dir, exe, err := a.offlineTarget(newKey)
	var exes []string
	if err == nil {
		exes = core.GameExes(dir, exe)
	}
	if err == nil && len(exes) == 0 {
		err = errors.New("there's no program in its new folder to block")
	}
	for i := range exes {
		exes[i] = plat.LongPath(exes[i])
	}
	if err == nil {
		err = a.runFirewall(core.FirewallMoveScript(core.OfflineTag(oldKey), core.OfflineTag(newKey), name, exes))
	}
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	delete(cfg.Offline, oldKey)
	if err == nil {
		if cfg.Offline == nil {
			cfg.Offline = map[string][]string{}
		}
		cfg.Offline[newKey] = exes
	}
	if serr := a.saveConfig(cfg); serr != nil && err == nil {
		err = serr
	}
	if err != nil {
		a.logf("%s moved but isn't kept offline in its new folder: %v", name, err)
		return "Windows Firewall wasn't changed for its new folder, so it can go online. Keep it offline again from Manage."
	}
	a.logf("%s is kept offline in its new folder: Windows Firewall blocks %d of its programs", name, len(exes))
	return ""
}
