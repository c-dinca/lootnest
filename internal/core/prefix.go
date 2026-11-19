package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// A Windows game on Linux runs through Wine or Proton, and keeps its saves
// inside a Wine prefix: a folder holding drive_c, which the game sees as
// C:\. Its %APPDATA% is drive_c/users/<user>/AppData/Roaming, so every
// Windows path Lootnest stores or suggests for that game is read inside the
// game's own prefix (docs/ROADMAP.md, section 9).

// Prefix is a Wine prefix and the Windows user games in it run as.
type Prefix struct {
	Root string // the folder that holds drive_c
	User string // the user folder under drive_c/users ("steamuser" for Proton)
}

// ProtonUser is the Windows user Proton runs every game as.
const ProtonUser = "steamuser"

// Drive is the prefix's C:\.
func (p Prefix) Drive() string { return filepath.Join(p.Root, "drive_c") }

// UserDir is the prefix's C:\Users\<user>.
func (p Prefix) UserDir() string { return filepath.Join(p.Drive(), "users", p.User) }

// IsPrefix tells a folder that is a Wine prefix: drive_c with Windows'
// folder or Wine's registry beside it.
func IsPrefix(dir string) bool {
	if dir == "" || !isDir(filepath.Join(dir, "drive_c")) {
		return false
	}
	return isFile(filepath.Join(dir, "system.reg")) || isDir(filepath.Join(dir, "drive_c", "windows"))
}

// OpenPrefix reads which user a prefix's games run as: Proton's steamuser
// when it's there, else the one user folder besides Public, else fallback
// (Wine names it after the Linux user).
func OpenPrefix(root, fallback string) Prefix {
	p := Prefix{Root: filepath.Clean(root), User: fallback}
	users := filepath.Join(p.Drive(), "users")
	if isDir(filepath.Join(users, ProtonUser)) {
		p.User = ProtonUser
		return p
	}
	ents, err := os.ReadDir(users)
	if err != nil {
		return p
	}
	var found []string
	for _, e := range ents {
		if e.IsDir() && !strings.EqualFold(e.Name(), "Public") && !strings.EqualFold(e.Name(), "Default") {
			found = append(found, e.Name())
		}
	}
	if len(found) == 1 || (p.User == "" && len(found) > 0) {
		p.User = found[0]
	}
	return p
}

// ProtonPrefix is the prefix Steam made for a game it runs through Proton,
// in the same library as the game (steamapps/compatdata/<appid>/pfx), or
// "" for a native game Proton never ran.
func ProtonPrefix(installDir string, appID int) string {
	if installDir == "" || appID <= 0 {
		return ""
	}
	apps := filepath.Dir(filepath.Dir(filepath.Clean(installDir))) // <lib>/steamapps
	p := filepath.Join(apps, "compatdata", strconv.Itoa(appID), "pfx")
	if IsPrefix(p) {
		return p
	}
	return ""
}

// InPrefix is this Env as a Windows game in p sees it: %APPDATA% and the
// other Windows folders are inside the prefix, and the Linux home, the
// prefix's own folders and Windows' folders in it can never be a game path.
// Steam's folder stays this system's, since Steam keeps userdata outside
// the prefix.
func (e *Env) InPrefix(p Prefix) *Env {
	ne := *e
	ne.OS = ""
	ne.Prefix = &p
	user := p.UserDir()
	drive := p.Drive()
	appData := filepath.Join(user, "AppData")
	docs := filepath.Join(user, "Documents")
	if !isDir(docs) && isDir(filepath.Join(user, "My Documents")) {
		docs = filepath.Join(user, "My Documents") // Wine before 5.0
	}
	vars := map[string]string{
		"APPDATA":           filepath.Join(appData, "Roaming"),
		"LOCALAPPDATA":      filepath.Join(appData, "Local"),
		"USERPROFILE":       user,
		"USERNAME":          p.User,
		"HOMEDRIVE":         drive,
		"SYSTEMDRIVE":       drive,
		"PUBLIC":            filepath.Join(drive, "users", "Public"),
		"PROGRAMDATA":       filepath.Join(drive, "ProgramData"),
		"ALLUSERSPROFILE":   filepath.Join(drive, "ProgramData"),
		"PROGRAMFILES":      filepath.Join(drive, "Program Files"),
		"PROGRAMFILES(X86)": filepath.Join(drive, "Program Files (x86)"),
		"WINDIR":            filepath.Join(drive, "windows"),
		"SYSTEMROOT":        filepath.Join(drive, "windows"),
		"TEMP":              filepath.Join(appData, "Local", "Temp"),
		"TMP":               filepath.Join(appData, "Local", "Temp"),
	}
	base := e.Getenv
	ne.Getenv = func(name string) string {
		if v, ok := vars[strings.ToUpper(name)]; ok {
			return v
		}
		if base != nil {
			return base(name)
		}
		return ""
	}
	ne.Home = user
	ne.Documents = docs
	ne.SavedGames = filepath.Join(user, "Saved Games")
	ne.LocalLow = filepath.Join(appData, "LocalLow")
	ne.ExtraProtected = append(append([]string(nil), e.ExtraProtected...),
		p.Root, drive, filepath.Join(drive, "users"), vars["PROGRAMFILES"], vars["PROGRAMFILES(X86)"], vars["WINDIR"])
	if e.Home != "" {
		ne.ExtraProtected = append(ne.ExtraProtected, e.Home)
	}
	return &ne
}

// For is the Env a game's paths are read in: this one, or inside the
// game's prefix when it has one (Game.Prefix, FoundGame.Prefix, stored
// with this Env's tokens).
func (e *Env) For(prefix string) (*Env, error) {
	if strings.TrimSpace(prefix) == "" {
		return e, nil
	}
	root, err := e.expand(prefix)
	if err != nil {
		return nil, fmt.Errorf("the game's Wine prefix: %w", err)
	}
	if !IsPrefix(root) {
		return nil, errors.New("the game's Wine prefix (" + root + ") isn't there any more")
	}
	user := ""
	if e.Getenv != nil {
		user = e.Getenv("USER")
	}
	return e.InPrefix(OpenPrefix(root, user)), nil
}

// matchCase finds p's folders and files as they're spelled inside root.
// Windows doesn't mind case and Wine follows it, so a path stored on
// Windows as AppData\Roaming\Game\Saves is .../AppData/Roaming/game/saves
// here when that's what the game made. A part that doesn't exist yet keeps
// the spelling given.
func matchCase(p, root string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return p
	}
	cur := root
	parts := strings.Split(rel, string(filepath.Separator))
	for i, part := range parts {
		next := filepath.Join(cur, part)
		if _, err := os.Lstat(next); err == nil {
			cur = next
			continue
		}
		ents, err := os.ReadDir(cur)
		found := ""
		if err == nil {
			for _, e := range ents {
				if strings.EqualFold(e.Name(), part) {
					found = e.Name()
					break
				}
			}
		}
		if found == "" {
			return filepath.Join(append([]string{cur}, parts[i:]...)...)
		}
		cur = filepath.Join(cur, found)
	}
	return cur
}
