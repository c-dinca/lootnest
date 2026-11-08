package core

import (
	"strings"
)

// ScanSources are the places the Library's scan looks, in the order the
// Sources panel lists them.
var ScanSources = []string{"steam", "epic", "gog", "xbox", "ea", "ubisoft", "battlenet", "riot", "amazon", "rockstar", "itch", "hydra", "folder", "shortcut"}

// SourceCheck is what the last scan saw of one source: how many games it
// found there, whether its launcher is on this PC (game folders and
// shortcuts always are), and why it found none.
type SourceCheck struct {
	Source    string `json:"source"`
	Found     int    `json:"found"`
	Installed bool   `json:"installed"`
	Path      string `json:"path,omitempty"`
	State     string `json:"state"` // ok, info or warn, as a DoctorStep's
	Reason    string `json:"reason"`
}

// launcherSources: the launchers' own entries in Windows' installed
// programs (names as launcherOf returns them), by the source they serve.
var launcherSources = map[string]string{
	"battle.net": "battlenet", "blizzard battle.net": "battlenet", "blizzard app": "battlenet",
	"ubisoft connect": "ubisoft", "uplay": "ubisoft", "ea app": "ea", "ea desktop": "ea", "origin": "ea",
	"riot client": "riot", "rockstar games launcher": "rockstar", "amazon games": "amazon",
	"epic games launcher": "epic", "gog galaxy": "gog", "itch": "itch", "hydra": "hydra", "hydralauncher": "hydra",
}

// InstalledLaunchers maps each source whose launcher is in Windows'
// installed programs to its install folder, which may be "".
func InstalledLaunchers(entries []UninstallEntry) map[string]string {
	out := map[string]string{}
	for _, e := range entries {
		if src := launcherSources[launcherOf(e.DisplayName)]; src != "" && out[src] == "" {
			out[src] = strings.Trim(strings.TrimSpace(e.InstallLocation), `"`)
		}
	}
	return out
}

// SourceChecks counts the scan's games by source and says, for each source
// with none, why: not installed, installed with no games, or its library
// couldn't be read. installed holds the sources found on this PC (with the
// folder Lootnest read, if any); errs the ones whose library failed.
func SourceChecks(found []FoundGame, installed map[string]string, errs map[string]error) []SourceCheck {
	n := map[string]int{}
	for _, f := range found {
		n[f.Source]++
	}
	out := make([]SourceCheck, 0, len(ScanSources))
	for _, src := range ScanSources {
		path, ok := installed[src]
		c := SourceCheck{Source: src, Found: n[src], Installed: ok || n[src] > 0 || src == "folder" || src == "shortcut", Path: path, State: "info"}
		switch {
		case c.Found > 0:
			c.State = "ok"
		case errs[src] != nil:
			c.State, c.Reason = "warn", "Couldn't read its library: "+shortErr(errs[src])
		case src == "folder":
			c.Reason = "No folders that look like games on this PC's drives"
		case src == "shortcut":
			c.Reason = "No shortcuts to games on the desktop or in the Start menu"
		case !c.Installed:
			c.Reason = "Not installed"
		default:
			c.Reason = "Installed, no games yet"
		}
		out = append(out, c)
	}
	return out
}

// shortErr is an error's first line, cut to fit a row.
func shortErr(err error) string {
	s, _, _ := strings.Cut(strings.TrimSpace(err.Error()), "\n")
	if r := []rune(s); len(r) > 120 {
		s = string(r[:119]) + "…"
	}
	return s
}
