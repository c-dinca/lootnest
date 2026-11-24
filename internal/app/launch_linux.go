package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// steamPlay starts a Steam game: Steam's own command (the Flatpak's when
// Steam is one) with -applaunch, else Steam's link.
func (a *App) steamPlay(appID int, args string) error {
	if cmd := core.SteamCommand(a.env.SteamPath, exec.LookPath); cmd != nil {
		argv := append(append(cmd[1:], "-applaunch", strconv.Itoa(appID)), core.SplitArgs(args)...)
		return startLinux(cmd[0], argv, a.env.SteamPath, nil)
	}
	return plat.ShellOpen(core.SteamRunURI(appID, args))
}

// runProgram starts a program for the game key. A Windows program runs
// through the runner chosen in Settings (core.RunnerFor), in the game's
// Wine prefix; Linux has no "as administrator" for a game.
func (a *App) runProgram(key, exe, args, dir string, admin bool) error {
	if admin {
		a.logf("Run as administrator isn't used on Linux; starting %s as you", filepath.Base(exe))
	}
	prefix, steamID := a.prefixRootOf(key), a.steamIDOf(key)
	a.mu.Lock()
	runner := a.cfg.Settings.LinuxRunner
	a.mu.Unlock()
	rc, err := core.RunnerFor(exe, core.SplitArgs(args), prefix, runner, steamID, exec.LookPath)
	if err != nil {
		return err
	}
	if rc.Argv[0] != exe {
		where := "Wine's own prefix"
		if prefix != "" {
			where = prefix
		}
		a.logf("Starting %s with %s, in %s", filepath.Base(exe), filepath.Base(rc.Argv[0]), where)
	}
	return startLinux(rc.Argv[0], rc.Argv[1:], dir, rc.Env)
}

// prefixRootOf is the Wine prefix of the game key on this PC, or "".
func (a *App) prefixRootOf(key string) string {
	stored := a.storedPrefix(key)
	if stored == "" {
		return ""
	}
	if e, err := a.env.For(stored); err == nil && e.Prefix != nil {
		return e.Prefix.Root
	}
	return ""
}

// steamIDOf is the game key's Steam app id, when it has one (for
// umu-launcher's fixes).
func (a *App) steamIDOf(key string) int {
	if id, ok := strings.CutPrefix(key, "game:"); ok {
		a.mu.Lock()
		defer a.mu.Unlock()
		if g := a.cfg.Game(id); g != nil {
			return core.GameAppID(g)
		}
		return 0
	}
	if f, ok := a.foundByKey(key); ok {
		return max(f.AppID, f.SteamID)
	}
	return 0
}

// startLinux starts a program in its own session, so it outlives
// Lootnest, and reaps it when it ends.
func startLinux(name string, args []string, dir string, env []string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
