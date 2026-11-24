package core

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"
)

// Starting games on Linux (docs/ROADMAP.md, section 9, L3): Steam's own
// command, and Windows programs through the runner the user has, Wine or
// Proton through umu-launcher, in the game's Wine prefix. Lootnest brings
// no runner of its own; these say how to call the one on the PC.

// Runners a Windows program can start with (Settings.LinuxRunner).
const (
	RunnerAuto = ""
	RunnerWine = "wine"
	RunnerUmu  = "umu"
)

// ErrNoRunner is the answer when a Windows program has nothing to run it.
var ErrNoRunner = errors.New("Starting a Windows game needs Wine or umu-launcher: install one of them, then try again")

// SteamCommand is how to start Steam with arguments on this system: the
// Flatpak when that's where Steam lives, else the steam command, else nil
// (then a steam:// link is what's left).
func SteamCommand(steamPath string, look func(string) (string, error)) []string {
	if strings.Contains(filepath.ToSlash(steamPath), "/.var/app/com.valvesoftware.Steam/") {
		if p, err := look("flatpak"); err == nil {
			return []string{p, "run", "com.valvesoftware.Steam"}
		}
	}
	for _, name := range []string{"steam", "/usr/games/steam"} {
		if p, err := look(name); err == nil {
			return []string{p}
		}
	}
	return nil
}

// SteamRunURI starts a Steam game through Steam's link, with launch
// arguments when there are any (steam://run/<id>//<args>/).
func SteamRunURI(appID int, args string) string {
	id := strconv.Itoa(appID)
	if strings.TrimSpace(args) == "" {
		return "steam://rungameid/" + id
	}
	return "steam://run/" + id + "//" + strings.ReplaceAll(strings.TrimSpace(args), "/", "%2F") + "/"
}

// WindowsProgram tells a program that needs Wine on Linux.
func WindowsProgram(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".exe", ".bat", ".msi", ".lnk":
		return true
	}
	return false
}

// SplitArgs splits a command line into arguments: spaces separate them,
// double or single quotes keep spaces inside one, a backslash before a quote
// keeps the quote.
func SplitArgs(s string) []string {
	var out []string
	var cur strings.Builder
	in, quote, has := false, byte(0), false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\' && i+1 < len(s) && (s[i+1] == '"' || s[i+1] == '\''):
			cur.WriteByte(s[i+1])
			i++
			has = true
		case in && c == quote:
			in = false
		case !in && (c == '"' || c == '\''):
			in, quote, has = true, c, true
		case !in && (c == ' ' || c == '\t' || c == '\n'):
			if has {
				out = append(out, cur.String())
				cur.Reset()
				has = false
			}
		default:
			cur.WriteByte(c)
			has = true
		}
	}
	if has {
		out = append(out, cur.String())
	}
	return out
}

// RunCommand is a program to start on Linux: the command, its arguments and
// what goes into its environment.
type RunCommand struct {
	Argv []string
	Env  []string // NAME=value, added to Lootnest's own
}

// RunnerFor is how to start a program on Linux. A native one starts as is;
// a Windows one through runner (RunnerAuto takes umu-launcher when it's
// there, else Wine), with WINEPREFIX set to the game's prefix when it has
// one. umu-launcher gets GAMEID too: umu-<Steam app id> lets it apply that
// game's fixes, else "0".
func RunnerFor(exe string, args []string, prefix, runner string, steamID int, look func(string) (string, error)) (RunCommand, error) {
	if !WindowsProgram(exe) {
		return RunCommand{Argv: append([]string{exe}, args...)}, nil
	}
	find := func(names ...string) string {
		for _, n := range names {
			if p, err := look(n); err == nil {
				return p
			}
		}
		return ""
	}
	umu, wine := find("umu-run"), find("wine")
	var cmd RunCommand
	switch {
	case (runner == RunnerUmu || runner == RunnerAuto) && umu != "":
		id := "0"
		if steamID > 0 {
			id = "umu-" + strconv.Itoa(steamID)
		}
		cmd = RunCommand{Argv: append([]string{umu, exe}, args...), Env: []string{"GAMEID=" + id}}
	case (runner == RunnerWine || runner == RunnerAuto) && wine != "":
		launch := []string{wine}
		if strings.EqualFold(filepath.Ext(exe), ".msi") {
			launch = append(launch, "msiexec", "/i")
		} else if strings.EqualFold(filepath.Ext(exe), ".lnk") || strings.EqualFold(filepath.Ext(exe), ".bat") {
			launch = append(launch, "start", "/unix")
		}
		cmd = RunCommand{Argv: append(append(launch, exe), args...)}
	default:
		if runner == RunnerUmu {
			return RunCommand{}, errors.New("umu-launcher (umu-run) isn't installed")
		}
		if runner == RunnerWine {
			return RunCommand{}, errors.New("Wine isn't installed")
		}
		return RunCommand{}, ErrNoRunner
	}
	if prefix != "" {
		cmd.Env = append(cmd.Env, "WINEPREFIX="+prefix)
	}
	return cmd, nil
}

// DefaultWinePrefix is Wine's own prefix for programs started without one:
// $WINEPREFIX, else ~/.wine, when it's there.
func (e *Env) DefaultWinePrefix() string {
	if e.Getenv != nil {
		if p := e.Getenv("WINEPREFIX"); filepath.IsAbs(p) && IsPrefix(p) {
			return filepath.Clean(p)
		}
	}
	if e.Home != "" {
		if p := filepath.Join(e.Home, ".wine"); IsPrefix(p) {
			return p
		}
	}
	return ""
}
