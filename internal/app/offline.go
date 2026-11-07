//go:build windows

package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lootnest/internal/core"
	"lootnest/internal/win"
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
	}
	go func() {
		out := map[string]any{"key": key, "on": on}
		defer func() { a.emit("offline", out) }()
		ps := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
		script := core.FirewallScript(core.OfflineTag(key), name, exes, on)
		args := "-NoProfile -NonInteractive -ExecutionPolicy Bypass -WindowStyle Hidden -EncodedCommand " + core.EncodePowerShell(script)
		code, err := win.RunElevatedAndWait(a.hwnd, ps, args)
		switch {
		case errors.Is(err, win.ErrCancelledByUser):
			out["error"] = "Nothing changed: Windows' prompt was cancelled."
			return
		case err != nil:
			out["error"] = "Windows Firewall couldn't be changed: " + err.Error()
			return
		case code != 0:
			out["error"] = fmt.Sprintf("Windows Firewall didn't take the rules (code %d). Nothing is blocked.", code)
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
