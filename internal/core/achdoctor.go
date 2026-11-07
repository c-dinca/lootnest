package core

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Achievement Doctor: why a game shows no achievements, step by step. It
// only reads: which Steam game it is, the tool next to it, the file that
// tool keeps, and where the names come from. The one change it can make,
// when asked, is the name a Goldberg-style tool shows in the game.
// ---------------------------------------------------------------------------

// DoctorStep is one line of the check.
type DoctorStep struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
	State  string `json:"state"` // ok, warn, fail, info
	Path   string `json:"path,omitempty"`
}

// ToolInfo is the Steam-compatible tool found next to a game.
type ToolInfo struct {
	Name     string `json:"name"`
	Reads    bool   `json:"reads"`              // Lootnest reads its achievement files
	Settings string `json:"settings,omitempty"` // a Goldberg-style steam_settings folder
}

var toolMarkers = []struct {
	file  string
	name  string
	reads bool
}{
	{"steam_settings", "Goldberg (GSE)", true},
	{"onlinefix.ini", "OnlineFix", true},
	{"onlinefix64.dll", "OnlineFix", true},
	{"cream_api.ini", "CreamAPI", true},
	{"smartsteamemu.ini", "SmartSteamEmu", false},
	{"ali213.ini", "ALI213", false},
	{"3dmgame.ini", "3DM", false},
	{"rld.ini", "RLD!", true},
	{"valve.ini", "RLD!", true},
	{"steam_emu.ini", "", true}, // read to tell CODEX, RUNE and the rest apart
}

// DetectTool looks for a Steam-compatible tool next to a game: in its
// folder and a few levels down (Unreal games keep theirs in
// <Game>\Binaries\Win64). Nil when there's none.
func DetectTool(dir string) *ToolInfo {
	if dir == "" {
		return nil
	}
	root := filepath.Clean(dir)
	var found *ToolInfo
	seen := 0
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		seen++
		if found != nil || seen > 20000 {
			return filepath.SkipAll
		}
		if err != nil {
			return filepath.SkipDir
		}
		name := strings.ToLower(d.Name())
		if d.IsDir() && p != root {
			if name == "steam_settings" {
				found = &ToolInfo{Name: "Goldberg (GSE)", Reads: true, Settings: p}
				return filepath.SkipAll
			}
			if skipForAppID[name] {
				return filepath.SkipDir
			}
			if rel, _ := filepath.Rel(root, p); strings.Count(rel, string(filepath.Separator)) >= 3 {
				return filepath.SkipDir
			}
			return nil
		}
		for _, m := range toolMarkers {
			if name != m.file {
				continue
			}
			t := &ToolInfo{Name: m.name, Reads: m.reads}
			if name == "steam_emu.ini" {
				b, _ := os.ReadFile(p)
				up := strings.ToUpper(string(b))
				switch {
				case strings.Contains(up, "CODEX"):
					t.Name = "CODEX"
				case strings.Contains(up, "RUNE"):
					t.Name = "RUNE"
				case strings.Contains(up, "SKIDROW"):
					t.Name = "SKIDROW"
				case strings.Contains(up, "EMPRESS"):
					t.Name = "EMPRESS"
				default:
					t.Name = "A Steam emulator (steam_emu.ini)"
				}
			}
			found = t
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// AchFileInfo is an achievement file a tool keeps for a game on this PC.
type AchFileInfo struct {
	Path     string    `json:"path"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
	Unlocked int       `json:"unlocked"`
}

// AchievementFiles lists every place a tool keeps a game's achievements,
// and which of them are there.
func (e *Env) AchievementFiles(appID int) (looked []string, there []AchFileInfo) {
	if appID <= 0 {
		return nil, nil
	}
	for _, f := range achFiles {
		b := e.achBase(f.base)
		if b == "" {
			continue
		}
		p := filepath.Join(b, achPath(f.path, appID))
		looked = append(looked, p)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			info := AchFileInfo{Path: p, Size: st.Size(), Modified: st.ModTime()}
			if raw, err := os.ReadFile(p); err == nil {
				info.Unlocked = len(ParseUnlocks(raw, f.format))
			}
			there = append(there, info)
		}
	}
	return looked, there
}

// ------------------------------------------------- the name in the game ---

var accountNameRe = regexp.MustCompile(`(?im)^\s*account_name\s*=\s*(.*?)\s*$`)

// ToolAccountName is the name a Goldberg-style tool shows in the game,
// from its steam_settings folder ("" when it sets none there).
func ToolAccountName(settings string) string {
	if settings == "" {
		return ""
	}
	if b, err := os.ReadFile(filepath.Join(settings, "configs.user.ini")); err == nil {
		if m := accountNameRe.FindSubmatch(b); m != nil {
			return string(m[1])
		}
	}
	if b, err := os.ReadFile(filepath.Join(settings, "force_account_name.txt")); err == nil {
		return strings.TrimSpace(string(b))
	}
	return ""
}

// cleanAccountName keeps a name a tool's settings file can hold: one line,
// at most 32 characters.
func cleanAccountName(n string) string {
	n = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || r == '[' || r == ']' || r == '=' {
			return -1
		}
		return r
	}, strings.TrimSpace(n))
	if r := []rune(n); len(r) > 32 {
		n = string(r[:32])
	}
	return strings.TrimSpace(n)
}

// SetToolAccountName sets the name a Goldberg-style tool shows in the game
// (and to friends on LAN) in its steam_settings folder: account_name in
// configs.user.ini ([user::general]), and force_account_name.txt where an
// older version uses it. The files it changes are copied to keep first.
func SetToolAccountName(settings, name, keep string) error {
	name = cleanAccountName(name)
	if settings == "" || name == "" {
		return os.ErrInvalid
	}
	ini := filepath.Join(settings, "configs.user.ini")
	legacy := filepath.Join(settings, "force_account_name.txt")
	for _, p := range []string{ini, legacy} {
		if fileThere(p) {
			if err := Mirror(p, filepath.Join(keep, filepath.Base(p))); err != nil {
				return err
			}
		}
	}
	b, _ := os.ReadFile(ini)
	s := string(b)
	switch {
	case accountNameRe.MatchString(s):
		s = accountNameRe.ReplaceAllLiteralString(s, "account_name="+name)
	case regexp.MustCompile(`(?im)^\s*\[user::general\]\s*$`).MatchString(s):
		s = regexp.MustCompile(`(?im)^(\s*\[user::general\]\s*)$`).ReplaceAllString(s, "${1}\naccount_name="+escapeDollar(name))
	default:
		if s != "" && !strings.HasSuffix(s, "\n") {
			s += "\n"
		}
		s += "[user::general]\naccount_name=" + name + "\n"
	}
	if err := writeFileAtomic(ini, []byte(s)); err != nil {
		return err
	}
	if fileThere(legacy) {
		return writeFileAtomic(legacy, []byte(name))
	}
	return nil
}

func escapeDollar(s string) string { return strings.ReplaceAll(s, "$", "$$") }

func fileThere(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
