package core

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Where things are on a Linux system, for the Linux Lootnest (docs/
// ROADMAP.md, section 9). These take the home folder and the environment
// as arguments, so they're tested on any system.

// XDGDataHome is $XDG_DATA_HOME, or ~/.local/share (the XDG Base
// Directory spec).
func XDGDataHome(home string, getenv func(string) string) string {
	return xdgBase(home, getenv, "XDG_DATA_HOME", filepath.Join(".local", "share"))
}

// XDGConfigHome is $XDG_CONFIG_HOME, or ~/.config.
func XDGConfigHome(home string, getenv func(string) string) string {
	return xdgBase(home, getenv, "XDG_CONFIG_HOME", ".config")
}

func xdgBase(home string, getenv func(string) string, name, def string) string {
	if getenv != nil {
		// The spec ignores a relative path.
		if v := getenv(name); filepath.IsAbs(v) {
			return filepath.Clean(v)
		}
	}
	if home == "" {
		return ""
	}
	return filepath.Join(home, def)
}

// xdgUserDefaults are the folders xdg-user-dirs makes when user-dirs.dirs
// doesn't say.
var xdgUserDefaults = map[string]string{
	"DESKTOP": "Desktop", "DOCUMENTS": "Documents", "DOWNLOAD": "Downloads",
	"MUSIC": "Music", "PICTURES": "Pictures", "VIDEOS": "Videos",
}

// XDGUserDir is one of the user's folders (DESKTOP, DOCUMENTS, DOWNLOAD,
// MUSIC, PICTURES, VIDEOS): from $XDG_<NAME>_DIR, then user-dirs.dirs in
// the config folder (where a desktop in another language names them), then
// the English default in the home folder.
func XDGUserDir(home string, getenv func(string) string, name string) string {
	key := "XDG_" + strings.ToUpper(name) + "_DIR"
	if getenv != nil {
		if v := getenv(key); filepath.IsAbs(v) {
			return filepath.Clean(v)
		}
	}
	if f, err := os.Open(filepath.Join(XDGConfigHome(home, getenv), "user-dirs.dirs")); err == nil {
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			k, v, ok := strings.Cut(line, "=")
			if !ok || strings.TrimSpace(k) != key {
				continue
			}
			v = strings.Trim(strings.TrimSpace(v), `"`)
			switch {
			case v == "$HOME" || v == "$HOME/":
				return "" // xdg-user-dirs turns a folder off by pointing it at home
			case strings.HasPrefix(v, "$HOME/"):
				return filepath.Join(home, filepath.FromSlash(v[len("$HOME/"):]))
			case filepath.IsAbs(v):
				return filepath.Clean(v)
			}
		}
	}
	if d, ok := xdgUserDefaults[strings.ToUpper(name)]; ok && home != "" {
		return filepath.Join(home, d)
	}
	return ""
}

// SteamLinuxPaths are where Steam keeps its folder on Linux, most usual
// first: the native package, the links it makes in ~/.steam, the Flatpak
// and the Snap.
func SteamLinuxPaths(home string) []string {
	if home == "" {
		return nil
	}
	return []string{
		filepath.Join(home, ".local", "share", "Steam"),
		filepath.Join(home, ".steam", "steam"),
		filepath.Join(home, ".steam", "root"),
		filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam"),
		filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", "data", "Steam"),
		filepath.Join(home, "snap", "steam", "common", ".local", "share", "Steam"),
	}
}

// FindSteamLinux is Steam's folder on this Linux system (the real folder,
// not the ~/.steam link to it), or "".
func FindSteamLinux(home string) string {
	for _, p := range SteamLinuxPaths(home) {
		if !isDir(filepath.Join(p, "steamapps")) {
			continue
		}
		if real, err := filepath.EvalSymlinks(p); err == nil {
			return real
		}
		return p
	}
	return ""
}

// NewLinuxEnv is the Env of a Linux system: native paths with %HOME% as
// their token, Steam where it's installed, and the system's and the home's
// own folders out of reach of any game path. Windows games' paths are read
// in their prefix (Env.For).
func NewLinuxEnv(home string, getenv func(string) string, dataDir string) *Env {
	home = filepath.Clean(home)
	steam := FindSteamLinux(home)
	e := &Env{
		Getenv:    getenv,
		Home:      home,
		Documents: XDGUserDir(home, getenv, "DOCUMENTS"),
		SteamPath: steam,
		DataDir:   dataDir,
		OS:        "linux",
		SteamUser: func() (string, error) { return SteamUserDir(steam, "", 0) },
	}
	for _, p := range []string{
		XDGDataHome(home, getenv), XDGConfigHome(home, getenv),
		filepath.Join(home, ".local"), filepath.Join(home, ".cache"), filepath.Join(home, ".steam"),
		filepath.Join(home, ".var"), filepath.Join(home, ".var", "app"), filepath.Join(home, "snap"),
		XDGUserDir(home, getenv, "DESKTOP"), XDGUserDir(home, getenv, "DOWNLOAD"),
		XDGUserDir(home, getenv, "PICTURES"), XDGUserDir(home, getenv, "VIDEOS"), XDGUserDir(home, getenv, "MUSIC"),
		filepath.Join(home, "Games"),
		"/home", "/opt", "/mnt", "/media", "/run/media",
	} {
		if p != "" {
			e.ExtraProtected = append(e.ExtraProtected, p)
		}
	}
	e.NeverInside = []string{"/usr", "/etc", "/bin", "/sbin", "/lib", "/lib64", "/boot", "/dev", "/proc", "/sys", "/run/user"}
	return e
}
