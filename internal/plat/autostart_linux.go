package plat

import (
	"os"
	"path/filepath"
	"strings"

	"lootnest/internal/core"
)

// Starting with the desktop: a .desktop file in ~/.config/autostart (the
// XDG Autostart specification).

func autostartFile(name string) string {
	return filepath.Join(core.XDGConfigHome(home(), os.Getenv), "autostart", strings.ToLower(name)+".desktop")
}

// Autostart is the command that starts with the desktop, if any.
func Autostart(name string) (string, bool) {
	b, err := os.ReadFile(autostartFile(name))
	if err != nil {
		return "", false
	}
	on := true
	cmd := ""
	for _, line := range strings.Split(string(b), "\n") {
		k, v, _ := strings.Cut(strings.TrimSpace(line), "=")
		switch k {
		case "Exec":
			cmd = v
		case "Hidden", "X-GNOME-Autostart-enabled":
			if (k == "Hidden" && v == "true") || (k == "X-GNOME-Autostart-enabled" && v == "false") {
				on = false
			}
		}
	}
	return cmd, on && cmd != ""
}

// SetAutostart starts command with the desktop, or stops it.
func SetAutostart(name, command string, on bool) error {
	f := autostartFile(name)
	if !on {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
		return err
	}
	body := "[Desktop Entry]\nType=Application\nName=" + name + "\nExec=" + command +
		"\nIcon=" + iconPath() + "\nTerminal=false\nX-GNOME-Autostart-enabled=true\n"
	tmp := f + ".tmp"
	if err := os.WriteFile(tmp, []byte(body), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, f)
}
